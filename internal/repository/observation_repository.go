package repository

import (
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
	return r.db.Create(observation).Error
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
	var total int64

	query := r.db.Model(&domains.Observation{})
	if datastreamID != nil {
		query = query.Where("datastream_id = ?", *datastreamID)
	}
	query = r.applyFilters(query, params, datastreamID != nil)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	err := query.Order("result_time desc").Find(&observations).Error
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
	return r.db.Save(observation).Error
}

func (r *ObservationRepository) Delete(id string) error {
	return r.db.Delete(&domains.Observation{}, "id = ?", id).Error
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

	if params.PhenomenonTime != nil {
		if params.PhenomenonTime.Start != nil && params.PhenomenonTime.End != nil {
			query = query.Where("phenomenon_time <= ? AND phenomenon_time >= ?", params.PhenomenonTime.End, params.PhenomenonTime.Start)
		} else if params.PhenomenonTime.Start != nil {
			query = query.Where("phenomenon_time >= ?", params.PhenomenonTime.Start)
		} else if params.PhenomenonTime.End != nil {
			query = query.Where("phenomenon_time <= ?", params.PhenomenonTime.End)
		}
	}

	if params.ResultTime != nil {
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
