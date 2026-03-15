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

var associations = []string{
	common_shared.OGCRel("parentSystem"),
	common_shared.OGCRel("sampleOf"),
}

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
			Links: serializeLinks(sf),
		}

		features = append(features, feature)
	}

	return features, nil
}

func serializeLinks(sf *domains.SamplingFeature) common_shared.Links {
	links := common_shared.Links{}

	if sf.Links != nil {
		links = sf.Links.FilterByRels(associations, false)
	}

	if sf.ParentSystemID != nil {
		link := common_shared.Link{
			Rel:  common_shared.OGCRel("parentSystem"),
			Href: "/systems/" + *sf.ParentSystemID,
		}
		if sf.ParentSystemUID != nil {
			link.UID = sf.ParentSystemUID
		}
		links = append(links, link)
	}

	if sf.SampleOf != nil {
		for _, link := range *sf.SampleOf {
			links = append(links, link)
		}
	}

	return links
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

	associationLinks := geoJSON.Links.FilterByRels(associations, true)

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
		sf.SampledFeatureLink = geoJSON.Properties.SampledFeatureLink
		sf.SampledFeatureID = geoJSON.Properties.SampledFeatureLink.GetId("samplingFeatures")
	}

	f.handleAssociationLinks(associationLinks, sf)

	return sf, nil
}

func (f *SamplingFeatureGeoJSONFormatter) handleAssociationLinks(sfLinks common_shared.Links, sf *domains.SamplingFeature) {
	if len(sfLinks) == 0 {
		return
	}

	sampleIDs := []string{}
	sampleUIDs := []string{}
	sampleOfLinks := common_shared.Links{}

	for _, link := range sfLinks {
		if common_shared.RelEquals(link.Rel, common_shared.OGCRel("parentSystem")) {
			sf.ParentSystemID = link.GetId("systems")
			if link.UID != nil {
				sf.ParentSystemUID = link.UID
			}
			continue
		}

		if common_shared.RelEquals(link.Rel, common_shared.OGCRel("sampleOf")) {
			sampleOfLinks = append(sampleOfLinks, link)
			if id := link.GetId("samplingFeatures"); id != nil {
				sampleIDs = append(sampleIDs, *id)
			}
			if link.UID != nil {
				sampleUIDs = append(sampleUIDs, *link.UID)
			}
		}
	}

	if len(sampleOfLinks) > 0 {
		sf.SampleOf = &sampleOfLinks
	}
	if len(sampleIDs) > 0 {
		sf.SampleOfIDs = &sampleIDs
	}
	if len(sampleUIDs) > 0 {
		sf.SampleOfUIDs = &sampleUIDs
	}
}
