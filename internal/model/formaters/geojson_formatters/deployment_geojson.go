package geojson_formatters

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// DeploymentGeoJSONFormatter handles serialization and deserialization of Deployment objects in GeoJSON format
type DeploymentGeoJSONFormatter struct {
	formaters.Formatter[domains.DeploymentGeoJSONFeature, *domains.Deployment]
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
	if deployment == nil {
		return domains.DeploymentGeoJSONFeature{}, fmt.Errorf("deployment cannot be nil")
	}

	features, err := f.SerializeAll(ctx, []*domains.Deployment{deployment})
	if err != nil {
		return domains.DeploymentGeoJSONFeature{}, err
	}
	if len(features) == 0 {
		return domains.DeploymentGeoJSONFeature{}, fmt.Errorf("deployment serialization produced no features")
	}
	return features[0], nil
}

func (f *DeploymentGeoJSONFormatter) SerializeAll(ctx context.Context, deployments []*domains.Deployment) ([]domains.DeploymentGeoJSONFeature, error) {
	if len(deployments) == 0 {
		return []domains.DeploymentGeoJSONFeature{}, nil
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

	var features []domains.DeploymentGeoJSONFeature
	for _, deployment := range deployments {
		if deployment == nil {
			continue
		}

		var systemLinks common_shared.Links
		for _, ds := range deployment.DeployedSystems {
			link := ds.System
			link.Href = formaters.ToFunctionalAssociationHref(link.Href)
			if link.Type == "" {
				link.Type = formaters.GeoJSONContentType
			}
			// Enrich deployedSystems@link from cache
			sysID := link.GetId("systems")
			if sysID != nil {
				if sys, ok := cache.Systems[*sysID]; ok {
					link.Title = sys.Name
					uid := string(sys.UniqueIdentifier)
					if uid != "" {
						link.UID = &uid
					}
				}
			}
			systemLinks = append(systemLinks, link)
		}

		var platformLink *common_shared.Link
		if deployment.Platform != nil {
			platform := deployment.Platform.System
			platform.Href = formaters.ToFunctionalAssociationHref(platform.Href)
			if platform.Type == "" {
				platform.Type = formaters.GeoJSONContentType
			}
			// Enrich platform@link from cache
			platID := platform.GetId("systems")
			if platID != nil {
				if sys, ok := cache.Systems[*platID]; ok {
					platform.Title = sys.Name
					uid := string(sys.UniqueIdentifier)
					if uid != "" {
						platform.UID = &uid
					}
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
			Links: formaters.AppendDeploymentAssociationLinks(deployment),
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

	associationLinks := formaters.DeploymentAssociationLinks(geoJSON.Links)

	deployment := &domains.Deployment{
		Links: common_shared.StripAssociationLinks(geoJSON.Links),
	}

	// Assign geometry
	if geoJSON.Geometry != nil {
		deployment.Geometry = geoJSON.Geometry
	}

	// Extract properties
	deployment.UniqueIdentifier = domains.UniqueID(geoJSON.Properties.UID)
	deployment.Name = geoJSON.Properties.Name
	deployment.Description = geoJSON.Properties.Description
	deployment.DeploymentType = geoJSON.Properties.FeatureType

	deployment.ValidTime = common_shared.NonEmptyTimeRange(geoJSON.Properties.ValidTime)

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

	formaters.ApplyDeploymentAssociationLinks(deployment, associationLinks)

	return deployment, nil
}
