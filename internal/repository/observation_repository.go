package repository

import (
	"context"
	"errors"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/resourcevalidation"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ObservationRepository handles Observation data access.
type ObservationRepository struct {
	db *gorm.DB
}

func NewObservationRepository(db *gorm.DB) *ObservationRepository {
	return &ObservationRepository{db: db}
}

func (r *ObservationRepository) Create(ctx context.Context, observation *domains.Observation) error {
	return r.CreateWithFormat(ctx, observation, "application/json")
}

func (r *ObservationRepository) CreateWithFormat(ctx context.Context, observation *domains.Observation, contentType string) error {
	// Default resultTime first: phenomenonTime falls back to it.
	if observation.ResultTime.IsZero() {
		observation.ResultTime = time.Now().UTC()
	}
	if observation.PhenomenonTime == nil {
		t := observation.ResultTime
		observation.PhenomenonTime = &t
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := validateLockedObservation(tx, observation, contentType); err != nil {
			return err
		}
		if err := tx.Create(observation).Error; err != nil {
			return err
		}
		return expandDatastreamTimeRanges(tx, observation.DatastreamID, *observation.PhenomenonTime, observation.ResultTime)
	})
}

// expandDatastreamTimeRanges widens the parent datastream's phenomenonTime and
// resultTime extents to include a newly written observation. LEAST/GREATEST
// ignore NULL columns, so this also seeds initially-empty ranges.
func expandDatastreamTimeRanges(tx *gorm.DB, datastreamID string, phenomenonTime, resultTime time.Time) error {
	return tx.Exec(`UPDATE datastreams SET
			phenomenon_time_start = LEAST(phenomenon_time_start, ?),
			phenomenon_time_end   = GREATEST(phenomenon_time_end, ?),
			result_time_start     = LEAST(result_time_start, ?),
			result_time_end       = GREATEST(result_time_end, ?)
		WHERE id = ?`,
		phenomenonTime, phenomenonTime, resultTime, resultTime, datastreamID).Error
}

// recomputeDatastreamTimeRanges recalculates the parent datastream's time
// extents from scratch after an observation update or delete (either can move
// a min/max inward). With no observations left the extents go NULL.
func recomputeDatastreamTimeRanges(tx *gorm.DB, datastreamID string) error {
	return tx.Exec(`UPDATE datastreams SET
			phenomenon_time_start = agg.pt_min,
			phenomenon_time_end   = agg.pt_max,
			result_time_start     = agg.rt_min,
			result_time_end       = agg.rt_max
		FROM (
			SELECT MIN(COALESCE(phenomenon_time, result_time)) AS pt_min,
			       MAX(COALESCE(phenomenon_time, result_time)) AS pt_max,
			       MIN(result_time) AS rt_min,
			       MAX(result_time) AS rt_max
			FROM observations WHERE datastream_id = ?
		) agg
		WHERE datastreams.id = ?`,
		datastreamID, datastreamID).Error
}

func (r *ObservationRepository) GetByID(ctx context.Context, id string) (*domains.Observation, error) {
	var observation domains.Observation
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&observation).Error
	if err != nil {
		return nil, err
	}
	return &observation, nil
}

func (r *ObservationRepository) List(ctx context.Context, params *queryparams.ObservationsQueryParams, datastreamID *string) ([]*domains.Observation, int64, error) {
	var observations []*domains.Observation

	query := r.db.WithContext(ctx).Model(&domains.Observation{})
	if datastreamID != nil {
		query = query.Where("datastream_id = ?", *datastreamID)
	}
	query = r.applyFilters(query, params, datastreamID != nil)

	// "latest" mode: return the single most-recent observation ordered by the
	// requested time column, bypassing normal pagination.
	latestResultTime := params.ResultTime != nil && params.ResultTime.Latest
	latestPhenomenonTime := params.PhenomenonTime != nil && params.PhenomenonTime.Latest
	if latestResultTime || latestPhenomenonTime {
		orderCol := "observations.result_time"
		if latestPhenomenonTime {
			orderCol = "observations.phenomenon_time"
		}
		err := query.Order(orderCol + " desc").Limit(1).Find(&observations).Error
		return observations, int64(len(observations)), err
	}

	return listPage(query, &params.QueryParams, CursorOrder{
		Columns:     []string{"observations.result_time", "observations.id"},
		Descending:  true,
		TimeColumns: map[int]bool{0: true},
	}, func(observation *domains.Observation) []string {
		return []string{queryparams.TimeCursorValue(observation.ResultTime), observation.ID}
	})
}

func (r *ObservationRepository) ListByDatastream(ctx context.Context, datastreamID string, params *queryparams.ObservationsQueryParams) ([]*domains.Observation, int64, error) {
	return r.List(ctx, params, &datastreamID)
}

