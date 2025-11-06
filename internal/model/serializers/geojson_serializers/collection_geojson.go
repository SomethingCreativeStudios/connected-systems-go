package geojson_serializers

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

type CollectionGeoJSONSerializer struct {
	serializers.Serializer[domains.CollectionGeoJSONFeature, *domains.Collection]
	repos *repository.Repositories
}

// NewCollectionGeoJSONSerializer constructs a serializer with required repository readers.
func NewCollectionGeoJSONSerializer(repos *repository.Repositories) *CollectionGeoJSONSerializer {
	return &CollectionGeoJSONSerializer{repos: repos}
}

func (s *CollectionGeoJSONSerializer) Serialize(ctx context.Context, collection *domains.Collection) (domains.CollectionGeoJSONFeature, error) {
	features, err := s.SerializeAll(ctx, []*domains.Collection{collection})
	if err != nil {
		return domains.CollectionGeoJSONFeature{}, err
	}
	return features[0], nil
}

func (s *CollectionGeoJSONSerializer) SerializeAll(ctx context.Context, collections []*domains.Collection) ([]domains.CollectionGeoJSONFeature, error) {
	if len(collections) == 0 {
		return []domains.CollectionGeoJSONFeature{}, nil
	}
	var features []domains.CollectionGeoJSONFeature
	for _, collection := range collections {
		feature := domains.CollectionGeoJSONFeature{
			Collection: domains.Collection{
				ID:          collection.ID,
				Title:       collection.Title,
				Description: collection.Description,
				Links:       collection.Links,
				Extent:      collection.Extent,
			},
		}
		features = append(features, feature)
	}
	return features, nil
}
