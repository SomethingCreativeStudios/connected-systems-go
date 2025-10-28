package model

// Property represents a sosa:ObservableProperty or sosa:ActuableProperty
type Property struct {
	Base

	// Core properties (from SOSA/SSN)
	UniqueIdentifier UniqueID `gorm:"type:varchar(255);uniqueIndex" json:"uid"`
	Name             string   `gorm:"type:varchar(255);not null" json:"name"`
	Description      string   `gorm:"type:text" json:"description,omitempty"`
	Definition       string   `gorm:"type:varchar(500)" json:"definition,omitempty"` // URI to property definition

	// Property type
	PropertyType string `gorm:"type:varchar(100)" json:"propertyType,omitempty"` // Observable, Actuable, etc.

	// Object type this property applies to
	ObjectType *string `gorm:"type:varchar(255)" json:"objectType,omitempty"`

	// Unit of measurement
	UnitOfMeasurement *string `gorm:"type:varchar(100)" json:"uom,omitempty"`

	// Links to related resources
	Links Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional properties
	Properties Properties `gorm:"type:jsonb" json:"properties,omitempty"`
}

// TableName specifies the table name
func (Property) TableName() string {
	return "properties"
}

// PropertyType constants (SOSA/SSN)
const (
	PropertyTypeObservable = "http://www.w3.org/ns/sosa/ObservableProperty"
	PropertyTypeActuable   = "http://www.w3.org/ns/sosa/ActuableProperty"
)

// PropertyGeoJSONFeature converts Deployment to GeoJSON Feature format
type PropertyGeoJSONFeature struct {
	Type       string                    `json:"type"`
	ID         string                    `json:"id"`
	Geometry   *Geometry                 `json:"geometry"`
	Properties PropertyGeoJSONProperties `json:"properties"`
	Links      Links                     `json:"links,omitempty"`
}

// PropertyGeoJSONProperties represents the properties object in GeoJSON
type PropertyGeoJSONProperties struct {
	UID               UniqueID `json:"uid"`
	Name              string   `json:"name"`
	Description       string   `json:"description,omitempty"`
	FeatureType       string   `json:"featureType,omitempty"`
	PropertyType      string   `json:"propertyType,omitempty"`
	Definition        string   `json:"definition,omitempty"`
	ObjectType        *string  `json:"objectType,omitempty"`
	UnitOfMeasurement *string  `json:"uom,omitempty"`
}

// ToGeoJSON converts Deployment model to GeoJSON Feature
func (d *Property) ToGeoJSON() PropertyGeoJSONFeature {
	return PropertyGeoJSONFeature{
		Type:     "Feature",
		ID:       d.ID,
		Geometry: nil, // Properties don't have geometry
		Properties: PropertyGeoJSONProperties{
			UID:               d.UniqueIdentifier,
			Name:              d.Name,
			Description:       d.Description,
			FeatureType:       "sosa:Property",
			PropertyType:      d.PropertyType,
			ObjectType:        d.ObjectType,
			Definition:        d.Definition,
			UnitOfMeasurement: d.UnitOfMeasurement,
		},
		Links: d.Links,
	}
}
