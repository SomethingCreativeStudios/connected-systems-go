package repository

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
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
func (r *PropertyRepository) Create(ctx context.Context, property *domains.Property) error {
	return r.db.WithContext(ctx).Create(property).Error
}

// GetByID retrieves a property by ID
func (r *PropertyRepository) GetByID(ctx context.Context, id string) (*domains.Property, error) {
	var property domains.Property
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&property).Error
	if err != nil {
		return nil, err
	}
	return &property, nil
}

// List retrieves properties with filtering
func (r *PropertyRepository) List(ctx context.Context, params *queryparams.PropertiesQueryParams) ([]*domains.Property, int64, error) {

	query := r.db.WithContext(ctx).Model(&domains.Property{})
	query = r.applyFilters(query, params)

	return listPage(query, &params.QueryParams, idOrder("properties"), idAnchor[*domains.Property])
}

// Update updates a property
func (r *PropertyRepository) Update(ctx context.Context, property *domains.Property) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return replaceByID(tx, property, nil)
	})
}

// Delete deletes a property
func (r *PropertyRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&domains.Property{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PropertyRepository) applyFilters(query *gorm.DB, params *queryparams.PropertiesQueryParams) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ? OR unique_identifier IN ?", params.IDs, params.IDs)
	}

	query = applyKeywordFilter(query, params.Q, "name", "description", "base_property", "object_type", "property_type")

	if len(params.ObjectType) > 0 {
		query = query.Where("object_type IN ?", params.ObjectType)
	}

	if len(params.BaseProperty) > 0 {
		query = query.Where("base_property IN ?", params.BaseProperty)
	}

	return applyCQLFilter(query, &params.QueryParams)
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
			var arr []map[string]any
			if err := json.Unmarshal(b, &arr); err != nil {
				continue
			}
			for _, el := range arr {
				if hrefVal, ok := el["href"].(string); ok {
					for _, sid := range systemIDs {
						if strings.Contains(hrefVal, sid) {
							result[sid] = append(result[sid], p)
						}
					}
				}
			}
		}
	}

	return result, nil
}
