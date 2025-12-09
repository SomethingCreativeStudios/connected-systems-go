package common_shared

import (
	"database/sql/driver"
	"encoding/json"
)

// SecurityConstraint represents a security tagging object. The SensorML schema
// requires a `type` property but allows additional arbitrary properties.
// We model the known `type` field and capture any other properties in the
// Extra map. Custom JSON marshal/unmarshal preserve both the typed field and
// arbitrary extras so round-tripping is lossless.
type SecurityConstraint struct {
	Type  string                 `json:"type"`
	Extra map[string]interface{} `json:"-"`
}

// UnmarshalJSON implements custom unmarshalling to extract the `type` field
// and preserve any other properties in the Extra map.
func (s *SecurityConstraint) UnmarshalJSON(b []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	if v, ok := m["type"]; ok {
		if str, ok := v.(string); ok {
			s.Type = str
		}
	}
	delete(m, "type")
	if len(m) == 0 {
		s.Extra = nil
	} else {
		s.Extra = m
	}
	return nil
}

// MarshalJSON writes out the `type` field and any extras captured in the
// Extra map.
func (s SecurityConstraint) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}
	if s.Extra != nil {
		for k, v := range s.Extra {
			m[k] = v
		}
	}
	m["type"] = s.Type
	return json.Marshal(m)
}

// SecurityConstraints is a JSONB-storable slice of SecurityConstraint
type SecurityConstraints []SecurityConstraint

// Value implements driver.Valuer for JSONB storage
func (s SecurityConstraints) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan implements sql.Scanner for JSONB retrieval
func (s *SecurityConstraints) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, s)
}
