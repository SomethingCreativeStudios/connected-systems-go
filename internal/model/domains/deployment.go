package domains

import (
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// Deployment represents a sosa:Deployment feature
type Deployment struct {
	Base
	CommonSSN

	DeploymentType string `gorm:"type:varchar(255)" json:"featureType,omitempty"`

	// Temporal - deployment period
	ValidTime *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:valid_time_" json:"validTime,omitempty"`

	// Spatial - deployment location
	Geometry *common_shared.GoGeom `gorm:"type:geometry" json:"geometry,omitempty"`

	// Associations
	ParentDeploymentID *string `gorm:"type:varchar(255);index" json:"-"`

	// Platform link (when provided in payload)
	Platform   *common_shared.Link `gorm:"type:jsonb" json:"platform,omitempty"`
	PlatformID *string             `gorm:"type:varchar(255);index" json:"-"`

	// Additional descriptive metadata from the SWE/System schema
	Lang                *string                           `gorm:"type:varchar(10)" json:"lang,omitempty"`
	Keywords            []string                          `gorm:"type:jsonb" json:"keywords,omitempty"`
	Identifiers         common_shared.Terms               `gorm:"type:jsonb" json:"identifiers,omitempty"`
	Classifiers         common_shared.Terms               `gorm:"type:jsonb" json:"classifiers,omitempty"`
	SecurityConstraints common_shared.SecurityConstraints `gorm:"type:jsonb" json:"securityConstraints,omitempty"`
	LegalConstraints    common_shared.LegalConstraints    `gorm:"type:jsonb" json:"legalConstraints,omitempty"`

	Characteristics []common_shared.CharacteristicGroup `gorm:"type:jsonb" json:"characteristics,omitempty"`
	Capabilities    []common_shared.CapabilityGroup     `gorm:"type:jsonb" json:"capabilities,omitempty"`
	Contacts        []common_shared.ContactWrapper      `gorm:"type:jsonb" json:"contacts,omitempty"`
	Documentation   common_shared.Documents             `gorm:"type:jsonb" json:"documents,omitempty"`
	History         common_shared.History               `gorm:"type:jsonb" json:"history,omitempty"`

	// DeployedSystems: list of systems deployed with optional configuration
	SystemIds       *[]string            `gorm:"-" json:"systemIds,omitempty"`
	DeployedSystems []DeployedSystemItem `gorm:"type:jsonb" json:"deployedSystems,omitempty"`

	// Links to related resources
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	Systems []System `gorm:"many2many:system_deployments;"`
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
	UID             UniqueID                 `json:"uid"`
	Name            string                   `json:"name"`
	Description     string                   `json:"description,omitempty"`
	FeatureType     string                   `json:"featureType,omitempty"`
	ValidTime       *common_shared.TimeRange `json:"validTime,omitempty"`
	Definition      string                   `json:"definition,omitempty"`
	Platform        *common_shared.Link      `json:"platform@link,omitempty"`
	DeployedSystems []DeployedSystemItem     `json:"deployedSystems@link,omitempty"`
}

// DeployedSystemItem represents an entry in the deployment's deployedSystems list
type DeployedSystemItem struct {
	Name          string                              `json:"name"`
	Description   string                              `json:"description,omitempty"`
	System        common_shared.Link                  `json:"system"`
	Configuration common_shared.ConfigurationSettings `json:"configuration,omitempty"`
}

// DeploymentSensorMLFeature represents a Deployment serialized in SensorML JSON format
type DeploymentSensorMLFeature struct {
	ID              string                   `json:"id"`
	Type            string                   `json:"type,omitempty"`
	Label           string                   `json:"label"`
	Description     string                   `json:"description,omitempty"`
	UniqueID        string                   `json:"uniqueId"`
	Definition      string                   `json:"definition,omitempty"`
	ValidTime       *common_shared.TimeRange `json:"validTime,omitempty"`
	Platform        *common_shared.Link      `json:"platform,omitempty"`
	DeployedSystems []DeployedSystemItem     `json:"deployedSystems,omitempty"`
	Links           common_shared.Links      `json:"links,omitempty"`

	Lang                *string                           `gorm:"type:varchar(10)" json:"lang,omitempty"`
	Keywords            []string                          `gorm:"type:jsonb" json:"keywords,omitempty"`
	Identifiers         common_shared.Terms               `gorm:"type:jsonb" json:"identifiers,omitempty"`
	Classifiers         common_shared.Terms               `gorm:"type:jsonb" json:"classifiers,omitempty"`
	SecurityConstraints common_shared.SecurityConstraints `gorm:"type:jsonb" json:"securityConstraints,omitempty"`
	LegalConstraints    common_shared.LegalConstraints    `gorm:"type:jsonb" json:"legalConstraints,omitempty"`

	Characteristics []common_shared.CharacteristicGroup `gorm:"type:jsonb" json:"characteristics,omitempty"`
	Capabilities    []common_shared.CapabilityGroup     `gorm:"type:jsonb" json:"capabilities,omitempty"`
	Contacts        []common_shared.ContactWrapper      `gorm:"type:jsonb" json:"contacts,omitempty"`
	Documentation   common_shared.Documents             `gorm:"type:jsonb" json:"documentation,omitempty"`
	History         common_shared.History               `gorm:"type:jsonb" json:"history,omitempty"`
}
