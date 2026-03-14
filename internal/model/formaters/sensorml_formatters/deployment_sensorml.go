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

// DeploymentSensorMLFormatter handles serialization and deserialization of Deployment objects in SensorML format
type DeploymentSensorMLFormatter struct {
	formaters.Formatter[domains.DeploymentSensorMLFeature, *domains.Deployment]
	repos *repository.Repositories
}

// NewDeploymentSensorMLFormatter constructs a formatter with required repository readers
func NewDeploymentSensorMLFormatter(repos *repository.Repositories) *DeploymentSensorMLFormatter {
	return &DeploymentSensorMLFormatter{repos: repos}
}

func (f *DeploymentSensorMLFormatter) ContentType() string {
	return SensorMLContentType
}

// --- Serialization ---

func (f *DeploymentSensorMLFormatter) Serialize(ctx context.Context, deployment *domains.Deployment) (domains.DeploymentSensorMLFeature, error) {
	features, err := f.SerializeAll(ctx, []*domains.Deployment{deployment})
	if err != nil {
		return domains.DeploymentSensorMLFeature{}, err
	}
	return features[0], nil
}

func (f *DeploymentSensorMLFormatter) SerializeAll(ctx context.Context, deployments []*domains.Deployment) ([]domains.DeploymentSensorMLFeature, error) {
	if len(deployments) == 0 {
		return []domains.DeploymentSensorMLFeature{}, nil
	}

	var features []domains.DeploymentSensorMLFeature
	for _, deployment := range deployments {
		feature := domains.DeploymentSensorMLFeature{
			ID:              deployment.ID,
			Type:            "Deployment",
			Label:           deployment.Name,
			Description:     deployment.Description,
			UniqueID:        string(deployment.UniqueIdentifier),
			Definition:      deployment.DeploymentType,
			ValidTime:       deployment.ValidTime,
			Platform:        deployment.Platform,
			DeployedSystems: deployment.DeployedSystems,
			Links:           deployment.Links,

			Lang:                deployment.Lang,
			Keywords:            deployment.Keywords,
			Identifiers:         deployment.Identifiers,
			Classifiers:         deployment.Classifiers,
			Contacts:            deployment.Contacts,
			Characteristics:     deployment.Characteristics,
			Capabilities:        deployment.Capabilities,
			SecurityConstraints: deployment.SecurityConstraints,
			LegalConstraints:    deployment.LegalConstraints,
			Documentation:       deployment.Documentation,
			History:             deployment.History,
		}
		features = append(features, feature)
	}
	return features, nil
}

// --- Deserialization ---

func (f *DeploymentSensorMLFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Deployment, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var sensorML domains.DeploymentSensorMLFeature
	if err := json.Unmarshal(body, &sensorML); err != nil {
		return nil, err
	}

	deployment := &domains.Deployment{
		Links: sensorML.Links,
	}

	if sensorML.Platform != nil {
		deployment.Platform = sensorML.Platform
		deployment.PlatformID = sensorML.Platform.System.GetId("systems")
	}

	if len(sensorML.DeployedSystems) > 0 {
		deployment.DeployedSystems = sensorML.DeployedSystems
		var systemIDs common_shared.StringArray
		for _, ds := range sensorML.DeployedSystems {
			systemIDs = append(systemIDs, *ds.System.GetId("systems"))
		}
		deployment.SystemIds = &systemIDs
	}

	deployment.UniqueIdentifier = domains.UniqueID(sensorML.UniqueID)
	deployment.Name = sensorML.Label
	deployment.Description = sensorML.Description

	if sensorML.Definition != "" {
		deployment.DeploymentType = sensorML.Definition
	} else if sensorML.Type != "" {
		deployment.DeploymentType = sensorML.Type
	}

	deployment.ValidTime = sensorML.ValidTime
	deployment.Lang = sensorML.Lang
	deployment.Keywords = sensorML.Keywords
	deployment.Identifiers = sensorML.Identifiers
	deployment.Classifiers = sensorML.Classifiers
	deployment.SecurityConstraints = sensorML.SecurityConstraints
	deployment.LegalConstraints = sensorML.LegalConstraints
	deployment.Characteristics = sensorML.Characteristics
	deployment.Capabilities = sensorML.Capabilities
	deployment.Contacts = sensorML.Contacts
	deployment.Documentation = sensorML.Documentation
	deployment.History = sensorML.History

	// Handle geometry from position field if present
	if geomObj, ok := raw["position"]; ok {
		if gb, err := json.Marshal(geomObj); err == nil {
			var gg common_shared.GoGeom
			if err := json.Unmarshal(gb, &gg); err == nil {
				deployment.Geometry = &gg
			}
		}
	}

	return deployment, nil
}
