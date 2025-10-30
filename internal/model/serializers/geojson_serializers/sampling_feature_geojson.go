package geojson_serializers

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// PropertyGeoJSONSerializer serializes domain Property objects into GeoJSON features.
// It accepts small repository interfaces so it can prefetch related data and avoid N+1 queries.
type SamplingFeatureGeoJSONSerializer struct {
	serializers.Serializer[domains.SamplingFeatureGeoJSONFeature, *domains.SamplingFeature]
	repos *repository.Repositories
}

// NewSamplingFeatureGeoJSONSerializer constructs a serializer with required repository readers.
func NewSamplingFeatureGeoJSONSerializer(repos *repository.Repositories) *SamplingFeatureGeoJSONSerializer {
	return &SamplingFeatureGeoJSONSerializer{repos: repos}
}

// ToFeatures converts a slice of domain sampling features into GeoJSON features. It will prefetch
// related deployments and properties if the corresponding repo readers are non-nil.
func (s *SamplingFeatureGeoJSONSerializer) Serialize(ctx context.Context, samplingFeature *domains.SamplingFeature) (domains.SamplingFeatureGeoJSONFeature, error) {
	features, err := s.SerializeAll(ctx, []*domains.SamplingFeature{samplingFeature})
	if err != nil {
		return domains.SamplingFeatureGeoJSONFeature{}, err
	}
	return features[0], nil
}

// ToFeatures converts a slice of domain sampling features into GeoJSON features. It will prefetch
// related deployments and properties if the corresponding repo readers are non-nil.
func (s *SamplingFeatureGeoJSONSerializer) SerializeAll(ctx context.Context, samplingFeatures []*domains.SamplingFeature) ([]domains.SamplingFeatureGeoJSONFeature, error) {
	if len(samplingFeatures) == 0 {
		return []domains.SamplingFeatureGeoJSONFeature{}, nil
	}

	features := make([]domains.SamplingFeatureGeoJSONFeature, 0, len(samplingFeatures))
	for _, sys := range samplingFeatures {
		if sys == nil {
			continue
		}

		// Start with domain-provided conversion
		f := s.convert(sys)

		features = append(features, f)
	}

	return features, nil
}

func (s *SamplingFeatureGeoJSONSerializer) convert(sf *domains.SamplingFeature) domains.SamplingFeatureGeoJSONFeature {
	return domains.SamplingFeatureGeoJSONFeature{
		Type:     "Feature",
		ID:       sf.ID,
		Geometry: sf.Geometry,
		Properties: domains.SamplingFeatureGeoJSONProperties{
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
