package json_formatters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

// DecodeCommandStatusReport decodes the native JSON representation of a
// command status report. requireComplete is used by Pub/Sub Resource Data,
// where read-only response fields must also be present on the wire.
func DecodeCommandStatusReport(reader io.Reader, requireComplete bool) (*domains.CommandStatusReport, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}

	statusCode, err := requiredStringField(fields, "statusCode")
	if err != nil {
		return nil, err
	}
	if !domains.CommandStatus(statusCode).IsValid() {
		return nil, fmt.Errorf("statusCode is invalid")
	}

	if reportTimeRaw, exists := fields["reportTime"]; exists {
		var reportTime string
		if err := json.Unmarshal(reportTimeRaw, &reportTime); err != nil || reportTime == "" {
			return nil, fmt.Errorf("reportTime must be an ISO 8601 string")
		}
		if _, err := time.Parse(time.RFC3339, reportTime); err != nil {
			return nil, fmt.Errorf("reportTime must be an ISO 8601 string")
		}
	} else if requireComplete {
		return nil, fmt.Errorf("reportTime is required for a complete Resource Data Message")
	}

	if requireComplete {
		if _, err := requiredStringField(fields, "id"); err != nil {
			return nil, fmt.Errorf("id is required for a complete Resource Data Message")
		}
		if _, err := requiredStringField(fields, "command@id"); err != nil {
			return nil, fmt.Errorf("command@id is required for a complete Resource Data Message")
		}
	}

	status, err := common_shared.DecodeWithFieldErrors[domains.CommandStatusReport](raw)
	if err != nil {
		return nil, err
	}
	if percentRaw, exists := fields["percentCompletion"]; exists {
		if bytes.Equal(bytes.TrimSpace(percentRaw), []byte("null")) {
			return nil, fmt.Errorf("percentCompletion must be a number")
		}
		if status.PercentCompletion == nil {
			return nil, fmt.Errorf("percentCompletion must be a number")
		}
		if *status.PercentCompletion < 0 || *status.PercentCompletion > 100 {
			return nil, fmt.Errorf("percentCompletion must be between 0 and 100")
		}
	}
	if executionTimeRaw, exists := fields["executionTime"]; exists && bytes.Equal(bytes.TrimSpace(executionTimeRaw), []byte("null")) {
		return nil, fmt.Errorf("executionTime must be a valid time period")
	}
	if messageRaw, exists := fields["message"]; exists {
		var message string
		if err := json.Unmarshal(messageRaw, &message); err != nil || message == "" {
			return nil, fmt.Errorf("message must be a non-empty string")
		}
	}
	if resultsRaw, exists := fields["results"]; exists {
		var results []json.RawMessage
		if bytes.Equal(bytes.TrimSpace(resultsRaw), []byte("null")) {
			return nil, fmt.Errorf("results must be an array")
		}
		if err := json.Unmarshal(resultsRaw, &results); err != nil {
			return nil, fmt.Errorf("results must be an array")
		}
	}

	status.ExecutionTime = common_shared.NonEmptyTimeRange(status.ExecutionTime)
	return &status, nil
}

func requiredStringField(fields map[string]json.RawMessage, name string) (string, error) {
	raw, exists := fields[name]
	if !exists {
		return "", fmt.Errorf("%s is required", name)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}
