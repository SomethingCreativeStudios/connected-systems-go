package repository

import (
	"strings"
	"time"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// SystemEventRepository handles system event data access.
type SystemEventRepository struct {
	db *gorm.DB
}

func NewSystemEventRepository(db *gorm.DB) *SystemEventRepository {
	return &SystemEventRepository{db: db}
}

func (r *SystemEventRepository) Create(event *domains.SystemEvent) error {
	normalizeSystemEventTime(event)
	return r.db.Create(event).Error
}

func (r *SystemEventRepository) GetByID(systemID, eventID string) (*domains.SystemEvent, error) {
	var event domains.SystemEvent
	err := r.db.Where("id = ? AND system_id = ?", eventID, systemID).First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *SystemEventRepository) List(params *queryparams.SystemEventsQueryParams, fixedSystemID *string) ([]*domains.SystemEvent, int64, error) {
	var events []*domains.SystemEvent
	var total int64

	query := r.db.Model(&domains.SystemEvent{})
	query = r.applyFilters(query, params, fixedSystemID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	err := query.Order("time_start desc, created_at desc").Find(&events).Error
	return events, total, err
}

func (r *SystemEventRepository) Update(event *domains.SystemEvent) error {
	normalizeSystemEventTime(event)
	return r.db.Save(event).Error
}

func (r *SystemEventRepository) Delete(systemID, eventID string) error {
	return r.db.Where("id = ? AND system_id = ?", eventID, systemID).Delete(&domains.SystemEvent{}).Error
}

func (r *SystemEventRepository) applyFilters(query *gorm.DB, params *queryparams.SystemEventsQueryParams, fixedSystemID *string) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ?", params.IDs)
	}

	if fixedSystemID != nil {
		query = query.Where("system_id = ?", *fixedSystemID)
	} else if len(params.System) > 0 {
		query = query.Where("system_id IN ?", params.System)
	}

	if len(params.EventType) > 0 {
		query = query.Where("definition IN ?", params.EventType)
	}

	if params.EventTime != nil {
		if params.EventTime.Start != nil && params.EventTime.End != nil {
			query = query.Where("time_start <= ? AND (time_end IS NULL OR time_end >= ?)", params.EventTime.End, params.EventTime.Start)
		} else if params.EventTime.Start != nil {
			query = query.Where("time_end IS NULL OR time_end >= ?", params.EventTime.Start)
		} else if params.EventTime.End != nil {
			query = query.Where("time_start <= ?", params.EventTime.End)
		}
	}

	if len(params.Keyword) > 0 || len(params.Q) > 0 {
		terms := append([]string{}, params.Keyword...)
		terms = append(terms, params.Q...)
		var clauses []string
		var args []interface{}
		for _, term := range terms {
			like := "%" + term + "%"
			clauses = append(clauses, "label ILIKE ?")
			args = append(args, like)
			clauses = append(clauses, "description ILIKE ?")
			args = append(args, like)
			clauses = append(clauses, "definition ILIKE ?")
			args = append(args, like)
			clauses = append(clauses, "CAST(properties AS text) ILIKE ?")
			args = append(args, like)
		}
		query = query.Where(strings.Join(clauses, " OR "), args...)
	}

	return query
}

func normalizeSystemEventTime(event *domains.SystemEvent) {
	if event == nil {
		return
	}

	if event.Time.Instant != nil {
		t := event.Time.Instant.UTC()
		event.TimeStart = &t
		event.TimeEnd = &t
		return
	}

	if event.Time.Range != nil {
		if event.Time.Range.Start != nil {
			t := event.Time.Range.Start.UTC()
			event.TimeStart = &t
		} else {
			event.TimeStart = nil
		}
		if event.Time.Range.End != nil {
			t := event.Time.Range.End.UTC()
			event.TimeEnd = &t
		} else {
			event.TimeEnd = nil
		}
		return
	}

	now := time.Now().UTC()
	event.Time.Instant = &now
	event.Time.Range = nil
	event.TimeStart = &now
	event.TimeEnd = &now
}
