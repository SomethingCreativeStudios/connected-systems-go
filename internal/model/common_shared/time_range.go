package common_shared

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"
)

// TimeRange represents a time period with start and end
type TimeRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// Value implements driver.Valuer for JSONB storage
func (tr TimeRange) Value() (driver.Value, error) {
	return json.Marshal(tr)
}

// Scan implements sql.Scanner for JSONB retrieval
func (tr *TimeRange) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, tr)
}

// Convert string representation to TimeRange
func ToTimeRange(timeValue string) TimeRange {
	if timeValue == "latest" || timeValue == "../.." || timeValue == ".." || timeValue == "now" {
		now := time.Now().UTC()
		return TimeRange{Start: &now, End: nil}
	}

	parts := strings.Split(timeValue, "/")

	if len(parts) == 2 {
		var startTime, endTime *time.Time

		if parts[0] != "" && parts[0] != ".." {
			t, _ := time.Parse(time.RFC3339, parts[0])
			startTime = &t
		}

		if parts[1] != "" && parts[1] != ".." {
			t, _ := time.Parse(time.RFC3339, parts[1])
			endTime = &t
		}

		return TimeRange{Start: startTime, End: endTime}
	}

	return TimeRange{Start: nil, End: nil}
}
