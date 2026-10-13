package geojson_formatters

import (
	"context"
	"io"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

// FeatureCollectionGeoJSONFormatter handles serialization and deserialization of Feature objects in GeoJSON format
type FeatureCollectionGeoJSONFormatter struct {
	formaters.Formatter[domains.CollectionGeoJSONFeature, *domains.Collection]
	links *formaters.LinkResolver
}

// NewFeatureCollectionGeoJSONFormatter constructs a formatter with required repository readers
func NewFeatureCollectionGeoJSONFormatter(links *formaters.LinkResolver) *FeatureCollectionGeoJSONFormatter {
	return &FeatureCollectionGeoJSONFormatter{links: links}
}

func (f *FeatureCollectionGeoJSONFormatter) ContentType() string {
	return mediatype.GeoJSON
}

// --- Serialization ---

func (f *FeatureCollectionGeoJSONFormatter) Serialize(ctx context.Context, collection *domains.Collection) (domains.CollectionGeoJSONFeature, error) {
	return formaters.SerializeOne(ctx, collection, f.SerializeAll)
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
