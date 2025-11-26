package common_shared

// Axis describes a single axis of a SpatialFrame.
type Axis struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SpatialFrame models the OpenAPI `SpatialFrame` schema. It includes the
// AbstractSweIdentifiable fields (id/label/description) and the required
// origin and axes fields.
type SpatialFrame struct {
	ID          string `json:"id,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`

	Origin string `json:"origin"`
	Axes   []Axis `json:"axes"`
}

// TemporalFrame models the OpenAPI `TemporalFrame` schema. It includes the
// AbstractSweIdentifiable fields and the required origin string.
type TemporalFrame struct {
	ID          string `json:"id,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`

	Origin string `json:"origin"`
}
