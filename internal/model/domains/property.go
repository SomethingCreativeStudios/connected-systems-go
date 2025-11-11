package domains

import (
	"net/http"

	"github.com/go-chi/render"
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

// PropertyGeoJSONFeature converts Deployment to GeoJSON Feature format
type PropertyGeoJSONFeature struct {
	Type       string                    `json:"type"`
	ID         string                    `json:"id"`
	Geometry   *common_shared.GoGeom     `json:"geometry"`
	Properties PropertyGeoJSONProperties `json:"properties"`
	Links      common_shared.Links       `json:"links,omitempty"`
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

func (Property) BuildFromRequest(r *http.Request, w http.ResponseWriter) (Property, error) {
	// Decode GeoJSON Feature format
	var geoJSON struct {
		Type       string                    `json:"type"`
		ID         string                    `json:"id,omitempty"`
		Properties PropertyGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.GoGeom     `json:"geometry,omitempty"`
		Links      common_shared.Links       `json:"links,omitempty"`
	}

	if err := render.DecodeJSON(r.Body, &geoJSON); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return Property{}, err
	}

	// Convert GeoJSON properties to Property model
	property := Property{
		Links: geoJSON.Links,
	}

	// If geometry provided (although Property doesn't store geometry currently), validate/convert it
	if geoJSON.Geometry != nil {
		// decoded into GoGeom; not stored on Property but validated by unmarshalling
		// (no further action required)
	}

	// Extract properties from the properties object
	property.UniqueIdentifier = UniqueID(geoJSON.Properties.UID)

	property.Name = geoJSON.Properties.Name
	property.Description = geoJSON.Properties.Description
	property.Definition = geoJSON.Properties.Definition
	property.PropertyType = geoJSON.Properties.PropertyType
	property.ObjectType = geoJSON.Properties.ObjectType
	property.Definition = geoJSON.Properties.Definition
	property.UnitOfMeasurement = geoJSON.Properties.UnitOfMeasurement

	return property, nil
}
