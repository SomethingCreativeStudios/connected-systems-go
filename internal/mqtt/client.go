package mqtt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/pubsub"
	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"go.uber.org/zap"
)

const (
	maxPendingPublications = 4096
	internalClientID       = "$csapi-internal"
)

// Delivery policy, also advertised through AsyncAPI.
const (
	// MaxPacketBytes caps any MQTT packet, above the configurable resource limit.
	MaxPacketBytes = 11 << 20
	// MaxQueuedMessages and MaxQueuedBytes bound each client's queue.
	MaxQueuedMessages = 1000
	MaxQueuedBytes    = 64 << 20
	// MaxPendingWrites bounds deliveries waiting on a connected client's socket.
	MaxPendingWrites = 128
	// MQTT311SessionExpiry applies to inactive MQTT 3.1.1 persistent sessions.
	MQTT311SessionExpiry = 24 * time.Hour
)

type pendingPublication struct {
	topic, media string
	payload      []byte
}

// Manager owns the embedded endpoint. Its interface exposes no Mochi types.
type Manager struct {
	cfg         config.MQTTConfig
	logger      *zap.Logger
	server      *mochi.Server
	internal    *mochi.Client
	ingress     pubsub.Ingress
	users       map[string][]byte
	store       *durableHook
	trusted     sync.Map
	running     atomic.Bool
	quiescing   atomic.Bool
	fatal       chan error
	done        chan struct{}
	closeOnce   sync.Once
	sendMu      sync.Mutex
	sessionMu   sync.Mutex
	maintenance sync.WaitGroup
	workMu      sync.Mutex
	work        sync.WaitGroup
	pending     chan pendingPublication
	lastQueued  sync.Map
}

func NewManager(cfg config.MQTTConfig, logger *zap.Logger) *Manager {
	return &Manager{cfg: cfg, logger: logger, fatal: make(chan error, 1), done: make(chan struct{}), pending: make(chan pendingPublication, maxPendingPublications)}
}

func (m *Manager) Fatal() <-chan error { return m.fatal }
func (m *Manager) IsConnected() bool   { return m != nil && m.running.Load() }

func (m *Manager) fail(err error) {
	if err == nil {
		return
	}
	m.logger.Error("Embedded MQTT failure", zap.Error(err))
	select {
	case m.fatal <- err:
	default:
	}
}

func (m *Manager) Start(ingress pubsub.Ingress) (err error) {
	if ingress == nil {
		return errors.New("MQTT ingress is required")
	}
	m.ingress = ingress
	m.users, err = loadAccounts(m.cfg.UsersFile, m.cfg.AllowAnonymous)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(m.cfg.DataDir, 0700); err != nil {
		return err
	}
	capabilities := mochi.NewDefaultServerCapabilities()
	capabilities.MinimumProtocolVersion = 4
	capabilities.MaximumPacketSize = MaxPacketBytes
	capabilities.MaximumInflight = MaxQueuedMessages
	capabilities.ReceiveMaximum = MaxQueuedMessages
	capabilities.MaximumClientWritesPending = MaxPendingWrites
	capabilities.RetainAvailable = 0
	capabilities.SharedSubAvailable = 0
	capabilities.MaximumMessageExpiryInterval = 0
	capabilities.Compatibilities.NoInheritedPropertiesOnAck = true
	m.server = mochi.New(&mochi.Options{Capabilities: capabilities, Logger: slog.New(&brokerLog{manager: m})})
	defer func() {
		if err != nil {
			m.Disconnect()
		}
	}()
	m.internal = m.server.NewClient(nil, mochi.LocalListener, internalClientID, true)
	m.internal.Properties.ProtocolVersion = 5
	m.trusted.Store(m.internal, true)
	m.store, err = newDurableHook(m, filepath.Join(m.cfg.DataDir, "state"))
	if err != nil {
		return err
	}
	if err = m.server.AddHook(&policyHook{manager: m}, nil); err != nil {
		return err
	}
	if err = m.server.AddHook(m.store, nil); err != nil {
		return err
	}
	if err = m.server.AddListener(&tcpListener{address: m.cfg.TCPAddress, wrap: m.flowConnection}); err != nil {
		return err
	}
	if err = m.server.AddListener(newWebsocketListener(m.cfg.WebsocketAddress, m.fail, m.flowConnection)); err != nil {
		return err
	}
	if err = m.server.Serve(); err != nil {
		return err
	}
	select {
	case err = <-m.fatal:
		return err
	default:
	}
	m.running.Store(true)
	m.maintenance.Add(2)
	go m.expireSessions()
	go m.publishPending()
	m.logger.Info("Embedded MQTT ready", zap.String("tcp", m.cfg.TCPAddress), zap.String("websocket", m.cfg.WebsocketAddress))
	return nil
}

func (m *Manager) beginOperation() bool {
	m.workMu.Lock()
	defer m.workMu.Unlock()
	if m.quiescing.Load() {
		return false
	}
	m.work.Add(1)
	return true
}

func (m *Manager) Quiesce() {
	if m == nil {
		return
	}
	m.workMu.Lock()
	m.quiescing.Store(true)
	m.workMu.Unlock()
	m.work.Wait()
}

