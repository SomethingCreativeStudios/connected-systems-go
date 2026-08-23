package pubsub

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/connected-systems-go/internal/config"
	"go.uber.org/zap"
)

const (
	resourceEventsSuffix      = ":events"
	batchResourceEventsSuffix = ":batch-events"
	cloudEventsVersion        = "1.0"
	cloudEventsJSONType       = "application/cloudevents+json"
)

// Transport is the protocol-specific surface needed by the Pub/Sub publisher.
// mqtt.Manager implements it, while tests can provide an in-memory transport.
type Transport interface {
	IsConnected() bool
	Publish(topic string, payload []byte)
}

type Operation string

const (
	OperationCreate Operation = "create"
	OperationUpdate Operation = "update"
	OperationDelete Operation = "delete"
)

// Change describes one committed CS API resource lifecycle operation.
type Change struct {
	ResourceType   string
	ResourceID     string
	Operation      Operation
	SubjectPath    string
	ParentPath     string
	CollectionPath string
	Time           time.Time
	Data           map[string]any
}

// CloudEvent is the Pub/Sub Resource Event and Batch Resource Event JSON envelope.
type CloudEvent struct {
	SpecVersion     string         `json:"specversion"`
	Type            string         `json:"type"`
	Source          string         `json:"source"`
	Subject         string         `json:"subject"`
	ID              string         `json:"id"`
	ParentID        string         `json:"parentId,omitempty"`
	Time            string         `json:"time"`
	DataContentType string         `json:"datacontenttype,omitempty"`
	Data            map[string]any `json:"data,omitempty"`
}

// Publisher emits Pub/Sub messages over the configured transport.
type Publisher struct {
	apiRoot   string
	cfg       config.PubSubConfig
	transport Transport
	logger    *zap.Logger
	now       func() time.Time
	newID     func() string

	batchMu      sync.Mutex
	batches      map[batchKey]*batch
	batchStop    chan struct{}
	batchDone    chan struct{}
	batchStarted bool
	batchClosed  bool
	closeOnce    sync.Once
}

type batchKey struct {
	resourceType string
	operation    Operation
	parentPath   string
	windowStart  time.Time
}

type batch struct {
	key            batchKey
	collectionPath string
	topic          string
	windowEnd      time.Time
	count          int
}

func NewPublisher(apiRoot string, cfg config.PubSubConfig, transport Transport, logger *zap.Logger) *Publisher {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Publisher{
		apiRoot:   strings.TrimRight(apiRoot, "/"),
		cfg:       cfg,
		transport: transport,
		logger:    logger,
		now:       func() time.Time { return time.Now().UTC() },
		newID:     func() string { return uuid.NewString() },
		batches:   make(map[batchKey]*batch),
	}
}

func (p *Publisher) ResourceDataEnabled() bool {
	return p != nil && p.cfg.ResourceData.Enabled && p.connected()
}

func (p *Publisher) ResourceEventsEnabled() bool {
	return p != nil && p.cfg.ResourceEvents.Enabled && p.connected()
}

func (p *Publisher) BatchResourceEventsEnabled() bool {
	return p != nil && p.cfg.BatchResourceEvents.Enabled && p.connected()
}

// LifecycleEventsEnabled reports whether at least one lifecycle-event class
// can currently accept changes.
func (p *Publisher) LifecycleEventsEnabled() bool {
	return p != nil && p.connected() && (p.cfg.ResourceEvents.Enabled || p.cfg.BatchResourceEvents.Enabled)
}

func (p *Publisher) connected() bool {
	return p.transport != nil && p.transport.IsConnected()
}

// PublishResourceData publishes one complete native resource representation.
// Callers are responsible for using the same formatter/validation contract as
// the HTTP API before passing the resource here.
func (p *Publisher) PublishResourceData(topic string, resource any) {
	if !p.ResourceDataEnabled() {
		return
	}
	payload, err := json.Marshal(resource)
	if err != nil {
		p.logger.Error("Failed to marshal Pub/Sub Resource Data Message", zap.String("topic", topic), zap.Error(err))
		return
	}
	p.transport.Publish(topic, payload)
}

// PublishChange routes one committed lifecycle change to either a Batch
// Resource Event or an individual Resource Event. Observations and commands
// use batches whenever that class is configured; all other resource types use
// individual events.
func (p *Publisher) PublishChange(change Change) {
	if p == nil {
		return
	}
	resourceType := strings.ToLower(strings.TrimSpace(change.ResourceType))
	if p.cfg.BatchResourceEvents.Enabled && (resourceType == "observation" || resourceType == "command") {
		if !p.connected() {
			return
		}
		if err := p.recordBatchChange(change, resourceType); err != nil {
			p.logger.Error("Failed to record Pub/Sub Batch Resource Event change", zap.Error(err))
		}
		return
	}
	p.PublishResourceEvent(change)
}

