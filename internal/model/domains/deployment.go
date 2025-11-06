package domains

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// Deployment represents a sosa:Deployment feature
type Deployment struct {
	Base
	CommonSSN

	DeploymentType string `gorm:"type:varchar(255)" json:"featureType,omitempty"`

	// Temporal - deployment period
	ValidTime *common_shared.TimeRange `gorm:"type:jsonb" json:"validTime,omitempty"`

	// Spatial - deployment location
	Geometry *common_shared.GoGeom `gorm:"type:geometry" json:"geometry,omitempty"`

	// Associations
	ParentDeploymentID *string `gorm:"type:varchar(255);index" json:"-"`

	// Links to related resources
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional properties
	Properties common_shared.Properties `gorm:"type:jsonb" json:"properties,omitempty"`
}

// TableName specifies the table name
func (Deployment) TableName() string {
	return "deployments"
}

// DeploymentType constant (SOSA/SSN)
const (
	DeploymentTypeDeployment = "http://www.w3.org/ns/ssn/Deployment"
)

// DeploymentGeoJSONFeature converts Deployment to GeoJSON Feature format
type DeploymentGeoJSONFeature struct {
	Type       string                      `json:"type"`
	ID         string                      `json:"id"`
	Geometry   *common_shared.GoGeom       `json:"geometry"`
	Properties DeploymentGeoJSONProperties `json:"properties"`
	Links      common_shared.Links         `json:"links,omitempty"`
}

// DeploymentGeoJSONProperties represents the properties object in GeoJSON
type DeploymentGeoJSONProperties struct {
	UID         UniqueID                 `json:"uid"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	FeatureType string                   `json:"featureType,omitempty"`
	ValidTime   *common_shared.TimeRange `json:"validTime,omitempty"`
}

func (Deployment) BuildFromRequest(r *http.Request, w http.ResponseWriter) (Deployment, error) {
	// Decode GeoJSON Feature format
	var geoJSON struct {
		Type       string                      `json:"type"`
		ID         string                      `json:"id,omitempty"`
		Properties DeploymentGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.Geometry     `json:"geometry,omitempty"`
		Links      common_shared.Links         `json:"links,omitempty"`
	}
	if err := render.DecodeJSON(r.Body, &geoJSON); err != nil {
		return Deployment{}, err
	}

	// Convert GeoJSON properties to Deployment model
	deployment := Deployment{
		Links: geoJSON.Links,
	}
	if geoJSON.Geometry != nil {
		gg := &common_shared.GoGeom{}
		if b, err := json.Marshal(geoJSON.Geometry); err == nil {
			_ = gg.UnmarshalJSON(b)
			deployment.Geometry = gg
		}
	}
	// Extract properties from the properties object
	deployment.UniqueIdentifier = UniqueID(geoJSON.Properties.UID)
	deployment.Name = geoJSON.Properties.Name
	deployment.Description = geoJSON.Properties.Description
	deployment.DeploymentType = geoJSON.Properties.FeatureType
	deployment.ValidTime = geoJSON.Properties.ValidTime

	return deployment, nil
}
