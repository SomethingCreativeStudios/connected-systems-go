package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// SystemHistoryRepository stores and retrieves historical system revisions.
type SystemHistoryRepository struct {
	db *gorm.DB
}

func NewSystemHistoryRepository(db *gorm.DB) *SystemHistoryRepository {
	return &SystemHistoryRepository{db: db}
}

func (r *SystemHistoryRepository) CreateFromSystem(ctx context.Context, system *domains.System) (*domains.SystemHistoryRevision, error) {
	if system == nil {
		return nil, fmt.Errorf("system is nil")
	}

	payload, err := json.Marshal(system)
	if err != nil {
		return nil, err
	}

	rev := &domains.SystemHistoryRevision{
		SystemID:  system.ID,
		Snapshot:  payload,
		ValidTime: system.ValidTime,
	}

	if err := r.db.WithContext(ctx).Create(rev).Error; err != nil {
		return nil, err
	}
	return rev, nil
}

func (r *SystemHistoryRepository) List(ctx context.Context, systemID string, params *queryparams.SystemHistoryQueryParams) ([]*domains.SystemHistoryRevision, int64, error) {
	var revisions []*domains.SystemHistoryRevision

	query := r.db.WithContext(ctx).Model(&domains.SystemHistoryRevision{}).Where("system_id = ?", systemID)
	query = r.applyFilters(query, params)

	if params.ValidTime != nil && params.ValidTime.Latest {
		err := query.Order("valid_time_start desc").Limit(1).Find(&revisions).Error
		return revisions, int64(len(revisions)), err
	}

	return listPage(query, &params.QueryParams, CursorOrder{
		Columns:     []string{"created_at", "id"},
		Descending:  true,
		TimeColumns: map[int]bool{0: true},
	}, func(revision *domains.SystemHistoryRevision) []string {
		return []string{queryparams.TimeCursorValue(revision.CreatedAt), revision.ID}
	})
}

func (r *SystemHistoryRepository) GetByID(ctx context.Context, systemID, revID string) (*domains.SystemHistoryRevision, error) {
	var rev domains.SystemHistoryRevision
	err := r.db.WithContext(ctx).Where("id = ? AND system_id = ?", revID, systemID).First(&rev).Error
	if err != nil {
		return nil, err
	}
	return &rev, nil
}

func (r *SystemHistoryRepository) UpdateSnapshot(ctx context.Context, systemID, revID string, snapshot *domains.System) error {
	if snapshot == nil {
		return fmt.Errorf("snapshot is nil")
	}

	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	updates := map[string]any{
		"snapshot": payload,
	}
	if snapshot.ValidTime != nil {
		updates["valid_time_start"] = snapshot.ValidTime.Start
		updates["valid_time_end"] = snapshot.ValidTime.End
	}

	return r.db.WithContext(ctx).Model(&domains.SystemHistoryRevision{}).
		Where("id = ? AND system_id = ?", revID, systemID).
		Updates(updates).Error
}

func (r *SystemHistoryRepository) Delete(ctx context.Context, systemID, revID string) error {
	return r.db.WithContext(ctx).Where("id = ? AND system_id = ?", revID, systemID).Delete(&domains.SystemHistoryRevision{}).Error
}

func (r *SystemHistoryRepository) DecodeRevisionSystem(rev *domains.SystemHistoryRevision) (*domains.System, error) {
	if rev == nil {
		return nil, fmt.Errorf("revision is nil")
	}

	var system domains.System
	if err := json.Unmarshal(rev.Snapshot, &system); err != nil {
		return nil, err
	}

	// For history resources, id acts as the revision identifier.
	system.ID = rev.ID
	return &system, nil
}

func (r *SystemHistoryRepository) applyFilters(query *gorm.DB, params *queryparams.SystemHistoryQueryParams) *gorm.DB {
	if params == nil {
		return query
	}

	query = applyValidTimeFilter(query, "system_history_revisions", params.ValidTime)

	query = applyKeywordFilter(query, append(append([]string{}, params.Keyword...), params.Q...), "CAST(snapshot AS text)")

	return applyCQLFilter(query, &params.QueryParams)
}
