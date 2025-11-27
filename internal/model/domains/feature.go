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
	Type       string                   `json:"type"`
	ID         string                   `json:"id"`
	Geometry   *common_shared.GoGeom    `json:"geometry"`
	Properties FeatureGeoJSONProperties `json:"properties"`
	Links      common_shared.Links      `json:"links,omitempty"`
}

// FeatureGeoJSONProperties represents the properties object in GeoJSON
type FeatureGeoJSONProperties struct {
	UID          UniqueID                 `json:"uid"`
	Name         string                   `json:"name"`
	Description  string                   `json:"description,omitempty"`
	DateTime     *time.Time               `json:"dateTime,omitempty"`
	ValidTime    *common_shared.TimeRange `json:"validTime,omitempty"`
	CollectionID string                   `json:"collectionId"`
	// Additional properties are merged from the Properties field
	AdditionalProperties map[string]interface{} `json:"-"` // Will be flattened into properties
}

// ToGeoJSON converts Feature domain to GeoJSON Feature
func (f Feature) ToGeoJSON() FeatureGeoJSONFeature {
	props := FeatureGeoJSONProperties{
		UID:          f.UniqueIdentifier,
		Name:         f.Name,
		Description:  f.Description,
		DateTime:     f.DateTime,
		ValidTime:    f.ValidTime,
		CollectionID: f.CollectionID,
	}

	return FeatureGeoJSONFeature{
		Type:       "Feature",
		ID:         f.ID,
		Geometry:   f.Geometry,
		Properties: props,
		Links:      f.Links,
	}
}
