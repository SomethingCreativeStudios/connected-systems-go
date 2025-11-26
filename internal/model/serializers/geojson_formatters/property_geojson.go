package geojson_formatters

import (
	"context"
	"encoding/json"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// PropertyGeoJSONFormatter handles serialization and deserialization of Property objects in GeoJSON format
type PropertyGeoJSONFormatter struct {
	serializers.Formatter[domains.PropertyGeoJSONFeature, *domains.Property]
	repos *repository.Repositories
}

// NewPropertyGeoJSONFormatter constructs a formatter with required repository readers
func NewPropertyGeoJSONFormatter(repos *repository.Repositories) *PropertyGeoJSONFormatter {
	return &PropertyGeoJSONFormatter{repos: repos}
}

func (f *PropertyGeoJSONFormatter) ContentType() string {
	return GeoJSONContentType
}

// --- Serialization ---

func (f *PropertyGeoJSONFormatter) Serialize(ctx context.Context, property *domains.Property) (domains.PropertyGeoJSONFeature, error) {
	features, err := f.SerializeAll(ctx, []*domains.Property{property})
	if err != nil {
		return domains.PropertyGeoJSONFeature{}, err
	}
	return features[0], nil
}

func (f *PropertyGeoJSONFormatter) SerializeAll(ctx context.Context, properties []*domains.Property) ([]domains.PropertyGeoJSONFeature, error) {
	if len(properties) == 0 {
		return []domains.PropertyGeoJSONFeature{}, nil
	}

	var features []domains.PropertyGeoJSONFeature
	for _, property := range properties {
		feature := domains.PropertyGeoJSONFeature{
			Type:     "Feature",
			ID:       property.ID,
			Geometry: nil, // Properties don't have spatial geometry
			Properties: domains.PropertyGeoJSONProperties{
				UID:               property.UniqueIdentifier,
				Name:              property.Name,
				Description:       property.Description,
				Definition:        property.Definition,
				PropertyType:      property.PropertyType,
				BaseProperty:      property.BaseProperty,
				ObjectType:        property.ObjectType,
				Statistic:         property.Statistic,
				Qualifiers:        property.Qualifiers,
				UnitOfMeasurement: property.UnitOfMeasurement,
			},
			Links: property.Links,
		}
		features = append(features, feature)
	}

	return features, nil
}

// --- Deserialization ---

func (f *PropertyGeoJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Property, error) {
	var geoJSON struct {
		Type       string                            `json:"type"`
		ID         string                            `json:"id,omitempty"`
		Properties domains.PropertyGeoJSONProperties `json:"properties"`
		Geometry   interface{}                       `json:"geometry,omitempty"`
		Links      common_shared.Links               `json:"links,omitempty"`
	}

	if err := json.NewDecoder(reader).Decode(&geoJSON); err != nil {
		return nil, err
	}

	property := &domains.Property{
		Links: geoJSON.Links,
	}

	// Extract properties
	property.UniqueIdentifier = domains.UniqueID(geoJSON.Properties.UID)
	property.Name = geoJSON.Properties.Name
	property.Description = geoJSON.Properties.Description
	property.Definition = geoJSON.Properties.Definition
	property.PropertyType = geoJSON.Properties.PropertyType
	property.BaseProperty = geoJSON.Properties.BaseProperty
	property.ObjectType = geoJSON.Properties.ObjectType
	property.Statistic = geoJSON.Properties.Statistic
	property.Qualifiers = geoJSON.Properties.Qualifiers
	property.UnitOfMeasurement = geoJSON.Properties.UnitOfMeasurement

	return property, nil
}
