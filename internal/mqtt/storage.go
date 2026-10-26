package mqtt

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	db "github.com/cockroachdb/pebble"
	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/storage"
	"github.com/mochi-mqtt/server/v2/packets"
	"go.uber.org/zap"
)

// Broker state is independent of the PostgreSQL resource/idempotency transaction.
// Metadata adds absolute expiry and the incoming QoS2 handshake to Mochi's store.
type sessionMetadata struct {
	Principal     string            `json:"principal"`
	ExpiresAt     int64             `json:"expires_at"`
	ExpirySeconds uint32            `json:"expiry_seconds"`
	Connected     bool              `json:"connected"`
	Incoming      map[uint16]string `json:"incoming,omitempty"`
}

type durableHook struct {
	pebbleStorageHook
	manager     *Manager
	mu          sync.Mutex
	metadata    *db.DB
	sessions    map[string]*sessionMetadata
	connections map[string]*mochi.Client
	restored    map[string]bool
	closed      sync.Once
	usageMu     sync.Mutex
	usage       map[string]*queueUsage
}

// queueUsage is a running total of a client's queued bytes, so admitting a
// delivery does not rescan every inflight message.
type queueUsage struct {
	total int64
	sizes map[uint16]int64
}

func newDurableHook(m *Manager, path string) (*durableHook, error) {
	metadata, err := db.Open(filepath.Join(path, "metadata"), &db.Options{})
	if err != nil {
		return nil, fmt.Errorf("open MQTT session metadata: %w", err)
	}
	h := &durableHook{manager: m, metadata: metadata, sessions: make(map[string]*sessionMetadata), restored: make(map[string]bool), connections: make(map[string]*mochi.Client), usage: make(map[string]*queueUsage)}
	iter, err := metadata.NewIter(nil)
	if err != nil {
		_ = metadata.Close()
		return nil, err
	}
	for iter.First(); iter.Valid(); iter.Next() {
		var session sessionMetadata
		if err = json.Unmarshal(iter.Value(), &session); err != nil {
			break
		}
		h.sessions[string(iter.Key())] = &session
	}
	if err == nil {
		err = iter.Error()
	}
	_ = iter.Close()
	if err != nil {
		_ = metadata.Close()
		return nil, fmt.Errorf("read MQTT session metadata: %w", err)
	}
	return h, nil
}

func (h *durableHook) Init(any) error {
	return h.open(filepath.Join(h.manager.cfg.DataDir, "state", "broker"), h.manager.fail)
}

func (h *durableHook) Stop() error {
	err := h.pebbleStorageHook.Stop()
	if err != nil {
		h.manager.fail(err)
	}
	return err
}

func (h *durableHook) Provides(b byte) bool {
	return b == mochi.OnPublishDropped || h.pebbleStorageHook.Provides(b)
}

func (h *durableHook) saveLocked(id string) bool {
	b, err := json.Marshal(h.sessions[id])
	if err == nil {
		err = h.metadata.Set([]byte(id), b, db.Sync)
	}
	if err != nil {
		h.manager.fail(fmt.Errorf("persist MQTT session metadata: %w", err))
		return false
	}
	return true
}

func (h *durableHook) deleteLocked(id string) {
	delete(h.sessions, id)
	if err := h.metadata.Delete([]byte(id), db.Sync); err != nil {
		h.manager.fail(err)
	}
}

// persistent reports whether a session can outlive its connection. Queued
// messages of other sessions are discarded on disconnect, so they are not stored.
func (h *durableHook) persistent(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.sessions[id]
	return s != nil && s.ExpirySeconds > 0
}

