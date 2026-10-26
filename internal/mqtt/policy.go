package mqtt

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/pubsub"
	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type policyHook struct {
	mochi.HookBase
	manager *Manager
}

func (*policyHook) ID() string { return "csapi-policy" }
func (*policyHook) Provides(b byte) bool {
	return bytes.Contains([]byte{mochi.OnConnect, mochi.OnConnectAuthenticate, mochi.OnSessionEstablish, mochi.OnACLCheck, mochi.OnPacketRead, mochi.OnPublish, mochi.OnPacketEncode}, []byte{b})
}

func (h *policyHook) OnConnect(cl *mochi.Client, pk packets.Packet) error {
	if pk.Connect.WillFlag || cl.ID == internalClientID || cl.ID == mochi.InlineClientId {
		_ = h.manager.server.SendConnack(cl, packets.ErrNotAuthorized, false, nil)
		return packets.ErrNotAuthorized
	}
	return nil
}

func (h *policyHook) OnConnectAuthenticate(cl *mochi.Client, pk packets.Packet) bool {
	m := h.manager
	if pk.Connect.UsernameFlag {
		hash, ok := m.users[string(pk.Connect.Username)]
		if !ok || bcrypt.CompareHashAndPassword(hash, pk.Connect.Password) != nil {
			return false
		}
	} else if !m.cfg.AllowAnonymous {
		return false
	}
	// Ownership admission and reservation must be atomic across simultaneous
	// CONNECTs. Mochi calls OnSessionEstablish immediately after a true result.
	m.sessionMu.Lock()
	m.purgeExpired(cl.ID)
	owner := m.store.owner(cl.ID)
	if owner != "" && owner != principal(cl) {
		m.sessionMu.Unlock()
		return false
	}
	return true
}

func (h *policyHook) OnSessionEstablish(cl *mochi.Client, pk packets.Packet) {
	m := h.manager
	defer m.sessionMu.Unlock()
	// Topic aliases are optional. Keep full topic names available to the final
	// permission check, including packets restored for reconnect replay.
	cl.Properties.Props.TopicAliasMaximum = 0
	// Native reconnect replay bypasses publishToClient's ACL. Remove revoked
	// queued messages before Mochi inherits the old session and replays it.
	if old, ok := m.server.Clients.Get(cl.ID); ok {
		for _, queued := range old.State.Inflight.GetAll(false) {
			if queued.FixedHeader.Type == packets.Publish {
				if queued.Properties.MessageExpiryInterval > 0 {
					queued.Expiry = queued.Created + int64(queued.Properties.MessageExpiryInterval)
					queued.ProtocolVersion = 5
				}
				if !m.allowed(cl, "read", queued.TopicName) || (queued.Expiry > 0 && queued.Expiry <= time.Now().Unix()) {
					if old.State.Inflight.Delete(queued.PacketID) {
						m.store.OnQosDropped(old, queued)
					}
				} else {
					old.State.Inflight.Set(queued)
				}
			}
		}
	}
	m.store.establish(cl, pk)
}

func (h *policyHook) OnACLCheck(cl *mochi.Client, topic string, write bool) bool {
	if _, trusted := h.manager.trusted.Load(cl); trusted {
		return true
	}
	access := "read"
	if write {
		access = "write"
	} else if strings.ContainsAny(topic, "+#") {
		access = "subscribe"
	}
	return h.manager.allowed(cl, access, topic)
}

func (h *policyHook) OnPublish(cl *mochi.Client, pk packets.Packet) (packets.Packet, error) {
	if _, trusted := h.manager.trusted.Load(cl); trusted {
		return pk, nil
	}
	// Network creates are consumed by OnPacketRead. No other path, including a
	// restored Will or an accidental local injection, can forward raw input.
	return pk, packets.ErrRejectPacket
}

func publicProperties(p packets.Properties) packets.Properties {
	p = p.Copy(false)
	p.TopicAlias = 0
	p.TopicAliasFlag = false
	users := make([]packets.UserProperty, 0, len(p.User))
	for _, property := range p.User {
		if property.Key != "idempotency-key" && !strings.HasPrefix(property.Key, "csapi-internal-") {
			users = append(users, property)
		}
	}
	p.User = users
	return p
}

