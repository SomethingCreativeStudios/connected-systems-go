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

// CommandRepository handles Command data access.
type CommandRepository struct {
	db *gorm.DB
}

// NewCommandRepository creates a new CommandRepository.
func NewCommandRepository(db *gorm.DB) *CommandRepository {
	return &CommandRepository{db: db}
}

// Create persists a new command. IssueTime is set to now if omitted.
func (r *CommandRepository) Create(ctx context.Context, cmd *domains.Command) error {
	if cmd.IssueTime == nil {
		now := time.Now().UTC()
		cmd.IssueTime = &now
	}
	if cmd.CurrentStatus == "" {
		cmd.CurrentStatus = domains.CommandStatusPending
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var parent domains.ControlStream
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", cmd.ControlStreamID).First(&parent).Error; err != nil {
			return err
		}
		if err := resourcevalidation.ValidateCommandAgainstControlStreamSchema(cmd, &parent); err != nil {
			return &ValidationError{Err: err}
		}
		if err := tx.Create(cmd).Error; err != nil {
			return err
		}
		return expandControlStreamTimeRanges(tx, cmd.ControlStreamID, cmd)
	})
}

// expandControlStreamTimeRanges widens the parent control stream's issueTime
// and executionTime extents to include a newly written command. LEAST/GREATEST
// ignore NULL args and columns, so nil execution bounds are no-ops and
// initially-empty ranges get seeded.
func expandControlStreamTimeRanges(tx *gorm.DB, controlStreamID string, cmd *domains.Command) error {
	var execStart, execEnd *time.Time
	if cmd.ExecutionTime != nil {
		execStart = cmd.ExecutionTime.Start
		execEnd = cmd.ExecutionTime.End
		if execEnd == nil {
			// Open-ended command execution counts as a point at its start.
			execEnd = execStart
		}
	}
	return tx.Exec(`UPDATE control_streams SET
			issue_time_start     = LEAST(issue_time_start, ?),
			issue_time_end       = GREATEST(issue_time_end, ?),
			execution_time_start = LEAST(execution_time_start, ?),
			execution_time_end   = GREATEST(execution_time_end, ?)
		WHERE id = ?`,
		cmd.IssueTime, cmd.IssueTime, execStart, execEnd, controlStreamID).Error
}

// recomputeControlStreamTimeRanges recalculates the parent control stream's
// time extents from scratch after a command update or delete. With no commands
// left the extents go NULL.
func recomputeControlStreamTimeRanges(tx *gorm.DB, controlStreamID string) error {
	return tx.Exec(`UPDATE control_streams SET
			issue_time_start     = agg.it_min,
			issue_time_end       = agg.it_max,
			execution_time_start = agg.et_min,
			execution_time_end   = agg.et_max
		FROM (
			SELECT MIN(issue_time) AS it_min,
			       MAX(issue_time) AS it_max,
			       MIN(execution_time_start) AS et_min,
			       MAX(COALESCE(execution_time_end, execution_time_start)) AS et_max
			FROM commands WHERE control_stream_id = ?
		) agg
		WHERE control_streams.id = ?`,
		controlStreamID, controlStreamID).Error
}

// GetByID retrieves a command by ID.
func (r *CommandRepository) GetByID(ctx context.Context, id string) (*domains.Command, error) {
	var cmd domains.Command
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&cmd).Error
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}

// List retrieves commands with filtering.
func (r *CommandRepository) List(ctx context.Context, params *queryparams.CommandsQueryParams, controlStreamID *string) ([]*domains.Command, int64, error) {
	var commands []*domains.Command

	query := r.db.WithContext(ctx).Model(&domains.Command{})
	if controlStreamID != nil {
		query = query.Where("control_stream_id = ?", *controlStreamID)
	}
	query = r.applyFilters(query, params, controlStreamID != nil)

	if params.IssueTime != nil && params.IssueTime.Latest {
		err := query.Order("issue_time desc").Limit(1).Find(&commands).Error
		return commands, int64(len(commands)), err
	}

	return listPage(query, &params.QueryParams, CursorOrder{
		Columns:     []string{"commands.issue_time", "commands.id"},
		Descending:  true,
		TimeColumns: map[int]bool{0: true},
	}, func(command *domains.Command) []string {
		return []string{cursorTimeValue(command.IssueTime), command.ID}
	})
}

// ListByControlStream retrieves all commands for a specific control stream.
func (r *CommandRepository) ListByControlStream(ctx context.Context, controlStreamID string, params *queryparams.CommandsQueryParams) ([]*domains.Command, int64, error) {
	return r.List(ctx, params, &controlStreamID)
}

