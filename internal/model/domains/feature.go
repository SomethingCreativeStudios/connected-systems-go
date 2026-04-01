package domains

import (
	"time"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// Feature represents a generic OGC API Features resource
// This is the base feature type for OGC API - Features Part 1
type Feature struct {
	CommonSSN
	Base

	// Collection ID that owns this feature
	CollectionID string `gorm:"type:varchar(255);not null;index" json:"-"`

	// Temporal
	DateTime  *time.Time               `gorm:"type:timestamptz" json:"dateTime,omitempty"`
	ValidTime *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:valid_time_" json:"validTime,omitempty"`

	// Spatial
	Geometry *common_shared.GoGeom `gorm:"type:geometry" json:"geometry,omitempty"`

	// Links to related resources
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional properties (arbitrary key-value pairs)
	Properties common_shared.Properties `gorm:"type:jsonb" json:"properties,omitempty"`
}

// TableName specifies the table name
func (Feature) TableName() string {
	return "features"
}

// FeatureGeoJSONFeature converts Feature to GeoJSON Feature format
type FeatureGeoJSONFeature struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Geometry   *common_shared.GoGeom  `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
	Links      common_shared.Links    `json:"links,omitempty"`
}

// ToGeoJSON converts Feature domain to GeoJSON Feature
func (f Feature) ToGeoJSON() FeatureGeoJSONFeature {
	// Start with extra properties so known fields always win
	props := make(map[string]interface{}, len(f.Properties)+6)
	for k, v := range f.Properties {
		props[k] = v
	}

	props["uid"] = string(f.UniqueIdentifier)
	props["name"] = f.Name
	props["collectionId"] = f.CollectionID
	if f.Description != "" {
		props["description"] = f.Description
	}
	if f.DateTime != nil {
		props["dateTime"] = f.DateTime
	}
	if f.ValidTime != nil {
		props["validTime"] = f.ValidTime
	}

	return FeatureGeoJSONFeature{
		Type:       "Feature",
		ID:         f.ID,
		Geometry:   f.Geometry,
		Properties: props,
		Links:      f.Links,
	}
}