func (h *policyHook) OnPacketEncode(cl *mochi.Client, pk packets.Packet) packets.Packet {
	if pk.FixedHeader.Type == packets.Connack && pk.ReasonCode == 0 {
		if conn, ok := cl.Net.Conn.(*flowConn); ok {
			conn.bind(cl)
		}
	}
	if pk.FixedHeader.Type == packets.Pubrel {
		// Mochi replaces the outgoing PUBLISH with PUBREL without a storage
		// callback, and its default unlimited-expiry setting gives this control
		// packet an immediate deadline. Persist the handshake phase explicitly.
		pk.Expiry = 0
		pk.Properties.MessageExpiryInterval = 0
		cl.State.Inflight.Set(pk)
		h.manager.store.OnQosPublish(cl, pk, pk.Created, 0)
	}
	if pk.FixedHeader.Type == packets.Publish && !h.manager.allowed(cl, "read", pk.TopicName) {
		// Also covers native inflight retransmission, which skips the normal ACL.
		cl.Stop(packets.ErrNotAuthorized)
	}
	// Strip only private user properties here. Outbound aliases are owned by
	// Mochi's connection and must survive encoding.
	props := publicProperties(pk.Properties)
	props.TopicAlias = pk.Properties.TopicAlias
	props.TopicAliasFlag = pk.Properties.TopicAliasFlag
	pk.Properties = props
	return pk
}

func (h *policyHook) OnPacketRead(cl *mochi.Client, pk packets.Packet) (packets.Packet, error) {
	if pk.FixedHeader.Type == packets.Puback || pk.FixedHeader.Type == packets.Pubcomp || (pk.FixedHeader.Type == packets.Pubrec && pk.ReasonCode >= 128) {
		if conn, ok := cl.Net.Conn.(*flowConn); ok {
			if err := conn.acknowledge(pk.PacketID); err != nil {
				cl.Stop(err)
			}
		}
	}
	if pk.FixedHeader.Type == packets.Disconnect {
		h.manager.store.disconnectExpiry(cl, pk)
		return pk, nil
	}
	if pk.FixedHeader.Type == packets.Pubrel {
		known := h.manager.store.completeIncoming(cl.ID, pk.PacketID)
		reason := byte(0)
		if !known && cl.Properties.ProtocolVersion == 5 {
			reason = packets.ErrPacketIdentifierNotFound.Code
		}
		_ = cl.WritePacket(packets.Packet{FixedHeader: packets.FixedHeader{Type: packets.Pubcomp}, PacketID: pk.PacketID, ReasonCode: reason})
		return h.consumed(), nil
	}
	if pk.FixedHeader.Type != packets.Publish {
		return pk, nil
	}
	h.receiveCreate(cl, pk)
	return h.consumed(), nil
}

// An error from OnPacketRead closes the connection in Mochi 2.7.9. After handling
// a packet, pass a broker-local QoS0 copy into a denied, non-advertised topic so
// native processing completes without another acknowledgement or inflight-state
// mutation. OnPublish independently refuses every untrusted publication.
func (h *policyHook) consumed() packets.Packet {
	return packets.Packet{FixedHeader: packets.FixedHeader{Type: packets.Publish}, TopicName: h.manager.cfg.NodeID + "/$consumed", Ignore: true}
}

// rejectionCode maps a create failure to the closest MQTT 5 reason code.
func rejectionCode(err error) packets.Code {
	var failure *pubsub.IngressError
	if !errors.As(err, &failure) {
		return packets.ErrUnspecifiedError
	}
	switch {
	case failure.Status == http.StatusForbidden:
		return packets.ErrNotAuthorized
	case failure.Status == http.StatusRequestEntityTooLarge:
		return packets.ErrPacketTooLarge
	case failure.Status == http.StatusNotFound:
		// The topic names a parent that does not exist.
		return packets.ErrTopicNameInvalid
	case failure.Status == http.StatusConflict || failure.Status == http.StatusGone:
		return packets.ErrImplementationSpecificError
	case failure.Status < http.StatusInternalServerError:
		return packets.ErrPayloadFormatInvalid
	}
	return packets.ErrUnspecifiedError
}