// Update updates a command.
func (r *CommandRepository) Update(ctx context.Context, cmd *domains.Command) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var stored domains.Command
		if err := tx.Where("id = ?", cmd.ID).First(&stored).Error; err != nil {
			return err
		}
		var parent domains.ControlStream
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", stored.ControlStreamID).First(&parent).Error; err != nil {
			return err
		}
		if err := resourcevalidation.ValidateCommandAgainstControlStreamSchema(cmd, &parent); err != nil {
			return &ValidationError{Err: err}
		}
		if err := replaceByID(tx, cmd, func(existing *domains.Command) {
			// The parent control stream, issueTime, currentStatus and
			// executionTime are server-owned.
			cmd.ControlStreamID = existing.ControlStreamID
			cmd.IssueTime = existing.IssueTime
			cmd.Sender = existing.Sender
			cmd.CurrentStatus = existing.CurrentStatus
			cmd.ExecutionTime = existing.ExecutionTime
		}); err != nil {
			return err
		}
		return recomputeControlStreamTimeRanges(tx, cmd.ControlStreamID)
	})
}

// Delete deletes a command.
func (r *CommandRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var controlStreamID string
		err := tx.Model(&domains.Command{}).Select("control_stream_id").
			Where("id = ?", id).First(&controlStreamID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := lockControlStream(tx, controlStreamID); err != nil {
			return err
		}
		if err := tx.Delete(&domains.CommandStatusReport{}, "command_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Delete(&domains.CommandResult{}, "command_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Delete(&domains.Command{}, "id = ?", id).Error; err != nil {
			return err
		}
		return recomputeControlStreamTimeRanges(tx, controlStreamID)
	})
}

// CreateStatus persists a command status report and updates the parent command current status.
func (r *CommandRepository) CreateStatus(ctx context.Context, status *domains.CommandStatusReport) error {
	if status.ReportTime.IsZero() {
		status.ReportTime = time.Now().UTC()
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockCommandStream(tx, status.CommandID); err != nil {
			return err
		}
		if err := tx.Create(status).Error; err != nil {
			return err
		}
		if err := tx.Model(&domains.Command{}).
			Where("id = ?", status.CommandID).
			Update("current_status", status.StatusCode).
			Error; err != nil {
			return err
		}
		return applyStatusExecutionTime(tx, status)
	})
}

// applyStatusExecutionTime copies a status report's executionTime onto the
// parent command (latest report wins) and refreshes the control stream's
// execution extent. Commands' executionTime is read-only for clients and is
// only ever populated through this path.
func applyStatusExecutionTime(tx *gorm.DB, status *domains.CommandStatusReport) error {
	if !status.ExecutionTime.HasBounds() {
		return nil
	}
	var controlStreamID string
	err := tx.Model(&domains.Command{}).Select("control_stream_id").
		Where("id = ?", status.CommandID).First(&controlStreamID).Error
	if err != nil {
		return err
	}
	if err := tx.Model(&domains.Command{}).
		Where("id = ?", status.CommandID).
		Updates(map[string]any{
			"execution_time_start": status.ExecutionTime.Start,
			"execution_time_end":   status.ExecutionTime.End,
		}).Error; err != nil {
		return err
	}
	return recomputeControlStreamTimeRanges(tx, controlStreamID)
}

// GetStatusByID retrieves a command status report by command ID and status ID.
func (r *CommandRepository) GetStatusByID(ctx context.Context, commandID, statusID string) (*domains.CommandStatusReport, error) {
	var status domains.CommandStatusReport
	err := r.db.WithContext(ctx).Where("command_id = ? AND id = ?", commandID, statusID).First(&status).Error
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// ListStatuses retrieves command status reports with filtering.
func (r *CommandRepository) ListStatuses(ctx context.Context, commandID string, params *queryparams.CommandStatusQueryParams) ([]*domains.CommandStatusReport, int64, error) {
	var statuses []*domains.CommandStatusReport

	query := r.db.WithContext(ctx).Model(&domains.CommandStatusReport{}).Where("command_id = ?", commandID)
	query = r.applyStatusFilters(query, params)
	query = applyCQLFilter(query, &params.QueryParams)

	if params.ReportTime != nil && params.ReportTime.Latest {
		err := query.Order("report_time desc").Limit(1).Find(&statuses).Error
		return statuses, int64(len(statuses)), err
	}

	return listPage(query, &params.QueryParams, CursorOrder{
		Columns:     []string{"report_time", "id"},
		Descending:  true,
		TimeColumns: map[int]bool{0: true},
	}, func(status *domains.CommandStatusReport) []string {
		return []string{queryparams.TimeCursorValue(status.ReportTime), status.ID}
	})
}

// UpdateStatus updates a command status report and the parent command current status.
func (r *CommandRepository) UpdateStatus(ctx context.Context, status *domains.CommandStatusReport) error {
	return r.updateStatus(ctx, status, false)
}

// UpdateStatusREST preserves reportTime from the locked row. MQTT Resource
// Data updates use UpdateStatus to retain the complete record's source time.
func (r *CommandRepository) UpdateStatusREST(ctx context.Context, status *domains.CommandStatusReport) error {
	return r.updateStatus(ctx, status, true)
}

func (r *CommandRepository) updateStatus(ctx context.Context, status *domains.CommandStatusReport, preserveReportTime bool) error {
	if status.ReportTime.IsZero() {
		status.ReportTime = time.Now().UTC()
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockCommandStream(tx, status.CommandID); err != nil {
			return err
		}
		if err := replaceByID(tx, status, func(existing *domains.CommandStatusReport) {
			status.CommandID = existing.CommandID
			if preserveReportTime {
				status.ReportTime = existing.ReportTime
			}
		}); err != nil {
			return err
		}
		if err := tx.Model(&domains.Command{}).
			Where("id = ?", status.CommandID).
			Update("current_status", status.StatusCode).
			Error; err != nil {
			return err
		}
		return applyStatusExecutionTime(tx, status)
	})
}

