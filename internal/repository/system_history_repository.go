package repository

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// SystemHistoryRepository stores and retrieves historical system revisions.
type SystemHistoryRepository struct {
	db *gorm.DB
}

func NewSystemHistoryRepository(db *gorm.DB) *SystemHistoryRepository {
	return &SystemHistoryRepository{db: db}
}

func (r *SystemHistoryRepository) CreateFromSystem(system *domains.System) (*domains.SystemHistoryRevision, error) {
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

	if err := r.db.Create(rev).Error; err != nil {
		return nil, err
	}
	return rev, nil
}

func (r *SystemHistoryRepository) List(systemID string, params *queryparams.SystemHistoryQueryParams) ([]*domains.SystemHistoryRevision, int64, error) {
	var revisions []*domains.SystemHistoryRevision
	var total int64

	query := r.db.Model(&domains.SystemHistoryRevision{}).Where("system_id = ?", systemID)
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

	err := query.Order("created_at desc").Find(&revisions).Error
	return revisions, total, err
}

func (r *SystemHistoryRepository) GetByID(systemID, revID string) (*domains.SystemHistoryRevision, error) {
	var rev domains.SystemHistoryRevision
	err := r.db.Where("id = ? AND system_id = ?", revID, systemID).First(&rev).Error
	if err != nil {
		return nil, err
	}
	return &rev, nil
}

func (r *SystemHistoryRepository) UpdateSnapshot(systemID, revID string, snapshot *domains.System) error {
	if snapshot == nil {
		return fmt.Errorf("snapshot is nil")
	}

	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"snapshot": payload,
	}
	if snapshot.ValidTime != nil {
		updates["valid_time_start"] = snapshot.ValidTime.Start
		updates["valid_time_end"] = snapshot.ValidTime.End
	}

	return r.db.Model(&domains.SystemHistoryRevision{}).
		Where("id = ? AND system_id = ?", revID, systemID).
		Updates(updates).Error
}

func (r *SystemHistoryRepository) Delete(systemID, revID string) error {
	return r.db.Where("id = ? AND system_id = ?", revID, systemID).Delete(&domains.SystemHistoryRevision{}).Error
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

	if params.ValidTime != nil {
		if params.ValidTime.Start != nil && params.ValidTime.End != nil {
			query = query.Where("valid_time_start <= ? AND (valid_time_end IS NULL OR valid_time_end >= ?)", params.ValidTime.End, params.ValidTime.Start)
		} else if params.ValidTime.Start != nil {
			query = query.Where("valid_time_end IS NULL OR valid_time_end >= ?", params.ValidTime.Start)
		} else if params.ValidTime.End != nil {
			query = query.Where("valid_time_start <= ?", params.ValidTime.End)
		}
	}

	if len(params.Keyword) > 0 || len(params.Q) > 0 {
		terms := append([]string{}, params.Keyword...)
		terms = append(terms, params.Q...)
		var clauses []string
		var args []interface{}
		for _, term := range terms {
			like := "%" + term + "%"
			clauses = append(clauses, "CAST(snapshot AS text) ILIKE ?")
			args = append(args, like)
		}
		query = query.Where(strings.Join(clauses, " OR "), args...)
	}

	return query
}
