package model

// Deployment represents a sosa:Deployment feature
type Deployment struct {
	Base

	// Core properties (from SOSA/SSN)
	UniqueIdentifier UniqueID `gorm:"type:varchar(255);uniqueIndex" json:"uid"`
	Name             string   `gorm:"type:varchar(255);not null" json:"name"`
	Description      string   `gorm:"type:text" json:"description,omitempty"`
	DeploymentType   string   `gorm:"type:varchar(255)" json:"featureType,omitempty"`

	// Temporal - deployment period
	ValidTime *TimeRange `gorm:"type:jsonb" json:"validTime,omitempty"`

	// Spatial - deployment location
	Geometry *Geometry `gorm:"type:jsonb" json:"geometry,omitempty"`

	// Associations
	ParentDeploymentID *string `gorm:"type:varchar(255);index" json:"-"`

	// Links to related resources
	Links Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional properties
	Properties Properties `gorm:"type:jsonb" json:"properties,omitempty"`
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
	Geometry   *Geometry                   `json:"geometry"`
	Properties DeploymentGeoJSONProperties `json:"properties"`
	Links      Links                       `json:"links,omitempty"`
}

// DeploymentGeoJSONProperties represents the properties object in GeoJSON
type DeploymentGeoJSONProperties struct {
	UID         UniqueID   `json:"uid"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	FeatureType string     `json:"featureType,omitempty"`
	ValidTime   *TimeRange `json:"validTime,omitempty"`
}

// ToGeoJSON converts Deployment model to GeoJSON Feature
func (d *Deployment) ToGeoJSON() DeploymentGeoJSONFeature {
	return DeploymentGeoJSONFeature{
		Type:     "Feature",
		ID:       d.ID,
		Geometry: d.Geometry,
		Properties: DeploymentGeoJSONProperties{
			UID:         d.UniqueIdentifier,
			Name:        d.Name,
			Description: d.Description,
			FeatureType: d.DeploymentType,
			ValidTime:   d.ValidTime,
		},
		Links: d.Links,
	}
}
