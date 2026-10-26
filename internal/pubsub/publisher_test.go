package pubsub

import (
	"encoding/json"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"go.uber.org/zap"
)

type publishedMessage struct {
	topic   string
	payload []byte
}

type recordingTransport struct {
	connected bool
	messages  []publishedMessage
}

type signalingTransport struct {
	messages chan publishedMessage
}

func (t *signalingTransport) IsConnected() bool { return true }
func (t *signalingTransport) Publish(topic string, payload []byte, _ string) {
	t.messages <- publishedMessage{topic: topic, payload: append([]byte(nil), payload...)}
}

func (t *recordingTransport) IsConnected() bool { return t.connected }
func (t *recordingTransport) Publish(topic string, payload []byte, _ string) {
	t.messages = append(t.messages, publishedMessage{topic: topic, payload: append([]byte(nil), payload...)})
}

func enabledConfig() config.PubSubConfig {
	return config.PubSubConfig{
		ResourceData:        config.PubSubFeatureConfig{Enabled: true},
		ResourceEvents:      config.PubSubFeatureConfig{Enabled: true},
		BatchResourceEvents: config.BatchResourceEventsConfig{Enabled: true, Window: time.Minute},
	}
}

func TestPublisherResourceEventCloudEventsShapeAndTopics(t *testing.T) {
	transport := &recordingTransport{connected: true}
	publisher := NewPublisher("https://example.org/api/", "node1", enabledConfig(), transport, zap.NewNop())
	publisher.now = func() time.Time { return time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC) }
	publisher.newID = func() string { return "event-1" }

	publisher.PublishResourceEvent(Change{TopicPath: "systems/sys-1/datastreams/ds-1/observations/obs-1", CollectionTopicPath: "systems/sys-1/datastreams/ds-1/observations",
		ResourceType:   "observation",
		ResourceID:     "obs-1",
		Operation:      OperationCreate,
		SubjectPath:    "/observations/obs-1",
		ParentPath:     "/datastreams/ds-1",
		CollectionPath: "/datastreams/ds-1/observations",
	})

	if len(transport.messages) != 1 {
		t.Fatalf("expected designated resource publication, got %d", len(transport.messages))
	}
	wantTopics := []string{"node1/systems/sys-1/datastreams/ds-1/observations/obs-1"}
	for i, want := range wantTopics {
		if transport.messages[i].topic != want {
			t.Fatalf("topic %d: expected %q, got %q", i, want, transport.messages[i].topic)
		}
	}

	var event CloudEvent
	if err := json.Unmarshal(transport.messages[0].payload, &event); err != nil {
		t.Fatalf("decode CloudEvent: %v", err)
	}
	if event.SpecVersion != "1.0" || event.Type != "org.ogc.api.csapi.observation.create" {
		t.Fatalf("unexpected CloudEvent identity: %#v", event)
	}
	if event.Source != "https://example.org/api" || event.Subject != "https://example.org/api/observations/obs-1" {
		t.Fatalf("unexpected CloudEvent URLs: %#v", event)
	}
	if event.ParentID != "https://example.org/api/datastreams/ds-1" || event.ID != "event-1" {
		t.Fatalf("unexpected parent/id: %#v", event)
	}
	if event.DataContentType != "" || event.Data != nil {
		t.Fatalf("minimal event should omit data fields: %#v", event)
	}
}

func TestPublisherResourceEventIncludesSummaryAndJSONContentType(t *testing.T) {
	transport := &recordingTransport{connected: true}
	publisher := NewPublisher("https://example.org/api", "node1", enabledConfig(), transport, zap.NewNop())
	publisher.newID = func() string { return "event-with-summary" }

	publisher.PublishResourceEvent(Change{TopicPath: "systems/sys-1", CollectionTopicPath: "systems",
		ResourceType:   "system",
		ResourceID:     "sys-1",
		Operation:      OperationUpdate,
		SubjectPath:    "/systems/sys-1",
		CollectionPath: "/systems",
		Data: BuildResourceEventSummary(
			"Temperature Sensor 01",
			"Air temperature sensor on the north wall",
			"urn:example:sensor:01",
		),
	})

	if len(transport.messages) != 1 {
		t.Fatalf("expected designated resource publication, got %d", len(transport.messages))
	}
	var event CloudEvent
	if err := json.Unmarshal(transport.messages[0].payload, &event); err != nil {
		t.Fatalf("decode CloudEvent: %v", err)
	}
	if event.DataContentType != "application/json" {
		t.Fatalf("expected JSON datacontenttype, got %q", event.DataContentType)
	}
	want := map[string]any{
		"name":        "Temperature Sensor 01",
		"description": "Air temperature sensor on the north wall",
		"uniqueId":    "urn:example:sensor:01",
	}
	if !reflect.DeepEqual(event.Data, want) {
		t.Fatalf("expected summary %#v, got %#v", want, event.Data)
	}
}

