package api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

func TestCommandResponse_EmptyExecutionTimeOmitted(t *testing.T) {
	cmd := &domains.Command{
		Base:          domains.Base{ID: "cmd-1"},
		ExecutionTime: &common_shared.TimeRange{},
	}

	data, err := json.Marshal(commandResponse(cmd))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(data), "executionTime") {
		t.Fatalf("expected executionTime to be omitted, got %s", data)
	}
}

func TestCommandStatusResponse_EmptyExecutionTimeOmitted(t *testing.T) {
	status := &domains.CommandStatusReport{
		Base:          domains.Base{ID: "status-1"},
		ReportTime:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		StatusCode:    domains.CommandStatusAccepted,
		ExecutionTime: &common_shared.TimeRange{},
	}

	data, err := json.Marshal(commandStatusResponse(status))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(data), "executionTime") {
		t.Fatalf("expected executionTime to be omitted, got %s", data)
	}
}