// reasonString describes a create failure without exposing server faults.
func reasonString(err error) string {
	var failure *pubsub.IngressError
	if !errors.As(err, &failure) || failure.Status >= http.StatusInternalServerError {
		return "resource operation failed"
	}
	return failure.Message
}

// problemInfo returns the Reason String a PUBACK or PUBREC may carry: none when
// the client asked for no problem information, and at most 256 valid UTF-8 bytes.
func problemInfo(cl *mochi.Client, reason string) string {
	props := cl.Properties.Props
	if props.RequestProblemInfoFlag && props.RequestProblemInfo == 0 {
		return ""
	}
	reason = strings.ToValidUTF8(reason, "")
	if len(reason) > 256 {
		cut := 256
		for cut > 0 && !utf8.RuneStart(reason[cut]) {
			cut--
		}
		reason = reason[:cut]
	}
	return reason
}

func rejectPublication(m *Manager, cl *mochi.Client, pk packets.Packet, code packets.Code, reason string) {
	if cl.Properties.ProtocolVersion == 5 && pk.FixedHeader.Qos > 0 {
		switch code {
		case packets.ErrUnspecifiedError, packets.ErrImplementationSpecificError, packets.ErrNotAuthorized, packets.ErrTopicNameInvalid, packets.ErrPacketIdentifierInUse, packets.ErrQuotaExceeded, packets.ErrPayloadFormatInvalid:
		default:
			// Protocol errors such as unsupported QoS or receive-maximum
			// violations are DISCONNECT reasons, not legal PUBACK/PUBREC codes.
			_ = m.server.DisconnectClient(cl, code)
			return
		}
		kind := byte(packets.Puback)
		if pk.FixedHeader.Qos == 2 {
			kind = packets.Pubrec
		}
		_ = cl.WritePacket(packets.Packet{FixedHeader: packets.FixedHeader{Type: kind}, PacketID: pk.PacketID, ReasonCode: code.Code, Properties: packets.Properties{ReasonString: problemInfo(cl, reason)}})
	} else if cl.Properties.ProtocolVersion == 5 {
		_ = m.server.DisconnectClient(cl, code)
	} else {
		// MQTT 3.1.1 has no server-to-client DISCONNECT packet or failure PUBACK.
		cl.Stop(code)
	}
}

