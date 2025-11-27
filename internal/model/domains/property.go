package domains

import (
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// Property represents a sosa:ObservableProperty or sosa:ActuableProperty
type Property struct {
	Base
	CommonSSN

	Definition string `gorm:"type:varchar(500)" json:"definition,omitempty"` // URI to property definition

	// Property type
	PropertyType string `gorm:"type:varchar(100)" json:"propertyType,omitempty"` // Observable, Actuable, etc.

	// Object type this property applies to
	ObjectType *string `gorm:"type:varchar(255)" json:"objectType,omitempty"`

	// Object type this property applies to
	BaseProperty *string `gorm:"type:varchar(255)" json:"baseProperty,omitempty"`

	// Statistic: URI pointing to definition of statistic applied to values
	Statistic *string `gorm:"type:varchar(255)" json:"statistic,omitempty"`

	// Qualifiers: additional data components used to further qualify the property
	Qualifiers []common_shared.ComponentWrapper `gorm:"type:jsonb" json:"qualifiers,omitempty"`

	// Unit of measurement
	UnitOfMeasurement *string `gorm:"type:varchar(100)" json:"uom,omitempty"`

	// Links to related resources
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional properties
	Properties common_shared.Properties `gorm:"type:jsonb" json:"properties,omitempty"`
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

type PropertySensorMLFeature struct {
	ID           string                           `json:"id"`
	Label        string                           `json:"label"`
	Description  string                           `json:"description,omitempty"`
	UniqueID     string                           `json:"uniqueId"`
	BaseProperty *string                          `json:"baseProperty,omitempty"`
	ObjectType   *string                          `json:"objectType,omitempty"`
	Statistic    *string                          `json:"statistic,omitempty"`
	Qualifiers   []common_shared.ComponentWrapper `json:"qualifiers,omitempty"`
	Links        common_shared.Links              `json:"links,omitempty"`
}

// PropertyGeoJSONFeature represents a Property serialized as GeoJSON Feature
type PropertyGeoJSONFeature struct {
	Type       string                    `json:"type"`
	ID         string                    `json:"id"`
	Geometry   interface{}               `json:"geometry"` // null for properties (no spatial component)
	Properties PropertyGeoJSONProperties `json:"properties"`
	Links      common_shared.Links       `json:"links,omitempty"`
}

// PropertyGeoJSONProperties represents the properties object in GeoJSON for a Property
type PropertyGeoJSONProperties struct {
	UID               UniqueID                         `json:"uid"`
	Name              string                           `json:"name"`
	Description       string                           `json:"description,omitempty"`
	Definition        string                           `json:"definition,omitempty"`
	PropertyType      string                           `json:"propertyType,omitempty"`
	BaseProperty      *string                          `json:"baseProperty,omitempty"`
	ObjectType        *string                          `json:"objectType,omitempty"`
	Statistic         *string                          `json:"statistic,omitempty"`
	Qualifiers        []common_shared.ComponentWrapper `json:"qualifiers,omitempty"`
	UnitOfMeasurement *string                          `json:"uom,omitempty"`
}