// account records pk's queued size and reports whether the client's queue still
// fits MaxQueuedBytes. A missed removal can only overcount, so the total is
// recomputed from the broker's inflight state before anything is rejected.
func (h *durableHook) account(cl *mochi.Client, pk packets.Packet) bool {
	h.usageMu.Lock()
	defer h.usageMu.Unlock()
	u := h.usage[cl.ID]
	if u == nil {
		u = &queueUsage{sizes: make(map[uint16]int64)}
		h.usage[cl.ID] = u
	}
	size := queuedSize(pk)
	u.total += size - u.sizes[pk.PacketID]
	u.sizes[pk.PacketID] = size
	if u.total <= MaxQueuedBytes || pk.FixedHeader.Type != packets.Publish {
		return true
	}
	u.total, u.sizes = 0, make(map[uint16]int64)
	for _, queued := range cl.State.Inflight.GetAll(false) {
		size = queuedSize(queued)
		u.sizes[queued.PacketID] = size
		u.total += size
	}
	return u.total <= MaxQueuedBytes
}

func (h *durableHook) release(id string, packetID uint16) {
	h.usageMu.Lock()
	defer h.usageMu.Unlock()
	if u := h.usage[id]; u != nil {
		u.total -= u.sizes[packetID]
		delete(u.sizes, packetID)
	}
}

func (h *durableHook) forget(id string) {
	h.usageMu.Lock()
	defer h.usageMu.Unlock()
	delete(h.usage, id)
}

func (h *durableHook) owner(id string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.sessions[id]
	if s == nil || (!s.Connected && s.ExpiresAt <= time.Now().Unix()) {
		return ""
	}
	return s.Principal
}

func (h *durableHook) establish(cl *mochi.Client, pk packets.Packet) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[cl.ID] = cl
	s := h.sessions[cl.ID]
	if s == nil || pk.Connect.Clean || (!s.Connected && s.ExpiresAt <= time.Now().Unix()) {
		s = &sessionMetadata{Incoming: make(map[uint16]string)}
		h.sessions[cl.ID] = s
		h.forget(cl.ID)
	}
	s.Principal = principal(cl)
	s.Connected = true
	s.ExpiresAt = 0
	s.ExpirySeconds = uint32(MQTT311SessionExpiry / time.Second)
	if cl.Properties.ProtocolVersion == 5 {
		s.ExpirySeconds = pk.Properties.SessionExpiryInterval
	}
	if cl.Properties.ProtocolVersion == 4 && pk.Connect.Clean {
		s.ExpirySeconds = 0
	}
	if !h.saveLocked(cl.ID) {
		cl.Stop(packets.ErrServerUnavailable)
	}
}

func (h *durableHook) disconnectExpiry(cl *mochi.Client, pk packets.Packet) {
	if cl.Properties.ProtocolVersion != 5 || !pk.Properties.SessionExpiryIntervalFlag {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	// A session that began with zero expiry cannot be extended at DISCONNECT
	// [MQTT-3.14.2-2]; its queued messages were never stored.
	if s := h.sessions[cl.ID]; s != nil && s.ExpirySeconds > 0 {
		s.ExpirySeconds = pk.Properties.SessionExpiryInterval
		h.saveLocked(cl.ID)
	}
}

func (h *durableHook) incoming(id string, packetID uint16) (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.sessions[id]
	if s == nil {
		return "", false
	}
	v, ok := s.Incoming[packetID]
	return v, ok
}
func (h *durableHook) incomingCount(id string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s := h.sessions[id]; s != nil {
		return len(s.Incoming)
	}
	return 0
}
func (h *durableHook) saveIncoming(cl *mochi.Client, packetID uint16, fingerprint string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	id := cl.ID
	// A taken-over connection may finish a committed create after its replacement
	// started a clean session. Its old packet ID must not poison the new session.
	if h.connections[id] != cl {
		return false
	}
	s := h.sessions[id]
	if s == nil {
		return false
	}
	if s.Incoming == nil {
		s.Incoming = make(map[uint16]string)
	}
	s.Incoming[packetID] = fingerprint
	return h.saveLocked(id)
}
func (h *durableHook) completeIncoming(id string, packetID uint16) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s := h.sessions[id]; s != nil {
		if _, known := s.Incoming[packetID]; known {
			delete(s.Incoming, packetID)
			h.saveLocked(id)
			return true
		}
	}
	return false
}

