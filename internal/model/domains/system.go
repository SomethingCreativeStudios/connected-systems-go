package domains

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// System represents a sosa:System feature
// Can be a Sensor, Actuator, Sampler, or Platform
type System struct {
	Base
	CommonSSN

	SystemType string  `gorm:"type:varchar(255);not null" json:"featureType"` // sosa:Sensor, sosa:Actuator, sosa:Platform, etc.
	AssetType  *string `gorm:"type:varchar(100)" json:"assetType,omitempty"`  // Equipment, Human, Platform, etc.

	// Temporal
	ValidTime *common_shared.TimeRange `gorm:"type:jsonb" json:"validTime,omitempty"`

	// Spatial
	// Use GoGeom wrapper which stores as PostGIS WKB/EWKB when possible
	Geometry *common_shared.GoGeom `gorm:"type:geometry" json:"geometry,omitempty"`

	// Associations (stored as links in JSON)
	ParentSystemID *string `gorm:"type:varchar(255);index" json:"-"`

	// Links to related resources
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional properties
	Properties common_shared.Properties `gorm:"type:jsonb" json:"properties,omitempty"`
}

// TableName specifies the table name
func (System) TableName() string {
	return "systems"
}

// SystemType constants (SOSA/SSN types)
const (
	SystemTypeSensor   = "http://www.w3.org/ns/sosa/Sensor"
	SystemTypeActuator = "http://www.w3.org/ns/sosa/Actuator"
	SystemTypeSampler  = "http://www.w3.org/ns/sosa/Sampler"
	SystemTypePlatform = "http://www.w3.org/ns/sosa/Platform"
	SystemTypeSystem   = "http://www.w3.org/ns/ssn/System"
)

// AssetType constants
const (
	AssetTypeEquipment  = "Equipment"
	AssetTypeHuman      = "Human"
	AssetTypePlatform   = "Platform"
	AssetTypeProcess    = "Process"
	AssetTypeSimulation = "Simulation"
)

// GeoJSONFeature converts System to GeoJSON Feature format
type SystemGeoJSONFeature struct {
	Type       string                  `json:"type"`
	ID         string                  `json:"id"`
	Geometry   *common_shared.GoGeom   `json:"geometry"`
	Properties SystemGeoJSONProperties `json:"properties"`
	Links      common_shared.Links     `json:"links,omitempty"`
}

// SystemGeoJSONProperties represents the properties object in GeoJSON
type SystemGeoJSONProperties struct {
	UID         UniqueID                 `json:"uid"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	FeatureType string                   `json:"featureType"`
	AssetType   *string                  `json:"assetType,omitempty"`
	ValidTime   *common_shared.TimeRange `json:"validTime,omitempty"`
	SystemKind  *common_shared.Link      `json:"systemKind@link,omitempty"`
}

func (System) BuildFromRequest(r *http.Request, w http.ResponseWriter) (System, error) {
	// Decode GeoJSON Feature format
	var geoJSON struct {
		Type       string                  `json:"type"`
		ID         string                  `json:"id,omitempty"`
		Properties map[string]interface{}  `json:"properties"`
		Geometry   *common_shared.Geometry `json:"geometry,omitempty"`
		Links      common_shared.Links     `json:"links,omitempty"`
	}

	if err := render.DecodeJSON(r.Body, &geoJSON); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return System{}, err
	}

	// Convert GeoJSON properties to System model
	system := System{
		Links: geoJSON.Links,
	}

	// convert geometry (GeoJSON) into GoGeom wrapper
	if geoJSON.Geometry != nil {
		gg := &common_shared.GoGeom{}
		// marshal the incoming geoJSON Geometry and unmarshal into GoGeom (uses toGeom)
		if b, err := json.Marshal(geoJSON.Geometry); err == nil {
			_ = gg.UnmarshalJSON(b)
			system.Geometry = gg
		}
	}

	// Extract properties from the properties object
	if uid, ok := geoJSON.Properties["uid"].(string); ok {
		system.UniqueIdentifier = UniqueID(uid)
	}
	if name, ok := geoJSON.Properties["name"].(string); ok {
		system.Name = name
	}
	if desc, ok := geoJSON.Properties["description"].(string); ok {
		system.Description = desc
	}
	if featureType, ok := geoJSON.Properties["featureType"].(string); ok {
		system.SystemType = featureType
	}
	if assetType, ok := geoJSON.Properties["assetType"].(string); ok {
		system.AssetType = &assetType
	}

	// Handle validTime if present
	if validTimeMap, ok := geoJSON.Properties["validTime"].(map[string]interface{}); ok {
		system.ValidTime = &common_shared.TimeRange{}
		if startStr, ok := validTimeMap["start"].(string); ok && startStr != "" {
			startTime, _ := time.Parse(time.RFC3339, startStr)
			system.ValidTime.Start = &startTime
		}
		if endStr, ok := validTimeMap["end"].(string); ok && endStr != "" {
			endTime, _ := time.Parse(time.RFC3339, endStr)
			system.ValidTime.End = &endTime
		}
	}

	return system, nil
}
