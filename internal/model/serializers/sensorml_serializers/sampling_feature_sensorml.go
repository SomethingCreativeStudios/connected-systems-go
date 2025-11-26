package sensorml_serializers

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// SamplingFeatureSensorMLSerializer serializes domain SamplingFeature objects into SensorML format.
type SamplingFeatureSensorMLSerializer struct {
	serializers.Serializer[domains.SamplingFeatureSensorMLFeature, *domains.SamplingFeature]
	repos *repository.Repositories
}

// NewSamplingFeatureSensorMLSerializer constructs a serializer with required repository readers.
func NewSamplingFeatureSensorMLSerializer(repos *repository.Repositories) *SamplingFeatureSensorMLSerializer {
	return &SamplingFeatureSensorMLSerializer{repos: repos}
}

func (s *SamplingFeatureSensorMLSerializer) Serialize(ctx context.Context, samplingFeature *domains.SamplingFeature) (domains.SamplingFeatureSensorMLFeature, error) {
	features, err := s.SerializeAll(ctx, []*domains.SamplingFeature{samplingFeature})
	if err != nil {
		return domains.SamplingFeatureSensorMLFeature{}, err
	}
	return features[0], nil
}

func (s *SamplingFeatureSensorMLSerializer) SerializeAll(ctx context.Context, samplingFeatures []*domains.SamplingFeature) ([]domains.SamplingFeatureSensorMLFeature, error) {
	if len(samplingFeatures) == 0 {
		return []domains.SamplingFeatureSensorMLFeature{}, nil
	}

	var features []domains.SamplingFeatureSensorMLFeature
	for _, sf := range samplingFeatures {
		feature := domains.SamplingFeatureSensorMLFeature{
			ID:                 sf.ID,
			Type:               sf.FeatureType,
			Label:              sf.Name,
			Description:        sf.Description,
			UniqueID:           string(sf.UniqueIdentifier),
			Definition:         sf.FeatureType, // FeatureType serves as definition URI
			ValidTime:          sf.ValidTime,
			SampledFeatureLink: sf.SampledFeatureLink,
			SampleOf:           sf.SampleOf,
			Links:              sf.Links,
		}
		features = append(features, feature)
	}
	return features, nil
}