// Publish queues a server publication without blocking the caller. One worker
// distributes the queue in order, so publications on a topic keep their order.
func (m *Manager) Publish(topic string, payload []byte, media string) {
	m.workMu.Lock()
	defer m.workMu.Unlock()
	if !m.IsConnected() {
		return
	}
	select {
	case m.pending <- pendingPublication{topic: topic, media: media, payload: payload}:
	default:
		m.logger.Warn("MQTT publication queue full; publication dropped", zap.String("topic", topic))
	}
}

func (m *Manager) publishPending() {
	defer m.maintenance.Done()
	for p := range m.pending {
		pk := packets.Packet{FixedHeader: packets.FixedHeader{Type: packets.Publish, Qos: 1}, TopicName: p.topic, Payload: p.payload, Properties: packets.Properties{ContentType: p.media}}
		if err := m.distribute(m.internal, pk); err != nil {
			m.logger.Warn("MQTT publication failed", zap.String("topic", p.topic), zap.Error(err))
		}
	}
}

func (m *Manager) distribute(origin *mochi.Client, pk packets.Packet) error {
	m.sendMu.Lock()
	defer m.sendMu.Unlock()
	defer m.lastQueued.Range(func(key, _ any) bool { m.lastQueued.Delete(key); return true })
	pk.PacketID = uint16(pk.FixedHeader.Qos)
	return m.server.InjectPacket(origin, pk)
}

func (m *Manager) Disconnect() {
	if m == nil {
		return
	}
	m.closeOnce.Do(func() {
		m.Quiesce()
		m.workMu.Lock()
		// The publication worker only runs after a successful Start. Closing its
		// queue lets it drain queued publications before the broker closes.
		if m.running.Swap(false) {
			close(m.pending)
		}
		m.workMu.Unlock()
		close(m.done)
		m.maintenance.Wait()
		if m.server != nil {
			if err := m.server.Close(); err != nil {
				m.logger.Warn("Close MQTT", zap.Error(err))
			}
		}
		if m.store != nil {
			m.store.closeMetadata()
		}
	})
}

func (m *Manager) purgeExpired(id string) {
	if m.store.owner(id) != "" {
		return
	}
	if cl, ok := m.server.Clients.Get(id); ok && cl.Closed() {
		cl.ClearInflights()
		m.server.UnsubscribeClient(cl)
		m.server.Clients.Delete(id)
		m.store.OnClientExpired(cl)
	}
}

func (m *Manager) allowed(cl *mochi.Client, access, topic string) bool {
	if !m.knownPrincipal(principal(cl)) {
		return false
	}
	return m.ingress.Authorize(context.Background(), principal(cl), access, topic) == nil
}

func principal(cl *mochi.Client) string {
	if len(cl.Properties.Username) == 0 {
		return "anonymous"
	}
	return "user:" + string(cl.Properties.Username)
}

func (m *Manager) knownPrincipal(p string) bool {
	if p == "anonymous" {
		return m.cfg.AllowAnonymous
	}
	if len(p) <= 5 || p[:5] != "user:" {
		return false
	}
	_, ok := m.users[p[5:]]
	return ok
}

func (m *Manager) expireSessions() {
	defer m.maintenance.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.done:
			return
		case <-ticker.C:
			if err := m.store.syncPending(); err != nil {
				m.fail(fmt.Errorf("sync MQTT queue: %w", err))
			}
			m.expireQueuedMessages(time.Now().Unix())
			for _, id := range m.store.expired(time.Now().Unix()) {
				m.sessionMu.Lock()
				m.purgeExpired(id)
				m.sessionMu.Unlock()
			}
		}
	}
}

func (m *Manager) expireQueuedMessages(now int64) {
	m.sessionMu.Lock()
	defer m.sessionMu.Unlock()
	m.sendMu.Lock()
	defer m.sendMu.Unlock()
	for _, cl := range m.server.Clients.GetAll() {
		if !cl.Closed() {
			continue
		}
		for _, pk := range cl.State.Inflight.GetAll(false) {
			if pk.FixedHeader.Type == packets.Publish && pk.Properties.MessageExpiryInterval > 0 && pk.Created+int64(pk.Properties.MessageExpiryInterval) <= now {
				if cl.State.Inflight.Delete(pk.PacketID) {
					m.store.OnQosDropped(cl, pk)
				}
			}
		}
	}
}

// brokerLog forwards Mochi's warnings and errors to zap. Listener and storage
// failures call Manager.fail directly, so no Mochi log line can stop the API.
type brokerLog struct {
	manager *Manager
	attrs   []slog.Attr
}

func (*brokerLog) Enabled(_ context.Context, level slog.Level) bool { return level >= slog.LevelWarn }
func (h *brokerLog) Handle(_ context.Context, r slog.Record) error {
	fields := []zap.Field{zap.String("component", "mochi")}
	for _, a := range h.attrs {
		fields = append(fields, zap.Any(a.Key, a.Value.Any()))
	}
	r.Attrs(func(a slog.Attr) bool { fields = append(fields, zap.Any(a.Key, a.Value.Any())); return true })
	if r.Level >= slog.LevelError {
		h.manager.logger.Error(r.Message, fields...)
	} else {
		h.manager.logger.Warn(r.Message, fields...)
	}
	return nil
}
func (h *brokerLog) WithAttrs(a []slog.Attr) slog.Handler {
	return &brokerLog{manager: h.manager, attrs: append(append([]slog.Attr{}, h.attrs...), a...)}
}
func (h *brokerLog) WithGroup(string) slog.Handler { return h }
