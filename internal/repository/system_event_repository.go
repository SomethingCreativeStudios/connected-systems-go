package repository

import (
	"context"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// SystemEventRepository handles system event data access.
type SystemEventRepository struct {
	db *gorm.DB
}

func NewSystemEventRepository(db *gorm.DB) *SystemEventRepository {
	return &SystemEventRepository{db: db}
}

func (r *SystemEventRepository) Create(ctx context.Context, event *domains.SystemEvent) error {
	normalizeSystemEventTime(event)
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *SystemEventRepository) GetByID(ctx context.Context, systemID, eventID string) (*domains.SystemEvent, error) {
	var event domains.SystemEvent
	err := r.db.WithContext(ctx).Where("id = ? AND system_id = ?", eventID, systemID).First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *SystemEventRepository) List(ctx context.Context, params *queryparams.SystemEventsQueryParams, fixedSystemID *string) ([]*domains.SystemEvent, int64, error) {
	var events []*domains.SystemEvent

	query := r.db.WithContext(ctx).Model(&domains.SystemEvent{})
	query = r.applyFilters(query, params, fixedSystemID)

	if params.EventTime != nil && params.EventTime.Latest {
		err := query.Order("time_start desc").Limit(1).Find(&events).Error
		return events, int64(len(events)), err
	}

	return listPage(query, &params.QueryParams, CursorOrder{
		Columns:     []string{"time_start", "created_at", "id"},
		Descending:  true,
		TimeColumns: map[int]bool{0: true, 1: true},
	}, func(event *domains.SystemEvent) []string {
		return []string{cursorTimeValue(event.TimeStart), queryparams.TimeCursorValue(event.CreatedAt), event.ID}
	})
}

func (r *SystemEventRepository) Update(ctx context.Context, event *domains.SystemEvent) error {
	normalizeSystemEventTime(event)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return replaceByID(tx, event, func(existing *domains.SystemEvent) {
			event.SystemID = existing.SystemID
		})
	})
}

func (r *SystemEventRepository) Delete(ctx context.Context, systemID, eventID string) error {
	return r.db.WithContext(ctx).Where("id = ? AND system_id = ?", eventID, systemID).Delete(&domains.SystemEvent{}).Error
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

	query = applyRangeFilter(query, "system_events.time_start", "system_events.time_end", params.EventTime)

	query = applyKeywordFilter(query, append(append([]string{}, params.Keyword...), params.Q...), "label", "description", "definition", "CAST(properties AS text)")

	return applyCQLFilter(query, &params.QueryParams)
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
