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

const SensorMLContentType = "application/sml+json"

// SystemSensorMLFormatter handles serialization and deserialization of System objects in SensorML format
type SystemSensorMLFormatter struct {
	formaters.Formatter[domains.SystemSensorMLFeature, *domains.System]
	repos *repository.Repositories
}

// NewSystemSensorMLFormatter constructs a formatter with required repository readers
func NewSystemSensorMLFormatter(repos *repository.Repositories) *SystemSensorMLFormatter {
	return &SystemSensorMLFormatter{repos: repos}
}

func (f *SystemSensorMLFormatter) ContentType() string {
	return SensorMLContentType
}

// --- Serialization ---

func (f *SystemSensorMLFormatter) Serialize(ctx context.Context, system *domains.System) (domains.SystemSensorMLFeature, error) {
	features, err := f.SerializeAll(ctx, []*domains.System{system})
	if err != nil {
		return domains.SystemSensorMLFeature{}, err
	}
	return features[0], nil
}

func (f *SystemSensorMLFormatter) SerializeAll(ctx context.Context, systems []*domains.System) ([]domains.SystemSensorMLFeature, error) {
	if len(systems) == 0 {
		return []domains.SystemSensorMLFeature{}, nil
	}

	var features []domains.SystemSensorMLFeature
	for _, system := range systems {

		feature := domains.SystemSensorMLFeature{
			ID:                   system.ID,
			Type:                 system.SystemType,
			Label:                system.Name,
			Description:          system.Description,
			UniqueID:             string(system.UniqueIdentifier),
			Lang:                 system.Lang,
			Keywords:             system.Keywords,
			Identifiers:          system.Identifiers,
			Classifiers:          system.Classifiers,
			SecurityConstraints:  system.SecurityConstraints,
			LegalConstraints:     system.LegalConstraints,
			Contacts:             system.Contacts,
			Documentation:        system.Documentation,
			History:              system.History,
			TypeOf:               system.TypeOf,
			Configuration:        system.Configuration,
			FeaturesOfInterest:   system.FeaturesOfInterest,
			Inputs:               system.Inputs,
			Outputs:              system.Outputs,
			Parameters:           system.Parameters,
			Modes:                system.Modes,
			Position:             system.Position,
			AttachedTo:           system.AttachedTo,
			LocalReferenceFrames: system.LocalReferenceFrames,
			LocalTimeFrames:      system.LocalTimeFrames,
			Links:                system.Links,
		}
		features = append(features, feature)
	}
	return features, nil
}

// --- Deserialization ---

func (f *SystemSensorMLFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.System, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var sml domains.SystemSensorMLFeature
	_ = json.Unmarshal(body, &sml)

	system := &domains.System{}

	if v, ok := raw["label"].(string); ok && v != "" {
		system.Name = v
	} else if v, ok := raw["name"].(string); ok && v != "" {
		system.Name = v
	}

	if v, ok := raw["description"].(string); ok {
		system.Description = v
	}

	if v, ok := raw["uniqueId"].(string); ok && v != "" {
		system.UniqueIdentifier = domains.UniqueID(v)
	} else if v, ok := raw["uid"].(string); ok && v != "" {
		system.UniqueIdentifier = domains.UniqueID(v)
	}

	if sml.Definition != "" {
		system.SystemType = sml.Definition
	} else if v, ok := raw["definition"].(string); ok {
		system.SystemType = v
	}

	if v, ok := raw["type"].(string); ok && system.SystemType == "" {
		system.SystemType = v
	}

	if v, ok := raw["assetType"].(string); ok {
		at := v
		system.AssetType = &at
	}

	if vt, ok := raw["validTime"]; ok {
		tr := common_shared.ParseTimeRange(vt)
		system.ValidTime = &tr
	}

	var geomObj interface{}
	if p, ok := raw["position"]; ok {
		geomObj = p
	} else if g, ok := raw["geometry"]; ok {
		geomObj = g
	}
	if geomObj != nil {
		if gb, err := json.Marshal(geomObj); err == nil {
			var gg common_shared.GoGeom
			if err := json.Unmarshal(gb, &gg); err == nil {
				system.Geometry = &gg
			}
		}
	}

	system.TypeOf = sml.TypeOf
	system.Configuration = sml.Configuration
	if sml.FeaturesOfInterest != nil {
		system.FeaturesOfInterest = sml.FeaturesOfInterest
	}
	system.Inputs = sml.Inputs
	system.Outputs = sml.Outputs
	system.Parameters = sml.Parameters
	system.Modes = sml.Modes
	system.AttachedTo = sml.AttachedTo
	system.LocalReferenceFrames = sml.LocalReferenceFrames
	system.LocalTimeFrames = sml.LocalTimeFrames
	system.Position = sml.Position
	system.Contacts = sml.Contacts
	system.Documentation = sml.Documentation
	system.History = sml.History

	if sml.FeaturesOfInterest != nil && len(sml.FeaturesOfInterest) > 0 {
		system.Links = append(system.Links, sml.FeaturesOfInterest...)
	}

	return system, nil
}
