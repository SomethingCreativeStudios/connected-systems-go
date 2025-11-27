package domains

import (
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// SamplingFeature represents a sosa:Sample feature
type SamplingFeature struct {
	Base
	CommonSSN

	FeatureType string `gorm:"type:varchar(255)" json:"featureType"`

	// Temporal
	ValidTime *common_shared.TimeRange `gorm:"embedded;embeddedPrefix:valid_time_" json:"validTime,omitempty"`

	// Spatial - sampling geometry
	Geometry *common_shared.GoGeom `gorm:"type:geometry" json:"geometry,omitempty"`

	// Associations
	// store parent system id; put FK constraint on the column to avoid duplicate constraint definitions
	ParentSystemID  *string `gorm:"type:varchar(255);index;" json:"parentSystemId,omitempty"`
	ParentSystemUID *string `gorm:"type:varchar(255)" json:"parentSystemUid,omitempty"`

	SampledFeatureID   *string             `gorm:"type:varchar(255);index" json:"featureId"`
	SampledFeatureUID  *string             `gorm:"type:varchar(255)" json:"featureUid,omitempty"`
	SampledFeatureLink *common_shared.Link `gorm:"type:jsonb" json:"sampledFeature@Link,omitempty"`

	SampleOfIDs  *[]string            `gorm:"-" json:"sampleOfIds,omitempty"`
	SampleOfUIDs *[]string            `gorm:"-" json:"sampleOfUids,omitempty"`
	SampleOf     *common_shared.Links `gorm:"type:jsonb" json:"sampleOf,omitempty"`

	// Links to related resources
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional properties
	Properties common_shared.Properties `gorm:"type:jsonb" json:"properties,omitempty"`

	// Optional back-reference to parent system. Don't specify foreignKey here since it's already
	// defined on the System side (System.SamplingFeatures with foreignKey:ParentSystemID).
	// Specifying it on both sides causes GORM to create duplicate/conflicting FK constraints.
	ParentSystem *System `gorm:"->;references:ID" json:"parentSystem,omitempty"`
}

// TableName specifies the table name
func (SamplingFeature) TableName() string {
	return "sampling_features"
}

// SamplingFeatureType constants (SOSA/SSN)
const (
	SamplingFeatureTypeSample = "http://www.w3.org/ns/sosa/Sample"
)

// SamplingFeatureGeoJSONFeature converts SamplingFeature to GeoJSON Feature format
type SamplingFeatureGeoJSONFeature struct {
	Type       string                           `json:"type"`
	ID         string                           `json:"id"`
	Geometry   *common_shared.GoGeom            `json:"geometry"`
	Properties SamplingFeatureGeoJSONProperties `json:"properties"`
	Links      common_shared.Links              `json:"links,omitempty"`
}

// SamplingFeatureGeoJSONProperties represents the properties object in GeoJSON
type SamplingFeatureGeoJSONProperties struct {
	UID                UniqueID                 `json:"uid"`
	Name               string                   `json:"name"`
	Description        string                   `json:"description,omitempty"`
	FeatureType        string                   `json:"featureType"`
	ValidTime          *common_shared.TimeRange `json:"validTime,omitempty"`
	SampledFeatureLink common_shared.Link       `json:"sampledFeature@Link,omitempty"`
}

// SamplingFeatureSensorMLFeature represents a SamplingFeature serialized in SensorML JSON format
type SamplingFeatureSensorMLFeature struct {
	ID                 string                   `json:"id"`
	Type               string                   `json:"type,omitempty"`
	Label              string                   `json:"label"`
	Description        string                   `json:"description,omitempty"`
	UniqueID           string                   `json:"uniqueId"`
	Definition         string                   `json:"definition,omitempty"`
	ValidTime          *common_shared.TimeRange `json:"validTime,omitempty"`
	SampledFeatureLink *common_shared.Link      `json:"sampledFeature,omitempty"`
	SampleOf           *common_shared.Links     `json:"sampleOf,omitempty"`
	Links              common_shared.Links      `json:"links,omitempty"`
}
