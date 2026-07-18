package json_formatters

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

// Empty validTime is omitted; empty phenomenonTime/resultTime serialize as
// explicit null (required, read-only, nullable per spec).
func TestDatastreamJSONSerialize_EmptyTimeRanges(t *testing.T) {
	systemID := "sys-1"
	formatter := NewDatastreamJSONFormatter(nil)
	datastream := &domains.Datastream{
		Base:           domains.Base{ID: "ds-1"},
		SystemID:       &systemID,
		ValidTime:      &common_shared.TimeRange{},
		PhenomenonTime: &common_shared.TimeRange{},
		ResultTime:     &common_shared.TimeRange{},
	}

	out, err := formatter.Serialize(context.Background(), datastream)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	body := string(data)
	if strings.Contains(body, "validTime") {
		t.Fatalf("expected validTime to be omitted, got %s", body)
	}
	for _, field := range []string{`"phenomenonTime":null`, `"resultTime":null`} {
		if !strings.Contains(body, field) {
			t.Fatalf("expected %s, got %s", field, body)
		}
	}
}

// Empty validTime is omitted; empty issueTime/executionTime serialize as
// explicit null (required, read-only, nullable per spec).
func TestControlStreamJSONSerialize_EmptyTimeRanges(t *testing.T) {
	formatter := NewControlStreamJSONFormatter(nil)
	controlStream := &domains.ControlStream{
		Base:          domains.Base{ID: "cs-1"},
		ValidTime:     &common_shared.TimeRange{},
		IssueTime:     &common_shared.TimeRange{},
		ExecutionTime: &common_shared.TimeRange{},
	}

	out, err := formatter.Serialize(context.Background(), controlStream)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	body := string(data)
	if strings.Contains(body, "validTime") {
		t.Fatalf("expected validTime to be omitted, got %s", body)
	}
	for _, field := range []string{`"issueTime":null`, `"executionTime":null`} {
		if !strings.Contains(body, field) {
			t.Fatalf("expected %s, got %s", field, body)
		}
	}
}
