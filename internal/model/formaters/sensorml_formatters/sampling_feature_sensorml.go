package sensorml_formatters

import (
	"context"
	"fmt"
	"io"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

// SamplingFeatureSensorMLFormatter handles serialization and deserialization of SamplingFeature objects in SensorML format
type SamplingFeatureSensorMLFormatter struct {
	formaters.Formatter[domains.SamplingFeatureSensorMLFeature, *domains.SamplingFeature]
	links *formaters.LinkResolver
}

// NewSamplingFeatureSensorMLFormatter constructs a formatter with required repository readers
func NewSamplingFeatureSensorMLFormatter(links *formaters.LinkResolver) *SamplingFeatureSensorMLFormatter {
	return &SamplingFeatureSensorMLFormatter{links: links}
}

func (f *SamplingFeatureSensorMLFormatter) ContentType() string {
	return mediatype.SensorML
}

// --- Serialization ---

func (f *SamplingFeatureSensorMLFormatter) Serialize(ctx context.Context, sf *domains.SamplingFeature) (domains.SamplingFeatureSensorMLFeature, error) {
	return formaters.SerializeOne(ctx, sf, f.SerializeAll)
}

func (f *SamplingFeatureSensorMLFormatter) SerializeAll(ctx context.Context, samplingFeatures []*domains.SamplingFeature) ([]domains.SamplingFeatureSensorMLFeature, error) {
	if len(samplingFeatures) == 0 {
		return []domains.SamplingFeatureSensorMLFeature{}, nil
	}

	var features []domains.SamplingFeatureSensorMLFeature
	for _, sf := range samplingFeatures {
		feature := domains.SamplingFeatureSensorMLFeature{
			ID:                 sf.ID,
			Type:               sf.FeatureType,
			Label:              sf.Name,
			Description:        sf.Description,
			UniqueID:           string(sf.UniqueIdentifier),
			Definition:         sf.FeatureType,
			ValidTime:          common_shared.NonEmptyTimeRange(sf.ValidTime),
			Position:           sf.Geometry,
			SampledFeatureLink: absolutizeLink(f.links, sf.SampledFeatureLink),
			SampleOf:           absolutizeLinks(f.links, sf.SampleOf),
			Links:              sf.Links,
		}
		features = append(features, feature)
	}
	return features, nil
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

// absolutizeLinks returns a copy of the links with absolute hrefs.
func absolutizeLinks(resolver *formaters.LinkResolver, links *common_shared.Links) *common_shared.Links {
	if links == nil {
		return nil
	}
	result := make(common_shared.Links, len(*links))
	for i, l := range *links {
		result[i] = l
		result[i].Href = resolver.Href(result[i].Href)
	}
	return &result
}

// absolutizeLinksValue returns a copy of the links (value type) with absolute hrefs.
func absolutizeLinksValue(resolver *formaters.LinkResolver, links common_shared.Links) common_shared.Links {
	if len(links) == 0 {
		return nil
	}
	result := make(common_shared.Links, len(links))
	for i, l := range links {
		result[i] = l
		result[i].Href = resolver.Href(result[i].Href)
	}
	return result
}

// --- Deserialization ---

func (f *SamplingFeatureSensorMLFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.SamplingFeature, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	sensorML, err := common_shared.DecodeWithFieldErrors[domains.SamplingFeatureSensorMLFeature](body)
	if err != nil {
		return nil, err
	}

	sf := &domains.SamplingFeature{
		Links: common_shared.StripAssociationLinks(sensorML.Links),
	}

	sf.UniqueIdentifier = domains.UniqueID(sensorML.UniqueID)

	sf.Name = sensorML.Label

	sf.Description = sensorML.Description

	if sensorML.Definition != "" {
		sf.FeatureType = sensorML.Definition
	} else if sensorML.Type != "" {
		sf.FeatureType = sensorML.Type
	}

	// Required root fields, mirroring the samplingFeature GeoJSON schema
	// (SensorML sampling features are a non-normative extension)
	if sf.UniqueIdentifier == "" {
		return nil, fmt.Errorf("uniqueId is required")
	}
	if sf.Name == "" {
		return nil, fmt.Errorf("label is required")
	}
	if sf.FeatureType == "" {
		return nil, fmt.Errorf("type or definition is required")
	}
	if sensorML.SampledFeatureLink == nil || sensorML.SampledFeatureLink.Href == "" {
		return nil, fmt.Errorf("sampledFeature is required")
	}

	sf.ValidTime = common_shared.NonEmptyTimeRange(sensorML.ValidTime)
	sf.Geometry = sensorML.Position
	sf.SampledFeatureLink = sensorML.SampledFeatureLink
	sf.SampleOf = sensorML.SampleOf

	// Handle sampled feature link
	if sensorML.SampledFeatureLink != nil && sensorML.SampledFeatureLink.Href != "" {
		sf.SampledFeatureID = sensorML.SampledFeatureLink.GetId("samplingFeatures")
	}

	return sf, nil
}
