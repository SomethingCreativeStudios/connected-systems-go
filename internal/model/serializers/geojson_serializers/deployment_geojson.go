package geojson_serializers

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// PropertyGeoJSONSerializer serializes domain Property objects into GeoJSON features.
// It accepts small repository interfaces so it can prefetch related data and avoid N+1 queries.
type DeploymentGeoJSONSerializer struct {
	serializers.Serializer[domains.DeploymentGeoJSONFeature, *domains.Deployment]
	repos *repository.Repositories
}

// NewDeploymentGeoJSONSerializer constructs a serializer with required repository readers.
func NewDeploymentGeoJSONSerializer(repos *repository.Repositories) *DeploymentGeoJSONSerializer {
	return &DeploymentGeoJSONSerializer{repos: repos}
}

// ToFeatures converts a slice of domain deployments into GeoJSON features. It will prefetch
// related deployments and properties if the corresponding repo readers are non-nil.
func (s *DeploymentGeoJSONSerializer) Serialize(ctx context.Context, deployment *domains.Deployment) (domains.DeploymentGeoJSONFeature, error) {
	features, err := s.SerializeAll(ctx, []*domains.Deployment{deployment})
	if err != nil {
		return domains.DeploymentGeoJSONFeature{}, err
	}
	return features[0], nil
}

// ToFeatures converts a slice of domain deployments into GeoJSON features. It will prefetch
// related deployments and properties if the corresponding repo readers are non-nil.
func (s *DeploymentGeoJSONSerializer) SerializeAll(ctx context.Context, deployments []*domains.Deployment) ([]domains.DeploymentGeoJSONFeature, error) {
	if len(deployments) == 0 {
		return []domains.DeploymentGeoJSONFeature{}, nil
	}

	features := make([]domains.DeploymentGeoJSONFeature, 0, len(deployments))
	for _, sys := range deployments {
		if sys == nil {
			continue
		}

		// Start with domain-provided conversion
		f := s.convert(sys)

		features = append(features, f)
	}

	return features, nil
}

func (s *DeploymentGeoJSONSerializer) convert(d *domains.Deployment) domains.DeploymentGeoJSONFeature {
	return domains.DeploymentGeoJSONFeature{
		Type:     "Feature",
		ID:       d.ID,
		Geometry: d.Geometry,
		Properties: domains.DeploymentGeoJSONProperties{
			UID:         d.UniqueIdentifier,
			Name:        d.Name,
			Description: d.Description,
			FeatureType: d.DeploymentType,
			ValidTime:   d.ValidTime,
		},
		Links: d.Links,
	}
}
