package mqtt

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/pubsub"
	"github.com/cockroachdb/pebble"
	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// readVarint decodes an MQTT variable byte integer.
func readVarint(b []byte) (value, size int) {
	for shift := 0; size < len(b); shift += 7 {
		v := b[size]
		size++
		value |= int(v&127) << shift
		if v < 128 {
			break
		}
	}
	return value, size
}

// decodePublish splits an MQTT 5 PUBLISH body.
func decodePublish(kind byte, body []byte) (topic string, packetID []byte, payload []byte) {
	n := int(binary.BigEndian.Uint16(body[:2]))
	topic, body = string(body[2:2+n]), body[2+n:]
	if kind&6 != 0 {
		packetID, body = body[:2], body[2:]
	}
	length, size := readVarint(body)
	return topic, packetID, body[size+length:]
}

func startManager(t testing.TB, cfg config.MQTTConfig, ingress pubsub.Ingress) *Manager {
	t.Helper()
	m := NewManager(cfg, zap.NewNop())
	require.NoError(t, m.Start(ingress))
	t.Cleanup(m.Disconnect)
	return m
}

func baseConfig(dir string) config.MQTTConfig {
	return config.MQTTConfig{AllowAnonymous: true, NodeID: "node1", TCPAddress: "127.0.0.1:0", WebsocketAddress: "127.0.0.1:0", DataDir: dir, IngestTimeout: time.Second, MaxMessageBytes: 10 << 20}
}

func TestBrokerPublishKeepsTopicOrder(t *testing.T) {
	m := testManager(t, t.TempDir(), nil, newTestIngress())
	c, code, _ := dialWire(t, m, 5, "ordered", "", "", 0, false)
	require.Zero(t, code)
	require.Equal(t, byte(2), c.subscribe(t, "node1/systems:data"))
	// Stay under MaxPendingWrites: a larger burst may outrun a starved writer and
	// be dropped, which is a different behavior from ordering.
	const n = MaxPendingWrites / 2
	for i := range n {
		m.Publish("node1/systems:data", []byte(strconv.Itoa(i)), "application/geo+json")
	}
	for want := range n {
		kind, body, err := c.packet()
		require.NoError(t, err)
		require.Equal(t, byte(3), kind>>4)
		_, packetID, payload := decodePublish(kind, body)
		require.Equal(t, strconv.Itoa(want), string(payload))
		require.NoError(t, c.send(0x40, packetID))
	}
}

// eventIngress publishes a lifecycle event through the create's Publish hook.
type eventIngress struct {
	*testIngress
	manager *Manager
}

func (e *eventIngress) Create(ctx context.Context, p pubsub.Publication) (pubsub.CreateResult, error) {
	result, err := e.testIngress.Create(ctx, p)
	result.Publish = func() {
		e.manager.Publish("node1/systems/1", []byte(`{"type":"org.ogc.api.csapi.system.create"}`), "application/cloudevents+json")
	}
	return result, err
}

func TestBrokerEventFollowsCanonicalData(t *testing.T) {
	ingress := &eventIngress{testIngress: newTestIngress()}
	ingress.manager = startManager(t, baseConfig(t.TempDir()), ingress)
	subscriber, code, _ := dialWire(t, ingress.manager, 5, "everything", "", "", 0, false)
	require.Zero(t, code)
	require.Equal(t, byte(2), subscriber.subscribe(t, "node1/#"))
	writer, code, _ := dialWire(t, ingress.manager, 5, "writer", "", "", 0, false)
	require.Zero(t, code)
	kind, _ := writer.create(t, "", 1, 3)
	require.Equal(t, byte(0x40), kind)
	var topics []string
	for range 2 {
		kind, body, err := subscriber.packet()
		require.NoError(t, err)
		topic, _, _ := decodePublish(kind, body)
		topics = append(topics, topic)
	}
	require.Equal(t, []string{"node1/systems:data", "node1/systems/1"}, topics)
}

// rejectingIngress fails every create with a fixed error.
type rejectingIngress struct{ err error }

func (rejectingIngress) Authorize(context.Context, string, string, string) error { return nil }
func (r rejectingIngress) Create(context.Context, pubsub.Publication) (pubsub.CreateResult, error) {
	return pubsub.CreateResult{}, r.err
}

