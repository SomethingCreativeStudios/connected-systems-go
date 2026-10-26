package mqtt

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/stretchr/testify/require"
)

func TestBrokerOutgoingQoS2ResumesPubrel(t *testing.T) {
	dir := t.TempDir()
	ingress := newTestIngress()
	m := testManager(t, dir, nil, ingress)
	subscriber, code, _ := dialWire(t, m, 5, "outgoing-qos2", "", "", 300, false)
	require.Zero(t, code)
	require.Equal(t, byte(2), subscriber.subscribe(t, "node1/systems:data"))
	writer, code, _ := dialWire(t, m, 5, "writer", "", "", 0, false)
	require.Zero(t, code)
	kind, _ := writer.create(t, "one-delivery", 2, 7)
	require.Equal(t, byte(0x50), kind)
	kind, body, err := subscriber.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0x34), kind)
	topicLength := int(binary.BigEndian.Uint16(body[:2]))
	mid := body[2+topicLength : 4+topicLength]
	require.NoError(t, subscriber.send(0x50, mid))
	kind, _, err = subscriber.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0x62), kind)
	require.NoError(t, subscriber.Close())
	writer.disconnect(t)
	m.Disconnect()
	m = testManager(t, dir, nil, ingress)
	subscriber, code, present := dialWire(t, m, 5, "outgoing-qos2", "", "", 300, false)
	require.Zero(t, code)
	require.True(t, present)
	kind, body, err = subscriber.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0x62), kind)
	require.Equal(t, mid, body[:2])
	require.NoError(t, subscriber.send(0x70, mid))
	require.Eventually(t, func() bool { cl, _ := m.server.Clients.Get("outgoing-qos2"); return cl.State.Inflight.Len() == 0 }, time.Second, time.Millisecond)
}

func TestBrokerReceiveMaximumAcrossReplay(t *testing.T) {
	dir := t.TempDir()
	m := testManager(t, dir, nil, newTestIngress())
	c, code, _ := dialWireWithReceiveMaximum(t, m, 5, "one-at-a-time", "", "", 300, false, 1)
	require.Zero(t, code)
	require.Equal(t, byte(2), c.subscribe(t, "node1/systems:data"))
	for range 5 {
		require.NoError(t, m.distribute(m.internal, packets.Packet{FixedHeader: packets.FixedHeader{Type: packets.Publish, Qos: 1}, TopicName: "node1/systems:data", Payload: []byte(`{"id":"paced"}`), Properties: packets.Properties{ContentType: "application/geo+json"}}))
	}
	kind, _, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(3), kind>>4)
	assertQuiet := func() {
		require.NoError(t, c.SetReadDeadline(time.Now().Add(80*time.Millisecond)))
		var b [1]byte
		_, err := io.ReadFull(c, b[:])
		var timeout net.Error
		require.True(t, errors.As(err, &timeout) && timeout.Timeout(), "receive maximum exceeded: %v", err)
	}
	assertQuiet()
	require.NoError(t, c.Close())
	m.Disconnect()
	m = testManager(t, dir, nil, newTestIngress())
	c, code, present := dialWireWithReceiveMaximum(t, m, 5, "one-at-a-time", "", "", 300, false, 1)
	require.Zero(t, code)
	require.True(t, present)
	for range 5 {
		kind, body, err := c.packet()
		require.NoError(t, err)
		require.Equal(t, byte(3), kind>>4)
		topicLength := int(binary.BigEndian.Uint16(body[:2]))
		mid := body[2+topicLength : 4+topicLength]
		assertQuiet()
		require.NoError(t, c.send(0x40, mid))
	}
	require.Eventually(t, func() bool { cl, _ := m.server.Clients.Get("one-at-a-time"); return cl.State.Inflight.Len() == 0 }, time.Second, time.Millisecond)
}

func TestBrokerMessageExpiryAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	m := testManager(t, dir, nil, newTestIngress())
	c, code, _ := dialWire(t, m, 5, "message-expiry", "", "", 300, false)
	require.Zero(t, code)
	require.Equal(t, byte(2), c.subscribe(t, "node1/systems:data"))
	c.disconnect(t)
	require.Eventually(t, func() bool { cl, _ := m.server.Clients.Get("message-expiry"); return cl.Closed() }, time.Second, time.Millisecond)
	require.NoError(t, m.distribute(m.internal, packets.Packet{FixedHeader: packets.FixedHeader{Type: packets.Publish, Qos: 1}, TopicName: "node1/systems:data", Payload: []byte(`{"id":"expires"}`), Properties: packets.Properties{ContentType: "application/geo+json", MessageExpiryInterval: 1}}))
	m.Disconnect()
	m = testManager(t, dir, nil, newTestIngress())
	time.Sleep(1100 * time.Millisecond)
	c, code, present := dialWire(t, m, 5, "message-expiry", "", "", 300, false)
	require.Zero(t, code)
	require.True(t, present)
	require.NoError(t, c.send(0xc0, nil))
	kind, _, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0xd0), kind)
}

func TestBrokerMQTT311PersistenceAndOneDayPolicy(t *testing.T) {
	dir := t.TempDir()
	m := testManager(t, dir, nil, newTestIngress())
	c, code, _ := dialWire(t, m, 4, "mqtt311-persistent", "", "", 300, false)
	require.Zero(t, code)
	require.Equal(t, byte(2), c.subscribe(t, "node1/systems:data"))
	c.disconnect(t)
	// Mochi marks the client closed before its disconnect hook records the expiry.
	require.Eventually(t, func() bool {
		m.store.mu.Lock()
		defer m.store.mu.Unlock()
		s := m.store.sessions["mqtt311-persistent"]
		return s != nil && !s.Connected
	}, time.Second, time.Millisecond)
	m.store.mu.Lock()
	require.EqualValues(t, 86400, m.store.sessions["mqtt311-persistent"].ExpirySeconds)
	expires := m.store.sessions["mqtt311-persistent"].ExpiresAt
	m.store.mu.Unlock()
	m.Publish("node1/systems:data", []byte(`{"id":"mqtt311-offline"}`), "application/geo+json")
	m.Disconnect()
	m = testManager(t, dir, nil, newTestIngress())
	m.store.mu.Lock()
	require.Equal(t, expires, m.store.sessions["mqtt311-persistent"].ExpiresAt)
	m.store.mu.Unlock()
	c, code, present := dialWire(t, m, 4, "mqtt311-persistent", "", "", 300, false)
	require.Zero(t, code)
	require.True(t, present)
	kind, body, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(3), kind>>4)
	require.Contains(t, string(body), "mqtt311-offline")
}
