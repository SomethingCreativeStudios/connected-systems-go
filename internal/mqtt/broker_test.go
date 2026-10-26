package mqtt

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/pubsub"
	"github.com/gorilla/websocket"
	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type testIngress struct {
	mu       sync.Mutex
	count    int
	results  map[string]pubsub.CreateResult
	disabled bool
}

func (s *testIngress) Authorize(_ context.Context, _ string, access, topic string) error {
	if s.disabled {
		return fmt.Errorf("disabled")
	}
	cfg := config.PubSubConfig{ResourceData: config.PubSubFeatureConfig{Enabled: true}, ResourceEvents: config.PubSubFeatureConfig{Enabled: true}}
	if access == "write" {
		if _, err := pubsub.ParseDataTopic("node1", topic); err != nil {
			return err
		}
		return nil
	}
	if pubsub.FilterOffered("node1", topic, cfg) {
		return nil
	}
	return fmt.Errorf("not offered")
}
func (s *testIngress) Create(_ context.Context, p pubsub.Publication) (pubsub.CreateResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ProtocolVersion == 4 {
		p.ContentType = "application/geo+json"
	}
	if p.ContentType != "application/geo+json" || !json.Valid(p.Payload) {
		return pubsub.CreateResult{}, fmt.Errorf("bad resource")
	}
	key := ""
	if p.IdempotencyKey != nil {
		key = p.Principal + ":" + *p.IdempotencyKey
	}
	if r, ok := s.results[key]; key != "" && ok {
		r.Replayed = true
		return r, nil
	}
	s.count++
	r := pubsub.CreateResult{Payload: []byte(fmt.Sprintf(`{"id":"%d"}`, s.count)), ContentType: p.ContentType}
	if key != "" {
		s.results[key] = r
	}
	return r, nil
}

func testManager(t testing.TB, dir string, edit func(*config.MQTTConfig), ingress *testIngress) *Manager {
	t.Helper()
	cfg := config.MQTTConfig{AllowAnonymous: true, NodeID: "node1", TCPAddress: "127.0.0.1:0", WebsocketAddress: "127.0.0.1:0", DataDir: dir, IngestTimeout: time.Second, MaxMessageBytes: 10 << 20}
	if edit != nil {
		edit(&cfg)
	}
	m := NewManager(cfg, zap.NewNop())
	require.NoError(t, m.Start(ingress))
	t.Cleanup(m.Disconnect)
	return m
}
func newTestIngress() *testIngress {
	return &testIngress{results: make(map[string]pubsub.CreateResult)}
}

type wireClient struct {
	net.Conn
	version byte
}

func wireString(s string) []byte {
	out := binary.BigEndian.AppendUint16(nil, uint16(len(s)))
	return append(out, []byte(s)...)
}
func wireLength(n int) []byte {
	var b []byte
	for {
		v := byte(n % 128)
		n /= 128
		if n > 0 {
			v |= 128
		}
		b = append(b, v)
		if n == 0 {
			return b
		}
	}
}
func (c *wireClient) send(kind byte, body []byte) error {
	_, err := c.Write(append(append([]byte{kind}, wireLength(len(body))...), body...))
	return err
}
func (c *wireClient) packet() (byte, []byte, error) {
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	var b [1]byte
	if _, err := io.ReadFull(c, b[:]); err != nil {
		return 0, nil, err
	}
	kind := b[0]
	n, shift := 0, 0
	for {
		if _, err := io.ReadFull(c, b[:]); err != nil {
			return 0, nil, err
		}
		n += int(b[0]&127) << shift
		shift += 7
		if b[0] < 128 {
			break
		}
	}
	body := make([]byte, n)
	_, err := io.ReadFull(c, body)
	return kind, body, err
}
func dialWire(t testing.TB, m *Manager, version byte, id, user, password string, expiry uint32, will bool) (*wireClient, byte, bool) {
	return dialWireWithReceiveMaximum(t, m, version, id, user, password, expiry, will, 65535)
}

