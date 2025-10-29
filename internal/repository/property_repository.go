package repository

import (
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// PropertyRepository handles Property data access
type PropertyRepository struct {
	db *gorm.DB
}

// NewPropertyRepository creates a new PropertyRepository
func NewPropertyRepository(db *gorm.DB) *PropertyRepository {
	return &PropertyRepository{db: db}
}

// Create creates a new property
func (r *PropertyRepository) Create(property *domains.Property) error {
	return r.db.Create(property).Error
}

// GetByID retrieves a property by ID
func (r *PropertyRepository) GetByID(id string) (*domains.Property, error) {
	var property domains.Property
	err := r.db.Where("id = ?", id).First(&property).Error
	if err != nil {
		return nil, err
	}
	return &property, nil
}

// List retrieves properties with filtering
func (r *PropertyRepository) List(params *queryparams.PropertiesQueryParams) ([]*domains.Property, int64, error) {
	var properties []*domains.Property
	var total int64

	query := r.db.Model(&domains.Property{})
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

	err := query.Find(&properties).Error
	return properties, total, err
}

// Update updates a property
func (r *PropertyRepository) Update(property *domains.Property) error {
	return r.db.Save(property).Error
}

// Delete deletes a property
func (r *PropertyRepository) Delete(id string) error {
	return r.db.Delete(&domains.Property{}, "id = ?", id).Error
}

func (r *PropertyRepository) applyFilters(query *gorm.DB, params *queryparams.PropertiesQueryParams) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("unique_identifier IN ?", params.IDs, params.IDs)
	}

	if params.Q != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ? OR property_type ILIKE ? OR object_type ILIKE ?", "%"+params.Q+"%", "%"+params.Q+"%", "%"+params.Q+"%", "%"+params.Q+"%")
	}

	if len(params.ObjectType) > 0 {
		query = query.Where("object_type IN ?", params.ObjectType)
	}

	if len(params.BaseProperty) > 0 {
		query = query.Where("base_property IN ?", params.BaseProperty)
	}

	return query
}
