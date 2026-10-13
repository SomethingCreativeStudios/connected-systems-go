package geojson_formatters

import (
	"context"
	"fmt"
	"io"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

// SamplingFeatureGeoJSONFormatter handles serialization and deserialization of SamplingFeature objects in GeoJSON format
type SamplingFeatureGeoJSONFormatter struct {
	formaters.Formatter[domains.SamplingFeatureGeoJSONFeature, *domains.SamplingFeature]
	links *formaters.LinkResolver
}

// NewSamplingFeatureGeoJSONFormatter constructs a formatter with required repository readers
func NewSamplingFeatureGeoJSONFormatter(links *formaters.LinkResolver) *SamplingFeatureGeoJSONFormatter {
	return &SamplingFeatureGeoJSONFormatter{links: links}
}

func (f *SamplingFeatureGeoJSONFormatter) ContentType() string {
	return mediatype.GeoJSON
}

// --- Serialization ---

func (f *SamplingFeatureGeoJSONFormatter) Serialize(ctx context.Context, sf *domains.SamplingFeature) (domains.SamplingFeatureGeoJSONFeature, error) {
	return formaters.SerializeOne(ctx, sf, f.SerializeAll)
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
				ValidTime:          common_shared.NonEmptyTimeRange(sf.ValidTime),
				SampledFeatureLink: absolutizeLink(f.links, sf.SampledFeatureLink),
			},
			Links: f.links.SamplingFeatureLinks(sf),
		}

		features = append(features, feature)
	}

	return features, nil
}

// --- Deserialization ---

func (f *SamplingFeatureGeoJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.SamplingFeature, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	geoJSON, err := common_shared.DecodeWithFieldErrors[struct {
		Type       string                                   `json:"type"`
		Properties domains.SamplingFeatureGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.GoGeom                    `json:"geometry,omitempty"`
		Links      common_shared.Links                      `json:"links,omitempty"`
	}](body)
	if err != nil {
		return nil, err
	}

	// ogc-rel links (parentSystem, sampleOf, ...) are server-generated, so they
	// are ignored on write; the parent system is only ever set through
	// /systems/{id}/samplingFeatures.
	sf := &domains.SamplingFeature{
		Links: common_shared.StripAssociationLinks(geoJSON.Links),
	}

	// Assign geometry
	if geoJSON.Geometry != nil {
		sf.Geometry = geoJSON.Geometry
	}

	// Required root fields per OGC CS Part 1 samplingFeature GeoJSON schema
	if geoJSON.Properties.UID == "" {
		return nil, fmt.Errorf("properties.uid is required")
	}
	if geoJSON.Properties.Name == "" {
		return nil, fmt.Errorf("properties.name is required")
	}
	if geoJSON.Properties.FeatureType == "" {
		return nil, fmt.Errorf("properties.featureType is required")
	}
	if geoJSON.Properties.SampledFeatureLink == nil || geoJSON.Properties.SampledFeatureLink.Href == "" {
		return nil, fmt.Errorf("properties.sampledFeature@link is required")
	}

	// Extract properties
	sf.UniqueIdentifier = domains.UniqueID(geoJSON.Properties.UID)
	sf.Name = geoJSON.Properties.Name
	sf.Description = geoJSON.Properties.Description
	sf.FeatureType = geoJSON.Properties.FeatureType
	sf.ValidTime = common_shared.NonEmptyTimeRange(geoJSON.Properties.ValidTime)

	// Handle sampled feature link - store both the link and extract ID
	if geoJSON.Properties.SampledFeatureLink != nil && geoJSON.Properties.SampledFeatureLink.Href != "" {
		link := *geoJSON.Properties.SampledFeatureLink
		if link.Rel == "" {
			link.Rel = common_shared.OGCRel("sampledFeature")
		}
		sf.SampledFeatureLink = &link
		sf.SampledFeatureID = link.GetId("items")
	}

	return sf, nil
}

// absolutizeLink returns a copy of the link with an absolute href and GeoJSON type.
func absolutizeLink(links *formaters.LinkResolver, link *common_shared.Link) *common_shared.Link {
	if link == nil {
		return nil
	}
	cp := *link
	cp.Href = links.Href(cp.Href)
	if cp.Type == "" {
		cp.Type = mediatype.GeoJSON
	}
	return &cp
}
