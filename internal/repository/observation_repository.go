package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// ObservationRepository handles Observation data access.
type ObservationRepository struct {
	db *gorm.DB
}

func NewObservationRepository(db *gorm.DB) *ObservationRepository {
	return &ObservationRepository{db: db}
}

func (r *ObservationRepository) Create(observation *domains.Observation) error {
	if observation.PhenomenonTime == nil {
		t := observation.ResultTime
		observation.PhenomenonTime = &t
	}
	if observation.ResultTime.IsZero() {
		now := time.Now().UTC()
		observation.ResultTime = now
		if observation.PhenomenonTime == nil {
			observation.PhenomenonTime = &now
		}
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
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

func (r *ObservationRepository) GetByID(id string) (*domains.Observation, error) {
	var observation domains.Observation
	err := r.db.Where("id = ?", id).First(&observation).Error
	if err != nil {
		return nil, err
	}
	return &observation, nil
}

func (r *ObservationRepository) List(params *queryparams.ObservationsQueryParams, datastreamID *string) ([]*domains.Observation, int64, error) {
	var observations []*domains.Observation

	query := r.db.Model(&domains.Observation{})
	if datastreamID != nil {
		query = query.Where("datastream_id = ?", *datastreamID)
	}
	query = r.applyFilters(query, params, datastreamID != nil)

	// "latest" mode: return the single most-recent observation ordered by the
	// requested time column, bypassing normal pagination.
	latestResultTime := params.ResultTime != nil && params.ResultTime.Latest
	latestPhenomenonTime := params.PhenomenonTime != nil && params.PhenomenonTime.Latest
	if latestResultTime || latestPhenomenonTime {
		orderCol := "result_time"
		if latestPhenomenonTime {
			orderCol = "phenomenon_time"
		}
		err := query.Order(orderCol + " desc").Limit(1).Find(&observations).Error
		return observations, int64(len(observations)), err
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query, err := ApplyCursorPagination(query, &params.QueryParams, CursorOrder{
		Columns:     []string{"result_time", "id"},
		Descending:  true,
		TimeColumns: map[int]bool{0: true},
	})
	if err != nil {
		return nil, 0, err
	}
	err = query.Find(&observations).Error
	observations = FinalizeCursorPage(observations, &params.QueryParams)
	params.Anchors = queryparams.CursorAnchorsFor(observations, func(observation *domains.Observation) []string {
		return []string{queryparams.TimeCursorValue(observation.ResultTime), observation.ID}
	})
	return observations, total, err
}

func (r *ObservationRepository) ListByDatastream(datastreamID string, params *queryparams.ObservationsQueryParams) ([]*domains.Observation, int64, error) {
	return r.List(params, &datastreamID)
}

func (r *ObservationRepository) Update(observation *domains.Observation) error {
	if observation.PhenomenonTime == nil {
		t := observation.ResultTime
		observation.PhenomenonTime = &t
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing domains.Observation
		if err := tx.Where("id = ?", observation.ID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
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

func (r *ObservationRepository) Delete(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var datastreamID string
		err := tx.Model(&domains.Observation{}).Select("datastream_id").
			Where("id = ?", id).First(&datastreamID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := tx.Delete(&domains.Observation{}, "id = ?", id).Error; err != nil {
			return err
		}
		return recomputeDatastreamTimeRanges(tx, datastreamID)
	})
}

func (r *ObservationRepository) applyFilters(query *gorm.DB, params *queryparams.ObservationsQueryParams, datastreamFixed bool) *gorm.DB {
	joinedDatastreams := false
	joinDatastreams := func(q *gorm.DB) *gorm.DB {
		if !joinedDatastreams {
			q = q.Joins("JOIN datastreams ON observations.datastream_id = datastreams.id")
			joinedDatastreams = true
		}
		return q
	}

	if len(params.IDs) > 0 {
		query = query.Where("id IN ?", params.IDs)
	}

	if !datastreamFixed && len(params.DataStream) > 0 {
		query = query.Where("datastream_id IN ?", params.DataStream)
	}

	if len(params.System) > 0 {
		query = joinDatastreams(query)
		query = query.Where("datastreams.system_id IN ?", params.System)
	}

	if params.PhenomenonTime != nil && !params.PhenomenonTime.Latest {
		if params.PhenomenonTime.Start != nil && params.PhenomenonTime.End != nil {
			query = query.Where("phenomenon_time <= ? AND phenomenon_time >= ?", params.PhenomenonTime.End, params.PhenomenonTime.Start)
		} else if params.PhenomenonTime.Start != nil {
			query = query.Where("phenomenon_time >= ?", params.PhenomenonTime.Start)
		} else if params.PhenomenonTime.End != nil {
			query = query.Where("phenomenon_time <= ?", params.PhenomenonTime.End)
		}
	}

	if params.ResultTime != nil && !params.ResultTime.Latest {
		if params.ResultTime.Start != nil && params.ResultTime.End != nil {
			query = query.Where("result_time <= ? AND result_time >= ?", params.ResultTime.End, params.ResultTime.Start)
		} else if params.ResultTime.Start != nil {
			query = query.Where("result_time >= ?", params.ResultTime.Start)
		} else if params.ResultTime.End != nil {
			query = query.Where("result_time <= ?", params.ResultTime.End)
		}
	}

	if len(params.FOI) > 0 {
		query = query.Where("sampling_feature_id IN ?", params.FOI)
	}

	if len(params.ObservedProperty) > 0 {
		query = joinDatastreams(query)
		for _, observedProperty := range params.ObservedProperty {
			query = query.Where("datastreams.observed_properties::text ILIKE ?", "%"+observedProperty+"%")
		}
	}

	if len(params.Q) > 0 {
		var clauses []string
		var args []interface{}
		for _, term := range params.Q {
			like := "%" + term + "%"
			clauses = append(clauses, "CAST(parameters AS text) ILIKE ?")
			args = append(args, like)
			clauses = append(clauses, "CAST(result AS text) ILIKE ?")
			args = append(args, like)
		}
		query = query.Where(strings.Join(clauses, " OR "), args...)
	}

	return query
}
