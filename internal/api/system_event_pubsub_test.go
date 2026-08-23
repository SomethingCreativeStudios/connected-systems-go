package api

import (
	"encoding/json"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/pubsub"
	"go.uber.org/zap"
)

type systemEventRecordingTransport struct {
	messages []systemEventPublishedMessage
}

type systemEventPublishedMessage struct {
	topic   string
	payload []byte
}

func (t *systemEventRecordingTransport) IsConnected() bool { return true }

func (t *systemEventRecordingTransport) Publish(topic string, payload []byte) {
	t.messages = append(t.messages, systemEventPublishedMessage{
		topic:   topic,
		payload: append([]byte(nil), payload...),
	})
}

func TestSystemEventLifecycleNormalizesLabelIntoResourceEventSummary(t *testing.T) {
	transport := &systemEventRecordingTransport{}
	publisher := pubsub.NewPublisher("https://example.org/api", config.PubSubConfig{
		ResourceEvents: config.PubSubFeatureConfig{Enabled: true},
	}, transport, zap.NewNop())
	handler := &SystemEventHandler{pubSubPublisher: publisher}

	handler.publishSystemEventLifecycle("sys-1", &domains.SystemEvent{
		Base:        domains.Base{ID: "event-1"},
		Label:       "Calibration complete",
		Description: "The sensor calibration completed successfully",
	}, pubsub.OperationCreate)

	if len(transport.messages) != 2 {
		t.Fatalf("expected collection and resource publications, got %d", len(transport.messages))
	}
	if transport.messages[0].topic != "systems/sys-1/events:events" {
		t.Fatalf("unexpected collection topic %q", transport.messages[0].topic)
	}
	var event pubsub.CloudEvent
	if err := json.Unmarshal(transport.messages[0].payload, &event); err != nil {
		t.Fatalf("decode CloudEvent: %v", err)
	}
	if event.DataContentType != "application/json" {
		t.Fatalf("expected JSON datacontenttype, got %q", event.DataContentType)
	}
	if event.Data["name"] != "Calibration complete" {
		t.Fatalf("expected label normalized to name, got %#v", event.Data)
	}
	if event.Data["description"] != "The sensor calibration completed successfully" {
		t.Fatalf("expected description in summary, got %#v", event.Data)
	}
}