func dialWireWithReceiveMaximum(t testing.TB, m *Manager, version byte, id, user, password string, expiry uint32, will bool, maximum uint16) (*wireClient, byte, bool) {
	t.Helper()
	listener, _ := m.server.Listeners.Get("tcp")
	conn, err := net.DialTimeout("tcp", listener.Address(), time.Second)
	require.NoError(t, err)
	c := &wireClient{Conn: conn, version: version}
	t.Cleanup(func() { _ = conn.Close() })
	flags := byte(2)
	if expiry > 0 {
		flags = 0
	}
	if user != "" {
		flags |= 192
	}
	if will {
		flags |= 4
	}
	b := append(wireString("MQTT"), version, flags, 0, 30)
	if version == 5 {
		var props []byte
		if expiry > 0 {
			props = append(props, 0x11)
			props = binary.BigEndian.AppendUint32(props, expiry)
		}
		if maximum != 65535 {
			props = append(props, 0x21)
			props = binary.BigEndian.AppendUint16(props, maximum)
		}
		b = append(b, wireLength(len(props))...)
		b = append(b, props...)
	}
	b = append(b, wireString(id)...)
	if will {
		if version == 5 {
			b = append(b, 0)
		}
		b = append(b, wireString("node1/systems:data")...)
		b = append(b, wireString(`{}`)...)
	}
	if user != "" {
		b = append(b, wireString(user)...)
		b = append(b, wireString(password)...)
	}
	require.NoError(t, c.send(0x10, b))
	kind, ack, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0x20), kind)
	require.GreaterOrEqual(t, len(ack), 2)
	return c, ack[1], ack[0] == 1
}
func (c *wireClient) subscribe(t testing.TB, topic string) byte {
	t.Helper()
	b := []byte{0, 1}
	if c.version == 5 {
		b = append(b, 0)
	}
	b = append(b, wireString(topic)...)
	b = append(b, 2)
	require.NoError(t, c.send(0x82, b))
	kind, ack, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0x90), kind)
	return ack[len(ack)-1]
}
func (c *wireClient) create(t *testing.T, key string, qos byte, id uint16) (byte, []byte) {
	t.Helper()
	b := wireString("node1/systems:data")
	if qos > 0 {
		b = binary.BigEndian.AppendUint16(b, id)
	}
	if c.version == 5 {
		props := append([]byte{3}, wireString("application/geo+json")...)
		if key != "" {
			props = append(props, 0x26)
			props = append(props, wireString("idempotency-key")...)
			props = append(props, wireString(key)...)
		}
		b = append(b, wireLength(len(props))...)
		b = append(b, props...)
	}
	b = append(b, []byte(`{"type":"Feature","properties":{}}`)...)
	require.NoError(t, c.send(0x30|(qos<<1), b))
	if qos == 0 {
		return 0, nil
	}
	kind, ack, err := c.packet()
	require.NoError(t, err)
	return kind, ack
}
func (c *wireClient) disconnect(t testing.TB) {
	t.Helper()
	require.NoError(t, c.send(0xe0, nil))
	require.NoError(t, c.Close())
}

func TestBrokerCreateAndQoS2RemainOnConnection(t *testing.T) {
	ingress := newTestIngress()
	m := testManager(t, t.TempDir(), nil, ingress)
	for _, version := range []byte{4, 5} {
		c, code, _ := dialWire(t, m, version, "writer"+strconv.Itoa(int(version)), "", "", 0, false)
		require.Zero(t, code)
		kind, ack := c.create(t, "key", 1, 7)
		require.Equal(t, byte(0x40), kind)
		require.Len(t, ack, 2)
		require.NoError(t, c.send(0xc0, nil))
		kind, _, err := c.packet()
		require.NoError(t, err)
		require.Equal(t, byte(0xd0), kind)
		if version == 5 {
			kind, _ = c.create(t, "two", 2, 9)
			require.Equal(t, byte(0x50), kind)
			kind, _ = c.create(t, "two", 2, 9)
			require.Equal(t, byte(0x50), kind)
			for range 2 {
				require.NoError(t, c.send(0x62, []byte{0, 9}))
				kind, _, err = c.packet()
				require.NoError(t, err)
				require.Equal(t, byte(0x70), kind)
			}
			kind, _ = c.create(t, "other", 2, 9)
			require.Equal(t, byte(0x50), kind)
		}
		c.disconnect(t)
	}
	ingress.mu.Lock()
	require.Equal(t, 4, ingress.count)
	ingress.mu.Unlock()
}

func TestBrokerPersistentStateAndExpiry(t *testing.T) {
	dir := t.TempDir()
	ingress := newTestIngress()
	m := testManager(t, dir, nil, ingress)
	c, code, _ := dialWire(t, m, 5, "persistent", "", "", 2, false)
	require.Zero(t, code)
	require.Equal(t, byte(2), c.subscribe(t, "node1/systems:data"))
	c.disconnect(t)
	require.Eventually(t, func() bool { return m.store.owner("persistent") == "anonymous" }, time.Second, 10*time.Millisecond)
	m.Publish("node1/systems:data", []byte(`{"id":"offline"}`), "application/geo+json")
	m.Disconnect()
	select {
	case err := <-m.Fatal():
		require.NoError(t, err)
	default:
	}
	m = testManager(t, dir, nil, ingress)
	c, code, present := dialWire(t, m, 5, "persistent", "", "", 2, false)
	require.Zero(t, code)
	require.True(t, present)
	kind, body, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(3), kind>>4)
	require.Contains(t, string(body), "offline")
	c.disconnect(t)
	m.Disconnect()
	time.Sleep(2100 * time.Millisecond)
	m = testManager(t, dir, nil, ingress)
	_, code, present = dialWire(t, m, 5, "persistent", "", "", 2, false)
	require.Zero(t, code)
	require.False(t, present)
}

