package domains

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/seriallizers"
)

// SamplingFeature represents a sosa:Sample feature
type SamplingFeature struct {
	Base
	CommonSSN
	seriallizers.GeoJsonSeriallizable[SystemGeoJSONFeature] `gorm:"-"` // <-- Ignore for GORM

	FeatureType string `gorm:"type:varchar(255)" json:"featureType"`

	// Temporal
	ValidTime *time.Time `gorm:"type:timestamp with time zone" json:"validTime,omitempty"`

	// Spatial - sampling geometry
	Geometry *common_shared.Geometry `gorm:"type:jsonb" json:"geometry,omitempty"`

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
	Geometry   *common_shared.Geometry          `json:"geometry"`
	Properties SamplingFeatureGeoJSONProperties `json:"properties"`
	Links      common_shared.Links              `json:"links,omitempty"`
}

// SamplingFeatureGeoJSONProperties represents the properties object in GeoJSON
type SamplingFeatureGeoJSONProperties struct {
	UID                UniqueID           `json:"uid"`
	Name               string             `json:"name"`
	Description        string             `json:"description,omitempty"`
	FeatureType        string             `json:"featureType"`
	ValidTime          string             `json:"validTime,omitempty"`
	SampledFeatureLink common_shared.Link `json:"sampledFeature@Link,omitempty"`
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
			ValidTime:   sf.ValidTime.String(),
			SampledFeatureLink: common_shared.Link{
				Href:  "features/" + *sf.SampledFeatureID,
				Type:  "application/geo+json",
				Title: "Sampled Feature",
			},
		},
		Links: sf.Links,
	}
}

func (SamplingFeature) BuildFromRequest(r *http.Request, w http.ResponseWriter) (SamplingFeature, error) {
	// Decode GeoJSON Feature format
	var geoJSON struct {
		Type       string                           `json:"type"`
		ID         string                           `json:"id,omitempty"`
		Properties SamplingFeatureGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.Geometry          `json:"geometry,omitempty"`
		Links      common_shared.Links              `json:"links,omitempty"`
	}

	if err := render.DecodeJSON(r.Body, &geoJSON); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return SamplingFeature{}, err
	}

	// Convert GeoJSON properties to System model
	samplingFeatures := SamplingFeature{
		Geometry: geoJSON.Geometry,
		Links:    geoJSON.Links,
	}

	// Extract properties from the properties object
	samplingFeatures.UniqueIdentifier = UniqueID(geoJSON.Properties.UID)

	samplingFeatures.Name = geoJSON.Properties.Name
	samplingFeatures.Description = geoJSON.Properties.Description
	samplingFeatures.FeatureType = geoJSON.Properties.FeatureType

	// Handle validTime if present
	if geoJSON.Properties.ValidTime == "now" {
		timeNow := time.Now()
		samplingFeatures.ValidTime = &timeNow
	} else if geoJSON.Properties.ValidTime != "" {
		parsedTime, _ := time.Parse(time.RFC3339, geoJSON.Properties.ValidTime)
		samplingFeatures.ValidTime = &parsedTime
	}

	parts := strings.Split(geoJSON.Properties.SampledFeatureLink.Href, "/")
	samplingFeatures.SampledFeatureID = &parts[len(parts)-1]

	return samplingFeatures, nil
}
