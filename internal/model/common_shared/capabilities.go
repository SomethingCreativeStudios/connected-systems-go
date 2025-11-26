package common_shared

import "encoding/json"

// CapabilityGroup models the "capabilities" group from the OpenAPI SWE-style
// schema. It largely mirrors CharacteristicGroup: identification metadata,
// an optional conditions array, and a required list of capability components.
type CapabilityGroup struct {
	ID          string `json:"id,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Definition  string `json:"definition,omitempty"`

	// When these capabilities apply
	Conditions []ComponentWrapper `json:"conditions,omitempty"`

	// The list of capability components (oneOf many component variants)
	Capabilities []ComponentWrapper `json:"capabilities"`
}

// CapabilityWrapper is an alias for ComponentWrapper to make intent explicit
// when reading code. It reuses the same unmarshalling and concrete variants
// defined in characteristics.go.
type CapabilityWrapper = ComponentWrapper

// --- Optional helpers ---

// AsComponents returns the concrete Component values (when available) for
// each CapabilityWrapper. Nil entries will be skipped.
func (cg *CapabilityGroup) AsComponents() []Component {
	res := make([]Component, 0, len(cg.Capabilities))
	for _, w := range cg.Capabilities {
		if w.Component != nil {
			res = append(res, w.Component)
		}
	}
	return res
}

// RawJSON returns the raw JSON for the capability at index i if present.
func (cg *CapabilityGroup) RawJSON(i int) json.RawMessage {
	if i < 0 || i >= len(cg.Capabilities) {
		return nil
	}
	return cg.Capabilities[i].Raw
}