func TestBrokerAccountsAndSessionOwnership(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "users.json")
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), 4)
	require.NoError(t, err)
	users, _ := json.Marshal([]account{{Username: "alice", PasswordHash: string(hash)}, {Username: "bob", PasswordHash: string(hash)}})
	require.NoError(t, os.WriteFile(file, users, 0600))
	m := testManager(t, filepath.Join(dir, "state"), func(c *config.MQTTConfig) { c.UsersFile = file }, newTestIngress())
	for _, password := range []string{"wrong", ""} {
		_, code, _ := dialWire(t, m, 5, "bad-"+password, "alice", password, 0, false)
		require.NotZero(t, code)
	}
	c, code, _ := dialWire(t, m, 5, "owned", "alice", "password", 300, false)
	require.Zero(t, code)
	c.disconnect(t)
	_, code, _ = dialWire(t, m, 5, "owned", "bob", "password", 300, false)
	require.NotZero(t, code)
	_, code, _ = dialWire(t, m, 5, "will", "alice", "password", 0, true)
	require.NotZero(t, code)
	_, code, _ = dialWire(t, m, 5, internalClientID, "alice", "password", 0, false)
	require.NotZero(t, code)
}

func TestAccountsRejectInvalidConfigurationAndHashCommand(t *testing.T) {
	file := filepath.Join(t.TempDir(), "users.json")
	for _, body := range []string{`null`, `[]`, `[{"username":"a","password_hash":"bad"}]`, `{} {}`} {
		require.NoError(t, os.WriteFile(file, []byte(body), 0600))
		_, err := loadAccounts(file, false)
		require.Error(t, err)
	}
	var output bytes.Buffer
	require.NoError(t, HashPassword(bytes.NewBufferString("with spaces\n"), &output))
	hash := bytes.TrimSpace(output.Bytes())
	cost, err := bcrypt.Cost(hash)
	require.NoError(t, err)
	require.Equal(t, 12, cost)
	require.NoError(t, bcrypt.CompareHashAndPassword(hash, []byte("with spaces")))
}

func TestBrokerQoS2SurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	ingress := newTestIngress()
	m := testManager(t, dir, nil, ingress)
	c, code, _ := dialWire(t, m, 5, "unfinished", "", "", 300, false)
	require.Zero(t, code)
	kind, _ := c.create(t, "survive", 2, 11)
	require.Equal(t, byte(0x50), kind)
	require.NoError(t, c.Close())
	m.Disconnect()
	m = testManager(t, dir, nil, ingress)
	c, code, present := dialWire(t, m, 5, "unfinished", "", "", 300, false)
	require.Zero(t, code)
	require.True(t, present)
	kind, _ = c.create(t, "survive", 2, 11)
	require.Equal(t, byte(0x50), kind)
	ingress.mu.Lock()
	require.Equal(t, 1, ingress.count)
	ingress.mu.Unlock()
	for range 2 {
		require.NoError(t, c.send(0x62, []byte{0, 11}))
		kind, _, err := c.packet()
		require.NoError(t, err)
		require.Equal(t, byte(0x70), kind)
	}
	kind, _ = c.create(t, "next-operation", 2, 11)
	require.Equal(t, byte(0x50), kind)
	ingress.mu.Lock()
	require.Equal(t, 2, ingress.count)
	ingress.mu.Unlock()
}

func TestBrokerFullDuplexQoS2PacketIDs(t *testing.T) {
	m := testManager(t, t.TempDir(), nil, newTestIngress())
	c, code, _ := dialWire(t, m, 5, "duplex", "", "", 0, false)
	require.Zero(t, code)
	require.Equal(t, byte(2), c.subscribe(t, "node1/systems:data"))
	first, one := c.create(t, "duplex-key", 2, 1)
	second, two, err := c.packet()
	require.NoError(t, err)
	require.ElementsMatch(t, []byte{0x34, 0x50}, []byte{first, second})
	body := one
	if second == 0x34 {
		body = two
	}
	topicLength := int(binary.BigEndian.Uint16(body[:2]))
	mid := body[2+topicLength : 4+topicLength]
	require.Equal(t, []byte{0, 1}, mid)
	require.NoError(t, c.send(0x50, mid))
	kind, _, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0x62), kind)
	require.NoError(t, c.send(0x70, mid))
	require.NoError(t, c.send(0x62, []byte{0, 1}))
	kind, _, err = c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0x70), kind)
	require.Zero(t, m.store.incomingCount("duplex"))
}