func (h *durableHook) OnDisconnect(cl *mochi.Client, err error, expire bool) {
	// During restore Mochi calls OnDisconnect for records it just loaded.
	// Their absolute expiry was already recovered by StoredClients.
	if cl.Net.Conn != nil {
		h.manager.sessionMu.Lock()
		defer h.manager.sessionMu.Unlock()
		h.mu.Lock()
		if h.connections[cl.ID] != cl {
			h.mu.Unlock()
			return
		}
		if s := h.sessions[cl.ID]; s != nil && !cl.IsTakenOver() {
			s.Connected = false
			s.ExpiresAt = time.Now().Unix() + int64(s.ExpirySeconds)
			if expire || s.ExpirySeconds == 0 {
				h.deleteLocked(cl.ID)
				h.forget(cl.ID)
			} else {
				h.saveLocked(cl.ID)
			}
		}
		delete(h.connections, cl.ID)
		h.mu.Unlock()
	}
	h.pebbleStorageHook.OnDisconnect(cl, err, expire)
}

func (h *durableHook) OnClientExpired(cl *mochi.Client) {
	h.mu.Lock()
	h.deleteLocked(cl.ID)
	h.mu.Unlock()
	h.forget(cl.ID)
	h.pebbleStorageHook.OnClientExpired(cl)
}

func (h *durableHook) expired(now int64) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var ids []string
	for id, s := range h.sessions {
		if !s.Connected && s.ExpiresAt <= now {
			ids = append(ids, id)
		}
	}
	return ids
}

