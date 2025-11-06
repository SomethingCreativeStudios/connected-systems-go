package domains

import (
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// Feature represents a generic OGC API Features resource
// This is the base feature type for OGC API - Features Part 1
type Feature struct {
	Base

	// Collection ID that owns this feature
	CollectionID string `gorm:"type:varchar(255);not null;index" json:"-"`

	// Unique identifier (business key, can be URN/URL)
	UniqueIdentifier UniqueID `gorm:"type:varchar(255);uniqueIndex;not null" json:"uid"`

	// Basic metadata
	Name        string `gorm:"type:varchar(255)" json:"name"`
	Description string `gorm:"type:text" json:"description,omitempty"`

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

// BuildFromRequest decodes GeoJSON Feature request into Feature domain model
func (Feature) BuildFromRequest(r *http.Request, w http.ResponseWriter) (Feature, error) {
	// Decode GeoJSON Feature format
	var geoJSON struct {
		Type       string                 `json:"type"`
		ID         string                 `json:"id,omitempty"`
		Properties map[string]interface{} `json:"properties"`
		Geometry   *common_shared.GoGeom  `json:"geometry,omitempty"`
		Links      common_shared.Links    `json:"links,omitempty"`
	}

	if err := render.DecodeJSON(r.Body, &geoJSON); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return Feature{}, err
	}

	// Convert GeoJSON properties to Feature model
	feature := Feature{
		Links:      geoJSON.Links,
		Properties: geoJSON.Properties,
	}
	// assign geometry (decoded directly into GoGeom)
	if geoJSON.Geometry != nil {
		feature.Geometry = geoJSON.Geometry
	}

	// Extract standard properties
	if uid, ok := geoJSON.Properties["uid"].(string); ok {
		feature.UniqueIdentifier = UniqueID(uid)
	}
	if name, ok := geoJSON.Properties["name"].(string); ok {
		feature.Name = name
	}
	if desc, ok := geoJSON.Properties["description"].(string); ok {
		feature.Description = desc
	}
	if collectionID, ok := geoJSON.Properties["collectionId"].(string); ok {
		feature.CollectionID = collectionID
	}

	// Parse dateTime if present
	if dtStr, ok := geoJSON.Properties["dateTime"].(string); ok {
		if dt, err := time.Parse(time.RFC3339, dtStr); err == nil {
			feature.DateTime = &dt
		}
	}

	// Parse validTime if present
	if validTimeMap, ok := geoJSON.Properties["validTime"].(map[string]interface{}); ok {
		validTime := &common_shared.TimeRange{}
		if start, ok := validTimeMap["start"].(string); ok {
			if t, err := time.Parse(time.RFC3339, start); err == nil {
				validTime.Start = &t
			}
		}
		if end, ok := validTimeMap["end"].(string); ok {
			if t, err := time.Parse(time.RFC3339, end); err == nil {
				validTime.End = &t
			}
		}
		feature.ValidTime = validTime
	}

	return feature, nil
}
