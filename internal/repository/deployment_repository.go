package repository

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// DeploymentRepository handles Deployment data access
type DeploymentRepository struct {
	db *gorm.DB
}

// NewDeploymentRepository creates a new DeploymentRepository
func NewDeploymentRepository(db *gorm.DB) *DeploymentRepository {
	return &DeploymentRepository{db: db}
}

// Create creates a new deployment
func (r *DeploymentRepository) Create(ctx context.Context, deployment *domains.Deployment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if deployment.ParentDeploymentID != nil {
			if err := requireParent(tx, &domains.Deployment{}, "deployment", *deployment.ParentDeploymentID); err != nil {
				return err
			}
		}
		if err := tx.Create(deployment).Error; err != nil {
			return err
		}
		return syncSystemDeployments(tx, deployment)
	})
}

// GetByID retrieves a deployment by ID
func (r *DeploymentRepository) GetByID(ctx context.Context, id string) (*domains.Deployment, error) {
	var deployment domains.Deployment
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&deployment).Error; err != nil {
		return nil, err
	}
	return &deployment, nil
}

// GetByIDs returns deployments keyed by ID (batch lookup)
func (r *DeploymentRepository) GetByIDs(ctx context.Context, ids []string) (map[string]*domains.Deployment, error) {
	result := make(map[string]*domains.Deployment)
	if len(ids) == 0 {
		return result, nil
	}

	var deployments []*domains.Deployment
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&deployments).Error; err != nil {
		return nil, err
	}

	for _, d := range deployments {
		if d != nil {
			result[d.ID] = d
		}
	}

	return result, nil
}

// List retrieves deployments with filtering
func (r *DeploymentRepository) List(ctx context.Context, params *queryparams.DeploymentsQueryParams, parentId *string) ([]*domains.Deployment, int64, error) {
	var deployments []*domains.Deployment

	query := r.db.WithContext(ctx).Model(&domains.Deployment{})
	query = r.applyFilters(query, params, parentId)

	if params.DateTime != nil && params.DateTime.Latest {
		err := query.Order("deployments.valid_time_start desc").Limit(1).Find(&deployments).Error
		return deployments, int64(len(deployments)), err
	}

	return listPage(query, &params.QueryParams, idOrder("deployments"), idAnchor[*domains.Deployment])
}

// Update replaces an existing deployment. It returns ErrNotFound rather than
// creating the deployment, and keeps the parent, which is only ever set
// through /deployments/{id}/subdeployments.
func (r *DeploymentRepository) Update(ctx context.Context, deployment *domains.Deployment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := replaceByIDChecked(tx, deployment, func(existing *domains.Deployment) error {
			deployment.ParentDeploymentID = existing.ParentDeploymentID
			projection, err := (&DeploymentRepository{db: tx}).DeployedSystemsByIDs(ctx, []string{deployment.ID})
			if err != nil {
				return err
			}
			direct := map[string]bool{}
			for _, item := range existing.DeployedSystems {
				if id := item.System.GetId("systems"); id != nil {
					direct[*id] = true
				}
			}
			inherited := map[string]bool{}
			for _, item := range projection[deployment.ID] {
				if id := item.System.GetId("systems"); id != nil && !direct[*id] {
					inherited[*id] = true
				}
			}
			owned := domains.DeployedSystemItems{}
			systemIDs := common_shared.StringArray{}
			for _, item := range deployment.DeployedSystems {
				id := item.System.GetId("systems")
				if id != nil && inherited[*id] {
					continue
				}
				owned = append(owned, item)
				if id != nil {
					systemIDs = append(systemIDs, *id)
				}
			}
			deployment.DeployedSystems = owned
			deployment.SystemIds = &systemIDs
			return nil
		}); err != nil {
			return err
		}
		return syncSystemDeployments(tx, deployment)
	})
}

