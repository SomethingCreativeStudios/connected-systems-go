package repository

import (
	"context"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
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
func (r *SamplingFeatureRepository) Create(sf *domains.SamplingFeature) error {
	return r.db.Create(sf).Error
}

// GetByID retrieves a sampling feature by ID
func (r *SamplingFeatureRepository) GetByID(id string) (*domains.SamplingFeature, error) {
	var sf domains.SamplingFeature
	err := r.db.Where("id = ?", id).First(&sf).Error
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
func (r *SamplingFeatureRepository) List(params *queryparams.SamplingFeatureQueryParams) ([]*domains.SamplingFeature, int64, error) {
	return r.ListSystem(params, nil)
}

// List retrieves sampling features with filtering
func (r *SamplingFeatureRepository) ListSystem(params *queryparams.SamplingFeatureQueryParams, systemID *string) ([]*domains.SamplingFeature, int64, error) {
	var features []*domains.SamplingFeature
	var total int64

	query := r.db.Model(&domains.SamplingFeature{})
	query = r.applyFilters(query, params, systemID)

	if params.DateTime != nil && params.DateTime.Latest {
		err := query.Order("valid_time_start desc").Limit(1).Find(&features).Error
		return features, int64(len(features)), err
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query, err := ApplyCursorPagination(query, &params.QueryParams, CursorOrderIDAsc)
	if err != nil {
		return nil, 0, err
	}
	err = query.Find(&features).Error
	features = FinalizeCursorPage(features, &params.QueryParams)
	params.Anchors = queryparams.CursorAnchorsFor(features, func(feature *domains.SamplingFeature) []string { return []string{feature.ID} })
	return features, total, err
}

// Update updates a sampling feature
func (r *SamplingFeatureRepository) Update(sf *domains.SamplingFeature) error {
	return r.db.Save(sf).Error
}

// Delete deletes a sampling feature
func (r *SamplingFeatureRepository) Delete(id string) error {
	result := r.db.Delete(&domains.SamplingFeature{}, "id = ?", id)
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

	if len(params.Q) > 0 {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+strings.Join(params.Q, "%")+"%", "%"+strings.Join(params.Q, "%")+"%")
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

	if params.Bbox != nil {
		query = query.Where("ST_Intersects(geometry, ST_MakeEnvelope(?, ?, ?, ?, 4326))", params.Bbox.MinX, params.Bbox.MinY, params.Bbox.MaxX, params.Bbox.MaxY)
	}

	if params.Geom != "" {
		query = query.Where("ST_Intersects(geometry, ST_GeomFromText(?, 4326))", params.Geom)
	}

	if len(params.FOI) > 0 {
		query = query.Joins("JOIN sampling_feature_fois sff ON sff.sampling_feature_id = sampling_features.id").
			Where("sff.foi_id IN ?", params.FOI)
	}

	if systemID != nil {
		query = query.Where("parent_system_id = ?", *systemID)
	}

	return query
}
