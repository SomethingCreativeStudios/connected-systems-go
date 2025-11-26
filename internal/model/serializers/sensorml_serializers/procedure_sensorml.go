package sensorml_serializers

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// ProcedureSensorMLSerializer serializes domain Procedure objects into SensorML format.
type ProcedureSensorMLSerializer struct {
	serializers.Serializer[domains.ProcedureSensorMLFeature, *domains.Procedure]
	repos *repository.Repositories
}

// NewProcedureSensorMLSerializer constructs a serializer with required repository readers.
func NewProcedureSensorMLSerializer(repos *repository.Repositories) *ProcedureSensorMLSerializer {
	return &ProcedureSensorMLSerializer{repos: repos}
}

func (s *ProcedureSensorMLSerializer) Serialize(ctx context.Context, procedure *domains.Procedure) (domains.ProcedureSensorMLFeature, error) {
	features, err := s.SerializeAll(ctx, []*domains.Procedure{procedure})
	if err != nil {
		return domains.ProcedureSensorMLFeature{}, err
	}
	return features[0], nil
}

func (s *ProcedureSensorMLSerializer) SerializeAll(ctx context.Context, procedures []*domains.Procedure) ([]domains.ProcedureSensorMLFeature, error) {
	if len(procedures) == 0 {
		return []domains.ProcedureSensorMLFeature{}, nil
	}

	var features []domains.ProcedureSensorMLFeature
	for _, procedure := range procedures {
		feature := domains.ProcedureSensorMLFeature{
			ID:                  procedure.ID,
			Type:                procedure.FeatureType,
			Label:               procedure.Name,
			Description:         procedure.Description,
			UniqueID:            string(procedure.UniqueIdentifier),
			Definition:          procedure.FeatureType, // FeatureType serves as definition URI
			Lang:                procedure.Lang,
			Keywords:            procedure.Keywords,
			Identifiers:         procedure.Identifiers,
			Classifiers:         procedure.Classifiers,
			SecurityConstraints: procedure.SecurityConstraints,
			LegalConstraints:    procedure.LegalConstraints,
			Characteristics:     procedure.Characteristics,
			Capabilities:        procedure.Capabilities,
			Contacts:            procedure.Contacts,
			Documentation:       procedure.Documentation,
			History:             procedure.History,
			TypeOf:              procedure.TypeOf,
			Configuration:       procedure.Configuration,
			FeaturesOfInterest:  procedure.FeaturesOfInterest,
			Inputs:              procedure.Inputs,
			Outputs:             procedure.Outputs,
			Parameters:          procedure.Parameters,
			Modes:               procedure.Modes,
			ValidTime:           procedure.ValidTime,
			Links:               procedure.Links,
		}
		features = append(features, feature)
	}
	return features, nil
}
