package mqtt

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/cockroachdb/pebble"
	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/storage"
	"github.com/mochi-mqtt/server/v2/packets"
)

// Use Mochi's storage records with a local Pebble hook. The bundled v2.7.9
// Pebble hook leaves its read iterators open. Owning the small storage adapter
// also lets us reject corrupt records instead of silently discarding sessions.
// No broker source is vendored or modified.
//
// Session and subscription records are written synchronously. Queued messages
// are written without a sync, like Mochi's default Pebble mode, so delivery does
// not wait on disk; syncPending makes them durable on the maintenance tick and
// at Stop.
type pebbleStorageHook struct {
	mochi.HookBase
	db      *pebble.DB
	fail    func(error)
	pending atomic.Bool
}

func (*pebbleStorageHook) ID() string { return "csapi-pebble" }
func (*pebbleStorageHook) Provides(b byte) bool {
	switch b {
	case mochi.OnSessionEstablished, mochi.OnDisconnect, mochi.OnSubscribed, mochi.OnUnsubscribed, mochi.OnQosPublish, mochi.OnQosComplete, mochi.OnQosDropped, mochi.OnClientExpired, mochi.StoredClients, mochi.StoredSubscriptions, mochi.StoredInflightMessages, mochi.StoredRetainedMessages:
		return true
	}
	return false
}
func (h *pebbleStorageHook) open(path string, fail func(error)) error {
	h.fail = fail
	var err error
	h.db, err = pebble.Open(path, &pebble.Options{})
	return err
}
func (h *pebbleStorageHook) Stop() error {
	if h.db == nil {
		return nil
	}
	return errors.Join(h.syncPending(), h.db.Close())
}

// syncPending flushes unsynced queued-message writes to the write-ahead log.
func (h *pebbleStorageHook) syncPending() error {
	if h.db == nil || !h.pending.Swap(false) {
		return nil
	}
	return h.db.LogData(nil, pebble.Sync)
}
func storeID(id string) string                 { return base64.RawURLEncoding.EncodeToString([]byte(id)) }
func clientKey(id string) string               { return "C/" + storeID(id) }
func subscriptionKey(id, filter string) string { return "S/" + storeID(id) + "/" + storeID(filter) }
func messageKey(id string, mid uint16) string {
	return "I/" + storeID(id) + "/" + strconv.Itoa(int(mid))
}
func (h *pebbleStorageHook) put(key string, value any, opts *pebble.WriteOptions) {
	b, err := json.Marshal(value)
	if err == nil {
		err = h.db.Set([]byte(key), b, opts)
	}
	if err != nil {
		h.fail(fmt.Errorf("persist MQTT record: %w", err))
	} else if opts == pebble.NoSync {
		h.pending.Store(true)
	}
}
func (h *pebbleStorageHook) remove(key string, opts *pebble.WriteOptions) {
	if err := h.db.Delete([]byte(key), opts); err != nil {
		h.fail(err)
	} else if opts == pebble.NoSync {
		h.pending.Store(true)
	}
}
func scanRecords[T any](h *pebbleStorageHook, prefix string) ([]T, error) {
	iter, err := h.db.NewIter(&pebble.IterOptions{LowerBound: []byte(prefix), UpperBound: []byte(prefix + "\xff")})
	if err != nil {
		return nil, err
	}
	var values []T
	for iter.First(); iter.Valid(); iter.Next() {
		var value T
		if err = json.Unmarshal(iter.Value(), &value); err != nil {
			break
		}
		values = append(values, value)
	}
	err = errors.Join(err, iter.Error(), iter.Close())
	return values, err
}

