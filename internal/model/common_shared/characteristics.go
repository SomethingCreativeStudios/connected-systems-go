package common_shared

import (
	"database/sql/driver"
	"encoding/json"
)

// CharacteristicGroup models a characteristics array item from the OpenAPI
// SWE-style schema. It contains identification metadata and lists of
// conditions and characteristics (components).
type CharacteristicGroup struct {
	ID              string             `json:"id,omitempty"`
	Label           string             `json:"label,omitempty"`
	Description     string             `json:"description,omitempty"`
	Definition      string             `json:"definition,omitempty"`
	Conditions      []ComponentWrapper `json:"conditions,omitempty"`
	Characteristics []ComponentWrapper `json:"characteristics,omitempty"`
}

// Component is an interface implemented by strongly-typed SWE component variants.
type Component interface {
	ComponentType() string
}

// ComponentWrappers is a slice of ComponentWrapper that implements GORM JSONB support
type ComponentWrappers []ComponentWrapper

// Value implements driver.Valuer for JSONB storage
func (c ComponentWrappers) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

// Scan implements sql.Scanner for JSONB retrieval
func (c *ComponentWrappers) Scan(value interface{}) error {
	if value == nil {
		*c = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, c)
}

// ComponentWrapper is a generic container for a component variant. It preserves
// the raw payload and unmarshals into a concrete typed component when possible.
type ComponentWrapper struct {
	// Common fields
	Type           string `json:"type,omitempty"`
	Definition     string `json:"definition,omitempty"`
	Label          string `json:"label,omitempty"`
	ReferenceFrame string `json:"referenceFrame,omitempty"`
	AxisID         string `json:"axisID,omitempty"`
	LocalFrame     string `json:"localFrame,omitempty"`

	Updatable *bool `json:"updateable,omitempty"`
	Optional  *bool `json:"optional,omitempty"`

	// Variant fields left as raw so callers can decode according to type
	UOM        json.RawMessage `json:"uom,omitempty"`
	Constraint json.RawMessage `json:"constraint,omitempty"`
	NilValues  json.RawMessage `json:"nilValues,omitempty"`
	Value      json.RawMessage `json:"value,omitempty"`

	// Concrete typed component (populated by UnmarshalJSON)
	Component Component `json:"-"`
	// Raw holds the original JSON value
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON implements custom unmarshalling to detect the component "type"
// discriminator and populate the appropriate concrete struct.
func (c *ComponentWrapper) UnmarshalJSON(b []byte) error {
	c.Raw = append([]byte(nil), b...)

	// first unmarshal common fields to inspect "type"
	typeOnly := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(b, &typeOnly); err != nil {
		return err
	}

	// populate the generic fields
	aux := struct {
		Type           string          `json:"type,omitempty"`
		Definition     string          `json:"definition,omitempty"`
		Label          string          `json:"label,omitempty"`
		ReferenceFrame string          `json:"referenceFrame,omitempty"`
		AxisID         string          `json:"axisID,omitempty"`
		LocalFrame     string          `json:"localFrame,omitempty"`
		UOM            json.RawMessage `json:"uom,omitempty"`
		Constraint     json.RawMessage `json:"constraint,omitempty"`
		NilValues      json.RawMessage `json:"nilValues,omitempty"`
		Value          json.RawMessage `json:"value,omitempty"`
	}{}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	c.Type = aux.Type
	c.Definition = aux.Definition
	c.Label = aux.Label
	c.ReferenceFrame = aux.ReferenceFrame
	c.AxisID = aux.AxisID
	c.LocalFrame = aux.LocalFrame
	c.UOM = aux.UOM
	c.Constraint = aux.Constraint
	c.NilValues = aux.NilValues
	c.Value = aux.Value

	// switch on type discriminator and unmarshal into concrete variant
	switch typeOnly.Type {
	case "Boolean":
		var v BooleanComponent
		if err := json.Unmarshal(b, &v); err == nil {
			c.Component = &v
			return nil
		}
	case "Count":
		var v CountComponent
		if err := json.Unmarshal(b, &v); err == nil {
			c.Component = &v
			return nil
		}
	case "Quantity":
		var v QuantityComponent
		if err := json.Unmarshal(b, &v); err == nil {
			c.Component = &v
			return nil
		}
	case "Time":
		var v TimeComponent
		if err := json.Unmarshal(b, &v); err == nil {
			c.Component = &v
			return nil
		}
	case "Category":
		var v CategoryComponent
		if err := json.Unmarshal(b, &v); err == nil {
			c.Component = &v
			return nil
		}
	case "Text":
		var v TextComponent
		if err := json.Unmarshal(b, &v); err == nil {
			c.Component = &v
			return nil
		}
	case "CountRange":
		var v CountRangeComponent
		if err := json.Unmarshal(b, &v); err == nil {
			c.Component = &v
			return nil
		}
	case "QuantityRange":
		var v QuantityRangeComponent
		if err := json.Unmarshal(b, &v); err == nil {
			c.Component = &v
			return nil
		}
	case "TimeRange":
		var v TimeRangeComponent
		if err := json.Unmarshal(b, &v); err == nil {
			c.Component = &v
			return nil
		}
	case "Vector":
		var v VectorComponent
		if err := json.Unmarshal(b, &v); err == nil {
			c.Component = &v
			return nil
		}
	case "Array":
		var v ArrayComponent
		if err := json.Unmarshal(b, &v); err == nil {
			c.Component = &v
			return nil
		}
	default:
		// unknown/unsupported type: leave Raw and generic fields populated
	}

	return nil
}

// --- Concrete component variants ---

// BooleanComponent implements a simple boolean component.
type BooleanComponent struct {
	Type       string `json:"type"`
	Definition string `json:"definition,omitempty"`
	Label      string `json:"label,omitempty"`
	Value      bool   `json:"value"`
}

func (BooleanComponent) ComponentType() string { return "Boolean" }

// CountComponent represents a discrete integer-count component.
type CountComponent struct {
	Type       string `json:"type"`
	Definition string `json:"definition,omitempty"`
	Label      string `json:"label,omitempty"`
	Value      int    `json:"value"`
}

func (CountComponent) ComponentType() string { return "Count" }

// QuantityComponent represents a numeric component with units.
type QuantityComponent struct {
	Type       string          `json:"type"`
	Definition string          `json:"definition,omitempty"`
	Label      string          `json:"label,omitempty"`
	UOM        json.RawMessage `json:"uom,omitempty"`
	Value      json.RawMessage `json:"value,omitempty"` // may be number or array
}

func (QuantityComponent) ComponentType() string { return "Quantity" }

// TimeComponent represents a time-valued component.
type TimeComponent struct {
	Type       string          `json:"type"`
	Definition string          `json:"definition,omitempty"`
	Label      string          `json:"label,omitempty"`
	UOM        json.RawMessage `json:"uom,omitempty"`
	Value      json.RawMessage `json:"value,omitempty"`
}

func (TimeComponent) ComponentType() string { return "Time" }

// CategoryComponent represents a categorical token component.
type CategoryComponent struct {
	Type       string `json:"type"`
	Definition string `json:"definition,omitempty"`
	Label      string `json:"label,omitempty"`
	Value      string `json:"value,omitempty"`
}

func (CategoryComponent) ComponentType() string { return "Category" }

// TextComponent represents a free-text component.
type TextComponent struct {
	Type       string `json:"type"`
	Definition string `json:"definition,omitempty"`
	Label      string `json:"label,omitempty"`
	Value      string `json:"value,omitempty"`
}

func (TextComponent) ComponentType() string { return "Text" }

// CountRangeComponent models a pair of integers representing a range.
type CountRangeComponent struct {
	Type       string `json:"type"`
	Definition string `json:"definition,omitempty"`
	Label      string `json:"label,omitempty"`
	Value      []int  `json:"value,omitempty"`
}

func (CountRangeComponent) ComponentType() string { return "CountRange" }

// QuantityRangeComponent models a numeric range (two values) with a UOM.
type QuantityRangeComponent struct {
	Type       string            `json:"type"`
	Definition string            `json:"definition,omitempty"`
	Label      string            `json:"label,omitempty"`
	UOM        json.RawMessage   `json:"uom,omitempty"`
	Value      []json.RawMessage `json:"value,omitempty"`
}

func (QuantityRangeComponent) ComponentType() string { return "QuantityRange" }

// TimeRangeComponent models a pair of time values.
type TimeRangeComponent struct {
	Type       string            `json:"type"`
	Definition string            `json:"definition,omitempty"`
	Label      string            `json:"label,omitempty"`
	UOM        json.RawMessage   `json:"uom,omitempty"`
	Value      []json.RawMessage `json:"value,omitempty"`
}

func (TimeRangeComponent) ComponentType() string { return "TimeRange" }

// VectorComponent models a vector component (coordinates list).
type VectorComponent struct {
	Type           string          `json:"type"`
	Definition     string          `json:"definition,omitempty"`
	Label          string          `json:"label,omitempty"`
	ReferenceFrame string          `json:"referenceFrame,omitempty"`
	LocalFrame     string          `json:"localFrame,omitempty"`
	Coordinates    json.RawMessage `json:"coordinates,omitempty"` // leave flexible
}

func (VectorComponent) ComponentType() string { return "Vector" }

// ArrayComponent models an ISO-11404 array component.
type ArrayComponent struct {
	Type         string          `json:"type"`
	Definition   string          `json:"definition,omitempty"`
	Label        string          `json:"label,omitempty"`
	ElementCount int             `json:"elementCount,omitempty"`
	Coordinates  json.RawMessage `json:"coordinates,omitempty"`
}

func (ArrayComponent) ComponentType() string { return "Array" }