// PublishResourceEvent emits one CloudEvents JSON Resource Event to the
// global, resource-type, and individual-resource discovery channels.
func (p *Publisher) PublishResourceEvent(change Change) {
	if !p.ResourceEventsEnabled() {
		return
	}
	event, err := p.NewResourceEvent(change)
	if err != nil {
		p.logger.Error("Failed to build Pub/Sub Resource Event", zap.Error(err))
		return
	}
	payload, err := json.Marshal(event)
	if err != nil {
		p.logger.Error("Failed to marshal Pub/Sub Resource Event", zap.Error(err))
		return
	}

	for _, topic := range ResourceEventTopics(change) {
		p.transport.Publish(topic, payload)
	}
}

func (p *Publisher) NewResourceEvent(change Change) (CloudEvent, error) {
	if p == nil {
		return CloudEvent{}, fmt.Errorf("Pub/Sub publisher is nil")
	}
	resourceType := strings.ToLower(strings.TrimSpace(change.ResourceType))
	if resourceType == "" {
		return CloudEvent{}, fmt.Errorf("resource type is required")
	}
	if change.ResourceID == "" {
		return CloudEvent{}, fmt.Errorf("resource ID is required")
	}
	switch change.Operation {
	case OperationCreate, OperationUpdate, OperationDelete:
	default:
		return CloudEvent{}, fmt.Errorf("unsupported resource operation %q", change.Operation)
	}
	if strings.TrimSpace(change.SubjectPath) == "" {
		return CloudEvent{}, fmt.Errorf("resource subject path is required")
	}

	eventTime := change.Time.UTC()
	if eventTime.IsZero() {
		eventTime = p.now()
	}
	event := CloudEvent{
		SpecVersion: cloudEventsVersion,
		Type:        fmt.Sprintf("org.ogc.api.consys.%s.%s", resourceType, change.Operation),
		Source:      p.apiRoot,
		Subject:     p.absoluteURL(change.SubjectPath),
		ID:          p.newID(),
		ParentID:    p.absoluteURL(change.ParentPath),
		Time:        eventTime.Format(time.RFC3339Nano),
		Data:        change.Data,
	}
	if len(change.Data) > 0 {
		event.DataContentType = "application/json"
	}
	return event, nil
}

func (p *Publisher) absoluteURL(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if parsed, err := url.Parse(path); err == nil && parsed.IsAbs() {
		return path
	}
	return p.apiRoot + "/" + strings.TrimLeft(path, "/")
}

func ResourceEventTopics(change Change) []string {
	paths := []string{change.CollectionPath, change.SubjectPath}
	topics := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		topic := topicForRESTPath(path, resourceEventsSuffix)
		if topic == "" {
			continue
		}
		if _, exists := seen[topic]; exists {
			continue
		}
		seen[topic] = struct{}{}
		topics = append(topics, topic)
	}
	return topics
}

func CloudEventsContentType() string {
	return cloudEventsJSONType
}

// BuildResourceEventSummary builds the optional JSON summary carried by a
// regular Resource Event. Empty values are omitted so callers can pass the
// persisted resource fields directly; a nil map means the event has no data
// member and therefore no datacontenttype attribute.
func BuildResourceEventSummary(name, description, uniqueID string) map[string]any {
	data := make(map[string]any, 3)
	if strings.TrimSpace(name) != "" {
		data["name"] = name
	}
	if strings.TrimSpace(description) != "" {
		data["description"] = description
	}
	if strings.TrimSpace(uniqueID) != "" {
		data["uniqueId"] = uniqueID
	}
	if len(data) == 0 {
		return nil
	}
	return data
}

func (p *Publisher) recordBatchChange(change Change, resourceType string) error {
	switch change.Operation {
	case OperationCreate, OperationUpdate, OperationDelete:
	default:
		return fmt.Errorf("unsupported resource operation %q", change.Operation)
	}

	parentKind := "datastreams"
	if resourceType == "command" {
		parentKind = "controlstreams"
	}
	_, err := nestedParentID(change.ParentPath, parentKind)
	if err != nil {
		return err
	}
	parentPath := strings.TrimRight(p.absoluteURL(change.ParentPath), "/")
	collectionPath := strings.TrimRight(strings.TrimSpace(change.CollectionPath), "/")
	if collectionPath == "" {
		return fmt.Errorf("collection path is required for a Batch Resource Event")
	}

	p.batchMu.Lock()
	defer p.batchMu.Unlock()
	if p.batchClosed {
		return fmt.Errorf("Pub/Sub publisher is closed")
	}
	// Read the clock while holding the same lock used by flushing so a change
	// arriving on a boundary cannot be inserted into an already-flushed window.
	now := p.now().UTC()
	windowStart, windowEnd := alignedWindow(now, p.cfg.BatchResourceEvents.Window)
	key := batchKey{
		resourceType: resourceType,
		operation:    change.Operation,
		parentPath:   parentPath,
		windowStart:  windowStart,
	}
	entry := p.batches[key]
	if entry == nil {
		entry = &batch{
			key:            key,
			collectionPath: collectionPath,
			topic:          topicForRESTPath(collectionPath, batchResourceEventsSuffix),
			windowEnd:      windowEnd,
		}
		p.batches[key] = entry
	}
	entry.count++
	p.startBatchWorkerLocked()
	return nil
}

