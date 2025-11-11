package domains

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// Procedure represents a sosa:Procedure feature (datasheet, methodology)
type Procedure struct {
	Base
	CommonSSN

	FeatureType string `gorm:"type:varchar(255)" json:"featureType,omitempty"`

	// Note: Procedures typically don't have location
	ProcedureType string `gorm:"type:varchar(255)" json:"procedureType,omitempty"`

	// Links to related resources
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	ControlledProperties []Property `gorm:"many2many:procedure_controlled_properties;" json:"-"`
	ObservedProperties   []Property `gorm:"many2many:procedure_observed_properties;" json:"-"`

	// Additional properties
	Properties common_shared.Properties `gorm:"type:jsonb" json:"properties,omitempty"`

	ValidTime *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:valid_time_" json:"validTime,omitempty"`
	Systems   []System                 `gorm:"many2many:system_procedures;"`
}

// TableName specifies the table name
func (Procedure) TableName() string {
	return "procedures"
}

// ProcedureType constants based on Table 16 from OGC API - Connected Systems
// Procedure Types (Methods)
const (
	ProcedureTypeObserving = "http://www.w3.org/ns/sosa/ObservingProcedure" // sosa:ObservingProcedure - An observation method
	ProcedureTypeSampling  = "http://www.w3.org/ns/sosa/SamplingProcedure"  // sosa:SamplingProcedure - A sampling method
	ProcedureTypeActuating = "http://www.w3.org/ns/sosa/ActuatingProcedure" // sosa:ActuatingProcedure - An actuation method
	ProcedureTypeProcedure = "http://www.w3.org/ns/sosa/Procedure"          // sosa:Procedure - Any other type of procedure or methodology
)

// System Datasheet Types
const (
	ProcedureTypeSensor   = "http://www.w3.org/ns/sosa/Sensor"   // sosa:Sensor - A sensor datasheet
	ProcedureTypeActuator = "http://www.w3.org/ns/sosa/Actuator" // sosa:Actuator - An actuator datasheet
	ProcedureTypeSampler  = "http://www.w3.org/ns/sosa/Sampler"  // sosa:Sampler - A sampler datasheet
	ProcedureTypePlatform = "http://www.w3.org/ns/sosa/Platform" // sosa:Platform - A platform datasheet
	ProcedureTypeSystem   = "http://www.w3.org/ns/sosa/System"   // sosa:System - Any other system datasheet
)

// ProcedureGeoJSONFeature converts Procedure to GeoJSON Feature format
type ProcedureGeoJSONFeature struct {
	Type       string                     `json:"type"`
	ID         string                     `json:"id"`
	Geometry   *common_shared.GoGeom      `json:"geometry"` // Always null for procedures
	Properties ProcedureGeoJSONProperties `json:"properties"`
	Links      common_shared.Links        `json:"links,omitempty"`
}

// ProcedureGeoJSONProperties represents the properties object in GeoJSON
type ProcedureGeoJSONProperties struct {
	UID         UniqueID                 `json:"uid"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	FeatureType string                   `json:"featureType,omitempty"`
	ValidTime   *common_shared.TimeRange `json:"validTime,omitempty"`
}

func (Procedure) BuildFromRequest(r *http.Request, w http.ResponseWriter) (Procedure, error) {
	// Decode GeoJSON Feature format
	var geoJSON struct {
		Type       string                     `json:"type"`
		ID         string                     `json:"id,omitempty"`
		Properties ProcedureGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.GoGeom      `json:"geometry,omitempty"`
		Links      common_shared.Links        `json:"links,omitempty"`
	}

	if err := render.DecodeJSON(r.Body, &geoJSON); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return Procedure{}, err
	}

	// Convert GeoJSON properties to Procedure model
	procedure := Procedure{
		Links: geoJSON.Links,
	}

	// Validate/convert geometry if provided (procedures usually don't have geometry)
	if geoJSON.Geometry != nil {
		// decoded into GoGeom; procedures normally don't store geometry
	}

	// Extract properties from the properties object
	procedure.UniqueIdentifier = UniqueID(geoJSON.Properties.UID)
	procedure.Name = geoJSON.Properties.Name
	procedure.Description = geoJSON.Properties.Description
	procedure.FeatureType = geoJSON.Properties.FeatureType
	procedure.ValidTime = geoJSON.Properties.ValidTime

	return procedure, nil
}
