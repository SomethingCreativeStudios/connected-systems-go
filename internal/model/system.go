package model

// System represents a sosa:System feature
// Can be a Sensor, Actuator, Sampler, or Platform
type System struct {
	Base

	// Core properties (from SOSA/SSN)
	UniqueIdentifier UniqueID `gorm:"type:varchar(255);uniqueIndex" json:"uid"`
	Name             string   `gorm:"type:varchar(255);not null" json:"name"`
	Description      string   `gorm:"type:text" json:"description,omitempty"`
	SystemType       string   `gorm:"type:varchar(255);not null" json:"featureType"` // sosa:Sensor, sosa:Actuator, sosa:Platform, etc.
	AssetType        *string  `gorm:"type:varchar(100)" json:"assetType,omitempty"`  // Equipment, Human, Platform, etc.

	// Temporal
	ValidTime *TimeRange `gorm:"type:jsonb" json:"validTime,omitempty"`

	// Spatial
	Geometry *Geometry `gorm:"type:jsonb" json:"geometry,omitempty"`

	// Associations (stored as links in JSON)
	ParentSystemID *string `gorm:"type:varchar(255);index" json:"-"`

	// Links to related resources
	Links Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional properties
	Properties Properties `gorm:"type:jsonb" json:"properties,omitempty"`
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
	Geometry   *Geometry               `json:"geometry"`
	Properties SystemGeoJSONProperties `json:"properties"`
	Links      Links                   `json:"links,omitempty"`
}

// SystemGeoJSONProperties represents the properties object in GeoJSON
type SystemGeoJSONProperties struct {
	UID         UniqueID   `json:"uid"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	FeatureType string     `json:"featureType"`
	AssetType   *string    `json:"assetType,omitempty"`
	ValidTime   *TimeRange `json:"validTime,omitempty"`
	SystemKind  *Link      `json:"systemKind@link,omitempty"`
}

// ToGeoJSON converts System model to GeoJSON Feature
func (s *System) ToGeoJSON() SystemGeoJSONFeature {
	extraLinks := Links{}

	// Add parent system link if applicable
	if s.ParentSystemID != nil {
		extraLinks = append(extraLinks, Link{
			Rel:  "ogc-rel:parentSystem",
			Href: "/systems/" + *s.ParentSystemID,
		})
	}

	// Combine existing links with extra links
	s.Links = append(s.Links, extraLinks...)

	return SystemGeoJSONFeature{
		Type:     "Feature",
		ID:       s.ID,
		Geometry: s.Geometry,
		Properties: SystemGeoJSONProperties{
			UID:         s.UniqueIdentifier,
			Name:        s.Name,
			Description: s.Description,
			FeatureType: s.SystemType,
			AssetType:   s.AssetType,
			ValidTime:   s.ValidTime,
		},
		Links: s.Links,
	}
}
