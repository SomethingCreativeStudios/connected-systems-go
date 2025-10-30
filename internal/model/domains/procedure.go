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

	// Links to related resources
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional properties
	Properties common_shared.Properties `gorm:"type:jsonb" json:"properties,omitempty"`
}

// TableName specifies the table name
func (Procedure) TableName() string {
	return "procedures"
}

// ProcedureType constants (SOSA/SSN)
const (
	ProcedureTypeProcedure = "http://www.w3.org/ns/sosa/Procedure"
)

// ProcedureGeoJSONFeature converts Procedure to GeoJSON Feature format
type ProcedureGeoJSONFeature struct {
	Type       string                     `json:"type"`
	ID         string                     `json:"id"`
	Geometry   *common_shared.Geometry    `json:"geometry"` // Always null for procedures
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
		Geometry   *common_shared.Geometry    `json:"geometry,omitempty"`
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

	// Extract properties from the properties object
	procedure.UniqueIdentifier = UniqueID(geoJSON.Properties.UID)
	procedure.Name = geoJSON.Properties.Name
	procedure.Description = geoJSON.Properties.Description
	procedure.FeatureType = geoJSON.Properties.FeatureType

	return procedure, nil
}
