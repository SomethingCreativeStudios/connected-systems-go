package sensorml_serializers

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// DeploymentSensorMLSerializer serializes domain Deployment objects into SensorML format.
type DeploymentSensorMLSerializer struct {
	serializers.Serializer[domains.DeploymentSensorMLFeature, *domains.Deployment]
	repos *repository.Repositories
}

// NewDeploymentSensorMLSerializer constructs a serializer with required repository readers.
func NewDeploymentSensorMLSerializer(repos *repository.Repositories) *DeploymentSensorMLSerializer {
	return &DeploymentSensorMLSerializer{repos: repos}
}

func (s *DeploymentSensorMLSerializer) Serialize(ctx context.Context, deployment *domains.Deployment) (domains.DeploymentSensorMLFeature, error) {
	features, err := s.SerializeAll(ctx, []*domains.Deployment{deployment})
	if err != nil {
		return domains.DeploymentSensorMLFeature{}, err
	}
	return features[0], nil
}

func (s *DeploymentSensorMLSerializer) SerializeAll(ctx context.Context, deployments []*domains.Deployment) ([]domains.DeploymentSensorMLFeature, error) {
	if len(deployments) == 0 {
		return []domains.DeploymentSensorMLFeature{}, nil
	}

	var features []domains.DeploymentSensorMLFeature
	for _, deployment := range deployments {
		feature := domains.DeploymentSensorMLFeature{
			ID:              deployment.ID,
			Type:            deployment.DeploymentType,
			Label:           deployment.Name,
			Description:     deployment.Description,
			UniqueID:        string(deployment.UniqueIdentifier),
			Definition:      deployment.DeploymentType, // DeploymentType serves as definition URI
			ValidTime:       deployment.ValidTime,
			Platform:        deployment.Platform,
			DeployedSystems: deployment.DeployedSystems,
			Links:           deployment.Links,
		}
		features = append(features, feature)
	}
	return features, nil
}
