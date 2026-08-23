package pubsub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"go.uber.org/zap"
)

type fixedResolver struct {
	change Change
	paths  []string
}

func (r *fixedResolver) Resolve(path string, operation Operation) (Change, error) {
	r.paths = append(r.paths, path)
	change := r.change
	change.Operation = operation
	return change, nil
}

func TestHTTPResourceEventMiddlewarePublishesAfterCreate(t *testing.T) {
	transport := &recordingTransport{connected: true}
	publisher := NewPublisher("https://example.org/api", enabledConfig(), transport, zap.NewNop())
	resolver := &fixedResolver{change: Change{
		ResourceType: "system", ResourceID: "sys-1", SubjectPath: "/systems/sys-1", CollectionPath: "/systems",
		Data: BuildResourceEventSummary("Created System", "Created description", "urn:example:created"),
	}}

	handler := HTTPResourceEventMiddleware(publisher, resolver, zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://example.org/api/systems/sys-1")
		w.WriteHeader(http.StatusCreated)
	}))
	req := httptest.NewRequest(http.MethodPost, "/systems", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if len(resolver.paths) != 1 || resolver.paths[0] != "https://example.org/api/systems/sys-1" {
		t.Fatalf("expected post-commit Location resolution, got %#v", resolver.paths)
	}
	if len(transport.messages) != 2 {
		t.Fatalf("expected Resource Event publications, got %d", len(transport.messages))
	}
	var event CloudEvent
	if err := json.Unmarshal(transport.messages[0].payload, &event); err != nil {
		t.Fatalf("decode CloudEvent: %v", err)
	}
	if event.Data["name"] != "Created System" || event.DataContentType != "application/json" {
		t.Fatalf("expected post-commit create summary, got %#v", event)
	}
}

func TestHTTPResourceEventMiddlewareResolvesUpdatedSummaryAfterHandler(t *testing.T) {
	transport := &recordingTransport{connected: true}
	publisher := NewPublisher("https://example.org/api", enabledConfig(), transport, zap.NewNop())
	resolver := &fixedResolver{change: Change{
		ResourceType: "system", ResourceID: "sys-1", SubjectPath: "/systems/sys-1", CollectionPath: "/systems",
	}}

	handler := HTTPResourceEventMiddleware(publisher, resolver, zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(resolver.paths) != 0 {
			t.Fatal("update must resolve the committed resource after the handler")
		}
		resolver.change.Data = BuildResourceEventSummary("Updated System", "Updated description", "urn:example:updated")
		w.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPut, "/systems/sys-1", nil))

	if len(transport.messages) != 2 {
		t.Fatalf("expected Resource Event publications, got %d", len(transport.messages))
	}
	var event CloudEvent
	if err := json.Unmarshal(transport.messages[0].payload, &event); err != nil {
		t.Fatalf("decode CloudEvent: %v", err)
	}
	if event.Data["name"] != "Updated System" || event.Data["uniqueId"] != "urn:example:updated" {
		t.Fatalf("expected post-commit update summary, got %#v", event.Data)
	}
}

func TestHTTPResourceEventMiddlewareResolvesDeleteBeforeHandler(t *testing.T) {
	transport := &recordingTransport{connected: true}
	publisher := NewPublisher("https://example.org/api", enabledConfig(), transport, zap.NewNop())
	resolver := &fixedResolver{change: Change{
		ResourceType: "observation", ResourceID: "obs-1", SubjectPath: "/observations/obs-1",
		ParentPath: "/datastreams/ds-1", CollectionPath: "/datastreams/ds-1/observations",
	}}

	handler := HTTPResourceEventMiddleware(publisher, resolver, zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(resolver.paths) != 1 {
			t.Fatal("delete must resolve the resource before it is removed")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodDelete, "/observations/obs-1", nil))
	publisher.Close()

	if len(transport.messages) != 1 || transport.messages[0].topic != "datastreams/ds-1/observations:batch-events" {
		t.Fatalf("expected one batched delete Resource Event, got %#v", transport.messages)
	}
}