func TestBuildResourceEventSummaryOmitsEmptyValues(t *testing.T) {
	if got := BuildResourceEventSummary("", " \t", ""); got != nil {
		t.Fatalf("expected nil summary for empty values, got %#v", got)
	}

	got := BuildResourceEventSummary("Datastream 01", "", "")
	want := map[string]any{"name": "Datastream 01"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected summary %#v, got %#v", want, got)
	}
}

func TestResourceEventTopicsFollowCanonicalRESTPaths(t *testing.T) {
	tests := []struct {
		name   string
		change Change
		want   []string
	}{
		{
			name:   "system",
			change: Change{TopicPath: "systems/sys-1", CollectionTopicPath: "systems", CollectionPath: "/systems", SubjectPath: "/systems/sys-1"},
			want:   []string{"systems/sys-1"},
		},
		{
			name:   "command status",
			change: Change{TopicPath: "systems/sys-1/controlstreams/cs-1/commands/cmd-1/status/status-1", CollectionTopicPath: "systems/sys-1/controlstreams/cs-1/commands/cmd-1/status", CollectionPath: "/commands/cmd-1/status", SubjectPath: "/commands/cmd-1/status/status-1"},
			want:   []string{"systems/sys-1/controlstreams/cs-1/commands/cmd-1/status/status-1"},
		},
		{
			name:   "system event",
			change: Change{TopicPath: "systems/sys-1/events/event-1", CollectionTopicPath: "systems/sys-1/events", CollectionPath: "/systems/sys-1/events", SubjectPath: "/systems/sys-1/events/event-1"},
			want:   []string{"systems/sys-1/events/event-1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResourceEventTopics(tt.change)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %#v, got %#v", tt.want, got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("topic %d: expected %q, got %q", i, tt.want[i], got[i])
				}
			}
		})
	}
}

func TestPublisherClassSwitchesAreIndependent(t *testing.T) {
	transport := &recordingTransport{connected: true}
	cfg := enabledConfig()
	cfg.ResourceEvents.Enabled = false
	publisher := NewPublisher("https://example.org/api", "node1", cfg, transport, zap.NewNop())

	publisher.PublishResourceEvent(Change{TopicPath: "systems/sys-1", CollectionTopicPath: "systems",
		ResourceType: "system", ResourceID: "sys-1", Operation: OperationDelete,
		SubjectPath: "/systems/sys-1", CollectionPath: "/systems",
	})
	publisher.PublishChange(Change{ResourceType: "system", ResourceID: "sys-1", Operation: OperationCreate, SubjectPath: "/systems/sys-1", TopicPath: "systems/sys-1", CollectionTopicPath: "systems", Representations: []Representation{{MediaType: "application/geo+json", Payload: []byte(`{"id":"sys-1"}`)}}})

	if len(transport.messages) != 1 || transport.messages[0].topic != "node1/systems:data" {
		t.Fatalf("expected only Resource Data publication, got %#v", transport.messages)
	}
}

func TestPublisherRequiresConnectedTransport(t *testing.T) {
	transport := &recordingTransport{}
	publisher := NewPublisher("https://example.org/api", "node1", enabledConfig(), transport, zap.NewNop())
	publisher.PublishChange(Change{ResourceType: "system", ResourceID: "sys-1", Operation: OperationCreate, SubjectPath: "/systems/sys-1", TopicPath: "systems/sys-1", CollectionTopicPath: "systems", Representations: []Representation{{MediaType: "application/geo+json", Payload: []byte(`{"id":"sys-1"}`)}}})
	if len(transport.messages) != 0 {
		t.Fatalf("expected no publications while disconnected, got %#v", transport.messages)
	}
}