// DeleteStatus deletes a command status report.
func (r *CommandRepository) DeleteStatus(ctx context.Context, commandID, statusID string) error {
	result := r.db.WithContext(ctx).Delete(&domains.CommandStatusReport{}, "command_id = ? AND id = ?", commandID, statusID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateResult persists a command result.
func (r *CommandRepository) CreateResult(ctx context.Context, result *domains.CommandResult) error {
	return r.db.WithContext(ctx).Create(result).Error
}

// GetResultByID retrieves a command result by command ID and result ID.
func (r *CommandRepository) GetResultByID(ctx context.Context, commandID, resultID string) (*domains.CommandResult, error) {
	var result domains.CommandResult
	err := r.db.WithContext(ctx).Where("command_id = ? AND id = ?", commandID, resultID).First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListResults retrieves command results with filtering.
func (r *CommandRepository) ListResults(ctx context.Context, commandID string, params *queryparams.CommandResultQueryParams) ([]*domains.CommandResult, int64, error) {

	query := r.db.WithContext(ctx).Model(&domains.CommandResult{}).Where("command_id = ?", commandID)
	if len(params.IDs) > 0 {
		query = query.Where("id IN ?", params.IDs)
	}
	query = applyCQLFilter(query, &params.QueryParams)

	return listPage(query, &params.QueryParams, CursorOrder{
		Columns:     []string{"created_at", "id"},
		Descending:  true,
		TimeColumns: map[int]bool{0: true},
	}, func(result *domains.CommandResult) []string {
		return []string{queryparams.TimeCursorValue(result.CreatedAt), result.ID}
	})
}

// UpdateResult updates a command result.
func (r *CommandRepository) UpdateResult(ctx context.Context, result *domains.CommandResult) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return replaceByID(tx, result, func(existing *domains.CommandResult) {
			result.CommandID = existing.CommandID
		})
	})
}

// DeleteResult deletes a command result.
func (r *CommandRepository) DeleteResult(ctx context.Context, commandID, resultID string) error {
	result := r.db.WithContext(ctx).Delete(&domains.CommandResult{}, "command_id = ? AND id = ?", commandID, resultID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CommandRepository) applyFilters(query *gorm.DB, params *queryparams.CommandsQueryParams, controlStreamFixed bool) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("commands.id IN ?", params.IDs)
	}

	if !controlStreamFixed && len(params.ControlStream) > 0 {
		query = query.Where("commands.control_stream_id IN ?", params.ControlStream)
	}

	// A semi-join rather than a JOIN: control_streams shares column names
	// (id, execution_time_*) with commands.
	if len(params.System) > 0 {
		query = query.Where("commands.control_stream_id IN (SELECT id FROM control_streams WHERE system_id IN ?)", params.System)
	}

	if len(params.FOI) > 0 {
		query = query.Where("commands.sampling_feature_id IN ?", params.FOI)
	}

	if len(params.CurrentStatus) > 0 {
		query = query.Where("commands.current_status IN ?", params.CurrentStatus)
	}

	query = applyInstantFilter(query, "commands.issue_time", params.IssueTime)

	query = applyRangeFilter(query, "commands.execution_time_start", "commands.execution_time_end", params.ExecutionTime)

	query = applyKeywordFilter(query, params.Q, "CAST(commands.parameters AS text)", "commands.sender")

	return applyCQLFilter(query, &params.QueryParams)
}

func (r *CommandRepository) applyStatusFilters(query *gorm.DB, params *queryparams.CommandStatusQueryParams) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ?", params.IDs)
	}

	if len(params.StatusCode) > 0 {
		query = query.Where("status_code IN ?", params.StatusCode)
	}

	query = applyInstantFilter(query, "command_status_reports.report_time", params.ReportTime)

	return query
}

func lockControlStream(tx *gorm.DB, id string) error {
	var stream domains.ControlStream
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id = ?", id).First(&stream).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func lockCommandStream(tx *gorm.DB, id string) error {
	var cmd domains.Command
	if err := tx.Select("id", "control_stream_id").Where("id = ?", id).First(&cmd).Error; err != nil {
		return err
	}
	return lockControlStream(tx, cmd.ControlStreamID)
}
