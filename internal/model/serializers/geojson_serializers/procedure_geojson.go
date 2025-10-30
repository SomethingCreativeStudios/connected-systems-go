package geojson_serializers

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// ProcedureGeoJSONSerializer serializes domain Procedure objects into GeoJSON features.
// It accepts small repository interfaces so it can prefetch related data and avoid N+1 queries.
type ProcedureGeoJSONSerializer struct {
	serializers.Serializer[domains.ProcedureGeoJSONFeature, *domains.Procedure]
	repos *repository.Repositories
}

// NewProcedureGeoJSONSerializer constructs a serializer with required repository readers.
func NewProcedureGeoJSONSerializer(repos *repository.Repositories) *ProcedureGeoJSONSerializer {
	return &ProcedureGeoJSONSerializer{repos: repos}
}

// ToFeatures converts a slice of domain procedures into GeoJSON features. It will prefetch
// related deployments and properties if the corresponding repo readers are non-nil.
func (s *ProcedureGeoJSONSerializer) Serialize(ctx context.Context, procedure *domains.Procedure) (domains.ProcedureGeoJSONFeature, error) {
	features, err := s.SerializeAll(ctx, []*domains.Procedure{procedure})
	if err != nil {
		return domains.ProcedureGeoJSONFeature{}, err
	}
	return features[0], nil
}

// ToFeatures converts a slice of domain procedures into GeoJSON features. It will prefetch
// related deployments and properties if the corresponding repo readers are non-nil.
func (s *ProcedureGeoJSONSerializer) SerializeAll(ctx context.Context, procedures []*domains.Procedure) ([]domains.ProcedureGeoJSONFeature, error) {
	if len(procedures) == 0 {
		return []domains.ProcedureGeoJSONFeature{}, nil
	}

	features := make([]domains.ProcedureGeoJSONFeature, 0, len(procedures))
	for _, sys := range procedures {
		if sys == nil {
			continue
		}

		// Start with domain-provided conversion
		f := s.convert(sys)

		features = append(features, f)
	}

	return features, nil
}

func (s *ProcedureGeoJSONSerializer) convert(p *domains.Procedure) domains.ProcedureGeoJSONFeature {
	return domains.ProcedureGeoJSONFeature{
		Type:     "Feature",
		ID:       p.ID,
		Geometry: nil, // Procedures don't have geometry
		Properties: domains.ProcedureGeoJSONProperties{
			UID:         p.UniqueIdentifier,
			Name:        p.Name,
			Description: p.Description,
			FeatureType: p.FeatureType,
		},
		Links: p.Links,
	}
}
