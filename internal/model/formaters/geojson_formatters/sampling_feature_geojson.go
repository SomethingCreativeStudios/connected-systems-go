package geojson_formatters

import (
	"context"
	"encoding/json"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// SamplingFeatureGeoJSONFormatter handles serialization and deserialization of SamplingFeature objects in GeoJSON format
type SamplingFeatureGeoJSONFormatter struct {
	formaters.Formatter[domains.SamplingFeatureGeoJSONFeature, *domains.SamplingFeature]
	repos *repository.Repositories
}

// NewSamplingFeatureGeoJSONFormatter constructs a formatter with required repository readers
func NewSamplingFeatureGeoJSONFormatter(repos *repository.Repositories) *SamplingFeatureGeoJSONFormatter {
	return &SamplingFeatureGeoJSONFormatter{repos: repos}
}

func (f *SamplingFeatureGeoJSONFormatter) ContentType() string {
	return GeoJSONContentType
}

// --- Serialization ---

func (f *SamplingFeatureGeoJSONFormatter) Serialize(ctx context.Context, sf *domains.SamplingFeature) (domains.SamplingFeatureGeoJSONFeature, error) {
	features, err := f.SerializeAll(ctx, []*domains.SamplingFeature{sf})
	if err != nil {
		return domains.SamplingFeatureGeoJSONFeature{}, err
	}
	return features[0], nil
}

func (f *SamplingFeatureGeoJSONFormatter) SerializeAll(ctx context.Context, samplingFeatures []*domains.SamplingFeature) ([]domains.SamplingFeatureGeoJSONFeature, error) {
	if len(samplingFeatures) == 0 {
		return []domains.SamplingFeatureGeoJSONFeature{}, nil
	}

	var features []domains.SamplingFeatureGeoJSONFeature
	for _, sf := range samplingFeatures {
		feature := domains.SamplingFeatureGeoJSONFeature{
			Type:     "Feature",
			ID:       sf.ID,
			Geometry: sf.Geometry,
			Properties: domains.SamplingFeatureGeoJSONProperties{
				UID:                sf.UniqueIdentifier,
				Name:               sf.Name,
				Description:        sf.Description,
				FeatureType:        sf.FeatureType,
				ValidTime:          sf.ValidTime,
				SampledFeatureLink: sf.SampledFeatureLink,
			},
			Links: formaters.AppendSamplingFeatureGeoJSONAssociationLinks(sf),
		}

		features = append(features, feature)
	}

	return features, nil
}

// --- Deserialization ---

func (f *SamplingFeatureGeoJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.SamplingFeature, error) {
	var geoJSON struct {
		Type       string                                   `json:"type"`
		Properties domains.SamplingFeatureGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.GoGeom                    `json:"geometry,omitempty"`
		Links      common_shared.Links                      `json:"links,omitempty"`
	}

	if err := json.NewDecoder(reader).Decode(&geoJSON); err != nil {
		return nil, err
	}

	associationLinks := formaters.SamplingFeatureGeoJSONAssociationLinks(geoJSON.Links)

	sf := &domains.SamplingFeature{
		Links: common_shared.StripAssociationLinks(geoJSON.Links),
	}

	// Assign geometry
	if geoJSON.Geometry != nil {
		sf.Geometry = geoJSON.Geometry
	}

	// Extract properties
	sf.UniqueIdentifier = domains.UniqueID(geoJSON.Properties.UID)
	sf.Name = geoJSON.Properties.Name
	sf.Description = geoJSON.Properties.Description
	sf.FeatureType = geoJSON.Properties.FeatureType
	sf.ValidTime = geoJSON.Properties.ValidTime

	// Handle sampled feature link - store both the link and extract ID
	if geoJSON.Properties.SampledFeatureLink != nil && geoJSON.Properties.SampledFeatureLink.Href != "" {
		link := *geoJSON.Properties.SampledFeatureLink
		if link.Rel == "" {
			link.Rel = common_shared.OGCRel("sampledFeature")
		}
		sf.SampledFeatureLink = &link
		sf.SampledFeatureID = link.GetId("items")
	}

	formaters.ApplySamplingFeatureGeoJSONAssociationLinks(sf, associationLinks)

	return sf, nil
}
