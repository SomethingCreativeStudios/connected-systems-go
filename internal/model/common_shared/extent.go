package common_shared

type Extent struct {
	Spatial  *BoundingBox `json:"spatial,omitempty"`
	Temporal *TimeRange   `json:"temporal,omitempty"`
}