func TestBrokerQueueBoundsAndRestore(t *testing.T) {
	for _, size := range []int{16, 8 << 20} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			dir := t.TempDir()
			m := testManager(t, dir, nil, newTestIngress())
			c, code, _ := dialWireWithReceiveMaximum(t, m, 5, "queue", "", "", 300, false, 1)
			require.Zero(t, code)
			require.Equal(t, byte(2), c.subscribe(t, "node1/systems:data"))
			c.disconnect(t)
			require.Eventually(t, func() bool { cl, ok := m.server.Clients.Get("queue"); return ok && cl.Closed() }, time.Second, time.Millisecond)
			n := MaxQueuedMessages + 1
			if size > 16 {
				n = 9
			}
			payload := bytes.Repeat([]byte("x"), size)
			for range n {
				require.NoError(t, m.distribute(m.internal, packets.Packet{FixedHeader: packets.FixedHeader{Type: packets.Publish, Qos: 1}, TopicName: "node1/systems:data", Payload: payload, Properties: packets.Properties{ContentType: "application/geo+json"}}))
			}
			cl, _ := m.server.Clients.Get("queue")
			before := cl.State.Inflight.Len()
			require.Positive(t, before)
			require.LessOrEqual(t, before, MaxQueuedMessages)
			var total int64
			for _, pk := range cl.State.Inflight.GetAll(false) {
				total += queuedSize(pk)
			}
			require.LessOrEqual(t, total, int64(MaxQueuedBytes))
			if size == 16 {
				require.Equal(t, MaxQueuedMessages, before)
			} else {
				require.Equal(t, 7, before)
			}
			m.Disconnect()
			m = testManager(t, dir, nil, newTestIngress())
			cl, _ = m.server.Clients.Get("queue")
			require.Equal(t, before, cl.State.Inflight.Len())
		})
	}
}

func TestBrokerWebsocketThroughTLSProxy(t *testing.T) {
	m := testManager(t, t.TempDir(), nil, newTestIngress())
	listener, _ := m.server.Listeners.Get("websocket")
	target, err := url.Parse("http://" + listener.Address())
	require.NoError(t, err)
	proxy := httptest.NewTLSServer(httputil.NewSingleHostReverseProxy(target))
	defer proxy.Close()
	roots := x509.NewCertPool()
	roots.AddCert(proxy.Certificate())
	dialer := websocket.Dialer{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, Subprotocols: []string{"mqtt"}}
	socket, _, err := dialer.Dial("wss"+proxy.URL[5:]+"/mqtt", nil)
	require.NoError(t, err)
	defer socket.Close()
	c := &wireClient{Conn: &websocketConn{Conn: socket.UnderlyingConn(), socket: socket}, version: 5}
	b := append(wireString("MQTT"), 5, 2, 0, 30, 0)
	b = append(b, wireString("websocket-client")...)
	require.NoError(t, c.send(0x10, b))
	kind, ack, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(0x20), kind)
	require.Zero(t, ack[1])
	require.Equal(t, byte(2), c.subscribe(t, "node1/systems:data"))
	m.Publish("node1/systems:data", []byte(`{"id":"websocket"}`), "application/geo+json")
	kind, body, err := c.packet()
	require.NoError(t, err)
	require.Equal(t, byte(3), kind>>4)
	require.Contains(t, string(body), "websocket")
}

func TestBrokerBindAndStorageFailures(t *testing.T) {
	for _, websocket := range []bool{false, true} {
		occupied, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		cfg := config.MQTTConfig{AllowAnonymous: true, NodeID: "node1", TCPAddress: "127.0.0.1:0", WebsocketAddress: "127.0.0.1:0", DataDir: t.TempDir(), MaxMessageBytes: 10 << 20}
		if websocket {
			cfg.WebsocketAddress = occupied.Addr().String()
		} else {
			cfg.TCPAddress = occupied.Addr().String()
		}
		m := NewManager(cfg, zap.NewNop())
		require.Error(t, m.Start(newTestIngress()))
		m.Disconnect()
		require.NoError(t, occupied.Close())
	}
}
