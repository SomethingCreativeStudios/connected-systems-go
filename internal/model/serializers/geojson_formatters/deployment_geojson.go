package geojson_formatters

import (
	"context"
	"encoding/json"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// DeploymentGeoJSONFormatter handles serialization and deserialization of Deployment objects in GeoJSON format
type DeploymentGeoJSONFormatter struct {
	serializers.Formatter[domains.DeploymentGeoJSONFeature, *domains.Deployment]
	repos *repository.Repositories
}

// NewDeploymentGeoJSONFormatter constructs a formatter with required repository readers
func NewDeploymentGeoJSONFormatter(repos *repository.Repositories) *DeploymentGeoJSONFormatter {
	return &DeploymentGeoJSONFormatter{repos: repos}
}

func (f *DeploymentGeoJSONFormatter) ContentType() string {
	return GeoJSONContentType
}

// --- Serialization ---

func (f *DeploymentGeoJSONFormatter) Serialize(ctx context.Context, deployment *domains.Deployment) (domains.DeploymentGeoJSONFeature, error) {
	features, err := f.SerializeAll(ctx, []*domains.Deployment{deployment})
	if err != nil {
		return domains.DeploymentGeoJSONFeature{}, err
	}
	return features[0], nil
}

func (f *DeploymentGeoJSONFormatter) SerializeAll(ctx context.Context, deployments []*domains.Deployment) ([]domains.DeploymentGeoJSONFeature, error) {
	if len(deployments) == 0 {
		return []domains.DeploymentGeoJSONFeature{}, nil
	}

	var features []domains.DeploymentGeoJSONFeature
	for _, deployment := range deployments {
		feature := domains.DeploymentGeoJSONFeature{
			Type:     "Feature",
			ID:       deployment.ID,
			Geometry: deployment.Geometry,
			Properties: domains.DeploymentGeoJSONProperties{
				UID:             deployment.UniqueIdentifier,
				Name:            deployment.Name,
				Description:     deployment.Description,
				FeatureType:     deployment.DeploymentType,
				ValidTime:       deployment.ValidTime,
				Definition:      deployment.DeploymentType,
				Platform:        deployment.Platform,
				DeployedSystems: deployment.DeployedSystems,
			},
			Links: deployment.Links,
		}
		features = append(features, feature)
	}

	return features, nil
}

// --- Deserialization ---

func (f *DeploymentGeoJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Deployment, error) {
	var geoJSON struct {
		Type       string                              `json:"type"`
		ID         string                              `json:"id,omitempty"`
		Properties domains.DeploymentGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.GoGeom               `json:"geometry,omitempty"`
		Links      common_shared.Links                 `json:"links,omitempty"`
	}

	if err := json.NewDecoder(reader).Decode(&geoJSON); err != nil {
		return nil, err
	}

	deployment := &domains.Deployment{
		Links: geoJSON.Links,
	}

	// Assign geometry
	if geoJSON.Geometry != nil {
		deployment.Geometry = geoJSON.Geometry
	}

	// Extract properties
	deployment.UniqueIdentifier = domains.UniqueID(geoJSON.Properties.UID)
	deployment.Name = geoJSON.Properties.Name
	deployment.Description = geoJSON.Properties.Description

	// Prefer explicit definition when present (matches schema semantics)
	if geoJSON.Properties.Definition != "" {
		deployment.DeploymentType = geoJSON.Properties.Definition
	} else {
		deployment.DeploymentType = geoJSON.Properties.FeatureType
	}

	deployment.ValidTime = geoJSON.Properties.ValidTime

	// Platform and DeployedSystems
	if geoJSON.Properties.Platform != nil {
		deployment.Platform = geoJSON.Properties.Platform
	}
	if len(geoJSON.Properties.DeployedSystems) > 0 {
		deployment.DeployedSystems = geoJSON.Properties.DeployedSystems
	}

	return deployment, nil
}