func nestedParentID(parentPath, wantCollection string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(parentPath))
	if err != nil {
		return "", fmt.Errorf("parse parent path: %w", err)
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 || parts[len(parts)-2] != wantCollection || parts[len(parts)-1] == "" {
		return "", fmt.Errorf("parent path %q must identify one %s resource", parentPath, wantCollection)
	}
	return parts[len(parts)-1], nil
}

func alignedWindow(at time.Time, window time.Duration) (time.Time, time.Time) {
	at = at.UTC()
	windowNanos := window.Nanoseconds()
	startNanos := at.UnixNano() - at.UnixNano()%windowNanos
	start := time.Unix(0, startNanos).UTC()
	return start, start.Add(window)
}

func (p *Publisher) startBatchWorkerLocked() {
	if p.batchStarted || p.batchClosed {
		return
	}
	p.batchStop = make(chan struct{})
	p.batchDone = make(chan struct{})
	p.batchStarted = true
	go p.runBatchWorker(p.batchStop, p.batchDone)
}

func (p *Publisher) runBatchWorker(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	for {
		now := p.now().UTC()
		_, next := alignedWindow(now, p.cfg.BatchResourceEvents.Window)
		delay := next.Sub(now)
		if delay <= 0 {
			delay = time.Nanosecond
		}
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			p.flushBatches(p.now().UTC(), false)
		case <-stop:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
	}
}

// Close stops batch aggregation and flushes any non-empty partial window. It
// is safe to call repeatedly and must run before the MQTT transport disconnects.
func (p *Publisher) Close() {
	if p == nil {
		return
	}
	p.closeOnce.Do(func() {
		p.batchMu.Lock()
		p.batchClosed = true
		started := p.batchStarted
		stop := p.batchStop
		done := p.batchDone
		p.batchMu.Unlock()

		if started {
			close(stop)
			<-done
		}
		p.flushBatches(p.now().UTC(), true)
	})
}

func (p *Publisher) flushBatches(at time.Time, includePartial bool) {
	at = at.UTC()
	p.batchMu.Lock()
	ready := make([]*batch, 0, len(p.batches))
	for key, entry := range p.batches {
		if includePartial || !entry.windowEnd.After(at) {
			ready = append(ready, entry)
			delete(p.batches, key)
		}
	}
	p.batchMu.Unlock()

	sort.Slice(ready, func(i, j int) bool {
		if ready[i].topic != ready[j].topic {
			return ready[i].topic < ready[j].topic
		}
		if ready[i].key.operation != ready[j].key.operation {
			return ready[i].key.operation < ready[j].key.operation
		}
		return ready[i].key.windowStart.Before(ready[j].key.windowStart)
	})
	for _, entry := range ready {
		windowEnd := entry.windowEnd
		if includePartial && at.Before(windowEnd) {
			windowEnd = at
		}
		p.publishBatch(entry, windowEnd, at)
	}
}

func (p *Publisher) publishBatch(entry *batch, windowEnd, publishedAt time.Time) {
	if entry == nil || entry.count == 0 || !p.connected() {
		return
	}
	event := CloudEvent{
		SpecVersion:     cloudEventsVersion,
		Type:            fmt.Sprintf("org.ogc.api.consys.%s.%s", entry.key.resourceType, entry.key.operation),
		Source:          p.apiRoot,
		Subject:         p.absoluteURL(entry.collectionPath),
		ID:              p.newID(),
		ParentID:        p.absoluteURL(entry.key.parentPath),
		Time:            publishedAt.Format(time.RFC3339Nano),
		DataContentType: "application/json",
		Data: map[string]any{
			"timerange": []string{
				entry.key.windowStart.Format(time.RFC3339Nano),
				windowEnd.Format(time.RFC3339Nano),
			},
			"count": entry.count,
		},
	}
	payload, err := json.Marshal(event)
	if err != nil {
		p.logger.Error("Failed to marshal Pub/Sub Batch Resource Event", zap.Error(err))
		return
	}
	p.transport.Publish(entry.topic, payload)
}

func topicForRESTPath(path, suffix string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if parsed, err := url.Parse(path); err == nil {
		path = parsed.Path
	}
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	return path + suffix
}
