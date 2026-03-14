package repository

import (
	"strings"
	"time"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
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
func (r *CommandRepository) Create(cmd *domains.Command) error {
	if cmd.IssueTime == nil {
		now := time.Now().UTC()
		cmd.IssueTime = &now
	}
	if cmd.CurrentStatus == "" {
		cmd.CurrentStatus = domains.CommandStatusPending
	}
	return r.db.Create(cmd).Error
}

// GetByID retrieves a command by ID.
func (r *CommandRepository) GetByID(id string) (*domains.Command, error) {
	var cmd domains.Command
	err := r.db.Where("id = ?", id).First(&cmd).Error
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}

// List retrieves commands with filtering.
func (r *CommandRepository) List(params *queryparams.CommandsQueryParams, controlStreamID *string) ([]*domains.Command, int64, error) {
	var commands []*domains.Command
	var total int64

	query := r.db.Model(&domains.Command{})
	if controlStreamID != nil {
		query = query.Where("control_stream_id = ?", *controlStreamID)
	}
	query = r.applyFilters(query, params, controlStreamID != nil)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	err := query.Order("issue_time desc").Find(&commands).Error
	return commands, total, err
}

// ListByControlStream retrieves all commands for a specific control stream.
func (r *CommandRepository) ListByControlStream(controlStreamID string, params *queryparams.CommandsQueryParams) ([]*domains.Command, int64, error) {
	return r.List(params, &controlStreamID)
}

// Update updates a command.
func (r *CommandRepository) Update(cmd *domains.Command) error {
	return r.db.Save(cmd).Error
}

// Delete deletes a command.
func (r *CommandRepository) Delete(id string) error {
	return r.db.Delete(&domains.Command{}, "id = ?", id).Error
}

func (r *CommandRepository) applyFilters(query *gorm.DB, params *queryparams.CommandsQueryParams, controlStreamFixed bool) *gorm.DB {
	joinedControlStreams := false
	joinControlStreams := func(q *gorm.DB) *gorm.DB {
		if !joinedControlStreams {
			q = q.Joins("JOIN control_streams ON commands.control_stream_id = control_streams.id")
			joinedControlStreams = true
		}
		return q
	}

	if len(params.IDs) > 0 {
		query = query.Where("commands.id IN ?", params.IDs)
	}

	if !controlStreamFixed && len(params.ControlStream) > 0 {
		query = query.Where("control_stream_id IN ?", params.ControlStream)
	}

	if len(params.System) > 0 {
		query = joinControlStreams(query)
		query = query.Where("control_streams.system_id IN ?", params.System)
	}

	if len(params.FOI) > 0 {
		query = query.Where("commands.sampling_feature_id IN ?", params.FOI)
	}

	if len(params.CurrentStatus) > 0 {
		query = query.Where("current_status IN ?", params.CurrentStatus)
	}

	if params.IssueTime != nil {
		if params.IssueTime.Start != nil && params.IssueTime.End != nil {
			query = query.Where("issue_time <= ? AND issue_time >= ?", params.IssueTime.End, params.IssueTime.Start)
		} else if params.IssueTime.Start != nil {
			query = query.Where("issue_time >= ?", params.IssueTime.Start)
		} else if params.IssueTime.End != nil {
			query = query.Where("issue_time <= ?", params.IssueTime.End)
		}
	}

	if params.ExecutionTime != nil {
		if params.ExecutionTime.Start != nil && params.ExecutionTime.End != nil {
			query = query.Where("execution_time_start <= ? AND (execution_time_end IS NULL OR execution_time_end >= ?)", params.ExecutionTime.End, params.ExecutionTime.Start)
		} else if params.ExecutionTime.Start != nil {
			query = query.Where("execution_time_end IS NULL OR execution_time_end >= ?", params.ExecutionTime.Start)
		} else if params.ExecutionTime.End != nil {
			query = query.Where("execution_time_start <= ?", params.ExecutionTime.End)
		}
	}

	if len(params.Q) > 0 {
		var clauses []string
		var args []interface{}
		for _, term := range params.Q {
			like := "%" + term + "%"
			clauses = append(clauses, "CAST(commands.parameters AS text) ILIKE ?")
			args = append(args, like)
			clauses = append(clauses, "commands.sender ILIKE ?")
			args = append(args, like)
		}
		query = query.Where(strings.Join(clauses, " OR "), args...)
	}

	return query
}
