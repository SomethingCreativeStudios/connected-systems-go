package mqtt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"

	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

// Mochi 2.7.9 force-replays all inflights on reconnect and removes a queued
// publication immediately after sending it through its native quota path. Keep
// its outgoing QoS state machine, but enforce the peer's receive window at the
// connection boundary. Queued IDs reference the existing durable inflight data;
// this gate keeps no second payload queue and does not implement an outbox.
type flowConn struct {
	net.Conn
	manager *Manager
	mu      sync.Mutex
	wireMu  sync.Mutex
	client  *mochi.Client
	limit   int
	active  map[uint16]bool
	queued  map[uint16]bool
	queue   []uint16
	closed  bool
}

func (m *Manager) flowConnection(conn net.Conn) net.Conn {
	return &flowConn{Conn: conn, manager: m, active: make(map[uint16]bool), queued: make(map[uint16]bool)}
}

func (c *flowConn) bind(cl *mochi.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.client = cl
	c.limit = int(cl.Properties.Props.ReceiveMaximum)
	if c.limit == 0 || c.limit > MaxQueuedMessages {
		c.limit = MaxQueuedMessages
	}
	for _, pk := range cl.State.Inflight.GetAll(false) {
		if pk.FixedHeader.Type == packets.Pubrel {
			c.active[pk.PacketID] = true
		}
	}
	// Incoming creates have independent receive accounting. Disable only the
	// native quota paths; the advertised receive maximum remains 1,000.
	cl.State.Inflight.ResetSendQuota(0)
	cl.State.Inflight.ResetReceiveQuota(1 << 30)
}

func publicationID(frame []byte) (uint16, error) {
	offset := 1
	for {
		if offset >= len(frame) {
			return 0, errors.New("short MQTT frame")
		}
		b := frame[offset]
		offset++
		if b < 128 {
			break
		}
	}
	if offset+2 > len(frame) {
		return 0, errors.New("short MQTT topic")
	}
	length := int(binary.BigEndian.Uint16(frame[offset : offset+2]))
	offset += 2 + length
	if offset+2 > len(frame) {
		return 0, errors.New("short MQTT packet identifier")
	}
	return binary.BigEndian.Uint16(frame[offset : offset+2]), nil
}

func (c *flowConn) writeFrame(frame []byte) (int, error) {
	c.wireMu.Lock()
	defer c.wireMu.Unlock()
	n := 0
	for n < len(frame) {
		written, err := c.Conn.Write(frame[n:])
		n += written
		if err != nil {
			return n, err
		}
		if written == 0 {
			return n, errors.New("empty MQTT socket write")
		}
	}
	return n, nil
}

// frameSize returns the length of the complete MQTT packet at the start of b.
func frameSize(b []byte) (int, error) {
	length, multiplier := 0, 1
	for i := 1; i < len(b) && i <= 4; i++ {
		length += int(b[i]&127) * multiplier
		if b[i] < 128 {
			if size := 1 + i + length; size <= len(b) {
				return size, nil
			}
			break
		}
		multiplier *= 128
	}
	return 0, errors.New("incomplete MQTT frame")
}

// Write gates every QoS 1/2 PUBLISH in b. Mochi coalesces queued packets into
// one write when its outbound queue backs up, so b can hold several frames.
func (c *flowConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []byte
	filtered := false
	for rest := b; len(rest) > 0; {
		size, err := frameSize(rest)
		if err != nil {
			return 0, err
		}
		frame := rest[:size]
		rest = rest[size:]
		send, err := c.admitLocked(frame)
		if err != nil {
			return 0, err
		}
		if send {
			out = append(out, frame...)
		} else {
			filtered = true
		}
	}
	if !filtered {
		out = b
	}
	if len(out) > 0 {
		if _, err := c.writeFrame(out); err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

// admitLocked reports whether frame may be written now. A QoS 1/2 PUBLISH
// beyond the peer's receive window is queued by packet ID instead.
func (c *flowConn) admitLocked(frame []byte) (bool, error) {
	if frame[0]>>4 != packets.Publish || (frame[0]>>1)&3 == 0 {
		return true, nil
	}
	mid, err := publicationID(frame)
	if err != nil {
		return false, err
	}
	if c.closed {
		return false, net.ErrClosed
	}
	if c.client == nil {
		return false, errors.New("MQTT publication before session establishment")
	}
	if _, exists := c.client.State.Inflight.Get(mid); !exists {
		return false, nil
	}
	if c.active[mid] {
		return true, nil
	}
	if c.queued[mid] {
		return false, nil
	}
	if len(c.active) >= c.limit {
		c.queued[mid] = true
		c.queue = append(c.queue, mid)
		return false, nil
	}
	c.active[mid] = true
	return true, nil
}

func (c *flowConn) acknowledge(mid uint16) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.active[mid] || c.closed {
		return nil
	}
	delete(c.active, mid)
	for len(c.queue) > 0 && len(c.active) < c.limit {
		next := c.queue[0]
		c.queue = c.queue[1:]
		delete(c.queued, next)
		pk, ok := c.client.State.Inflight.Get(next)
		if !ok || pk.FixedHeader.Type != packets.Publish {
			continue
		}
		if !c.manager.allowed(c.client, "read", pk.TopicName) {
			c.drop(pk)
			continue
		}
		if pk.Properties.MessageExpiryInterval > 0 {
			expires := pk.Created + int64(pk.Properties.MessageExpiryInterval)
			remaining := expires - time.Now().Unix()
			if remaining <= 0 {
				c.drop(pk)
				continue
			}
			pk.Properties.MessageExpiryInterval = uint32(remaining)
		}
		pk.ProtocolVersion = c.client.Properties.ProtocolVersion
		pk.Mods.MaxSize = c.client.Properties.Props.MaximumPacketSize
		var frame bytes.Buffer
		if err := pk.PublishEncode(&frame); err != nil {
			c.drop(pk)
			continue
		}
		c.active[next] = true
		if _, err := c.writeFrame(frame.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (c *flowConn) drop(pk packets.Packet) {
	if c.client.State.Inflight.Delete(pk.PacketID) {
		c.manager.store.OnQosDropped(c.client, pk)
	}
}

func (c *flowConn) Close() error {
	// Close the underlying socket first to unblock a writer holding mu.
	err := c.Conn.Close()
	c.mu.Lock()
	c.closed = true
	c.queue = nil
	clear(c.queued)
	clear(c.active)
	c.mu.Unlock()
	return err
}
