package sensorml_formatters

import (
	"context"
	"encoding/json"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// SamplingFeatureSensorMLFormatter handles serialization and deserialization of SamplingFeature objects in SensorML format
type SamplingFeatureSensorMLFormatter struct {
	formaters.Formatter[domains.SamplingFeatureSensorMLFeature, *domains.SamplingFeature]
	repos *repository.Repositories
}

// NewSamplingFeatureSensorMLFormatter constructs a formatter with required repository readers
func NewSamplingFeatureSensorMLFormatter(repos *repository.Repositories) *SamplingFeatureSensorMLFormatter {
	return &SamplingFeatureSensorMLFormatter{repos: repos}
}

func (f *SamplingFeatureSensorMLFormatter) ContentType() string {
	return SensorMLContentType
}

// --- Serialization ---

func (f *SamplingFeatureSensorMLFormatter) Serialize(ctx context.Context, sf *domains.SamplingFeature) (domains.SamplingFeatureSensorMLFeature, error) {
	features, err := f.SerializeAll(ctx, []*domains.SamplingFeature{sf})
	if err != nil {
		return domains.SamplingFeatureSensorMLFeature{}, err
	}
	return features[0], nil
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
			ValidTime:          sf.ValidTime,
			SampledFeatureLink: absolutizeLink(sf.SampledFeatureLink),
			SampleOf:           absolutizeLinks(sf.SampleOf),
			Links:              sf.Links,
		}
		features = append(features, feature)
	}
	return features, nil
}

// absolutizeLink returns a copy of the link with an absolute href.
func absolutizeLink(link *common_shared.Link) *common_shared.Link {
	if link == nil {
		return nil
	}
	cp := *link
	cp.Href = formaters.ToFunctionalAssociationHref(cp.Href)
	return &cp
}

// absolutizeLinks returns a copy of the links with absolute hrefs.
func absolutizeLinks(links *common_shared.Links) *common_shared.Links {
	if links == nil {
		return nil
	}
	result := make(common_shared.Links, len(*links))
	for i, l := range *links {
		result[i] = l
		result[i].Href = formaters.ToFunctionalAssociationHref(result[i].Href)
	}
	return &result
}

// absolutizeLinksValue returns a copy of the links (value type) with absolute hrefs.
func absolutizeLinksValue(links common_shared.Links) common_shared.Links {
	if len(links) == 0 {
		return nil
	}
	result := make(common_shared.Links, len(links))
	for i, l := range links {
		result[i] = l
		result[i].Href = formaters.ToFunctionalAssociationHref(result[i].Href)
	}
	return result
}

// --- Deserialization ---

func (f *SamplingFeatureSensorMLFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.SamplingFeature, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
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

	if sensorML.Label != "" {
		sf.Name = sensorML.Label
	} else if v, ok := raw["name"].(string); ok {
		sf.Name = v
	}

	sf.Description = sensorML.Description

	if sensorML.Definition != "" {
		sf.FeatureType = sensorML.Definition
	} else if sensorML.Type != "" {
		sf.FeatureType = sensorML.Type
	}

	sf.ValidTime = sensorML.ValidTime
	sf.SampledFeatureLink = sensorML.SampledFeatureLink
	sf.SampleOf = sensorML.SampleOf

	// Handle sampled feature link
	if sensorML.SampledFeatureLink != nil && sensorML.SampledFeatureLink.Href != "" {
		sf.SampledFeatureID = sensorML.SampledFeatureLink.GetId("samplingFeatures")
	}

	// Handle geometry from position field if present
	if geomObj, ok := raw["position"]; ok {
		if gb, err := json.Marshal(geomObj); err == nil {
			var gg common_shared.GoGeom
			if err := json.Unmarshal(gb, &gg); err == nil {
				sf.Geometry = &gg
			}
		}
	}

	return sf, nil
}