// syncSystemDeployments replaces the system_deployments junction rows for a deployment
// based on the current DeployedSystems list.
func syncSystemDeployments(tx *gorm.DB, deployment *domains.Deployment) error {
	if err := tx.Exec("DELETE FROM system_deployments WHERE deployment_id = ?", deployment.ID).Error; err != nil {
		return err
	}
	for _, ds := range deployment.DeployedSystems {
		systemID := hrefLastSegment(ds.System.Href)
		if systemID == "" {
			continue
		}
		if err := tx.Exec(
			"INSERT INTO system_deployments (system_id, deployment_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
			systemID, deployment.ID,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

// hrefLastSegment extracts the final path segment from a URL or path string.
func hrefLastSegment(href string) string {
	href = strings.TrimRight(href, "/")
	if idx := strings.LastIndex(href, "/"); idx >= 0 {
		return href[idx+1:]
	}
	return href
}

// Delete deletes a deployment
// The checks and every delete share one transaction, so a refused or failed
// delete leaves the deployment and its system links exactly as they were.
func (r *DeploymentRepository) Delete(ctx context.Context, id string, cascade bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if cascade {
			if exists, err := hasRows(tx, &domains.Deployment{}, "id = ?", id); err != nil {
				return err
			} else if !exists {
				return ErrNotFound
			}
			return r.deleteCascade(tx, id)
		}

		if has, err := hasRows(tx, &domains.Deployment{}, "parent_deployment_id = ?", id); err != nil {
			return err
		} else if has {
			return ErrHasChildren
		}

		if err := tx.Exec("DELETE FROM system_deployments WHERE deployment_id = ?", id).Error; err != nil {
			return err
		}

		result := tx.Delete(&domains.Deployment{}, "id = ?", id)
		if result.Error != nil {
			if isFKViolation(result.Error) {
				return ErrHasChildren
			}
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func (r *DeploymentRepository) deleteCascade(tx *gorm.DB, deploymentID string) error {
	var childIDs []string
	if err := tx.Model(&domains.Deployment{}).Where("parent_deployment_id = ?", deploymentID).Pluck("id", &childIDs).Error; err != nil {
		return err
	}

	for _, childID := range childIDs {
		if err := r.deleteCascade(tx, childID); err != nil {
			return err
		}
	}

	if err := tx.Exec("DELETE FROM system_deployments WHERE deployment_id = ?", deploymentID).Error; err != nil {
		return err
	}

	return tx.Delete(&domains.Deployment{}, "id = ?", deploymentID).Error
}

// Delete all deployments - for testing purposes
func (r *DeploymentRepository) DeleteAll(ctx context.Context) error {
	return r.db.WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&domains.Deployment{}).Error
}

func (r *DeploymentRepository) applyFilters(query *gorm.DB, params *queryparams.DeploymentsQueryParams, parentId *string) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("deployments.id IN ? OR deployments.unique_identifier IN ?", params.IDs, params.IDs)
	}

	query = applyKeywordFilter(query, params.Q, "deployments.name", "deployments.description")

	query = applyValidTimeFilter(query, "deployments", params.DateTime)

	query = applySpatialIntersectionFilters(query, "deployments.geometry", params.Bbox, params.Geom)

	if len(params.ControlledProperty) > 0 {
		query = query.Joins("JOIN procedure_controlled_properties ON procedures.id = procedure_controlled_properties.procedure_id").
			Where("procedure_controlled_properties.property_id IN ?", params.ControlledProperty)
	}

	if len(params.ObservedProperty) > 0 {
		query = query.Joins("JOIN procedure_observed_properties ON procedures.id = procedure_observed_properties.procedure_id").
			Where("procedure_observed_properties.property_id IN ?", params.ObservedProperty)
	}

	if len(params.System) > 0 {
		for _, systemID := range params.System {
			if systemID == "" {
				continue
			}

			needle, err := json.Marshal([]string{systemID})
			if err != nil {
				continue
			}

			query = query.Where("deployments.system_ids @> ?::jsonb", string(needle))
		}
	}

	if len(params.Foi) > 0 {
		query = query.Where(`EXISTS (SELECT 1 FROM system_deployments sd
			JOIN sampling_features sf ON sf.parent_system_id = sd.system_id
			WHERE sd.deployment_id = deployments.id AND (sf.sampled_feature_id IN ? OR sf.id IN ?))`, params.Foi, params.Foi)
	}

	if len(params.Parent) > 0 && parentId == nil {
		query = query.Where("deployments.parent_deployment_id IN ?", params.Parent)
	}

	if parentId != nil {
		if params.Recursive {
			// Recursive traversal from parent to include all descendants.
			query = query.Where(`deployments.id IN (
				WITH RECURSIVE deployment_descendants AS (
					SELECT id FROM deployments WHERE parent_deployment_id = ?
					UNION ALL
					SELECT d.id
					FROM deployments d
					JOIN deployment_descendants dd ON d.parent_deployment_id = dd.id
				)
				SELECT id FROM deployment_descendants
			)`, *parentId)
		} else {
			// Direct children only
			query = query.Where("deployments.parent_deployment_id = ?", *parentId)
		}
	} else {
		if params.Recursive {
			// Canonical recursive search should include top-level deployments and all descendants.
		} else {
			query = query.Where("deployments.parent_deployment_id IS NULL OR deployments.parent_deployment_id = ''")
		}
	}

	return applyCQLFilter(query, &params.QueryParams)
}
