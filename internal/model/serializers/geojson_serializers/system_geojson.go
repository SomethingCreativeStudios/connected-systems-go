package geojson_serializers

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// SystemGeoJSONSerializer serializes domain System objects into GeoJSON features.
// It accepts small repository interfaces so it can prefetch related data and avoid N+1 queries.
type SystemGeoJSONSerializer struct {
	serializers.Serializer[domains.SystemGeoJSONFeature, *domains.System]
	repos *repository.Repositories
}

// NewSystemGeoJSONSerializer constructs a serializer with required repository readers.
func NewSystemGeoJSONSerializer(repos *repository.Repositories) *SystemGeoJSONSerializer {
	return &SystemGeoJSONSerializer{repos: repos}
}

// ToFeatures converts a slice of domain systems into GeoJSON features. It will prefetch
// related deployments and properties if the corresponding repo readers are non-nil.
func (s *SystemGeoJSONSerializer) Serialize(ctx context.Context, system *domains.System) (domains.SystemGeoJSONFeature, error) {
	features, err := s.SerializeAll(ctx, []*domains.System{system})
	if err != nil {
		return domains.SystemGeoJSONFeature{}, err
	}
	return features[0], nil
}

// ToFeatures converts a slice of domain systems into GeoJSON features. It will prefetch
// related deployments and properties if the corresponding repo readers are non-nil.
func (s *SystemGeoJSONSerializer) SerializeAll(ctx context.Context, systems []*domains.System) ([]domains.SystemGeoJSONFeature, error) {
	if len(systems) == 0 {
		return []domains.SystemGeoJSONFeature{}, nil
	}

	ids := make([]string, 0, len(systems))
	for _, ss := range systems {
		if ss != nil {
			ids = append(ids, ss.ID)
		}
	}

	var (
		deployMap map[string][]*domains.Deployment
		propMap   map[string][]*domains.Property
		err       error
	)

	if s.repos.Deployment != nil {
		deployMap, err = s.repos.Deployment.GetBySystemIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
	} else {
		deployMap = map[string][]*domains.Deployment{}
	}

	if s.repos.Property != nil {
		propMap, err = s.repos.Property.GetBySystemIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
	} else {
		propMap = map[string][]*domains.Property{}
	}

	features := make([]domains.SystemGeoJSONFeature, 0, len(systems))
	for _, sys := range systems {
		if sys == nil {
			continue
		}

		// Start with domain-provided conversion
		f := s.convert(sys)

		// Optionally attach counts or links derived from prefetch data
		if deps := deployMap[sys.ID]; len(deps) > 0 {
			// add a link to indicate deployments existence
			f.Links = append(f.Links, common_shared.Link{
				Href: "/deployments/" + sys.ID,
			})
		}

		if props := propMap[sys.ID]; len(props) > 0 {
			f.Links = append(f.Links, common_shared.Link{
				Href: "/properties/" + sys.ID,
			})
		}

		features = append(features, f)
	}

	return features, nil
}

func (s *SystemGeoJSONSerializer) convert(system *domains.System) domains.SystemGeoJSONFeature {
	extraLinks := common_shared.Links{}

	// Add parent system link if applicable
	if system.ParentSystemID != nil {
		extraLinks = append(extraLinks, common_shared.Link{
			Rel:  "parentSystem",
			Href: "/systems/" + *system.ParentSystemID,
		})
	}

	// Combine existing links with extra links
	system.Links = append(system.Links, extraLinks...)

	return domains.SystemGeoJSONFeature{
		Type:     "Feature",
		ID:       system.ID,
		Geometry: system.Geometry,
		Properties: domains.SystemGeoJSONProperties{
			UID:         system.UniqueIdentifier,
			Name:        system.Name,
			Description: system.Description,
			FeatureType: system.SystemType,
			AssetType:   system.AssetType,
			ValidTime:   system.ValidTime,
		},
		Links: system.Links,
	}
}
