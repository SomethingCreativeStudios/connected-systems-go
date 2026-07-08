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
//
// Latest signals "most recent record only" semantics (the ?resultTime=latest
// query keyword). When true, Start/End are ignored by the repository and the
// query uses ORDER BY <time_col> DESC LIMIT 1 instead of a WHERE filter.
// Latest is never serialized to JSON and is only meaningful in query params.
type TimeRange struct {
	Start  *time.Time `json:"start,omitempty"`
	End    *time.Time `json:"end,omitempty"`
	Latest bool       `json:"-"`
}

// HasBounds reports whether the range has at least one concrete bound.
func (tr *TimeRange) HasBounds() bool {
	return tr != nil && (tr.Start != nil || tr.End != nil)
}

// NonEmptyTimeRange returns nil for unbounded ranges so json:",omitempty" can
// omit empty time fields instead of emitting values such as [null].
func NonEmptyTimeRange(tr *TimeRange) *TimeRange {
	if tr == nil || !tr.HasBounds() {
		return nil
	}
	return tr
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
//
// Non-null, non-".." elements that are not valid RFC3339 strings are rejected.
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
			s, ok := arr[0].(string)
			if !ok {
				return fmt.Errorf("time array element 0 must be a string or null")
			}
			if s != "" && s != ".." {
				t, err := time.Parse(time.RFC3339, s)
				if err != nil {
					return fmt.Errorf("invalid time %q: must be RFC3339 or \"..\"", s)
				}
				tr.Start = &t
			}
		}
		if len(arr) > 1 && arr[1] != nil {
			s, ok := arr[1].(string)
			if !ok {
				return fmt.Errorf("time array element 1 must be a string or null")
			}
			if s != "" && s != ".." {
				t, err := time.Parse(time.RFC3339, s)
				if err != nil {
					return fmt.Errorf("invalid time %q: must be RFC3339 or \"..\"", s)
				}
				tr.End = &t
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
		if obj.Start != nil && *obj.Start != "" && *obj.Start != ".." {
			t, err := time.Parse(time.RFC3339, *obj.Start)
			if err != nil {
				return fmt.Errorf("invalid start time %q: must be RFC3339 or \"..\"", *obj.Start)
			}
			tr.Start = &t
		}
		if obj.End != nil && *obj.End != "" && *obj.End != ".." {
			t, err := time.Parse(time.RFC3339, *obj.End)
			if err != nil {
				return fmt.Errorf("invalid end time %q: must be RFC3339 or \"..\"", *obj.End)
			}
			tr.End = &t
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
// into a TimeRange. "latest" sets Latest=true (most-recent-record semantics). "now" maps to
// Start=now. ".." / "../.." means unbounded (nil-nil, no filter).
func ToTimeRange(timeValue string) TimeRange {
	if timeValue == "latest" {
		return TimeRange{Latest: true}
	}
	if timeValue == "now" {
		now := time.Now().UTC()
		return TimeRange{Start: &now, End: nil}
	}
	if timeValue == "../.." || timeValue == ".." {
		return TimeRange{}
	}

	parts := strings.Split(timeValue, "/")

	if len(parts) == 2 {
		var startTime, endTime *time.Time

		if parts[0] != "" && parts[0] != ".." {
			if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
				startTime = &t
			}
		}

		if parts[1] != "" && parts[1] != ".." {
			if t, err := time.Parse(time.RFC3339, parts[1]); err == nil {
				endTime = &t
			}
		}

		return TimeRange{Start: startTime, End: endTime}
	}

	// Single value (no slash): treat as start time only
	if len(parts) == 1 && parts[0] != "" && parts[0] != ".." {
		if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
			return TimeRange{Start: &t, End: nil}
		}
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

// ParseTimeRangeStrict parses a query parameter value into a TimeRange, returning
// an error when a non-special value cannot be parsed as RFC3339. Use this for
// request query parameter validation where garbage input should produce HTTP 400.
// Accepts string or []string. Special values ("latest", "now", "..", "../..") are
// accepted. ".." is allowed in range parts to mean unbounded.
func ParseTimeRangeStrict(value interface{}) (TimeRange, error) {
	if value == nil {
		return TimeRange{}, nil
	}
	switch v := value.(type) {
	case string:
		return toTimeRangeStrict(v)
	case []string:
		return toTimeRangeFromSliceStrict(v)
	default:
		return TimeRange{}, fmt.Errorf("unsupported time range format")
	}
}

func toTimeRangeStrict(s string) (TimeRange, error) {
	if s == "latest" {
		return TimeRange{Latest: true}, nil
	}
	if s == "now" {
		now := time.Now().UTC()
		return TimeRange{Start: &now, End: nil}, nil
	}
	if s == "../.." || s == ".." {
		return TimeRange{}, nil
	}
	parts := strings.Split(s, "/")
	if len(parts) == 2 {
		return toTimeRangeFromSliceStrict(parts)
	}
	// Single value (no slash): treat as start time only
	if len(parts) == 1 {
		start, err := parseRangePart(parts[0], "start")
		if err != nil {
			return TimeRange{}, err
		}
		return TimeRange{Start: start}, nil
	}
	return TimeRange{}, fmt.Errorf("invalid time range %q: expected RFC3339, RFC3339/RFC3339, RFC3339/.., or ../RFC3339", s)
}

func toTimeRangeFromSliceStrict(parts []string) (TimeRange, error) {
	// A single element is treated as a standalone value: delegate to toTimeRangeStrict
	// so special keywords ("latest", "now", "..") and slash-delimited ranges both work.
	if len(parts) == 1 {
		return toTimeRangeStrict(parts[0])
	}
	var tr TimeRange
	if len(parts) > 0 {
		start, err := parseRangePart(parts[0], "start")
		if err != nil {
			return TimeRange{}, err
		}
		tr.Start = start
	}
	if len(parts) > 1 {
		end, err := parseRangePart(parts[1], "end")
		if err != nil {
			return TimeRange{}, err
		}
		tr.End = end
	}
	return tr, nil
}

// parseRangePart parses one side of a time range interval.
// "" and ".." mean unbounded (nil). "now" resolves to the current time.
// "latest" in a range endpoint is treated as unbounded (open-ended).
// Any other value must be a valid RFC3339 string.
func parseRangePart(s, side string) (*time.Time, error) {
	switch s {
	case "", "..", "latest":
		return nil, nil
	case "now":
		now := time.Now().UTC()
		return &now, nil
	default:
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("invalid %s time %q: expected RFC3339, \"now\", or \"..\"", side, s)
		}
		return &t, nil
	}
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
