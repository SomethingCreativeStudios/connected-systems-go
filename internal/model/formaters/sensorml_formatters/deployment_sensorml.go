package sensorml_formatters

import (
	"context"
	"fmt"
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

	// Collect system IDs for batch enrichment (platform + deployedSystems)
	systemIDs := make([]string, 0, len(deployments)*2)
	for _, deployment := range deployments {
		if deployment == nil {
			continue
		}
		if deployment.Platform != nil {
			id := deployment.Platform.System.GetId("systems")
			if id != nil && *id != "" {
				systemIDs = append(systemIDs, *id)
			}
		}
		for _, ds := range deployment.DeployedSystems {
			id := ds.System.GetId("systems")
			if id != nil && *id != "" {
				systemIDs = append(systemIDs, *id)
			}
		}
	}

	// Build resource cache for enriching inline links
	cache := formaters.NewResourceCache()
	if f.repos != nil && len(systemIDs) > 0 {
		_ = cache.FetchParentSystems(ctx, f.repos.System, systemIDs)
	}

	var features []domains.DeploymentSensorMLFeature
	for _, deployment := range deployments {
		// Absolutize inline link hrefs
		var platform *domains.DeployedSystemItem
		if deployment.Platform != nil {
			cp := *deployment.Platform
			cp.System.Href = formaters.ToFunctionalAssociationHref(cp.System.Href)
			if cp.System.Type == "" {
				cp.System.Type = formaters.GeoJSONContentType
			}
			// Enrich platform system link from cache
			sysID := cp.System.GetId("systems")
			if sysID != nil {
				if sys, ok := cache.Systems[*sysID]; ok {
					cp.System.Title = sys.Name
					uid := string(sys.UniqueIdentifier)
					if uid != "" {
						cp.System.UID = &uid
					}
				}
			}
			platform = &cp
		}
		var deployedSystems []domains.DeployedSystemItem
		for _, ds := range deployment.DeployedSystems {
			cp := ds
			cp.System.Href = formaters.ToFunctionalAssociationHref(cp.System.Href)
			if cp.System.Type == "" {
				cp.System.Type = formaters.GeoJSONContentType
			}
			// Enrich deployedSystems system link from cache
			sysID := cp.System.GetId("systems")
			if sysID != nil {
				if sys, ok := cache.Systems[*sysID]; ok {
					cp.System.Title = sys.Name
					uid := string(sys.UniqueIdentifier)
					if uid != "" {
						cp.System.UID = &uid
					}
				}
			}
			deployedSystems = append(deployedSystems, cp)
		}

		feature := domains.DeploymentSensorMLFeature{
			ID:              deployment.ID,
			Type:            "Deployment",
			Label:           deployment.Name,
			Description:     deployment.Description,
			UniqueID:        string(deployment.UniqueIdentifier),
			Definition:      deployment.DeploymentType,
			ValidTime:       common_shared.NonEmptyTimeRange(deployment.ValidTime),
			Location:        deployment.Geometry,
			Platform:        platform,
			DeployedSystems: deployedSystems,
			Links:           formaters.AppendDeploymentAssociationLinks(deployment),

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

	sensorML, err := common_shared.DecodeWithFieldErrors[domains.DeploymentSensorMLFeature](body)
	if err != nil {
		return nil, err
	}

	associationLinks := formaters.DeploymentAssociationLinks(sensorML.Links)

	deployment := &domains.Deployment{
		Links: common_shared.StripAssociationLinks(sensorML.Links),
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

	// Required root fields per OGC CS Part 1 deployment SensorML schema
	if sensorML.Type == "" {
		return nil, fmt.Errorf("type is required")
	}
	if deployment.UniqueIdentifier == "" {
		return nil, fmt.Errorf("uniqueId is required")
	}
	if deployment.Name == "" {
		return nil, fmt.Errorf("label is required")
	}
	if sensorML.Definition == "" {
		return nil, fmt.Errorf("definition is required")
	}

	deployment.ValidTime = common_shared.NonEmptyTimeRange(sensorML.ValidTime)
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

	// Handle geometry from "location" (OGC CS spec) or "position" field
	geomRaw, ok := raw["location"]
	if !ok {
		geomRaw, ok = raw["position"]
	}
	if ok {
		if gb, err := json.Marshal(geomRaw); err == nil {
			var gg common_shared.GoGeom
			if err := json.Unmarshal(gb, &gg); err == nil {
				deployment.Geometry = &gg
			}
		}
	}

	formaters.ApplyDeploymentAssociationLinks(deployment, associationLinks)

	return deployment, nil
}
