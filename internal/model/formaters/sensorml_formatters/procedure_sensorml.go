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

// ProcedureSensorMLFormatter handles serialization and deserialization of Procedure objects in SensorML format
type ProcedureSensorMLFormatter struct {
	formaters.Formatter[domains.ProcedureSensorMLFeature, *domains.Procedure]
	repos *repository.Repositories
}

// NewProcedureSensorMLFormatter constructs a formatter with required repository readers
func NewProcedureSensorMLFormatter(repos *repository.Repositories) *ProcedureSensorMLFormatter {
	return &ProcedureSensorMLFormatter{repos: repos}
}

func (f *ProcedureSensorMLFormatter) ContentType() string {
	return SensorMLContentType
}

// --- Serialization ---

func (f *ProcedureSensorMLFormatter) Serialize(ctx context.Context, procedure *domains.Procedure) (domains.ProcedureSensorMLFeature, error) {
	features, err := f.SerializeAll(ctx, []*domains.Procedure{procedure})
	if err != nil {
		return domains.ProcedureSensorMLFeature{}, err
	}
	return features[0], nil
}

func (f *ProcedureSensorMLFormatter) SerializeAll(ctx context.Context, procedures []*domains.Procedure) ([]domains.ProcedureSensorMLFeature, error) {
	if len(procedures) == 0 {
		return []domains.ProcedureSensorMLFeature{}, nil
	}

	var features []domains.ProcedureSensorMLFeature
	for _, procedure := range procedures {
		feature := domains.ProcedureSensorMLFeature{
			ID:                   procedure.ID,
			Type:                 procedure.ProcessType,
			Label:                procedure.Name,
			Description:          procedure.Description,
			UniqueID:             string(procedure.UniqueIdentifier),
			Definition:           procedure.ProcedureType,
			Lang:                 procedure.Lang,
			Keywords:             procedure.Keywords,
			Identifiers:          procedure.Identifiers,
			Classifiers:          procedure.Classifiers,
			SecurityConstraints:  procedure.SecurityConstraints,
			LegalConstraints:     procedure.LegalConstraints,
			Characteristics:      procedure.Characteristics,
			Capabilities:         procedure.Capabilities,
			Contacts:             procedure.Contacts,
			Documentation:        procedure.Documentation,
			History:              procedure.History,
			TypeOf:               procedure.TypeOf,
			Configuration:        procedure.Configuration,
			FeaturesOfInterest:   procedure.FeaturesOfInterest,
			Inputs:               procedure.Inputs,
			Outputs:              procedure.Outputs,
			Parameters:           procedure.Parameters,
			Modes:                procedure.Modes,
			Method:               procedure.Method,
			Components:           procedure.Components,
			Connections:          procedure.Connections,
			AttachedTo:           procedure.AttachedTo,
			LocalReferenceFrames: procedure.LocalReferenceFrames,
			LocalTimeFrames:      procedure.LocalTimeFrames,
			ValidTime:            procedure.ValidTime,
			Links:                procedure.Links,
		}
		features = append(features, feature)
	}
	return features, nil
}

// --- Deserialization ---

func (f *ProcedureSensorMLFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Procedure, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var sensorML domains.ProcedureSensorMLFeature
	if err := json.Unmarshal(body, &sensorML); err != nil {
		return nil, err
	}

	procedure := &domains.Procedure{
		Links: sensorML.Links,
	}

	procedure.UniqueIdentifier = domains.UniqueID(sensorML.UniqueID)

	if sensorML.Label != "" {
		procedure.Name = sensorML.Label
	} else if v, ok := raw["name"].(string); ok {
		procedure.Name = v
	}

	procedure.Description = sensorML.Description

	if sensorML.Definition != "" {
		procedure.ProcedureType = sensorML.Definition
	}
	if sensorML.Type != "" {
		procedure.ProcessType = sensorML.Type
	}

	procedure.Lang = sensorML.Lang
	procedure.Keywords = sensorML.Keywords
	procedure.Identifiers = sensorML.Identifiers
	procedure.Classifiers = sensorML.Classifiers
	procedure.SecurityConstraints = sensorML.SecurityConstraints
	procedure.LegalConstraints = sensorML.LegalConstraints
	procedure.Characteristics = sensorML.Characteristics
	procedure.Capabilities = sensorML.Capabilities
	procedure.Contacts = sensorML.Contacts
	procedure.Documentation = sensorML.Documentation
	procedure.History = sensorML.History
	procedure.TypeOf = sensorML.TypeOf
	procedure.Configuration = sensorML.Configuration
	procedure.FeaturesOfInterest = sensorML.FeaturesOfInterest
	procedure.Inputs = sensorML.Inputs
	procedure.Outputs = sensorML.Outputs
	procedure.Parameters = sensorML.Parameters
	procedure.Modes = sensorML.Modes
	procedure.Method = sensorML.Method
	procedure.Components = sensorML.Components
	procedure.Connections = sensorML.Connections
	procedure.AttachedTo = sensorML.AttachedTo
	procedure.LocalReferenceFrames = sensorML.LocalReferenceFrames
	procedure.LocalTimeFrames = sensorML.LocalTimeFrames
	procedure.ValidTime = sensorML.ValidTime

	// Handle validTime from raw if not in structured form
	if procedure.ValidTime == nil {
		if vt, ok := raw["validTime"]; ok {
			tr := common_shared.ParseTimeRange(vt)
			procedure.ValidTime = &tr
		}
	}

	return procedure, nil
}