func TestPublisherBatchesObservationChangesByWindowCollectionAndOperation(t *testing.T) {
	transport := &recordingTransport{connected: true}
	publisher := NewPublisher("https://example.org/api", "node1", enabledConfig(), transport, zap.NewNop())
	publisher.now = func() time.Time { return time.Date(2026, 8, 22, 12, 0, 30, 0, time.UTC) }
	publisher.newID = func() string { return "batch-1" }

	for range 2 {
		publisher.PublishChange(Change{TopicPath: "systems/sys-1/datastreams/ds-1/observations/obs-1", CollectionTopicPath: "systems/sys-1/datastreams/ds-1/observations",
			ResourceType: "observation", ResourceID: "obs-1", Operation: OperationCreate,
			SubjectPath: "/observations/obs-1", ParentPath: "/datastreams/ds-1",
			CollectionPath: "/datastreams/ds-1/observations",
		})
	}
	publisher.PublishChange(Change{TopicPath: "systems/sys-1/datastreams/ds-1/observations/obs-1", CollectionTopicPath: "systems/sys-1/datastreams/ds-1/observations",
		ResourceType: "observation", ResourceID: "obs-1", Operation: OperationUpdate,
		SubjectPath: "/observations/obs-1", ParentPath: "/datastreams/ds-1",
		CollectionPath: "/datastreams/ds-1/observations",
	})
	publisher.PublishChange(Change{TopicPath: "systems/sys-1/datastreams/ds-2/observations/obs-2", CollectionTopicPath: "systems/sys-1/datastreams/ds-2/observations",
		ResourceType: "observation", ResourceID: "obs-2", Operation: OperationCreate,
		SubjectPath: "/observations/obs-2", ParentPath: "/datastreams/ds-2",
		CollectionPath: "/datastreams/ds-2/observations",
	})

	if len(transport.messages) != 0 {
		t.Fatalf("batch changes published before the window closed: %#v", transport.messages)
	}
	publisher.flushBatches(time.Date(2026, 8, 22, 12, 1, 0, 0, time.UTC), false)
	publisher.Close()

	if len(transport.messages) != 3 {
		t.Fatalf("expected three separately keyed batches, got %d", len(transport.messages))
	}
	var create CloudEvent
	if err := json.Unmarshal(transport.messages[0].payload, &create); err != nil {
		t.Fatalf("decode batch event: %v", err)
	}
	if transport.messages[0].topic != "node1/systems/sys-1/datastreams/ds-1/observations" {
		t.Fatalf("unexpected batch topic %q", transport.messages[0].topic)
	}
	if create.Type != "org.ogc.api.csapi.observation.create" || create.Subject != "https://example.org/api/datastreams/ds-1/observations" {
		t.Fatalf("unexpected batch identity: %#v", create)
	}
	if create.ParentID != "https://example.org/api/datastreams/ds-1" || create.DataContentType != "application/json" {
		t.Fatalf("unexpected batch context: %#v", create)
	}
	if got := create.Data["count"]; got != float64(2) {
		t.Fatalf("expected count 2, got %#v", got)
	}
	wantRange := []any{"2026-08-22T12:00:00Z", "2026-08-22T12:01:00Z"}
	if got := create.Data["timerange"]; !equalAnySlices(got, wantRange) {
		t.Fatalf("expected timerange %#v, got %#v", wantRange, got)
	}
}