func TestRejectionCodesAndReasons(t *testing.T) {
	for _, c := range []struct {
		err    error
		code   packets.Code
		reason string
	}{
		{&pubsub.IngressError{Status: 400, Message: "properties.uid is required"}, packets.ErrPayloadFormatInvalid, "properties.uid is required"},
		{&pubsub.IngressError{Status: 403, Message: "forbidden"}, packets.ErrNotAuthorized, "forbidden"},
		{&pubsub.IngressError{Status: 404, Message: "parent missing"}, packets.ErrTopicNameInvalid, "parent missing"},
		{&pubsub.IngressError{Status: 409, Message: "key reused"}, packets.ErrImplementationSpecificError, "key reused"},
		{&pubsub.IngressError{Status: 410, Message: "gone"}, packets.ErrImplementationSpecificError, "gone"},
		{&pubsub.IngressError{Status: 413, Message: "too large"}, packets.ErrPacketTooLarge, "too large"},
		{&pubsub.IngressError{Status: 500, Message: "pq: connection refused"}, packets.ErrUnspecifiedError, "resource operation failed"},
		{errors.New("unexpected"), packets.ErrUnspecifiedError, "resource operation failed"},
	} {
		require.Equal(t, c.code, rejectionCode(c.err), c.err.Error())
		require.Equal(t, c.reason, reasonString(c.err), c.err.Error())
	}
	cl := &mochi.Client{}
	long := strings.Repeat("é", 200)
	reason := problemInfo(cl, long)
	require.LessOrEqual(t, len(reason), 256)
	require.True(t, strings.HasPrefix(long, reason))
	cl.Properties.Props.RequestProblemInfoFlag = true
	require.Empty(t, problemInfo(cl, "detail"))
}

func TestBrokerRejectionCarriesReasonString(t *testing.T) {
	err := &pubsub.IngressError{Status: 400, Message: "properties.uid is required"}
	m := startManager(t, baseConfig(t.TempDir()), rejectingIngress{err: err})
	c, code, _ := dialWire(t, m, 5, "rejected", "", "", 0, false)
	require.Zero(t, code)
	kind, ack := c.create(t, "", 1, 5)
	require.Equal(t, byte(0x40), kind)
	require.Equal(t, []byte{0, 5}, ack[:2])
	require.Equal(t, packets.ErrPayloadFormatInvalid.Code, ack[2])
	length, size := readVarint(ack[3:])
	props := ack[3+size : 3+size+length]
	require.True(t, bytes.Contains(props, append([]byte{packets.PropReasonString}, wireString(err.Message)...)), "%x", props)
}

func TestBrokerRetainedCreateDisconnects(t *testing.T) {
	m := testManager(t, t.TempDir(), nil, newTestIngress())
	c, code, _ := dialWire(t, m, 5, "retainer", "", "", 0, false)
	require.Zero(t, code)
	b := wireString("node1/systems:data")
	b = binary.BigEndian.AppendUint16(b, 4)
	props := append([]byte{3}, wireString("application/geo+json")...)
	b = append(b, wireLength(len(props))...)
	b = append(b, props...)
	b = append(b, `{"type":"Feature","properties":{}}`...)
	require.NoError(t, c.send(0x33, b))
	kind, body, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0xe0), kind)
	require.Equal(t, packets.ErrRetainNotSupported.Code, body[0])
}

// flakyListener fails Accept a few times before serving connections.
type flakyListener struct {
	net.Listener
	errs  []error
	conns chan net.Conn
}

func (l *flakyListener) Accept() (net.Conn, error) {
	if len(l.errs) > 0 {
		err := l.errs[0]
		l.errs = l.errs[1:]
		return nil, err
	}
	conn, ok := <-l.conns
	if !ok {
		return nil, net.ErrClosed
	}
	return conn, nil
}

func TestTCPListenerRetriesAcceptErrors(t *testing.T) {
	fake := &flakyListener{errs: []error{syscall.EMFILE, syscall.ECONNABORTED}, conns: make(chan net.Conn, 1)}
	l := &tcpListener{Listener: fake, log: slog.New(slog.DiscardHandler), wrap: func(c net.Conn) net.Conn { return c }}
	established := make(chan string, 1)
	served := make(chan struct{})
	go func() {
		l.Serve(func(id string, _ net.Conn) error { established <- id; return nil })
		close(served)
	}()
	server, client := net.Pipe()
	defer client.Close()
	fake.conns <- server
	select {
	case id := <-established:
		require.Equal(t, "tcp", id)
	case <-time.After(2 * time.Second):
		t.Fatal("listener stopped after a temporary accept error")
	}
	close(fake.conns)
	select {
	case <-served:
	case <-time.After(2 * time.Second):
		t.Fatal("listener did not stop after close")
	}
}

func storedMessages(t testing.TB, m *Manager, id string) int {
	t.Helper()
	prefix := []byte("I/" + storeID(id) + "/")
	iter, err := m.store.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: append(bytes.Clone(prefix), 0xff)})
	require.NoError(t, err)
	n := 0
	for iter.First(); iter.Valid(); iter.Next() {
		n++
	}
	require.NoError(t, iter.Close())
	return n
}