func (r *ObservationRepository) Update(ctx context.Context, observation *domains.Observation) error {
	return r.UpdateWithFormat(ctx, observation, "application/json")
}

func (r *ObservationRepository) UpdateWithFormat(ctx context.Context, observation *domains.Observation, contentType string) error {
	if observation.PhenomenonTime == nil {
		t := observation.ResultTime
		observation.PhenomenonTime = &t
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domains.Observation
		if err := tx.Where("id = ?", observation.ID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		// Lock the parent before touching the child, matching schema writes.
		observation.DatastreamID = existing.DatastreamID
		if err := validateLockedObservation(tx, observation, contentType); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", observation.ID).First(&existing).Error; err != nil {
			return err
		}

		// Preserve lifecycle fields across a replacement. result_time is part of
		// the composite primary key, so moving an observation to another chunk
		// must delete the old tuple and insert the replacement atomically.
		observation.CreatedAt = existing.CreatedAt
		observation.UpdatedAt = time.Now().UTC()
		if !observation.ResultTime.Equal(existing.ResultTime) {
			if err := tx.Where("id = ? AND result_time = ?", existing.ID, existing.ResultTime).
				Delete(&domains.Observation{}).Error; err != nil {
				return err
			}
			if err := tx.Create(observation).Error; err != nil {
				return err
			}
		} else if err := updateObservationRow(tx, observation); err != nil {
			return err
		}

		if existing.DatastreamID != observation.DatastreamID {
			if err := recomputeDatastreamTimeRanges(tx, existing.DatastreamID); err != nil {
				return err
			}
		}
		return recomputeDatastreamTimeRanges(tx, observation.DatastreamID)
	})
}

func updateObservationRow(tx *gorm.DB, observation *domains.Observation) error {
	// Target the stable API ID and original time tuple rather than using Save,
	// which would otherwise treat the composite replacement key as a new row.
	updates := map[string]any{
		"datastream_id":       observation.DatastreamID,
		"sampling_feature_id": observation.SamplingFeatureID,
		"procedure_link":      observation.ProcedureLink,
		"phenomenon_time":     observation.PhenomenonTime,
		"result_time":         observation.ResultTime,
		"parameters":          observation.Parameters,
		"result":              observation.Result,
		"result_link":         observation.ResultLink,
		"updated_at":          time.Now().UTC(),
	}
	return tx.Model(&domains.Observation{}).
		Where("id = ? AND result_time = ?", observation.ID, observation.ResultTime).
		Updates(updates).Error
}

func (r *ObservationRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var datastreamID string
		err := tx.Model(&domains.Observation{}).Select("datastream_id").
			Where("id = ?", id).First(&datastreamID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		var parent domains.Datastream
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id = ?", datastreamID).First(&parent).Error; err != nil {
			return err
		}
		if err := tx.Delete(&domains.Observation{}, "id = ?", id).Error; err != nil {
			return err
		}
		return recomputeDatastreamTimeRanges(tx, datastreamID)
	})
}

func (r *ObservationRepository) applyFilters(query *gorm.DB, params *queryparams.ObservationsQueryParams, datastreamFixed bool) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("observations.id IN ?", params.IDs)
	}

	if !datastreamFixed && len(params.DataStream) > 0 {
		query = query.Where("observations.datastream_id IN ?", params.DataStream)
	}

	// Datastream-level filters are semi-joins so they can use the
	// observations.datastream_id index and never clash with datastream columns.
	if len(params.System) > 0 {
		query = query.Where("observations.datastream_id IN (SELECT id FROM datastreams WHERE system_id IN ?)", params.System)
	}

	query = applyInstantFilter(query, "observations.phenomenon_time", params.PhenomenonTime)

	query = applyInstantFilter(query, "observations.result_time", params.ResultTime)

	if len(params.FOI) > 0 {
		query = query.Where("observations.sampling_feature_id IN ?", params.FOI)
	}

	if len(params.ObservedProperty) > 0 {
		match, args := ilikeAll("observed_properties::text", params.ObservedProperty)
		query = query.Where("observations.datastream_id IN (SELECT id FROM datastreams WHERE "+match+")", args...)
	}

	query = applyKeywordFilter(query, params.Q, "CAST(observations.parameters AS text)", "CAST(observations.result AS text)")

	return applyCQLFilter(query, &params.QueryParams)
}

func validateLockedObservation(tx *gorm.DB, observation *domains.Observation, contentType string) error {
	var parent domains.Datastream
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", observation.DatastreamID).First(&parent).Error; err != nil {
		return err
	}
	if err := resourcevalidation.ValidateObservationAgainstDatastreamSchema(observation, &parent, contentType); err != nil {
		return &ValidationError{Err: err}
	}
	return nil
}

func lockDatastream(tx *gorm.DB, id string) error {
	var stream domains.Datastream
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id = ?", id).First(&stream).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