func TestPublisherBatchModeReplacesOnlyObservationAndCommandEvents(t *testing.T) {
	transport := &recordingTransport{connected: true}
	publisher := NewPublisher("https://example.org/api", "node1", enabledConfig(), transport, zap.NewNop())
	publisher.now = func() time.Time { return time.Date(2026, 8, 22, 12, 0, 30, 0, time.UTC) }

	publisher.PublishChange(Change{TopicPath: "systems/sys-1/controlstreams/cs-1/commands/cmd-1", CollectionTopicPath: "systems/sys-1/controlstreams/cs-1/commands",
		ResourceType: "command", ResourceID: "cmd-1", Operation: OperationDelete,
		SubjectPath: "/commands/cmd-1", ParentPath: "/controlstreams/cs-1",
		CollectionPath: "/controlstreams/cs-1/commands",
	})
	publisher.PublishChange(Change{TopicPath: "systems/sys-1/controlstreams/cs-1/commands/cmd-1/status/status-1", CollectionTopicPath: "systems/sys-1/controlstreams/cs-1/commands/cmd-1/status",
		ResourceType: "commandstatus", ResourceID: "status-1", Operation: OperationCreate,
		SubjectPath: "/commands/cmd-1/status/status-1", ParentPath: "/commands/cmd-1",
		CollectionPath: "/commands/cmd-1/status",
	})

	if len(transport.messages) != 1 {
		t.Fatalf("expected only the immediate command-status event, got %d messages", len(transport.messages))
	}
	publisher.Close()
	if len(transport.messages) != 2 || transport.messages[1].topic != "node1/systems/sys-1/controlstreams/cs-1/commands" {
		t.Fatalf("expected one command batch after close, got %#v", transport.messages)
	}
}

func TestPublisherFallsBackToIndividualEventWhenBatchingIsDisabled(t *testing.T) {
	transport := &recordingTransport{connected: true}
	cfg := enabledConfig()
	cfg.BatchResourceEvents.Enabled = false
	publisher := NewPublisher("https://example.org/api", "node1", cfg, transport, zap.NewNop())

	publisher.PublishChange(Change{TopicPath: "systems/sys-1/datastreams/ds-1/observations/obs-1", CollectionTopicPath: "systems/sys-1/datastreams/ds-1/observations",
		ResourceType: "observation", ResourceID: "obs-1", Operation: OperationCreate,
		SubjectPath: "/observations/obs-1", ParentPath: "/datastreams/ds-1",
		CollectionPath: "/datastreams/ds-1/observations",
	})

	if len(transport.messages) != 1 {
		t.Fatalf("expected individual Resource Event scopes, got %d", len(transport.messages))
	}
}

func TestPublisherBatchingRequiresResourceEvents(t *testing.T) {
	transport := &recordingTransport{connected: true}
	cfg := enabledConfig()
	cfg.ResourceEvents.Enabled = false
	publisher := NewPublisher("https://example.org/api", "node1", cfg, transport, zap.NewNop())
	publisher.now = func() time.Time { return time.Date(2026, 8, 22, 12, 0, 30, 0, time.UTC) }

	publisher.PublishChange(Change{TopicPath: "systems/sys-1/datastreams/ds-1/observations/obs-1", CollectionTopicPath: "systems/sys-1/datastreams/ds-1/observations",
		ResourceType: "observation", ResourceID: "obs-1", Operation: OperationCreate,
		SubjectPath: "/observations/obs-1", ParentPath: "/datastreams/ds-1",
		CollectionPath: "/datastreams/ds-1/observations",
	})
	publisher.PublishChange(Change{TopicPath: "systems/sys-1", CollectionTopicPath: "systems",
		ResourceType: "system", ResourceID: "sys-1", Operation: OperationCreate,
		SubjectPath: "/systems/sys-1", CollectionPath: "/systems",
	})
	publisher.Close()

	if len(transport.messages) != 0 {
		t.Fatalf("batching must require Resource Events, got %d messages", len(transport.messages))
	}
}

func TestPublisherCloseFlushesPartialWindowOnce(t *testing.T) {
	transport := &recordingTransport{connected: true}
	publisher := NewPublisher("https://example.org/api", "node1", enabledConfig(), transport, zap.NewNop())
	var now atomic.Int64
	now.Store(time.Date(2026, 8, 22, 12, 0, 15, 0, time.UTC).UnixNano())
	publisher.now = func() time.Time { return time.Unix(0, now.Load()).UTC() }
	publisher.PublishChange(Change{TopicPath: "systems/sys-1/controlstreams/cs-1/commands/cmd-1", CollectionTopicPath: "systems/sys-1/controlstreams/cs-1/commands",
		ResourceType: "command", ResourceID: "cmd-1", Operation: OperationCreate,
		SubjectPath: "/commands/cmd-1", ParentPath: "/controlstreams/cs-1",
		CollectionPath: "/controlstreams/cs-1/commands",
	})
	now.Store(time.Date(2026, 8, 22, 12, 0, 40, 0, time.UTC).UnixNano())

	publisher.Close()
	publisher.Close()

	if len(transport.messages) != 1 {
		t.Fatalf("expected one idempotent shutdown flush, got %d", len(transport.messages))
	}
	var event CloudEvent
	if err := json.Unmarshal(transport.messages[0].payload, &event); err != nil {
		t.Fatalf("decode batch event: %v", err)
	}
	wantRange := []any{"2026-08-22T12:00:00Z", "2026-08-22T12:00:40Z"}
	if got := event.Data["timerange"]; !equalAnySlices(got, wantRange) {
		t.Fatalf("expected partial timerange %#v, got %#v", wantRange, got)
	}
}

