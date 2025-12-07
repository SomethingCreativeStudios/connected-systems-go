package common_shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// ConfigurationSettings represents the 'configuration' object used during a
// deployment. Fields are modeled after the SensorML-based schema used by the
// conformance suite.
type ConfigurationSettings struct {
	SetValues      []SetValue      `json:"setValues,omitempty"`
	SetArrayValues []SetArrayValue `json:"setArrayValues,omitempty"`
	SetModes       []SetMode       `json:"setModes,omitempty"`
	SetConstraints []Constraint    `json:"setConstraints,omitempty"`
	SetStatus      []SetStatus     `json:"setStatus,omitempty"`
}

// SetValue represents an object with a reference and a scalar value (number or string).
type SetValue struct {
	Ref   string      `json:"ref"`
	Value interface{} `json:"value"` // number or string per schema
}

// SetArrayValue represents an object with a reference and an array value.
type SetArrayValue struct {
	Ref   string        `json:"ref"`
	Value []interface{} `json:"value"`
}

// SetMode represents a mode assignment with a reference and a mode value.
type SetMode struct {
	Ref   string `json:"ref"`
	Value string `json:"value"`
}

// Constraint is a flexible representation for setConstraints items. The schema
// allows several variants (AllowedTokens, AllowedValues, AllowedTimes) and
// permits additional properties; this struct exposes commonly used fields and
// will ignore unknown properties during JSON unmarshalling.
// AllowedTokens represents either an enum list or a regex pattern for tokens.
type AllowedTokens struct {
	Type    string   `json:"type,omitempty"`
	Values  []string `json:"values,omitempty"`
	Pattern string   `json:"pattern,omitempty"`
}

// AllowedValues represents enumerated values and/or numeric intervals.
// ValueItem represents a capability value item which may be a number or a string.
type ValueItem struct {
	Number *float64 `json:"-"`
	String *string  `json:"-"`
}

func (v *ValueItem) UnmarshalJSON(b []byte) error {
	// try number
	var num float64
	if err := json.Unmarshal(b, &num); err == nil {
		v.Number = &num
		return nil
	}
	// try string
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		v.String = &s
		return nil
	}
	return fmt.Errorf("ValueItem: unsupported JSON type: %s", string(b))
}

func (v ValueItem) MarshalJSON() ([]byte, error) {
	if v.Number != nil {
		return json.Marshal(*v.Number)
	}
	if v.String != nil {
		return json.Marshal(*v.String)
	}
	return []byte("null"), nil
}

// AllowedValues represents enumerated values and/or numeric intervals.
type AllowedValues struct {
	Type               string        `json:"type,omitempty"`
	Values             []ValueItem   `json:"values,omitempty"`
	Intervals          [][]ValueItem `json:"intervals,omitempty"`
	SignificantFigures *int          `json:"significantFigures,omitempty"`
}

// AllowedTimes represents time values or time intervals.
// AllowedTimes represents time values or time intervals (ISO8601 strings).
type AllowedTimes struct {
	Type               string     `json:"type,omitempty"`
	Values             []string   `json:"values,omitempty"`
	Intervals          [][]string `json:"intervals,omitempty"`
	SignificantFigures *int       `json:"significantFigures,omitempty"`
}

// Constraint is a discriminated wrapper that holds one of several concrete
// constraint variants. Raw preserves the original JSON when available.
type Constraint struct {
	Type string `json:"type,omitempty"`

	Ref string `json:"ref,omitempty"`

	Tokens *AllowedTokens `json:"allowedTokens,omitempty"`
	Values *AllowedValues `json:"allowedValues,omitempty"`
	Times  *AllowedTimes  `json:"allowedTimes,omitempty"`
}

// UnmarshalJSON implements custom unmarshalling to populate the appropriate
// concrete variant based on the `type` discriminator (or available fields).
func (c *Constraint) UnmarshalJSON(b []byte) error {
	// quick probe for type and ref
	var probe struct {
		Type string `json:"type"`
		Ref  string `json:"ref"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return err
	}
	c.Type = probe.Type
	c.Ref = probe.Ref

	switch probe.Type {
	case "AllowedTokens":
		var at AllowedTokens
		if err := json.Unmarshal(b, &at); err != nil {
			return err
		}
		c.Tokens = &at
	case "AllowedValues":
		var av AllowedValues
		if err := json.Unmarshal(b, &av); err != nil {
			return err
		}
		c.Values = &av
	case "AllowedTimes":
		var at AllowedTimes
		if err := json.Unmarshal(b, &at); err != nil {
			return err
		}
		c.Times = &at
	default:
		// If no type provided, attempt to detect fields
		var guess map[string]json.RawMessage
		if err := json.Unmarshal(b, &guess); err != nil {
			return err
		}
		if _, ok := guess["values"]; ok {
			// try AllowedValues first
			var av AllowedValues
			if err := json.Unmarshal(b, &av); err == nil {
				c.Values = &av
				return nil
			}
			// try AllowedTimes
			var at AllowedTimes
			if err := json.Unmarshal(b, &at); err == nil {
				c.Times = &at
				return nil
			}
		}
	}
	return nil
}

// MarshalJSON emits the preserved Raw bytes when available, otherwise the
// concrete variant is marshalled.
func (c Constraint) MarshalJSON() ([]byte, error) {
	out := make(map[string]interface{})
	if c.Type != "" {
		out["type"] = c.Type
	}
	if c.Ref != "" {
		out["ref"] = c.Ref
	}
	switch c.Type {
	case "AllowedTokens":
		if c.Tokens != nil {
			// merge token fields
			if c.Tokens.Values != nil {
				out["values"] = c.Tokens.Values
			}
			if c.Tokens.Pattern != "" {
				out["pattern"] = c.Tokens.Pattern
			}
		}
	case "AllowedValues":
		if c.Values != nil {
			out["values"] = c.Values.Values
			out["intervals"] = c.Values.Intervals
			out["significantFigures"] = c.Values.SignificantFigures
		}
	case "AllowedTimes":
		if c.Times != nil {
			out["values"] = c.Times.Values
			out["intervals"] = c.Times.Intervals
			out["significantFigures"] = c.Times.SignificantFigures
		}
	default:
		// if no type, try to emit whichever concrete is present
		if c.Tokens != nil {
			out["values"] = c.Tokens.Values
			if c.Tokens.Pattern != "" {
				out["pattern"] = c.Tokens.Pattern
			}
		} else if c.Values != nil {
			out["values"] = c.Values.Values
			out["intervals"] = c.Values.Intervals
			out["significantFigures"] = c.Values.SignificantFigures
		} else if c.Times != nil {
			out["values"] = c.Times.Values
			out["intervals"] = c.Times.Intervals
			out["significantFigures"] = c.Times.SignificantFigures
		}
	}
	return json.Marshal(out)
}

// SetStatus represents enable/disable status setting for a referenced item.
type SetStatus struct {
	Ref   string `json:"ref"`
	Value string `json:"value"` // enum: enabled | disabled
}

// Value implements driver.Valuer to store ConfigurationSettings as JSONB.
func (c ConfigurationSettings) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements sql.Scanner to load ConfigurationSettings from a JSONB column.
func (c *ConfigurationSettings) Scan(value interface{}) error {
	if value == nil {
		*c = ConfigurationSettings{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, c)
}
