package repository

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
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
func (r *DeploymentRepository) Create(deployment *domains.Deployment) error {
	if err := r.db.Create(deployment).Error; err != nil {
		return err
	}
	r.syncSystemDeployments(deployment)
	return nil
}

// GetByID retrieves a deployment by ID
func (r *DeploymentRepository) GetByID(id string) (*domains.Deployment, error) {
	var deployment domains.Deployment
	err := r.db.Where("id = ?", id).First(&deployment).Error
	if err != nil {
		return nil, err
	}

	// Enrich deployment with associations
	deploymentPtr := &deployment
	deploymentPtr = r.findAssociations(deploymentPtr)

	return deploymentPtr, nil
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
func (r *DeploymentRepository) List(params *queryparams.DeploymentsQueryParams, parentId *string) ([]*domains.Deployment, int64, error) {
	var deployments []*domains.Deployment
	var total int64

	query := r.db.Model(&domains.Deployment{})
	query = r.applyFilters(query, params, parentId)

	if params.DateTime != nil && params.DateTime.Latest {
		err := query.Order("valid_time_start desc").Limit(1).Find(&deployments).Error
		return deployments, int64(len(deployments)), err
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query, err := ApplyCursorPagination(query, &params.QueryParams, CursorOrderIDAsc)
	if err != nil {
		return nil, 0, err
	}
	err = query.Find(&deployments).Error
	deployments = FinalizeCursorPage(deployments, &params.QueryParams)
	params.Anchors = queryparams.CursorAnchorsFor(deployments, func(deployment *domains.Deployment) []string { return []string{deployment.ID} })

	// Enrich deployments with associations
	for i, deployment := range deployments {
		deployments[i] = r.findAssociations(deployment)
	}

	return deployments, total, err
}

// Update updates a deployment
func (r *DeploymentRepository) Update(deployment *domains.Deployment) error {
	if err := r.db.Save(deployment).Error; err != nil {
		return err
	}
	r.syncSystemDeployments(deployment)
	return nil
}

// syncSystemDeployments replaces the system_deployments junction rows for a deployment
// based on the current DeployedSystems list.
func (r *DeploymentRepository) syncSystemDeployments(deployment *domains.Deployment) {
	r.db.Exec("DELETE FROM system_deployments WHERE deployment_id = ?", deployment.ID)
	for _, ds := range deployment.DeployedSystems {
		systemID := hrefLastSegment(ds.System.Href)
		if systemID == "" {
			continue
		}
		r.db.Exec(
			"INSERT INTO system_deployments (system_id, deployment_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
			systemID, deployment.ID,
		)
	}
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
func (r *DeploymentRepository) Delete(id string, cascade bool) error {
	if !cascade {
		var childCount int64
		if err := r.db.Model(&domains.Deployment{}).Where("parent_deployment_id = ?", id).Count(&childCount).Error; err != nil {
			return err
		}
		if childCount > 0 {
			return ErrHasChildren
		}

		if err := r.db.Exec("DELETE FROM system_deployments WHERE deployment_id = ?", id).Error; err != nil {
			return err
		}

		result := r.db.Delete(&domains.Deployment{}, "id = ?", id)
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
	}

	var count int64
	if err := r.db.Model(&domains.Deployment{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		return r.deleteCascade(tx, id)
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
func (r *DeploymentRepository) DeleteAll() error {
	return r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&domains.Deployment{}).Error
}

func (r *DeploymentRepository) applyFilters(query *gorm.DB, params *queryparams.DeploymentsQueryParams, parentId *string) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ? OR unique_identifier IN ?", params.IDs, params.IDs)
	}

	if len(params.Q) > 0 {
		var clauses []string
		var args []interface{}
		for _, term := range params.Q {
			clauses = append(clauses, "name ILIKE ?")
			args = append(args, "%"+term+"%")
			clauses = append(clauses, "description ILIKE ?")
			args = append(args, "%"+term+"%")
		}
		query = query.Where(strings.Join(clauses, " OR "), args...)
	}

	if params.DateTime != nil && !params.DateTime.Latest {
		// Only add conditions if start/end are not nil
		if params.DateTime.Start != nil && params.DateTime.End != nil {
			query = query.Where("valid_time_start <= ? AND (valid_time_end IS NULL OR valid_time_end >= ?)", params.DateTime.End, params.DateTime.Start)
		} else if params.DateTime.Start != nil {
			query = query.Where("valid_time_end IS NULL OR valid_time_end >= ?", params.DateTime.Start)
		} else if params.DateTime.End != nil {
			query = query.Where("valid_time_start <= ?", params.DateTime.End)
		}
	}

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

			query = query.Where("system_ids @> ?::jsonb", string(needle))
		}
	}

	if len(params.Foi) > 0 {
		query = query.
			Joins("JOIN system_deployments sd ON sd.deployment_id = deployments.id").
			Joins("JOIN sampling_features sf ON sf.parent_system_id = sd.system_id").
			Where("sf.sampled_feature_id IN ? OR sf.id IN ?", params.Foi, params.Foi).
			Distinct()
	}

	if len(params.Parent) > 0 && parentId == nil {
		query = query.Where("parent_deployment_id IN ?", params.Parent)
	}

	if parentId != nil {
		if params.Recursive {
			// Recursive traversal from parent to include all descendants.
			query = query.Where(`id IN (
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
			query = query.Where("parent_deployment_id = ?", *parentId)
		}
	} else {
		if params.Recursive {
			// Canonical recursive search should include top-level deployments and all descendants.
		} else {
			query = query.Where("parent_deployment_id IS NULL OR parent_deployment_id = ''")
		}
	}

	return query
}

func (r *DeploymentRepository) findAssociations(deployment *domains.Deployment) *domains.Deployment {
	subDeployments := findAllChildren(r.db, deployment.ID)
	if len(subDeployments) == 0 {
		return deployment
	}

	allSubLinks := []common_shared.Link{}

	for _, subDeployment := range subDeployments {
		allSubLinks = append(allSubLinks, subDeployment.Links...)
	}

	samplingFeatures := findLinks(common_shared.OGCRel("samplingFeatures"), allSubLinks)
	featuresOfInterest := findLinks(common_shared.OGCRel("featuresOfInterest"), allSubLinks)
	deployedSystems := findLinks(common_shared.OGCRel("deployedSystems"), allSubLinks)

	deployment.Links = append(deployment.Links, samplingFeatures...)
	deployment.Links = append(deployment.Links, featuresOfInterest...)
	deployment.Links = append(deployment.Links, deployedSystems...)

	// De-duplicate links, just in case person sends back duplicates
	deployment.Links = deDuplicateLinks(deployment.Links)

	return deployment
}

func findLinks(rel string, links common_shared.Links) []common_shared.Link {
	var results []common_shared.Link
	for _, link := range links {
		if common_shared.RelEquals(link.Rel, rel) {
			results = append(results, link)
		}
	}
	return results
}

func deDuplicateLinks(links []common_shared.Link) []common_shared.Link {
	seen := make(map[string]bool)
	var result []common_shared.Link

	for _, link := range links {
		key := link.Rel + "|" + link.Href
		if !seen[key] {
			seen[key] = true
			result = append(result, link)
		}
	}

	return result
}

// stringIndex returns the index of substr in s or -1
func stringIndex(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func findAllChildren(db *gorm.DB, parentID string) []domains.Deployment {
	var children []domains.Deployment

	// Look up deployment_closures for all descendants
	db.Table("deployments").
		Joins("JOIN deployment_closures ON deployments.id = deployment_closures.descendant_id").
		Where("deployment_closures.ancestor_id = ? AND deployment_closures.depth > 0", parentID).
		Find(&children)

	return children
}
