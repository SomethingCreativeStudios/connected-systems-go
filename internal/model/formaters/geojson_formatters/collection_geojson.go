package geojson_formatters

import (
	"context"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// FeatureCollectionGeoJSONFormatter handles serialization and deserialization of Feature objects in GeoJSON format
type FeatureCollectionGeoJSONFormatter struct {
	formaters.Formatter[domains.CollectionGeoJSONFeature, *domains.Collection]
	repos *repository.Repositories
}

// NewFeatureCollectionGeoJSONFormatter constructs a formatter with required repository readers
func NewFeatureCollectionGeoJSONFormatter(repos *repository.Repositories) *FeatureCollectionGeoJSONFormatter {
	return &FeatureCollectionGeoJSONFormatter{repos: repos}
}

func (f *FeatureCollectionGeoJSONFormatter) ContentType() string {
	return GeoJSONContentType
}

// --- Serialization ---

func (f *FeatureCollectionGeoJSONFormatter) Serialize(ctx context.Context, collection *domains.Collection) (domains.CollectionGeoJSONFeature, error) {
	features, err := f.SerializeAll(ctx, []*domains.Collection{collection})

	if err != nil {
		return domains.CollectionGeoJSONFeature{}, err
	}

	return features[0], nil
}

func (f *FeatureCollectionGeoJSONFormatter) SerializeAll(ctx context.Context, collections []*domains.Collection) ([]domains.CollectionGeoJSONFeature, error) {
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

// --- Deserialization ---

func (f *FeatureCollectionGeoJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Collection, error) {
	// Deserialization not implemented yet
	return nil, nil
}
