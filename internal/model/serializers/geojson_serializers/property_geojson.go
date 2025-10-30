package geojson_serializers

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// PropertyGeoJSONSerializer serializes domain Property objects into GeoJSON features.
// It accepts small repository interfaces so it can prefetch related data and avoid N+1 queries.
type PropertyGeoJSONSerializer struct {
	serializers.Serializer[domains.PropertyGeoJSONFeature, *domains.Property]
	repos *repository.Repositories
}

// NewPropertyGeoJSONSerializer constructs a serializer with required repository readers.
func NewPropertyGeoJSONSerializer(repos *repository.Repositories) *PropertyGeoJSONSerializer {
	return &PropertyGeoJSONSerializer{repos: repos}
}

// ToFeatures converts a slice of domain propertys into GeoJSON features. It will prefetch
// related deployments and properties if the corresponding repo readers are non-nil.
func (s *PropertyGeoJSONSerializer) Serialize(ctx context.Context, property *domains.Property) (domains.PropertyGeoJSONFeature, error) {
	features, err := s.SerializeAll(ctx, []*domains.Property{property})
	if err != nil {
		return domains.PropertyGeoJSONFeature{}, err
	}
	return features[0], nil
}

// ToFeatures converts a slice of domain propertys into GeoJSON features. It will prefetch
// related deployments and properties if the corresponding repo readers are non-nil.
func (s *PropertyGeoJSONSerializer) SerializeAll(ctx context.Context, propertys []*domains.Property) ([]domains.PropertyGeoJSONFeature, error) {
	if len(propertys) == 0 {
		return []domains.PropertyGeoJSONFeature{}, nil
	}

	features := make([]domains.PropertyGeoJSONFeature, 0, len(propertys))
	for _, sys := range propertys {
		if sys == nil {
			continue
		}

		// Start with domain-provided conversion
		f := s.convert(sys)

		features = append(features, f)
	}

	return features, nil
}

func (s *PropertyGeoJSONSerializer) convert(d *domains.Property) domains.PropertyGeoJSONFeature {
	return domains.PropertyGeoJSONFeature{
		Type:     "Feature",
		ID:       d.ID,
		Geometry: nil, // Properties don't have geometry
		Properties: domains.PropertyGeoJSONProperties{
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
