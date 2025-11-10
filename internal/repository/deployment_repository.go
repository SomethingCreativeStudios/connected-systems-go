package repository

import (
	"context"
	"encoding/json"
	"strings"

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
	return r.db.Create(deployment).Error
}

// GetByID retrieves a deployment by ID
func (r *DeploymentRepository) GetByID(id string) (*domains.Deployment, error) {
	var deployment domains.Deployment
	err := r.db.Where("id = ?", id).First(&deployment).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

// List retrieves deployments with filtering
func (r *DeploymentRepository) List(params *queryparams.DeploymentsQueryParams) ([]*domains.Deployment, int64, error) {
	var deployments []*domains.Deployment
	var total int64

	query := r.db.Model(&domains.Deployment{})
	query = r.applyFilters(query, params)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	err := query.Find(&deployments).Error
	return deployments, total, err
}

// Update updates a deployment
func (r *DeploymentRepository) Update(deployment *domains.Deployment) error {
	return r.db.Save(deployment).Error
}

// Delete deletes a deployment
func (r *DeploymentRepository) Delete(id string) error {
	return r.db.Delete(&domains.Deployment{}, "id = ?", id).Error
}

func (r *DeploymentRepository) applyFilters(query *gorm.DB, params *queryparams.DeploymentsQueryParams) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ? OR unique_identifier IN ?", params.IDs, params.IDs)
	}
	if len(params.Q) > 0 {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+strings.Join(params.Q, "%")+"%", "%"+strings.Join(params.Q, "%")+"%")
	}
	return query
}

// GetBySystemIDs returns deployments grouped by system UID/ID by inspecting the
// properties JSON for deployed system links. This is a best-effort implementation
// to avoid adding hard foreign-key constraints; it performs a text search on the
// JSONB properties and then inspects the JSON to map deployments to system IDs.
func (r *DeploymentRepository) GetBySystemIDs(ctx context.Context, systemIDs []string) (map[string][]*domains.Deployment, error) {
	result := make(map[string][]*domains.Deployment)
	if len(systemIDs) == 0 {
		return result, nil
	}

	// Build a WHERE clause that checks properties as text for any of the system IDs
	// e.g. properties::text ILIKE '%urn:example:system:...%'
	query := r.db.WithContext(ctx).Model(&domains.Deployment{})
	first := true
	for _, id := range systemIDs {
		like := "%" + id + "%"
		if first {
			query = query.Where("properties::text ILIKE ?", like)
			first = false
		} else {
			query = query.Or("properties::text ILIKE ?", like)
		}
	}

	var deployments []*domains.Deployment
	if err := query.Find(&deployments).Error; err != nil {
		return nil, err
	}

	// Inspect each deployment.Properties to find deployedSystems@link hrefs
	for _, d := range deployments {
		if d == nil {
			continue
		}
		if d.Properties == nil {
			continue
		}

		// Look for deployedSystems@link which should be an array of objects with href
		if raw, ok := d.Properties["deployedSystems@link"]; ok {
			// marshal/unmarshal to normalize types
			b, _ := json.Marshal(raw)
			var arr []map[string]interface{}
			if err := json.Unmarshal(b, &arr); err == nil {
				for _, el := range arr {
					if hrefVal, ok := el["href"].(string); ok {
						for _, sid := range systemIDs {
							if hrefVal == sid || contains(hrefVal, sid) {
								result[sid] = append(result[sid], d)
							}
						}
					}
				}
			}
		}
	}

	return result, nil
}

// contains is a helper to check substring (case-sensitive)
func contains(href, sub string) bool {
	return len(href) >= len(sub) && (href == sub || (len(sub) > 0 && (stringIndex(href, sub) >= 0)))
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
