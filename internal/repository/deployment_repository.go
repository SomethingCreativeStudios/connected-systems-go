package repository

import (
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
	if params.Q != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+params.Q+"%", "%"+params.Q+"%")
	}
	return query
}
