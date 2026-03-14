package repository

import (
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// ControlStreamRepository handles ControlStream data access.
type ControlStreamRepository struct {
	db *gorm.DB
}

// NewControlStreamRepository creates a new ControlStreamRepository.
func NewControlStreamRepository(db *gorm.DB) *ControlStreamRepository {
	return &ControlStreamRepository{db: db}
}

// Create creates a new control stream.
func (r *ControlStreamRepository) Create(cs *domains.ControlStream) error {
	normalizeControlStreamRefs(cs)
	return r.db.Create(cs).Error
}

// GetByID retrieves a control stream by ID.
func (r *ControlStreamRepository) GetByID(id string) (*domains.ControlStream, error) {
	var cs domains.ControlStream
	err := r.db.Where("id = ?", id).First(&cs).Error
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

// List retrieves control streams with filtering.
func (r *ControlStreamRepository) List(params *queryparams.ControlStreamsQueryParams, systemID *string) ([]*domains.ControlStream, int64, error) {
	var controlStreams []*domains.ControlStream
	var total int64

	query := r.db.Model(&domains.ControlStream{})
	query = r.applyFilters(query, params, systemID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	err := query.Find(&controlStreams).Error
	return controlStreams, total, err
}

// Update updates a control stream.
func (r *ControlStreamRepository) Update(cs *domains.ControlStream) error {
	normalizeControlStreamRefs(cs)
	return r.db.Save(cs).Error
}

// Delete deletes a control stream.
func (r *ControlStreamRepository) Delete(id string) error {
	return r.db.Delete(&domains.ControlStream{}, "id = ?", id).Error
}

// GetSchema retrieves only the schema for a control stream.
func (r *ControlStreamRepository) GetSchema(id string) (*domains.ControlStreamSchema, error) {
	var cs domains.ControlStream
	err := r.db.Select("id", "schema").Where("id = ?", id).First(&cs).Error
	if err != nil {
		return nil, err
	}
	return cs.Schema, nil
}

// UpdateSchema updates only the schema of a control stream.
func (r *ControlStreamRepository) UpdateSchema(id string, schema *domains.ControlStreamSchema) error {
	return r.db.Model(&domains.ControlStream{}).Where("id = ?", id).Update("schema", schema).Error
}

func (r *ControlStreamRepository) applyFilters(query *gorm.DB, params *queryparams.ControlStreamsQueryParams, systemID *string) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ? OR unique_identifier IN ?", params.IDs, params.IDs)
	}

	if len(params.Q) > 0 {
		var clauses []string
		var args []interface{}
		for _, term := range params.Q {
			like := "%" + term + "%"
			clauses = append(clauses, "name ILIKE ?")
			args = append(args, like)
			clauses = append(clauses, "description ILIKE ?")
			args = append(args, like)
			clauses = append(clauses, "input_name ILIKE ?")
			args = append(args, like)
		}
		query = query.Where(strings.Join(clauses, " OR "), args...)
	}

	if params.IssueTime != nil {
		if params.IssueTime.Start != nil && params.IssueTime.End != nil {
			query = query.Where("issue_time_start <= ? AND (issue_time_end IS NULL OR issue_time_end >= ?)", params.IssueTime.End, params.IssueTime.Start)
		} else if params.IssueTime.Start != nil {
			query = query.Where("issue_time_end IS NULL OR issue_time_end >= ?", params.IssueTime.Start)
		} else if params.IssueTime.End != nil {
			query = query.Where("issue_time_start <= ?", params.IssueTime.End)
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

	if systemID != nil {
		query = query.Where("system_id = ?", *systemID)
	} else if len(params.System) > 0 {
		query = query.Where("system_id IN ?", params.System)
	}

	if len(params.FOI) > 0 {
		query = query.Where("sampling_feature_id IN ?", params.FOI)
	}

	if len(params.ControlledProperty) > 0 {
		for _, cp := range params.ControlledProperty {
			query = query.Where("controlled_properties::text ILIKE ?", "%"+cp+"%")
		}
	}

	return query
}

func normalizeControlStreamRefs(cs *domains.ControlStream) {
	if cs == nil {
		return
	}

	if cs.SystemLink != nil {
		cs.SystemID = cs.SystemLink.GetId("systems")
	}
	if cs.ProcedureLink != nil {
		cs.ProcedureID = cs.ProcedureLink.GetId("procedures")
	}
	if cs.DeploymentLink != nil {
		cs.DeploymentID = cs.DeploymentLink.GetId("deployments")
	}
	if cs.FeatureOfInterest != nil {
		cs.FeatureOfInterestID = cs.FeatureOfInterest.GetId("features")
	}
	if cs.SamplingFeatureLink != nil {
		cs.SamplingFeatureID = cs.SamplingFeatureLink.GetId("samplingFeatures")
	}
}