func (h *pebbleStorageHook) OnSessionEstablished(cl *mochi.Client, _ packets.Packet) {
	p := cl.Properties.Props
	h.put(clientKey(cl.ID), storage.Client{ID: cl.ID, Username: cl.Properties.Username, Listener: cl.Net.Listener, ProtocolVersion: cl.Properties.ProtocolVersion, Clean: cl.Properties.Clean, Properties: storage.ClientProperties{
		SessionExpiryInterval: p.SessionExpiryInterval, SessionExpiryIntervalFlag: p.SessionExpiryIntervalFlag, MaximumPacketSize: p.MaximumPacketSize, ReceiveMaximum: p.ReceiveMaximum, TopicAliasMaximum: p.TopicAliasMaximum, RequestProblemInfo: p.RequestProblemInfo, RequestProblemInfoFlag: p.RequestProblemInfoFlag,
	}}, pebble.Sync)
}
func (h *pebbleStorageHook) OnDisconnect(cl *mochi.Client, _ error, expire bool) {
	if cl.IsTakenOver() {
		return
	}
	if expire {
		h.OnClientExpired(cl)
	} else {
		h.OnSessionEstablished(cl, packets.Packet{})
	}
}
func (h *pebbleStorageHook) OnSubscribed(cl *mochi.Client, pk packets.Packet, codes []byte) {
	for i, sub := range pk.Filters {
		if i >= len(codes) || codes[i] > 2 {
			continue
		}
		h.put(subscriptionKey(cl.ID, sub.Filter), storage.Subscription{Client: cl.ID, Filter: sub.Filter, Qos: codes[i], Identifier: sub.Identifier, NoLocal: sub.NoLocal, RetainHandling: sub.RetainHandling, RetainAsPublished: sub.RetainAsPublished}, pebble.Sync)
	}
}
func (h *pebbleStorageHook) OnUnsubscribed(cl *mochi.Client, pk packets.Packet) {
	for _, sub := range pk.Filters {
		h.remove(subscriptionKey(cl.ID, sub.Filter), pebble.Sync)
	}
}
func (h *pebbleStorageHook) OnQosPublish(cl *mochi.Client, pk packets.Packet, sent int64, _ int) {
	p := pk.Properties
	h.put(messageKey(cl.ID, pk.PacketID), storage.Message{Client: cl.ID, Origin: pk.Origin, PacketID: pk.PacketID, TopicName: pk.TopicName, Payload: pk.Payload, FixedHeader: pk.FixedHeader, Created: pk.Created, Sent: sent, Properties: storage.MessageProperties{
		ContentType: p.ContentType, CorrelationData: p.CorrelationData, ResponseTopic: p.ResponseTopic, User: p.User, MessageExpiryInterval: p.MessageExpiryInterval, SubscriptionIdentifier: p.SubscriptionIdentifier, PayloadFormat: p.PayloadFormat, PayloadFormatFlag: p.PayloadFormatFlag,
	}}, pebble.NoSync)
}
func (h *pebbleStorageHook) OnQosComplete(cl *mochi.Client, pk packets.Packet) {
	h.remove(messageKey(cl.ID, pk.PacketID), pebble.NoSync)
}
func (h *pebbleStorageHook) OnQosDropped(cl *mochi.Client, pk packets.Packet) {
	h.OnQosComplete(cl, pk)
}
func (h *pebbleStorageHook) OnClientExpired(cl *mochi.Client) {
	batch := h.db.NewBatch()
	defer batch.Close()
	_ = batch.Delete([]byte(clientKey(cl.ID)), nil)
	for _, prefix := range []string{"S/" + storeID(cl.ID) + "/", "I/" + storeID(cl.ID) + "/"} {
		iter, err := h.db.NewIter(&pebble.IterOptions{LowerBound: []byte(prefix), UpperBound: []byte(prefix + "\xff")})
		if err != nil {
			h.fail(err)
			return
		}
		for iter.First(); iter.Valid(); iter.Next() {
			_ = batch.Delete(iter.Key(), nil)
		}
		err = errors.Join(iter.Error(), iter.Close())
		if err != nil {
			h.fail(err)
			return
		}
	}
	if err := batch.Commit(pebble.Sync); err != nil {
		h.fail(err)
	}
}
func (h *pebbleStorageHook) StoredClients() ([]storage.Client, error) {
	return scanRecords[storage.Client](h, "C/")
}
func (h *pebbleStorageHook) StoredSubscriptions() ([]storage.Subscription, error) {
	return scanRecords[storage.Subscription](h, "S/")
}
func (h *pebbleStorageHook) StoredInflightMessages() ([]storage.Message, error) {
	return scanRecords[storage.Message](h, "I/")
}
func (*pebbleStorageHook) StoredRetainedMessages() ([]storage.Message, error) { return nil, nil }
