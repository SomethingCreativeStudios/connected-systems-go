package sensorml_serializers

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// SystemSensorMLSerializer serializes domain System objects into SensorML format.
type SystemSensorMLSerializer struct {
	serializers.Serializer[domains.SystemSensorMLFeature, *domains.System]
	repos *repository.Repositories
}

// NewSystemSensorMLSerializer constructs a serializer with required repository readers.
func NewSystemSensorMLSerializer(repos *repository.Repositories) *SystemSensorMLSerializer {
	return &SystemSensorMLSerializer{repos: repos}
}

func (s *SystemSensorMLSerializer) Serialize(ctx context.Context, system *domains.System) (domains.SystemSensorMLFeature, error) {
	features, err := s.SerializeAll(ctx, []*domains.System{system})
	if err != nil {
		return domains.SystemSensorMLFeature{}, err
	}
	return features[0], nil
}

func (s *SystemSensorMLSerializer) SerializeAll(ctx context.Context, systems []*domains.System) ([]domains.SystemSensorMLFeature, error) {
	if len(systems) == 0 {
		return []domains.SystemSensorMLFeature{}, nil
	}

	var features []domains.SystemSensorMLFeature
	for _, system := range systems {
		// Get Definition from SMLProperties if available

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