func incomingFingerprint(pk packets.Packet) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(pk.Payload))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	if value == nil {
		return "", errors.New("resource must be an object")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return "", errors.New("resource must be one JSON object")
	}
	key := ""
	for _, p := range pk.Properties.User {
		if p.Key == "idempotency-key" {
			key = p.Val
		}
	}
	b, err := json.Marshal([]any{pk.TopicName, mediatype.Normalize(pk.Properties.ContentType), key, value})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func (h *policyHook) receiveCreate(cl *mochi.Client, pk packets.Packet) {
	m := h.manager
	if !m.beginOperation() {
		rejectPublication(m, cl, pk, packets.ErrServerShuttingDown, "")
		return
	}
	defer m.work.Done()
	if code := pk.PublishValidate(m.server.Options.Capabilities.TopicAliasMaximum); code != packets.CodeSuccess {
		rejectPublication(m, cl, pk, code, "")
		return
	}
	if pk.Properties.TopicAliasFlag {
		pk.TopicName = cl.State.TopicAliases.Inbound.Set(pk.Properties.TopicAlias, pk.TopicName)
	}
	if !m.allowed(cl, "write", pk.TopicName) {
		rejectPublication(m, cl, pk, packets.ErrNotAuthorized, "topic is not a writable Resource Data topic")
		return
	}
	// Retain Available is 0 and the resource limit is advertised, so MQTT 5 uses
	// the DISCONNECT reasons for both.
	if pk.FixedHeader.Retain {
		rejectPublication(m, cl, pk, packets.ErrRetainNotSupported, "")
		return
	}
	if int64(len(pk.Payload)) > m.cfg.MaxMessageBytes {
		rejectPublication(m, cl, pk, packets.ErrPacketTooLarge, "")
		return
	}
	var key *string
	for _, property := range pk.Properties.User {
		if strings.HasPrefix(property.Key, "csapi-internal-") {
			rejectPublication(m, cl, pk, packets.ErrNotAuthorized, "reserved user property")
			return
		}
		if property.Key == "idempotency-key" {
			if key != nil || property.Val == "" {
				rejectPublication(m, cl, pk, packets.ErrPayloadFormatInvalid, "exactly one nonempty idempotency-key is allowed")
				return
			}
			v := property.Val
			key = &v
		}
	}
	var fingerprint string
	if pk.FixedHeader.Qos == 2 {
		if cl.Properties.ProtocolVersion != 5 || key == nil {
			rejectPublication(m, cl, pk, packets.ErrQosNotSupported, "")
			return
		}
		var err error
		fingerprint, err = incomingFingerprint(pk)
		if err != nil {
			rejectPublication(m, cl, pk, packets.ErrPayloadFormatInvalid, err.Error())
			return
		}
		if prior, exists := m.store.incoming(cl.ID, pk.PacketID); exists {
			if prior != fingerprint {
				rejectPublication(m, cl, pk, packets.ErrPacketIdentifierInUse, "")
				return
			}
			_ = cl.WritePacket(packets.Packet{FixedHeader: packets.FixedHeader{Type: packets.Pubrec}, PacketID: pk.PacketID})
			return
		}
		if m.store.incomingCount(cl.ID) >= MaxQueuedMessages {
			rejectPublication(m, cl, pk, packets.ErrReceiveMaximum, "")
			return
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.cfg.IngestTimeout)
	defer cancel()
	result, err := m.ingress.Create(ctx, pubsub.Publication{Principal: principal(cl), Topic: pk.TopicName, ProtocolVersion: int(cl.Properties.ProtocolVersion), QoS: int(pk.FixedHeader.Qos), Retained: pk.FixedHeader.Retain, ContentType: pk.Properties.ContentType, IdempotencyKey: key, Payload: pk.Payload})
	if err != nil {
		m.logger.Warn("Rejected MQTT create", zap.String("principal", principal(cl)), zap.String("topic", pk.TopicName), zap.Error(err))
		rejectPublication(m, cl, pk, rejectionCode(err), reasonString(err))
		return
	}
	if pk.FixedHeader.Qos == 2 && !m.store.saveIncoming(cl, pk.PacketID, fingerprint) {
		cl.Stop(packets.ErrServerUnavailable)
		return
	}
	out := pk.Copy(false)
	out.Payload = result.Payload
	out.Properties = publicProperties(pk.Properties)
	out.Properties.ContentType = result.ContentType
	out.FixedHeader.Dup = false
	out.FixedHeader.Retain = false
	// Preserve origin for MQTT No Local, while trust is tied to this in-process
	// object rather than the externally chosen identifier.
	origin := m.server.NewClient(nil, mochi.LocalListener, cl.ID, true)
	origin.Properties.ProtocolVersion = 5
	m.trusted.Store(origin, true)
	err = m.distribute(origin, out)
	m.trusted.Delete(origin)
	if err != nil {
		m.fail(fmt.Errorf("distribute canonical resource: %w", err))
		cl.Stop(packets.ErrServerUnavailable)
		return
	}
	if result.Publish != nil {
		result.Publish()
	}
	if pk.FixedHeader.Qos > 0 {
		kind := byte(packets.Puback)
		if pk.FixedHeader.Qos == 2 {
			kind = packets.Pubrec
		}
		_ = cl.WritePacket(packets.Packet{FixedHeader: packets.FixedHeader{Type: kind}, PacketID: pk.PacketID})
	}
}
