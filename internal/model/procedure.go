package model

// Procedure represents a sosa:Procedure feature (datasheet, methodology)
type Procedure struct {
	Base

	// Core properties (from SOSA/SSN)
	UniqueIdentifier UniqueID `gorm:"type:varchar(255);uniqueIndex" json:"uid"`
	Name             string   `gorm:"type:varchar(255);not null" json:"name"`
	Description      string   `gorm:"type:text" json:"description,omitempty"`
	ProcedureType    string   `gorm:"type:varchar(255)" json:"featureType,omitempty"`

	// Temporal
	ValidTime *TimeRange `gorm:"type:jsonb" json:"validTime,omitempty"`

	// Note: Procedures typically don't have location

	// Links to related resources
	Links Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional properties
	Properties Properties `gorm:"type:jsonb" json:"properties,omitempty"`
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
	Geometry   *Geometry                  `json:"geometry"` // Always null for procedures
	Properties ProcedureGeoJSONProperties `json:"properties"`
	Links      Links                      `json:"links,omitempty"`
}

// ProcedureGeoJSONProperties represents the properties object in GeoJSON
type ProcedureGeoJSONProperties struct {
	UID         UniqueID   `json:"uid"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	FeatureType string     `json:"featureType,omitempty"`
	ValidTime   *TimeRange `json:"validTime,omitempty"`
}

// ToGeoJSON converts Procedure model to GeoJSON Feature
func (p *Procedure) ToGeoJSON() ProcedureGeoJSONFeature {
	return ProcedureGeoJSONFeature{
		Type:     "Feature",
		ID:       p.ID,
		Geometry: nil, // Procedures don't have geometry
		Properties: ProcedureGeoJSONProperties{
			UID:         p.UniqueIdentifier,
			Name:        p.Name,
			Description: p.Description,
			FeatureType: p.ProcedureType,
			ValidTime:   p.ValidTime,
		},
		Links: p.Links,
	}
}
