package domains

import (
	"encoding/json"

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

	SystemKindID *string `gorm:"type:varchar(255);index" json:"-"`

	// Additional SWE/System metadata mapped from the JSON Schema
	Lang                *string                    `gorm:"type:varchar(10)" json:"lang,omitempty"`
	Keywords            []string                   `gorm:"type:jsonb" json:"keywords,omitempty"`
	Identifiers         common_shared.Terms        `gorm:"type:jsonb" json:"identifiers,omitempty"`
	Classifiers         common_shared.Terms        `gorm:"type:jsonb" json:"classifiers,omitempty"`
	SecurityConstraints []common_shared.Properties `gorm:"type:jsonb" json:"securityConstraints,omitempty"`
	LegalConstraints    []common_shared.Properties `gorm:"type:jsonb" json:"legalConstraints,omitempty"`

	// Documentation/contacts/history at top-level (also present in SMLProperties)
	Contacts      []common_shared.ContactWrapper `gorm:"type:jsonb" json:"contacts,omitempty"`
	Documentation common_shared.Documents        `gorm:"type:jsonb" json:"documentation,omitempty"`
	History       common_shared.History          `gorm:"type:jsonb" json:"history,omitempty"`

	// Process-level fields (also mirrored inside SensorML properties)
	TypeOf             *common_shared.Link  `gorm:"type:jsonb" json:"typeOf,omitempty"`
	Configuration      json.RawMessage      `gorm:"type:jsonb" json:"configuration,omitempty"`
	FeaturesOfInterest common_shared.Links  `gorm:"type:jsonb" json:"featuresOfInterest,omitempty"`
	Inputs             common_shared.IOList `gorm:"type:jsonb" json:"inputs,omitempty"`
	Outputs            common_shared.IOList `gorm:"type:jsonb" json:"outputs,omitempty"`
	Parameters         common_shared.IOList `gorm:"type:jsonb" json:"parameters,omitempty"`
	Modes              json.RawMessage      `gorm:"type:jsonb" json:"modes,omitempty"`

	// Spatial frame / position
	AttachedTo           *common_shared.Link           `gorm:"type:jsonb" json:"attachedTo,omitempty"`
	LocalReferenceFrames []common_shared.SpatialFrame  `gorm:"type:jsonb" json:"localReferenceFrames,omitempty"`
	LocalTimeFrames      []common_shared.TemporalFrame `gorm:"type:jsonb" json:"localTimeFrames,omitempty"`
	Position             json.RawMessage               `gorm:"type:jsonb" json:"position,omitempty"`

	// Links to related resources
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	SystemKind Procedure `gorm:"foreignKey:SystemKindID;" json:"-"`

	// Associations
	Procedures  []Procedure  `gorm:"many2many:system_procedures;"`
	Deployments []Deployment `gorm:"many2many:system_deployments;"`
	//SamplingFeatures []SamplingFeature `gorm:"foreignKey:ParentSystemID;"`
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
	UID                  UniqueID                       `json:"uid"`
	Name                 string                         `json:"name"`
	Description          string                         `json:"description,omitempty"`
	FeatureType          string                         `json:"featureType"`
	AssetType            *string                        `json:"assetType,omitempty"`
	ValidTime            *common_shared.TimeRange       `json:"validTime,omitempty"`
	SystemKind           *common_shared.Link            `json:"systemKind@link,omitempty"`
	Lang                 *string                        `json:"lang,omitempty"`
	Keywords             []string                       `json:"keywords,omitempty"`
	Identifiers          common_shared.Terms            `json:"identifiers,omitempty"`
	Classifiers          common_shared.Terms            `json:"classifiers,omitempty"`
	Contacts             []common_shared.ContactWrapper `json:"contacts,omitempty"`
	Documentation        common_shared.Documents        `json:"documentation,omitempty"`
	History              common_shared.History          `json:"history,omitempty"`
	TypeOf               *common_shared.Link            `json:"typeOf,omitempty"`
	Configuration        json.RawMessage                `json:"configuration,omitempty"`
	FeaturesOfInterest   common_shared.Links            `json:"featuresOfInterest,omitempty"`
	Inputs               common_shared.IOList           `json:"inputs,omitempty"`
	Outputs              common_shared.IOList           `json:"outputs,omitempty"`
	Parameters           common_shared.IOList           `json:"parameters,omitempty"`
	Modes                json.RawMessage                `json:"modes,omitempty"`
	AttachedTo           *common_shared.Link            `json:"attachedTo,omitempty"`
	LocalReferenceFrames []common_shared.SpatialFrame   `json:"localReferenceFrames,omitempty"`
	LocalTimeFrames      []common_shared.TemporalFrame  `json:"localTimeFrames,omitempty"`
	Position             json.RawMessage                `json:"position,omitempty"`
}

// SystemSensorMLFeature represents a System serialized in SensorML JSON format
type SystemSensorMLFeature struct {
	ID                   string                         `json:"id"`
	Type                 string                         `json:"type"`
	Label                string                         `json:"label"`
	Description          string                         `json:"description,omitempty"`
	UniqueID             string                         `json:"uniqueId"`
	Lang                 *string                        `json:"lang,omitempty"`
	Keywords             []string                       `json:"keywords,omitempty"`
	Identifiers          common_shared.Terms            `json:"identifiers,omitempty"`
	Classifiers          common_shared.Terms            `json:"classifiers,omitempty"`
	SecurityConstraints  []common_shared.Properties     `json:"securityConstraints,omitempty"`
	LegalConstraints     []common_shared.Properties     `json:"legalConstraints,omitempty"`
	Contacts             []common_shared.ContactWrapper `json:"contacts,omitempty"`
	Documentation        common_shared.Documents        `json:"documentation,omitempty"`
	History              common_shared.History          `json:"history,omitempty"`
	Definition           string                         `json:"definition,omitempty"`
	TypeOf               *common_shared.Link            `json:"typeOf,omitempty"`
	Configuration        json.RawMessage                `json:"configuration,omitempty"`
	FeaturesOfInterest   common_shared.Links            `json:"featuresOfInterest,omitempty"`
	Inputs               common_shared.IOList           `json:"inputs,omitempty"`
	Outputs              common_shared.IOList           `json:"outputs,omitempty"`
	Parameters           common_shared.IOList           `json:"parameters,omitempty"`
	Modes                json.RawMessage                `json:"modes,omitempty"`
	Position             json.RawMessage                `json:"position,omitempty"`
	AttachedTo           *common_shared.Link            `json:"attachedTo,omitempty"`
	LocalReferenceFrames []common_shared.SpatialFrame   `json:"localReferenceFrames,omitempty"`
	LocalTimeFrames      []common_shared.TemporalFrame  `json:"localTimeFrames,omitempty"`
	Links                common_shared.Links            `json:"links,omitempty"`
}
