package common_shared

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TimeRange represents a time period with start and end.
// It is intentionally a plain struct (no driver.Valuer/sql.Scanner)
// so it can be embedded into domain models and have Start/End
// stored as separate DB columns via GORM's `embedded` support.
type TimeRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// MarshalJSON serializes TimeRange as a JSON array [start, end].
// Each element is an RFC3339 string or null when missing.
func (tr TimeRange) MarshalJSON() ([]byte, error) {
	var s interface{}
	var e interface{}
	if tr.Start != nil {
		s = tr.Start.Format(time.RFC3339)
	} else {
		s = nil
	}
	if tr.End != nil {
		e = tr.End.Format(time.RFC3339)
	} else {
		e = nil
	}

	if e == nil {
		return json.Marshal([]interface{}{s})
	}

	return json.Marshal([]interface{}{s, e})
}

// UnmarshalJSON supports multiple input shapes for backwards compatibility:
// - JSON array: [start, end] where elements are RFC3339 strings or null
// - JSON object: {"start":"...","end":"..."}
// - JSON string: "start/end" (existing ToTimeRange string format)
func (tr *TimeRange) UnmarshalJSON(b []byte) error {
	// allow null
	if len(b) == 0 || string(b) == "null" {
		*tr = TimeRange{}
		return nil
	}

	// try array form
	var arr []interface{}
	if err := json.Unmarshal(b, &arr); err == nil {
		if len(arr) > 0 && arr[0] != nil {
			if s, ok := arr[0].(string); ok {
				if t, err := time.Parse(time.RFC3339, s); err == nil {
					tr.Start = &t
				}
			}
		}
		if len(arr) > 1 && arr[1] != nil {
			if s, ok := arr[1].(string); ok {
				if t, err := time.Parse(time.RFC3339, s); err == nil {
					tr.End = &t
				}
			}
		}
		return nil
	}

	// try object form {"start":"...","end":"..."}
	var obj struct {
		Start *string `json:"start,omitempty"`
		End   *string `json:"end,omitempty"`
	}
	if err := json.Unmarshal(b, &obj); err == nil {
		if obj.Start != nil && *obj.Start != "" {
			if t, err := time.Parse(time.RFC3339, *obj.Start); err == nil {
				tr.Start = &t
			}
		}
		if obj.End != nil && *obj.End != "" {
			if t, err := time.Parse(time.RFC3339, *obj.End); err == nil {
				tr.End = &t
			}
		}
		return nil
	}

	// try string form
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*tr = ToTimeRange(s)
		return nil
	}

	return fmt.Errorf("unsupported TimeRange JSON format")
}

// ToTimeRange converts string/time-range expressions (e.g. "2020-01-01T00:00:00Z/2020-02-01T00:00:00Z")
// into a TimeRange. Special values like "now" or "latest" map to a Start = now, End = nil.
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

// ToTimeRangeFromSlice accepts a string slice where index 0 is the start
// and index 1 is the end. Missing elements or empty strings are treated
// as nil. This is useful when query parameters are provided as repeated
// values (e.g. ?dateTime=2020-01-01T00:00:00Z&dateTime=2020-02-01T00:00:00Z).
func ToTimeRangeFromSlice(parts []string) TimeRange {
	if len(parts) == 0 {
		return TimeRange{Start: nil, End: nil}
	}

	var startTime, endTime *time.Time

	if parts[0] != "" && parts[0] != ".." {
		if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
			startTime = &t
		}
	}

	if len(parts) > 1 {
		if parts[1] != "" && parts[1] != ".." {
			if t, err := time.Parse(time.RFC3339, parts[1]); err == nil {
				endTime = &t
			}
		}
	}

	return TimeRange{Start: startTime, End: endTime}
}

// ParseTimeRange accepts a value of various possible shapes and returns a TimeRange.
// Supported inputs:
// - string (e.g. "start/end" or special values)
// - []string where index 0 = start, index 1 = end
// - []interface{} where elements are strings
// - map[string]interface{} with keys "start" and/or "end"
func ParseTimeRange(value interface{}) TimeRange {
	if value == nil {
		return TimeRange{}
	}

	switch v := value.(type) {
	case string:
		return ToTimeRange(v)
	case []string:
		return ToTimeRangeFromSlice(v)
	case []interface{}:
		parts := make([]string, 0, 2)
		for i := 0; i < len(v) && i < 2; i++ {
			if s, ok := v[i].(string); ok {
				parts = append(parts, s)
			} else {
				parts = append(parts, "")
			}
		}
		return ToTimeRangeFromSlice(parts)
	case map[string]interface{}:
		startStr, _ := v["start"].(string)
		endStr, _ := v["end"].(string)
		return ToTimeRange(startStr + "/" + endStr)
	default:
		return TimeRange{}
	}
}