func TestHTTPResourceEventMiddlewarePublishesPreDeleteSummary(t *testing.T) {
	transport := &recordingTransport{connected: true}
	publisher := NewPublisher("https://example.org/api", enabledConfig(), transport, zap.NewNop())
	resolver := &fixedResolver{change: Change{
		ResourceType: "system", ResourceID: "sys-1", SubjectPath: "/systems/sys-1", CollectionPath: "/systems",
		Data: BuildResourceEventSummary("Deleted System", "Deleted description", "urn:example:deleted"),
	}}

	handler := HTTPResourceEventMiddleware(publisher, resolver, zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(resolver.paths) != 1 {
			t.Fatal("delete must resolve the summary before the resource is removed")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodDelete, "/systems/sys-1", nil))

	if len(transport.messages) != 2 {
		t.Fatalf("expected Resource Event publications, got %d", len(transport.messages))
	}
	var event CloudEvent
	if err := json.Unmarshal(transport.messages[0].payload, &event); err != nil {
		t.Fatalf("decode CloudEvent: %v", err)
	}
	if event.Data["name"] != "Deleted System" || event.Data["description"] != "Deleted description" {
		t.Fatalf("expected pre-delete summary, got %#v", event.Data)
	}
}

func TestSummaryForResolvedResourceCoversDescriptiveResources(t *testing.T) {
	common := domains.CommonSSN{
		Name:             "Resource name",
		Description:      "Resource description",
		UniqueIdentifier: domains.UniqueID("urn:example:resource"),
	}
	withUniqueID := map[string]any{
		"name":        "Resource name",
		"description": "Resource description",
		"uniqueId":    "urn:example:resource",
	}
	withoutUniqueID := map[string]any{
		"name":        "Resource name",
		"description": "Resource description",
	}
	tests := []struct {
		name     string
		resource any
		want     map[string]any
	}{
		{name: "system and subsystem", resource: &domains.System{CommonSSN: common}, want: withUniqueID},
		{name: "deployment", resource: &domains.Deployment{CommonSSN: common}, want: withUniqueID},
		{name: "procedure", resource: &domains.Procedure{CommonSSN: common}, want: withUniqueID},
		{name: "property", resource: &domains.Property{CommonSSN: common}, want: withUniqueID},
		{name: "sampling feature", resource: &domains.SamplingFeature{CommonSSN: common}, want: withUniqueID},
		{name: "control stream", resource: &domains.ControlStream{CommonSSN: common}, want: withUniqueID},
		{name: "datastream", resource: &domains.Datastream{Name: common.Name, Description: common.Description}, want: withoutUniqueID},
		{name: "system event label normalized", resource: &domains.SystemEvent{Label: common.Name, Description: common.Description}, want: withoutUniqueID},
		{name: "observation has no summary", resource: &domains.Observation{}, want: nil},
		{name: "command has no summary", resource: &domains.Command{}, want: nil},
		{name: "command status has no summary", resource: &domains.CommandStatusReport{}, want: nil},
		{name: "command result has no summary", resource: &domains.CommandResult{}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summaryForResolvedResource(tt.resource)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %#v, got %#v", tt.want, got)
			}
		})
	}
}

func TestHTTPResourceEventMiddlewareSkipsFailedMutation(t *testing.T) {
	transport := &recordingTransport{connected: true}
	publisher := NewPublisher("https://example.org/api", enabledConfig(), transport, zap.NewNop())
	resolver := &fixedResolver{change: Change{
		ResourceType: "system", ResourceID: "sys-1", SubjectPath: "/systems/sys-1", CollectionPath: "/systems",
	}}
	handler := HTTPResourceEventMiddleware(publisher, resolver, zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPut, "/systems/sys-1", nil))
	if len(transport.messages) != 0 {
		t.Fatalf("failed mutation published %d messages", len(transport.messages))
	}
}
