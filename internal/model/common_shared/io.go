package common_shared

import (
	"database/sql/driver"
	"encoding/json"
)

// ObservablePropertyInline models the inline ObservableProperty variant
// used in the OpenAPI `inputs`/`outputs` oneOf. It is intentionally small
// and mirrors the common fields used by the API (type, definition, label).
type ObservablePropertyInline struct {
	Type       string `json:"type,omitempty"`
	Definition string `json:"definition,omitempty"`
	Label      string `json:"label,omitempty"`
}

// IOItem is a generic wrapper for an item in `inputs` or `outputs` which may
// be either a DataComponent (represented by ComponentWrapper) or an
// ObservablePropertyInline. The Raw field preserves the original JSON.
type IOItem struct {
	Component  *ComponentWrapper         `json:"-"`
	Observable *ObservablePropertyInline `json:"-"`
	Raw        json.RawMessage           `json:"-"`
}

// UnmarshalJSON detects which variant is present and populates the
// appropriate field.
func (io *IOItem) UnmarshalJSON(b []byte) error {
	io.Raw = append([]byte(nil), b...)

	// quick probe for "type"
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(b, &probe); err == nil {
		if probe.Type == "ObservableProperty" {
			var op ObservablePropertyInline
			if err := json.Unmarshal(b, &op); err != nil {
				return err
			}
			io.Observable = &op
			return nil
		}
	}

	// otherwise try to unmarshal as ComponentWrapper
	var cw ComponentWrapper
	if err := json.Unmarshal(b, &cw); err == nil {
		io.Component = &cw
		return nil
	}

	// unknown shape; keep Raw populated
	return nil
}

// IsComponent returns true if this IOItem contains a component payload.
func (io IOItem) IsComponent() bool { return io.Component != nil }

// IsObservable returns true if this IOItem contains an observable property.
func (io IOItem) IsObservable() bool { return io.Observable != nil }

// MarshalJSON implements custom marshalling so that IOItem serializes to the
// original JSON payload when possible. This avoids emitting empty objects
// ("{}") for list entries when only internal fields are populated.
func (io IOItem) MarshalJSON() ([]byte, error) {
	// Prefer the preserved raw payload if present
	if len(io.Raw) > 0 {
		return io.Raw, nil
	}

	// Fall back to concrete variants
	if io.Observable != nil {
		return json.Marshal(io.Observable)
	}
	if io.Component != nil {
		return json.Marshal(io.Component)
	}

	// Nothing to emit; return empty object
	return []byte("{}"), nil
}

// IOList is a JSONB-friendly slice of IOItem values.
type IOList []IOItem

// Value implements driver.Valuer for storing IOList as JSONB in the DB.
func (l IOList) Value() (driver.Value, error) {
	return json.Marshal(l)
}

// Scan implements sql.Scanner to load IOList from a JSONB column.
func (l *IOList) Scan(value interface{}) error {
	if value == nil {
		*l = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, l)
}

// IOComponentChoice matches the OpenAPI schema name `IOComponentChoice`.
// It's an alias to `IOItem` to make intent explicit when used in domain models.
type IOComponentChoice = IOItem
