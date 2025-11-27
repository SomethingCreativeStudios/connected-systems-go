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
			SampledFeatureLink: sf.SampledFeatureLink,
			SampleOf:           sf.SampleOf,
			Links:              sf.Links,
		}
		features = append(features, feature)
	}
	return features, nil
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

	var sensorML domains.SamplingFeatureSensorMLFeature
	if err := json.Unmarshal(body, &sensorML); err != nil {
		return nil, err
	}

	sf := &domains.SamplingFeature{
		Links: sensorML.Links,
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

	// Process links for parent system and sampleOf relationships
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
