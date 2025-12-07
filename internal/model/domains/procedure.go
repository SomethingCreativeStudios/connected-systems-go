package domains

import (
	"encoding/json"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// Procedure represents a sosa:Procedure feature (datasheet, methodology)
type Procedure struct {
	Base
	CommonSSN

	// Note: Procedures typically don't have location
	ProcedureType string `gorm:"type:varchar(255)" json:"procedureType,omitempty"`
	ProcessType   string `gorm:"type:varchar(255)" json:"type,omitempty"` // SimpleProcess, AggregateProcess, PhysicalSystem, PhysicalComponent

	// Links to related resources
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional descriptive metadata from the SWE/System schema
	Lang                *string                    `gorm:"type:varchar(10)" json:"lang,omitempty"`
	Keywords            []string                   `gorm:"type:jsonb" json:"keywords,omitempty"`
	Identifiers         common_shared.Terms        `gorm:"type:jsonb" json:"identifiers,omitempty"`
	Classifiers         common_shared.Terms        `gorm:"type:jsonb" json:"classifiers,omitempty"`
	SecurityConstraints []common_shared.Properties `gorm:"type:jsonb" json:"securityConstraints,omitempty"`
	LegalConstraints    []common_shared.Properties `gorm:"type:jsonb" json:"legalConstraints,omitempty"`

	Characteristics []common_shared.CharacteristicGroup `gorm:"type:jsonb" json:"characteristics,omitempty"`
	Capabilities    []common_shared.CapabilityGroup     `gorm:"type:jsonb" json:"capabilities,omitempty"`
	Contacts        []common_shared.ContactWrapper      `gorm:"type:jsonb" json:"contacts,omitempty"`
	Documentation   common_shared.Documents             `gorm:"type:jsonb" json:"documentation,omitempty"`
	History         common_shared.History               `gorm:"type:jsonb" json:"history,omitempty"`

	TypeOf        *common_shared.Link `gorm:"type:jsonb" json:"typeOf,omitempty"`
	Configuration json.RawMessage     `gorm:"type:jsonb" json:"configuration,omitempty"`

	FeaturesOfInterest common_shared.Links `gorm:"type:jsonb" json:"featuresOfInterest,omitempty"`

	Inputs     common_shared.IOList `gorm:"type:jsonb" json:"inputs,omitempty"`
	Outputs    common_shared.IOList `gorm:"type:jsonb" json:"outputs,omitempty"`
	Parameters common_shared.IOList `gorm:"type:jsonb" json:"parameters,omitempty"`
	Method     common_shared.Method `gorm:"type:jsonb" json:"method,omitempty"`
	Modes      json.RawMessage      `gorm:"type:jsonb" json:"modes,omitempty"`

	// Aggregate Process fields
	Components  json.RawMessage `gorm:"type:jsonb" json:"components,omitempty"`
	Connections json.RawMessage `gorm:"type:jsonb" json:"connections,omitempty"`

	AttachedTo           *common_shared.Link           `gorm:"type:jsonb" json:"attachedTo,omitempty"`
	LocalReferenceFrames []common_shared.SpatialFrame  `gorm:"type:jsonb" json:"localReferenceFrames,omitempty"`
	LocalTimeFrames      []common_shared.TemporalFrame `gorm:"type:jsonb" json:"localTimeFrames,omitempty"`

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

// ProcedureSensorMLFeature represents a Procedure serialized in SensorML JSON format
type ProcedureSensorMLFeature struct {
	ID                   string                              `json:"id"`
	Type                 string                              `json:"type,omitempty"`
	Label                string                              `json:"label"`
	Description          string                              `json:"description,omitempty"`
	UniqueID             string                              `json:"uniqueId"`
	Definition           string                              `json:"definition,omitempty"`
	Lang                 *string                             `json:"lang,omitempty"`
	Keywords             []string                            `json:"keywords,omitempty"`
	Identifiers          common_shared.Terms                 `json:"identifiers,omitempty"`
	Classifiers          common_shared.Terms                 `json:"classifiers,omitempty"`
	SecurityConstraints  []common_shared.Properties          `json:"securityConstraints,omitempty"`
	LegalConstraints     []common_shared.Properties          `json:"legalConstraints,omitempty"`
	Characteristics      []common_shared.CharacteristicGroup `json:"characteristics,omitempty"`
	Capabilities         []common_shared.CapabilityGroup     `json:"capabilities,omitempty"`
	Contacts             []common_shared.ContactWrapper      `json:"contacts,omitempty"`
	Documentation        common_shared.Documents             `json:"documentation,omitempty"`
	History              common_shared.History               `json:"history,omitempty"`
	TypeOf               *common_shared.Link                 `json:"typeOf,omitempty"`
	Configuration        json.RawMessage                     `json:"configuration,omitempty"`
	FeaturesOfInterest   common_shared.Links                 `json:"featuresOfInterest,omitempty"`
	Inputs               common_shared.IOList                `json:"inputs,omitempty"`
	Outputs              common_shared.IOList                `json:"outputs,omitempty"`
	Parameters           common_shared.IOList                `json:"parameters,omitempty"`
	Modes                json.RawMessage                     `json:"modes,omitempty"`
	Method               common_shared.Method                `json:"method,omitempty"`
	Components           json.RawMessage                     `json:"components,omitempty"`
	Connections          json.RawMessage                     `json:"connections,omitempty"`
	AttachedTo           *common_shared.Link                 `json:"attachedTo,omitempty"`
	LocalReferenceFrames []common_shared.SpatialFrame        `json:"localReferenceFrames,omitempty"`
	LocalTimeFrames      []common_shared.TemporalFrame       `json:"localTimeFrames,omitempty"`
	Position             json.RawMessage                     `json:"position,omitempty"`
	ValidTime            *common_shared.TimeRange            `json:"validTime,omitempty"`
	Links                common_shared.Links                 `json:"links,omitempty"`
}
