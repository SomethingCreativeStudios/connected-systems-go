package json_formatters

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

func TestDatastreamJSONSerialize_EmptyTimeRangesOmitted(t *testing.T) {
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
	for _, field := range []string{"validTime", "phenomenonTime", "resultTime"} {
		if strings.Contains(body, field) {
			t.Fatalf("expected %s to be omitted, got %s", field, body)
		}
	}
}

func TestControlStreamJSONSerialize_EmptyTimeRangesOmitted(t *testing.T) {
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
	for _, field := range []string{"validTime", "issueTime", "executionTime"} {
		if strings.Contains(body, field) {
			t.Fatalf("expected %s to be omitted, got %s", field, body)
		}
	}
}