func (h *durableHook) StoredClients() ([]storage.Client, error) {
	clients, err := h.pebbleStorageHook.StoredClients()
	if err != nil {
		return nil, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	accepted := clients[:0]
	now := time.Now().Unix()
	for _, cl := range clients {
		s := h.sessions[cl.ID]
		if s == nil || !h.manager.knownPrincipal(s.Principal) || s.ExpirySeconds == 0 {
			h.pebbleStorageHook.OnClientExpired(&mochi.Client{ID: cl.ID})
			h.deleteLocked(cl.ID)
			continue
		}
		if s.Connected {
			s.Connected = false
			s.ExpiresAt = now + int64(s.ExpirySeconds)
			h.saveLocked(cl.ID)
		}
		if s.ExpiresAt <= now {
			h.pebbleStorageHook.OnClientExpired(&mochi.Client{ID: cl.ID})
			h.deleteLocked(cl.ID)
			continue
		}
		cl.Will = storage.ClientWill{}
		if cl.ProtocolVersion == 5 {
			cl.Properties.SessionExpiryIntervalFlag = true
			cl.Properties.SessionExpiryInterval = s.ExpirySeconds
		}
		h.restored[cl.ID] = true
		accepted = append(accepted, cl)
	}
	return accepted, nil
}

func (h *durableHook) StoredSubscriptions() ([]storage.Subscription, error) {
	values, err := h.pebbleStorageHook.StoredSubscriptions()
	if err != nil {
		return nil, err
	}
	accepted := values[:0]
	for _, v := range values {
		if !h.restored[v.Client] || v.Qos > 2 {
			continue
		}
		cl, ok := h.manager.server.Clients.Get(v.Client)
		if ok && h.manager.allowed(cl, "subscribe", v.Filter) {
			accepted = append(accepted, v)
		}
	}
	return accepted, nil
}

func queuedSize(pk packets.Packet) int64 {
	// Count payload, topic and public metadata rather than just JSON bytes.
	return int64(len(pk.Payload)+len(pk.TopicName)+len(pk.Properties.ContentType)+len(pk.Properties.CorrelationData)+len(pk.Properties.ResponseTopic)+32+5*len(pk.Properties.SubscriptionIdentifier)) + userPropertiesSize(pk.Properties.User)
}
func userPropertiesSize(values []packets.UserProperty) int64 {
	var n int64
	for _, v := range values {
		n += int64(len(v.Key) + len(v.Val) + 5)
	}
	return n
}

func (h *durableHook) StoredInflightMessages() ([]storage.Message, error) {
	values, err := h.pebbleStorageHook.StoredInflightMessages()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(values, func(i, j int) bool { return values[i].Created < values[j].Created })
	count := map[string]int{}
	sizes := map[string]int64{}
	accepted := values[:0]
	for _, v := range values {
		cl, ok := h.manager.server.Clients.Get(v.Client)
		if !ok || !h.restored[v.Client] {
			continue
		}
		pk := v.ToPacket()
		if pk.FixedHeader.Type == packets.Publish {
			if !h.manager.allowed(cl, "read", pk.TopicName) || int64(len(pk.Payload)) > h.manager.cfg.MaxMessageBytes {
				h.OnQosDropped(cl, pk)
				continue
			}
			if pk.Properties.MessageExpiryInterval > 0 && pk.Created+int64(pk.Properties.MessageExpiryInterval) <= time.Now().Unix() {
				h.OnQosDropped(cl, pk)
				continue
			}
		}
		size := queuedSize(pk)
		if count[v.Client] >= MaxQueuedMessages || sizes[v.Client]+size > MaxQueuedBytes {
			h.OnQosDropped(cl, pk)
			continue
		}
		count[v.Client]++
		sizes[v.Client] += size
		h.account(cl, pk)
		accepted = append(accepted, v)
	}
	return accepted, nil
}

func (*durableHook) StoredRetainedMessages() ([]storage.Message, error) { return nil, nil }

func (h *durableHook) OnQosPublish(cl *mochi.Client, pk packets.Packet, sent int64, resends int) {
	if pk.FixedHeader.Type == packets.Publish {
		if !h.manager.allowed(cl, "read", pk.TopicName) {
			cl.State.Inflight.Delete(pk.PacketID)
			h.OnQosDropped(cl, pk)
			cl.Stop(packets.ErrNotAuthorized)
			return
		}
		if !h.account(cl, pk) {
			if cl.State.Inflight.Delete(pk.PacketID) {
				atomic.AddInt64(&h.manager.server.Info.Inflight, -1)
				cl.State.Inflight.IncreaseSendQuota()
			}
			h.OnQosDropped(cl, pk)
			h.manager.logger.Warn("MQTT client queue byte limit reached", zap.String("client", cl.ID))
			if !cl.Closed() {
				cl.Stop(packets.ErrQuotaExceeded)
			}
			return
		}
		if !pk.FixedHeader.Dup {
			h.manager.lastQueued.Store(cl.ID, pk)
		}
	} else {
		// A PUBREL replaces its PUBLISH under the same packet ID.
		h.account(cl, pk)
	}
	if h.persistent(cl.ID) {
		h.pebbleStorageHook.OnQosPublish(cl, pk, sent, resends)
	}
}

func (h *durableHook) OnQosComplete(cl *mochi.Client, pk packets.Packet) {
	h.release(cl.ID, pk.PacketID)
	h.pebbleStorageHook.OnQosComplete(cl, pk)
}

func (h *durableHook) OnQosDropped(cl *mochi.Client, pk packets.Packet) {
	h.release(cl.ID, pk.PacketID)
	h.pebbleStorageHook.OnQosDropped(cl, pk)
}

func (h *durableHook) OnPublishDropped(cl *mochi.Client, _ packets.Packet) {
	if candidate, ok := h.manager.lastQueued.LoadAndDelete(cl.ID); ok {
		h.OnQosDropped(cl, candidate.(packets.Packet))
	}
}

func (h *durableHook) closeMetadata() {
	h.closed.Do(func() {
		if err := h.metadata.Close(); err != nil {
			h.manager.logger.Warn("Close MQTT metadata", zap.Error(err))
		}
	})
}
