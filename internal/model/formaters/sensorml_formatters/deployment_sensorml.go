package sensorml_formatters

import (
	"context"
	"fmt"
	"io"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

// DeploymentSensorMLFormatter handles serialization and deserialization of Deployment objects in SensorML format
type DeploymentSensorMLFormatter struct {
	formaters.Formatter[domains.DeploymentSensorMLFeature, *domains.Deployment]
	links *formaters.LinkResolver
}

// NewDeploymentSensorMLFormatter constructs a formatter with required repository readers
func NewDeploymentSensorMLFormatter(links *formaters.LinkResolver) *DeploymentSensorMLFormatter {
	return &DeploymentSensorMLFormatter{links: links}
}

func (f *DeploymentSensorMLFormatter) ContentType() string {
	return mediatype.SensorML
}

// --- Serialization ---

func (f *DeploymentSensorMLFormatter) Serialize(ctx context.Context, deployment *domains.Deployment) (domains.DeploymentSensorMLFeature, error) {
	return formaters.SerializeOne(ctx, deployment, f.SerializeAll)
}

func (f *DeploymentSensorMLFormatter) SerializeAll(ctx context.Context, deployments []*domains.Deployment) ([]domains.DeploymentSensorMLFeature, error) {
	if len(deployments) == 0 {
		return []domains.DeploymentSensorMLFeature{}, nil
	}

	cache, err := f.links.Lookup(ctx, formaters.DeploymentReferences(deployments))
	if err != nil {
		return nil, err
	}

	var features []domains.DeploymentSensorMLFeature
	for _, deployment := range deployments {
		// Absolutize inline link hrefs
		var platform *domains.DeployedSystemItem
		if deployment.Platform != nil {
			cp := *deployment.Platform
			cp.System.Href = f.links.Href(cp.System.Href)
			if cp.System.Type == "" {
				cp.System.Type = mediatype.GeoJSON
			}
			// Enrich platform system link from cache
			sysID := f.links.LocalID(cp.System, "systems")
			if sysID != nil {
				if sys, ok := cache.Systems[*sysID]; ok {
					cp.System.Describe(sys.Name, string(sys.UniqueIdentifier))
				}
			}
			platform = &cp
		}
		var deployedSystems []domains.DeployedSystemItem
		for _, ds := range cache.DeployedSystems(deployment) {
			cp := ds
			cp.System.Href = f.links.Href(cp.System.Href)
			if cp.System.Type == "" {
				cp.System.Type = mediatype.GeoJSON
			}
			// Enrich deployedSystems system link from cache
			sysID := f.links.LocalID(cp.System, "systems")
			if sysID != nil {
				if sys, ok := cache.Systems[*sysID]; ok {
					cp.System.Describe(sys.Name, string(sys.UniqueIdentifier))
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
			Links:           f.links.DeploymentLinks(deployment, cache),

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

	sensorML, err := common_shared.DecodeWithFieldErrors[domains.DeploymentSensorMLFeature](body)
	if err != nil {
		return nil, err
	}

	// ogc-rel links are server-generated, so they are ignored on write; the
	// parent deployment is only ever set through /deployments/{id}/subdeployments.
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
	deployment.Geometry = sensorML.Location

	return deployment, nil
}
