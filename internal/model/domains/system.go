package domains

import (
	"net/http"

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
	ValidTime *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:valid_time_" json:"validTime,omitempty"`

	// Spatial
	// Use GoGeom wrapper which stores as PostGIS WKB/EWKB when possible
	Geometry *common_shared.GoGeom `gorm:"type:geometry" json:"geometry,omitempty"`

	// Associations (stored as links in JSON)
	ParentSystemID *string `gorm:"type:varchar(255);index" json:"-"`

	// Links to related resources
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional properties
	Properties common_shared.Properties `gorm:"type:jsonb" json:"properties,omitempty"`

	// Associations
	Procedures       []Procedure       `gorm:"many2many:system_procedures;"`
	Deployments      []Deployment      `gorm:"many2many:system_deployments;"`
	SamplingFeatures []SamplingFeature `gorm:"foreignKey:ParentSystemID;"`
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
		Properties SystemGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.GoGeom   `json:"geometry,omitempty"`
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

	// assign geometry (decoded directly into GoGeom)
	if geoJSON.Geometry != nil {
		system.Geometry = geoJSON.Geometry
	}

	// Extract properties from the properties object
	system.UniqueIdentifier = UniqueID(geoJSON.Properties.UID)
	system.Name = geoJSON.Properties.Name
	system.Description = geoJSON.Properties.Description
	system.SystemType = geoJSON.Properties.FeatureType
	system.AssetType = geoJSON.Properties.AssetType
	system.ValidTime = geoJSON.Properties.ValidTime

	// if vt, ok := geoJSON.Properties["validTime"]; ok {
	// 	tr := common_shared.ParseTimeRange(vt)
	// 	system.ValidTime = &tr
	// }

	return system, nil
}
