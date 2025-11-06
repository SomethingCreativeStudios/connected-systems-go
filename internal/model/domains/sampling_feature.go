package domains

import (
	"net/http"
	"strings"

	"github.com/go-chi/render"
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
	ParentSystemID *string `gorm:"type:varchar(255);index" json:"-"`

	SampledFeatureID *string   `gorm:"type:varchar(255);index" json:"featureId"`
	SampleOf         *[]string `gorm:"type:varchar(255)[]" json:"sampleOf,omitempty"`

	// Links to related resources
	Links common_shared.Links `gorm:"type:jsonb" json:"links,omitempty"`

	// Additional properties
	Properties common_shared.Properties `gorm:"type:jsonb" json:"properties,omitempty"`
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

func (SamplingFeature) BuildFromRequest(r *http.Request, w http.ResponseWriter) (SamplingFeature, error) {
	// Decode GeoJSON Feature format
	var geoJSON struct {
		Type       string                           `json:"type"`
		ID         string                           `json:"id,omitempty"`
		Properties SamplingFeatureGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.GoGeom            `json:"geometry,omitempty"`
		Links      common_shared.Links              `json:"links,omitempty"`
	}

	if err := render.DecodeJSON(r.Body, &geoJSON); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return SamplingFeature{}, err
	}

	// Convert GeoJSON properties to System model
	samplingFeatures := SamplingFeature{
		Links: geoJSON.Links,
	}
	if geoJSON.Geometry != nil {
		samplingFeatures.Geometry = geoJSON.Geometry
	}

	// Extract properties from the properties object
	samplingFeatures.UniqueIdentifier = UniqueID(geoJSON.Properties.UID)

	samplingFeatures.Name = geoJSON.Properties.Name
	samplingFeatures.Description = geoJSON.Properties.Description
	samplingFeatures.FeatureType = geoJSON.Properties.FeatureType
	samplingFeatures.ValidTime = geoJSON.Properties.ValidTime

	parts := strings.Split(geoJSON.Properties.SampledFeatureLink.Href, "/")
	samplingFeatures.SampledFeatureID = &parts[len(parts)-1]

	return samplingFeatures, nil
}
