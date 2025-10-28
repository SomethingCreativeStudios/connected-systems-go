package model

// SamplingFeature represents a sosa:Sample feature
type SamplingFeature struct {
	Base

	// Core properties (from SOSA/SSN)
	UniqueIdentifier UniqueID `gorm:"type:varchar(255);uniqueIndex" json:"uid"`
	Name             string   `gorm:"type:varchar(255);not null" json:"name"`
	Description      string   `gorm:"type:text" json:"description,omitempty"`
	FeatureType      string   `gorm:"type:varchar(255)" json:"featureType"`

	// Temporal
	ValidTime *TimeRange `gorm:"type:jsonb" json:"validTime,omitempty"`

	// Spatial - sampling geometry
	Geometry *Geometry `gorm:"type:jsonb" json:"geometry,omitempty"`

	// Associations
	ParentSystemID *string `gorm:"type:varchar(255);index" json:"-"`

	// Links to related resources
	Links Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional properties
	Properties Properties `gorm:"type:jsonb" json:"properties,omitempty"`
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
	Geometry   *Geometry                        `json:"geometry"`
	Properties SamplingFeatureGeoJSONProperties `json:"properties"`
	Links      Links                            `json:"links,omitempty"`
}

// SamplingFeatureGeoJSONProperties represents the properties object in GeoJSON
type SamplingFeatureGeoJSONProperties struct {
	UID         UniqueID   `json:"uid"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	FeatureType string     `json:"featureType"`
	ValidTime   *TimeRange `json:"validTime,omitempty"`
}

// ToGeoJSON converts SamplingFeature model to GeoJSON Feature
func (sf *SamplingFeature) ToGeoJSON() SamplingFeatureGeoJSONFeature {
	return SamplingFeatureGeoJSONFeature{
		Type:     "Feature",
		ID:       sf.ID,
		Geometry: sf.Geometry,
		Properties: SamplingFeatureGeoJSONProperties{
			UID:         sf.UniqueIdentifier,
			Name:        sf.Name,
			Description: sf.Description,
			FeatureType: sf.FeatureType,
			ValidTime:   sf.ValidTime,
		},
		Links: sf.Links,
	}
}
