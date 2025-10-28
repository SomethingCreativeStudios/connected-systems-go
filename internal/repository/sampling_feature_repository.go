package repository

import (
	"github.com/yourusername/connected-systems-go/internal/model"
	"gorm.io/gorm"
)

// SamplingFeatureRepository handles SamplingFeature data access
type SamplingFeatureRepository struct {
	db *gorm.DB
}

// NewSamplingFeatureRepository creates a new SamplingFeatureRepository
func NewSamplingFeatureRepository(db *gorm.DB) *SamplingFeatureRepository {
	return &SamplingFeatureRepository{db: db}
}

// Create creates a new sampling feature
func (r *SamplingFeatureRepository) Create(sf *model.SamplingFeature) error {
	return r.db.Create(sf).Error
}

// GetByID retrieves a sampling feature by ID
func (r *SamplingFeatureRepository) GetByID(id string) (*model.SamplingFeature, error) {
	var sf model.SamplingFeature
	err := r.db.Where("id = ?", id).First(&sf).Error
	if err != nil {
		return nil, err
	}
	return &sf, nil
}

// List retrieves sampling features with filtering
func (r *SamplingFeatureRepository) List(params *QueryParams) ([]*model.SamplingFeature, int64, error) {
	var features []*model.SamplingFeature
	var total int64

	query := r.db.Model(&model.SamplingFeature{})
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

	err := query.Find(&features).Error
	return features, total, err
}

// Update updates a sampling feature
func (r *SamplingFeatureRepository) Update(sf *model.SamplingFeature) error {
	return r.db.Save(sf).Error
}

// Delete deletes a sampling feature
func (r *SamplingFeatureRepository) Delete(id string) error {
	return r.db.Delete(&model.SamplingFeature{}, "id = ?", id).Error
}

func (r *SamplingFeatureRepository) applyFilters(query *gorm.DB, params *QueryParams) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ? OR unique_identifier IN ?", params.IDs, params.IDs)
	}
	if params.Q != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+params.Q+"%", "%"+params.Q+"%")
	}
	if len(params.Parent) > 0 {
		query = query.Where("parent_system_id IN ?", params.Parent)
	}
	return query
}
