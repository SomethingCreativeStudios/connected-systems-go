package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
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
func (r *FeatureRepository) Create(feature *domains.Feature) error {
	return r.db.Create(feature).Error
}

// GetByID retrieves a feature by ID
func (r *FeatureRepository) GetByID(id string) (*domains.Feature, error) {
	var feature domains.Feature
	err := r.db.Where("id = ?", id).First(&feature).Error
	if err != nil {
		return nil, err
	}
	return &feature, nil
}

// GetByCollectionAndID retrieves a feature by collection ID and feature ID
func (r *FeatureRepository) GetByCollectionAndID(collectionID, featureID string) (*domains.Feature, error) {
	var feature domains.Feature
	err := r.db.Where("collection_id = ? AND id = ?", collectionID, featureID).First(&feature).Error
	if err != nil {
		return nil, err
	}
	return &feature, nil
}

// List retrieves features with filtering
func (r *FeatureRepository) List(params *queryparams.FeatureQueryParams) ([]*domains.Feature, int64, error) {
	var features []*domains.Feature
	var total int64

	query := r.db.Model(&domains.Feature{})
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

// ListByCollection retrieves features in a specific collection with filtering
func (r *FeatureRepository) ListByCollection(collectionID string, params *queryparams.FeatureQueryParams) ([]*domains.Feature, int64, error) {
	var features []*domains.Feature
	var total int64

	query := r.db.Model(&domains.Feature{}).Where("collection_id = ?", collectionID)
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

// Update updates a feature
func (r *FeatureRepository) Update(feature *domains.Feature) error {
	return r.db.Save(feature).Error
}

// Delete deletes a feature
func (r *FeatureRepository) Delete(id string) error {
	return r.db.Delete(&domains.Feature{}, "id = ?", id).Error
}

func (r *FeatureRepository) applyFilters(query *gorm.DB, params *queryparams.FeatureQueryParams) *gorm.DB {
	// Text search
	if len(params.Q) > 0 {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+strings.Join(params.Q, "%")+"%", "%"+strings.Join(params.Q, "%")+"%")
	}

	// Bounding box filter (OGC bbox parameter)
	if params.BBox != nil && len(params.BBox) >= 4 {
		// PostGIS ST_Intersects on geometry JSONB
		// Format: [minLon, minLat, maxLon, maxLat] or [minLon, minLat, minZ, maxLon, maxLat, maxZ]
		minLon, minLat, maxLon, maxLat := params.BBox[0], params.BBox[1], params.BBox[2], params.BBox[3]
		bboxWKT := fmt.Sprintf("POLYGON((%f %f, %f %f, %f %f, %f %f, %f %f))",
			minLon, minLat,
			maxLon, minLat,
			maxLon, maxLat,
			minLon, maxLat,
			minLon, minLat,
		)
		// Assuming geometry is stored as JSONB with GeoJSON structure
		// For production, consider using PostGIS geometry column instead
		query = query.Where("ST_Intersects(ST_GeomFromGeoJSON(geometry::text), ST_GeomFromText(?, 4326))", bboxWKT)
	}

	// DateTime filter (OGC datetime parameter)
	if params.DateTime != nil {
		if params.DateTime.Start != nil && params.DateTime.End != nil {
			// Interval: both start and end
			query = query.Where("date_time >= ? AND date_time <= ?", params.DateTime.Start, params.DateTime.End)
		} else if params.DateTime.Start != nil {
			// Open-ended: start only
			query = query.Where("date_time >= ?", params.DateTime.Start)
		} else if params.DateTime.End != nil {
			// Open-ended: end only
			query = query.Where("date_time <= ?", params.DateTime.End)
		}
	}

	return query
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