func TestBrokerStoresQueuesOnlyForPersistentSessions(t *testing.T) {
	m := testManager(t, t.TempDir(), nil, newTestIngress())
	clean, code, _ := dialWire(t, m, 5, "clean", "", "", 0, false)
	require.Zero(t, code)
	durable, code, _ := dialWire(t, m, 5, "durable", "", "", 300, false)
	require.Zero(t, code)
	for _, c := range []*wireClient{clean, durable} {
		require.Equal(t, byte(2), c.subscribe(t, "node1/systems:data"))
	}
	require.NoError(t, m.distribute(m.internal, packets.Packet{FixedHeader: packets.FixedHeader{Type: packets.Publish, Qos: 1}, TopicName: "node1/systems:data", Payload: []byte(`{"id":"1"}`), Properties: packets.Properties{ContentType: "application/geo+json"}}))
	var packetID []byte
	for _, c := range []*wireClient{clean, durable} {
		kind, body, err := c.packet()
		require.NoError(t, err)
		_, packetID, _ = decodePublish(kind, body)
	}
	require.Zero(t, storedMessages(t, m, "clean"))
	require.Equal(t, 1, storedMessages(t, m, "durable"))
	require.NoError(t, durable.send(0x40, packetID))
	require.Eventually(t, func() bool { return storedMessages(t, m, "durable") == 0 }, time.Second, time.Millisecond)
	m.store.usageMu.Lock()
	defer m.store.usageMu.Unlock()
	require.Zero(t, m.store.usage["durable"].total)
	require.Empty(t, m.store.usage["durable"].sizes)
}

// BenchmarkDistributeQoS1 measures one publication fanned out to persistent
// QoS 1 subscribers that acknowledge every delivery.
func BenchmarkDistributeQoS1(b *testing.B) {
	m := testManager(b, b.TempDir(), nil, newTestIngress())
	const subscribers = 20
	for i := range subscribers {
		c, code, _ := dialWire(b, m, 5, fmt.Sprintf("subscriber-%d", i), "", "", 300, false)
		require.Zero(b, code)
		require.Equal(b, byte(2), c.subscribe(b, "node1/systems:data"))
		go func() {
			for {
				kind, body, err := c.packet()
				if err != nil {
					return
				}
				if kind>>4 == 3 {
					_, packetID, _ := decodePublish(kind, body)
					if c.send(0x40, packetID) != nil {
						return
					}
				}
			}
		}()
	}
	pk := packets.Packet{FixedHeader: packets.FixedHeader{Type: packets.Publish, Qos: 1}, TopicName: "node1/systems:data", Payload: []byte(`{"id":"bench"}`), Properties: packets.Properties{ContentType: "application/geo+json"}}
	for b.Loop() {
		if err := m.distribute(m.internal, pk); err != nil {
			b.Fatal(err)
		}
	}
}

// captureConn records each socket write.
type captureConn struct {
	net.Conn
	writes [][]byte
}

func (c *captureConn) Write(b []byte) (int, error) {
	c.writes = append(c.writes, bytes.Clone(b))
	return len(b), nil
}
func (*captureConn) Close() error { return nil }

func TestFlowGateSplitsCoalescedWrites(t *testing.T) {
	m := testManager(t, t.TempDir(), nil, newTestIngress())
	cl := m.server.NewClient(nil, mochi.LocalListener, "coalesced", true)
	cl.Properties.ProtocolVersion = 5
	cl.Properties.Props.ReceiveMaximum = 1
	var coalesced []byte
	for mid := uint16(1); mid <= 3; mid++ {
		pk := packets.Packet{FixedHeader: packets.FixedHeader{Type: packets.Publish, Qos: 1}, PacketID: mid, TopicName: "node1/systems:data", Payload: []byte(strconv.Itoa(int(mid))), ProtocolVersion: 5}
		cl.State.Inflight.Set(pk)
		var frame bytes.Buffer
		require.NoError(t, pk.PublishEncode(&frame))
		coalesced = append(coalesced, frame.Bytes()...)
	}
	socket := &captureConn{}
	gate := m.flowConnection(socket).(*flowConn)
	gate.bind(cl)
	payload := func(write []byte) string {
		size, err := frameSize(write)
		require.NoError(t, err)
		require.Equal(t, len(write), size, "one frame per gated write")
		_, header := readVarint(write[1:])
		_, _, body := decodePublish(write[0], write[1+header:])
		return string(body)
	}
	n, err := gate.Write(coalesced)
	require.NoError(t, err)
	require.Equal(t, len(coalesced), n)
	require.Len(t, socket.writes, 1)
	require.Equal(t, "1", payload(socket.writes[0]))
	require.NoError(t, gate.acknowledge(1))
	require.Len(t, socket.writes, 2)
	require.Equal(t, "2", payload(socket.writes[1]))
}
