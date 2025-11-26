package geojson_serializers

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// PropertyGeoJSONSerializer serializes domain Property objects into GeoJSON features.
type PropertyGeoJSONSerializer struct {
	serializers.Serializer[domains.PropertyGeoJSONFeature, *domains.Property]
	repos *repository.Repositories
}

// NewPropertyGeoJSONSerializer constructs a serializer with required repository readers.
func NewPropertyGeoJSONSerializer(repos *repository.Repositories) *PropertyGeoJSONSerializer {
	return &PropertyGeoJSONSerializer{repos: repos}
}

func (s *PropertyGeoJSONSerializer) Serialize(ctx context.Context, property *domains.Property) (domains.PropertyGeoJSONFeature, error) {
	features, err := s.SerializeAll(ctx, []*domains.Property{property})
	if err != nil {
		return domains.PropertyGeoJSONFeature{}, err
	}
	return features[0], nil
}

func (s *PropertyGeoJSONSerializer) SerializeAll(ctx context.Context, properties []*domains.Property) ([]domains.PropertyGeoJSONFeature, error) {
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
