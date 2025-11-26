package common_shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// HistoryTime represents either an instant (RFC3339 string) or a time range.
// It marshals/unmarshals to the flexible shapes accepted by the API.
type HistoryTime struct {
	Instant *time.Time `json:"-"`
	Range   *TimeRange `json:"-"`
}

// UnmarshalJSON accepts either a string instant, a TimeRange-compatible
// array/object, or a string range like "start/end".
func (ht *HistoryTime) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*ht = HistoryTime{}
		return nil
	}

	// try string (instant or compact range)
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		if s == "" {
			*ht = HistoryTime{}
			return nil
		}
		// try parse as RFC3339 instant
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			ht.Instant = &t
			ht.Range = nil
			return nil
		}
		// fallback: interpret string as a range (start/end)
		tr := ToTimeRange(s)
		ht.Range = &tr
		ht.Instant = nil
		return nil
	}

	// try TimeRange shape (array/object). Leverage TimeRange.UnmarshalJSON
	var tr TimeRange
	if err := json.Unmarshal(b, &tr); err == nil {
		ht.Range = &tr
		ht.Instant = nil
		return nil
	}

	return fmt.Errorf("unsupported history time JSON")
}

// MarshalJSON emits either an RFC3339 instant string or the TimeRange JSON
// representation.
func (ht HistoryTime) MarshalJSON() ([]byte, error) {
	if ht.Instant != nil {
		return json.Marshal(ht.Instant.Format(time.RFC3339))
	}
	if ht.Range != nil {
		return json.Marshal(ht.Range)
	}
	return json.Marshal(nil)
}

// HistoryEvent represents a time-tagged event with optional metadata and
// additional properties. It mirrors the OpenAPI `history` item definition
// but keeps component `properties` flexible by reusing ComponentWrapper.
type HistoryEvent struct {
	ID            string             `json:"id,omitempty"`
	Label         string             `json:"label,omitempty"`
	Description   string             `json:"description,omitempty"`
	Definition    string             `json:"definition,omitempty"`
	Identifiers   []Term             `json:"identifiers,omitempty"`
	Classifiers   []Term             `json:"classifiers,omitempty"`
	Contacts      []ContactWrapper   `json:"contacts,omitempty"`
	Documentation Documents          `json:"documentation,omitempty"`
	Time          HistoryTime        `json:"time,omitempty"`
	Properties    []ComponentWrapper `json:"properties,omitempty"`
	Configuration json.RawMessage    `json:"configuration,omitempty"`
}

// History is a JSONB-friendly slice of HistoryEvent values.
type History []HistoryEvent

// Value implements driver.Valuer for storing History as JSONB in the DB.
func (h History) Value() (driver.Value, error) {
	return json.Marshal(h)
}

// Scan implements sql.Scanner to load History from a JSONB column.
func (h *History) Scan(value interface{}) error {
	if value == nil {
		*h = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, h)
}