func TestPublisherAggregatesConcurrentChanges(t *testing.T) {
	transport := &recordingTransport{connected: true}
	publisher := NewPublisher("https://example.org/api", "node1", enabledConfig(), transport, zap.NewNop())
	publisher.now = func() time.Time { return time.Date(2026, 8, 22, 12, 0, 30, 0, time.UTC) }

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			publisher.PublishChange(Change{TopicPath: "systems/sys-1/datastreams/ds-1/observations/obs", CollectionTopicPath: "systems/sys-1/datastreams/ds-1/observations",
				ResourceType: "observation", ResourceID: "obs", Operation: OperationCreate,
				SubjectPath: "/observations/obs", ParentPath: "/datastreams/ds-1",
				CollectionPath: "/datastreams/ds-1/observations",
			})
		})
	}
	wg.Wait()
	publisher.Close()

	if len(transport.messages) != 1 {
		t.Fatalf("expected one concurrent batch, got %d", len(transport.messages))
	}
	var event CloudEvent
	if err := json.Unmarshal(transport.messages[0].payload, &event); err != nil {
		t.Fatalf("decode batch event: %v", err)
	}
	if got := event.Data["count"]; got != float64(100) {
		t.Fatalf("expected count 100, got %#v", got)
	}
}

// The window timer runs on synctest's fake clock, so a realistic one-minute
// window closes as soon as every goroutine is waiting on it.
func TestPublisherAutomaticallyFlushesAtWindowBoundary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		transport := &signalingTransport{messages: make(chan publishedMessage, 1)}
		publisher := NewPublisher("https://example.org/api", "node1", enabledConfig(), transport, zap.NewNop())
		defer publisher.Close()

		publisher.PublishChange(Change{TopicPath: "systems/sys-1/datastreams/ds-1/observations/obs-1", CollectionTopicPath: "systems/sys-1/datastreams/ds-1/observations",
			ResourceType: "observation", ResourceID: "obs-1", Operation: OperationCreate,
			SubjectPath: "/observations/obs-1", ParentPath: "/datastreams/ds-1",
			CollectionPath: "/datastreams/ds-1/observations",
		})

		select {
		case message := <-transport.messages:
			if message.topic != "node1/systems/sys-1/datastreams/ds-1/observations" {
				t.Fatalf("unexpected automatic batch topic %q", message.topic)
			}
		case <-time.After(2 * time.Minute):
			t.Fatal("batch was not published after its window closed")
		}
	})
}

func TestPublisherCloseDoesNotPublishEmptyBatch(t *testing.T) {
	transport := &recordingTransport{connected: true}
	publisher := NewPublisher("https://example.org/api", "node1", enabledConfig(), transport, zap.NewNop())
	publisher.Close()
	if len(transport.messages) != 0 {
		t.Fatalf("empty publisher emitted %#v", transport.messages)
	}
}

func TestAlignedWindowUsesUTCClockBoundaries(t *testing.T) {
	start, end := alignedWindow(time.Date(2026, 8, 22, 12, 0, 59, 999, time.FixedZone("offset", -4*60*60)), time.Minute)
	if start.Format(time.RFC3339Nano) != "2026-08-22T16:00:00Z" || end.Format(time.RFC3339Nano) != "2026-08-22T16:01:00Z" {
		t.Fatalf("unexpected aligned window %s - %s", start, end)
	}
}

func equalAnySlices(got any, want []any) bool {
	values, ok := got.([]any)
	if !ok || len(values) != len(want) {
		return false
	}
	for i := range want {
		if values[i] != want[i] {
			return false
		}
	}
	return true
}
