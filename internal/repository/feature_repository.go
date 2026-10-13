package repository

import (
	"context"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// FeatureRepository handles Feature data access
type FeatureRepository struct {
	db *gorm.DB
}

// NewFeatureRepository creates a new FeatureRepository
func NewFeatureRepository(db *gorm.DB) *FeatureRepository {
	return &FeatureRepository{db: db}
}

// Create creates a new feature
func (r *FeatureRepository) Create(ctx context.Context, feature *domains.Feature) error {
	return r.db.WithContext(ctx).Create(feature).Error
}

// GetByID retrieves a feature by ID
func (r *FeatureRepository) GetByID(ctx context.Context, id string) (*domains.Feature, error) {
	var feature domains.Feature
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&feature).Error
	if err != nil {
		return nil, err
	}
	return &feature, nil
}

// GetByCollectionAndID retrieves a feature by collection ID and feature ID
func (r *FeatureRepository) GetByCollectionAndID(ctx context.Context, collectionID, featureID string) (*domains.Feature, error) {
	var feature domains.Feature
	err := r.db.WithContext(ctx).Where("collection_id = ? AND id = ?", collectionID, featureID).First(&feature).Error
	if err != nil {
		return nil, err
	}
	return &feature, nil
}

// List retrieves features with filtering
func (r *FeatureRepository) List(ctx context.Context, params *queryparams.FeatureQueryParams) ([]*domains.Feature, int64, error) {

	query := r.db.WithContext(ctx).Model(&domains.Feature{})
	query = r.applyFilters(query, params)

	return listPage(query, &params.QueryParams, idOrder("features"), idAnchor[*domains.Feature])
}

// ListByCollection retrieves features in a specific collection with filtering
func (r *FeatureRepository) ListByCollection(ctx context.Context, collectionID string, params *queryparams.FeatureQueryParams) ([]*domains.Feature, int64, error) {

	query := r.db.WithContext(ctx).Model(&domains.Feature{}).Where("collection_id = ?", collectionID)
	query = r.applyFilters(query, params)

	return listPage(query, &params.QueryParams, idOrder("features"), idAnchor[*domains.Feature])
}

// Update updates a feature
func (r *FeatureRepository) Update(ctx context.Context, feature *domains.Feature) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return replaceByID(tx, feature, func(existing *domains.Feature) {
			feature.CollectionID = existing.CollectionID
		})
	})
}

// Delete deletes a feature
func (r *FeatureRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domains.Feature{}, "id = ?", id).Error
}

func (r *FeatureRepository) applyFilters(query *gorm.DB, params *queryparams.FeatureQueryParams) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ? OR unique_identifier IN ?", params.IDs, params.IDs)
	}

	// Text search
	query = applyKeywordFilter(query, params.Q, "name", "description")

	query = applySpatialIntersectionFilters(query, "features.geometry", params.BBox, "")

	// DateTime filter (OGC datetime parameter)
	query = applyInstantFilter(query, "features.date_time", params.DateTime)

	return applyCQLFilter(query, &params.QueryParams)
}

// GetByIDs returns features keyed by ID (batch lookup)
func (r *FeatureRepository) GetByIDs(ctx context.Context, ids []string) (map[string]*domains.Feature, error) {
	result := make(map[string]*domains.Feature)
	if len(ids) == 0 {
		return result, nil
	}

	var features []*domains.Feature
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&features).Error; err != nil {
		return nil, err
	}

	for _, f := range features {
		if f != nil {
			result[f.ID] = f
		}
	}

	return result, nil
}
