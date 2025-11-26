package geojson_formatters

import (
	"context"
	"encoding/json"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// SamplingFeatureGeoJSONFormatter handles serialization and deserialization of SamplingFeature objects in GeoJSON format
type SamplingFeatureGeoJSONFormatter struct {
	serializers.Formatter[domains.SamplingFeatureGeoJSONFeature, *domains.SamplingFeature]
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
		var sampledFeatureLink common_shared.Link
		if sf.SampledFeatureLink != nil {
			sampledFeatureLink = *sf.SampledFeatureLink
		}

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
				SampledFeatureLink: sampledFeatureLink,
			},
			Links: sf.Links,
		}
		features = append(features, feature)
	}

	return features, nil
}

// --- Deserialization ---

func (f *SamplingFeatureGeoJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.SamplingFeature, error) {
	var geoJSON struct {
		Type       string                                   `json:"type"`
		ID         string                                   `json:"id,omitempty"`
		Properties domains.SamplingFeatureGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.GoGeom                    `json:"geometry,omitempty"`
		Links      common_shared.Links                      `json:"links,omitempty"`
	}

	if err := json.NewDecoder(reader).Decode(&geoJSON); err != nil {
		return nil, err
	}

	sf := &domains.SamplingFeature{
		Links: geoJSON.Links,
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

	// Handle sampled feature link
	if geoJSON.Properties.SampledFeatureLink.Href != "" {
		sf.SampledFeatureID = geoJSON.Properties.SampledFeatureLink.GetId("samplingFeatures")
	}

	// Handle links (parentSystem, sampleOf, etc.) - process link relations
	sampleIds := []string{}
	sampleUids := []string{}

	for _, link := range sf.Links {
		if link.Rel == "parentSystem" {
			sf.ParentSystemID = link.GetId("systems")
			if link.UID != nil {
				sf.ParentSystemUID = link.UID
			}
		}

		if link.Rel == "sampleOf" {
			if id := link.GetId("samplingFeatures"); id != nil {
				sampleIds = append(sampleIds, *id)
			}
			if link.UID != nil {
				sampleUids = append(sampleUids, *link.UID)
			}
		}
	}

	if len(sampleIds) > 0 {
		sf.SampleOfIDs = &sampleIds
	}
	if len(sampleUids) > 0 {
		sf.SampleOfUIDs = &sampleUids
	}

	return sf, nil
}
