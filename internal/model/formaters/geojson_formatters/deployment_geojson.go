package geojson_formatters

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

// DeploymentGeoJSONFormatter handles serialization and deserialization of Deployment objects in GeoJSON format
type DeploymentGeoJSONFormatter struct {
	formaters.Formatter[domains.DeploymentGeoJSONFeature, *domains.Deployment]
	links *formaters.LinkResolver
}

// NewDeploymentGeoJSONFormatter constructs a formatter with required repository readers
func NewDeploymentGeoJSONFormatter(links *formaters.LinkResolver) *DeploymentGeoJSONFormatter {
	return &DeploymentGeoJSONFormatter{links: links}
}

func (f *DeploymentGeoJSONFormatter) ContentType() string {
	return mediatype.GeoJSON
}

// --- Serialization ---

func (f *DeploymentGeoJSONFormatter) Serialize(ctx context.Context, deployment *domains.Deployment) (domains.DeploymentGeoJSONFeature, error) {
	return formaters.SerializeOne(ctx, deployment, f.SerializeAll)
}

func (f *DeploymentGeoJSONFormatter) SerializeAll(ctx context.Context, deployments []*domains.Deployment) ([]domains.DeploymentGeoJSONFeature, error) {
	if len(deployments) == 0 {
		return []domains.DeploymentGeoJSONFeature{}, nil
	}

	cache, err := f.links.Lookup(ctx, formaters.DeploymentReferences(deployments))
	if err != nil {
		return nil, err
	}

	var features []domains.DeploymentGeoJSONFeature
	for _, deployment := range deployments {
		if deployment == nil {
			continue
		}

		var systemLinks common_shared.Links
		for _, ds := range cache.DeployedSystems(deployment) {
			link := ds.System
			link.Href = f.links.Href(link.Href)
			if link.Type == "" {
				link.Type = mediatype.GeoJSON
			}
			// Enrich deployedSystems@link from cache
			sysID := f.links.LocalID(link, "systems")
			if sysID != nil {
				if sys, ok := cache.Systems[*sysID]; ok {
					link.Describe(sys.Name, string(sys.UniqueIdentifier))
				}
			}
			systemLinks = append(systemLinks, link)
		}

		var platformLink *common_shared.Link
		if deployment.Platform != nil {
			platform := deployment.Platform.System
			platform.Href = f.links.Href(platform.Href)
			if platform.Type == "" {
				platform.Type = mediatype.GeoJSON
			}
			// Enrich platform@link from cache
			platID := f.links.LocalID(platform, "systems")
			if platID != nil {
				if sys, ok := cache.Systems[*platID]; ok {
					platform.Describe(sys.Name, string(sys.UniqueIdentifier))
				}
			}
			platformLink = &platform
		}

		feature := domains.DeploymentGeoJSONFeature{
			Type:     "Feature",
			ID:       deployment.ID,
			Geometry: deployment.Geometry,
			Properties: domains.DeploymentGeoJSONProperties{
				UID:             deployment.UniqueIdentifier,
				Name:            deployment.Name,
				Description:     deployment.Description,
				FeatureType:     deployment.DeploymentType,
				ValidTime:       common_shared.NonEmptyTimeRange(deployment.ValidTime),
				Definition:      deployment.DeploymentType,
				Platform:        platformLink,
				DeployedSystems: systemLinks,
			},
			Links: f.links.DeploymentLinks(deployment, cache),
		}
		features = append(features, feature)
	}

	return features, nil
}

// --- Deserialization ---

func (f *DeploymentGeoJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Deployment, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	geoJSON, err := common_shared.DecodeWithFieldErrors[struct {
		Type       string                              `json:"type"`
		ID         string                              `json:"id,omitempty"`
		Properties domains.DeploymentGeoJSONProperties `json:"properties"`
		Geometry   *common_shared.GoGeom               `json:"geometry,omitempty"`
		Links      common_shared.Links                 `json:"links,omitempty"`
	}](body)
	if err != nil {
		return nil, err
	}

	// ogc-rel links are server-generated, so they are ignored on write; the
	// parent deployment is only ever set through /deployments/{id}/subdeployments.
	deployment := &domains.Deployment{
		Links: common_shared.StripAssociationLinks(geoJSON.Links),
	}

	// Assign geometry
	if geoJSON.Geometry != nil {
		deployment.Geometry = geoJSON.Geometry
	}

	// Required root fields per OGC CS Part 1 deployment GeoJSON schema
	if geoJSON.Properties.UID == "" {
		return nil, fmt.Errorf("properties.uid is required")
	}
	if geoJSON.Properties.Name == "" {
		return nil, fmt.Errorf("properties.name is required")
	}
	if geoJSON.Properties.FeatureType == "" {
		return nil, fmt.Errorf("properties.featureType is required")
	}

	// Extract properties
	deployment.UniqueIdentifier = domains.UniqueID(geoJSON.Properties.UID)
	deployment.Name = geoJSON.Properties.Name
	deployment.Description = geoJSON.Properties.Description
	deployment.DeploymentType = geoJSON.Properties.FeatureType

	deployment.ValidTime = common_shared.NonEmptyTimeRange(geoJSON.Properties.ValidTime)
	if deployment.ValidTime == nil {
		return nil, fmt.Errorf("properties.validTime is required")
	}

	// Platform and DeployedSystems
	if geoJSON.Properties.Platform != nil {
		deployment.PlatformID = geoJSON.Properties.Platform.GetId("systems")
		var platformItem domains.DeployedSystemItem
		platformItem.System = *geoJSON.Properties.Platform
		deployment.Platform = &platformItem
	}

	if len(geoJSON.Properties.DeployedSystems) > 0 {
		var systemIDs common_shared.StringArray
		for _, ds := range geoJSON.Properties.DeployedSystems {
			systemID := ds.GetId("systems")
			if systemID == nil {
				continue
			}

			systemIDs = append(systemIDs, *systemID)

			var deployedItem domains.DeployedSystemItem
			deployedItem.Name = "System_" + strings.ReplaceAll(*systemID, "-", "_")
			deployedItem.System = ds
			deployment.DeployedSystems = append(deployment.DeployedSystems, deployedItem)
		}

		if len(systemIDs) > 0 {
			deployment.SystemIds = &systemIDs
		}
	}

	return deployment, nil
}
