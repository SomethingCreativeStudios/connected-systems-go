package repository

import (
	"context"
	"encoding/json"
	"strings"

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

	if len(params.Q) > 0 {
		query = query.Where("name ILIKE ? OR description ILIKE ? OR property_type ILIKE ? OR object_type ILIKE ?", "%"+strings.Join(params.Q, "%")+"%", "%"+strings.Join(params.Q, "%")+"%", "%"+strings.Join(params.Q, "%")+"%", "%"+strings.Join(params.Q, "%")+"%")
	}

	if len(params.ObjectType) > 0 {
		query = query.Where("object_type IN ?", params.ObjectType)
	}

	if len(params.BaseProperty) > 0 {
		query = query.Where("base_property IN ?", params.BaseProperty)
	}

	return query
}

// GetBySystemIDs returns properties grouped by system ID by inspecting JSON properties
func (r *PropertyRepository) GetBySystemIDs(ctx context.Context, systemIDs []string) (map[string][]*domains.Property, error) {
	result := make(map[string][]*domains.Property)
	if len(systemIDs) == 0 {
		return result, nil
	}

	query := r.db.WithContext(ctx).Model(&domains.Property{})
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

	var props []*domains.Property
	if err := query.Find(&props).Error; err != nil {
		return nil, err
	}

	for _, p := range props {
		if p == nil || p.Properties == nil {
			continue
		}
		// attempt to find linking hrefs in properties
		for _, rawVal := range p.Properties {
			b, _ := json.Marshal(rawVal)
			var arr []map[string]interface{}
			if err := json.Unmarshal(b, &arr); err != nil {
				continue
			}
			for _, el := range arr {
				if hrefVal, ok := el["href"].(string); ok {
					for _, sid := range systemIDs {
						if hrefVal == sid || stringIndex(hrefVal, sid) >= 0 {
							result[sid] = append(result[sid], p)
						}
					}
				}
			}
		}
	}

	return result, nil
}

// stringIndex is provided in deployment_repository.go; reuse the package-level helper.
