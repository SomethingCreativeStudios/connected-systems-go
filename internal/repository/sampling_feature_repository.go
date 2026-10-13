package repository

import (
	"context"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
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
func (r *SamplingFeatureRepository) Create(ctx context.Context, sf *domains.SamplingFeature) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if sf.SampledFeatureLink != nil {
			var err error
			sf.SampledFeatureID, err = resolveFOILink(ctx, tx, sf.SampledFeatureLink)
			if err != nil {
				return err
			}
		}
		if sf.ParentSystemID != nil {
			if err := requireParent(tx, &domains.System{}, "system", *sf.ParentSystemID); err != nil {
				return err
			}
		}
		return tx.Create(sf).Error
	})
}

// GetByID retrieves a sampling feature by ID
func (r *SamplingFeatureRepository) GetByID(ctx context.Context, id string) (*domains.SamplingFeature, error) {
	var sf domains.SamplingFeature
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&sf).Error
	if err != nil {
		return nil, err
	}
	return &sf, nil
}

// GetByIDs returns sampling features keyed by ID (batch lookup)
func (r *SamplingFeatureRepository) GetByIDs(ctx context.Context, ids []string) (map[string]*domains.SamplingFeature, error) {
	result := make(map[string]*domains.SamplingFeature)
	if len(ids) == 0 {
		return result, nil
	}

	var sfs []*domains.SamplingFeature
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&sfs).Error; err != nil {
		return nil, err
	}

	for _, sf := range sfs {
		if sf != nil {
			result[sf.ID] = sf
		}
	}

	return result, nil
}

// List retrieves sampling features with filtering
func (r *SamplingFeatureRepository) List(ctx context.Context, params *queryparams.SamplingFeatureQueryParams) ([]*domains.SamplingFeature, int64, error) {
	return r.ListSystem(ctx, params, nil)
}

// List retrieves sampling features with filtering
func (r *SamplingFeatureRepository) ListSystem(ctx context.Context, params *queryparams.SamplingFeatureQueryParams, systemID *string) ([]*domains.SamplingFeature, int64, error) {
	var features []*domains.SamplingFeature

	query := r.db.WithContext(ctx).Model(&domains.SamplingFeature{})
	query = r.applyFilters(query, params, systemID)

	if params.DateTime != nil && params.DateTime.Latest {
		err := query.Order("valid_time_start desc").Limit(1).Find(&features).Error
		return features, int64(len(features)), err
	}

	return listPage(query, &params.QueryParams, idOrder("sampling_features"), idAnchor[*domains.SamplingFeature])
}

// Update replaces an existing sampling feature. It returns ErrNotFound rather
// than creating one, and keeps the parent system, which is only ever set
// through /systems/{id}/samplingFeatures.
func (r *SamplingFeatureRepository) Update(ctx context.Context, sf *domains.SamplingFeature) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return replaceByIDChecked(tx, sf, func(existing *domains.SamplingFeature) error {
			if sf.SampledFeatureLink != nil {
				var err error
				sf.SampledFeatureID, err = resolveFOILink(ctx, tx, sf.SampledFeatureLink)
				if err != nil {
					return err
				}
			}
			sf.ParentSystemID = existing.ParentSystemID
			sf.ParentSystemUID = existing.ParentSystemUID
			return nil
		})
	})
}

// Delete deletes a sampling feature
func (r *SamplingFeatureRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&domains.SamplingFeature{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SamplingFeatureRepository) applyFilters(query *gorm.DB, params *queryparams.SamplingFeatureQueryParams, systemID *string) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ? OR unique_identifier IN ?", params.IDs, params.IDs)
	}

	query = applyKeywordFilter(query, params.Q, "sampling_features.name", "sampling_features.description")

	query = applyValidTimeFilter(query, "sampling_features", params.DateTime)

	query = applySpatialIntersectionFilters(query, "sampling_features.geometry", params.Bbox, params.Geom)

	if len(params.FOI) > 0 {
		// The feature of interest of a sampling feature is the real-world
		// feature it samples.
		query = query.Where("sampling_features.sampled_feature_id IN ?", params.FOI)
	}

	if systemID != nil {
		query = query.Where("parent_system_id IN ("+systemTreeSQL("?")+")", *systemID)
	}

	return applyCQLFilter(query, &params.QueryParams)
}
