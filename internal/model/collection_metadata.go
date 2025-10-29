package model

import "github.com/yourusername/connected-systems-go/internal/model/common_shared"

// CollectionMetadata represents OGC API Collections metadata
type CollectionMetadata struct {
	ID          string              `json:"id"`
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Links       common_shared.Links `json:"links"`
	Extent      *Extent             `json:"extent,omitempty"`
	ItemType    string              `json:"itemType,omitempty"`    // e.g., "feature"
	FeatureType string              `json:"featureType,omitempty"` // e.g., "sosa:System"
	CRS         []string            `json:"crs,omitempty"`
}

// Extent represents spatial and temporal extent
type Extent struct {
	Spatial  *SpatialExtent  `json:"spatial,omitempty"`
	Temporal *TemporalExtent `json:"temporal,omitempty"`
}

// SpatialExtent represents a bounding box
type SpatialExtent struct {
	Bbox [][]float64 `json:"bbox"` // Array of bboxes [minx, miny, maxx, maxy]
	CRS  string      `json:"crs,omitempty"`
}

// TemporalExtent represents a time interval
type TemporalExtent struct {
	Interval [][]string `json:"interval"` // Array of time intervals [[start, end]]
	TRS      string     `json:"trs,omitempty"`
}

// LandingPage represents the API landing page
type LandingPage struct {
	Title       string              `json:"title"`
	Description string              `json:"description,omitempty"`
	Links       common_shared.Links `json:"links"`
}

// ConformanceDeclaration represents conformance classes
type ConformanceDeclaration struct {
	ConformsTo []string `json:"conformsTo"`
}
