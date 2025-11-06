package geojson_serializers

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

// FeatureGeoJSONSerializer converts Feature domain to GeoJSON
type FeatureGeoJSONSerializer struct {
	// No additional dependencies needed for basic features
}

// NewFeatureGeoJSONSerializer creates a new FeatureGeoJSONSerializer
func NewFeatureGeoJSONSerializer() *FeatureGeoJSONSerializer {
	return &FeatureGeoJSONSerializer{}
}

// Serialize converts a single Feature to GeoJSON Feature
func (s *FeatureGeoJSONSerializer) Serialize(ctx context.Context, feature *domains.Feature) (domains.FeatureGeoJSONFeature, error) {
	return feature.ToGeoJSON(), nil
}

// SerializeAll converts multiple Features to GeoJSON Features (with batch prefetch if needed)
func (s *FeatureGeoJSONSerializer) SerializeAll(ctx context.Context, features []*domains.Feature) ([]domains.FeatureGeoJSONFeature, error) {
	result := make([]domains.FeatureGeoJSONFeature, 0, len(features))

	for _, feature := range features {
		if feature == nil {
			continue
		}
		result = append(result, feature.ToGeoJSON())
	}

	return result, nil
}
