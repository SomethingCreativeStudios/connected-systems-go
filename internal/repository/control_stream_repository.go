package repository

import (
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
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
	// issueTime and executionTime are read-only extents derived from commands;
	// a new control stream has none.
	cs.IssueTime = nil
	cs.ExecutionTime = nil
	// Seed the per-format schema registry with the initial schema.
	cs.Schemas = nil
	if cs.Schema != nil {
		cs.Schemas = cs.Schemas.Upsert(*cs.Schema)
	}
	normalizeControlStreamRefs(cs)
	r.populateSystemAssociations(cs)
	if err := r.db.Create(cs).Error; err != nil {
		return err
	}
	if cs.SystemID != nil && cs.ID != "" {
		r.db.Exec(
			"INSERT INTO system_controlstreams (system_id, control_stream_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
			*cs.SystemID, cs.ID,
		)
	}
	return nil
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

	if params.IssueTime != nil && params.IssueTime.Latest {
		err := query.Order("issue_time_start desc").Limit(1).Find(&controlStreams).Error
		return controlStreams, int64(len(controlStreams)), err
	}

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
// Procedure and deployment are locked (server-derived from system). SamplingFeature can be
// changed by the client; featureOfInterest is re-derived from the updated samplingFeature.
func (r *ControlStreamRepository) Update(cs *domains.ControlStream) error {
	var existing domains.ControlStream
	if err := r.db.Select("id", "procedure_link", "procedure_id", "deployment_link", "deployment_id",
		"issue_time_start", "issue_time_end", "execution_time_start", "execution_time_end", "schemas").
		Where("id = ?", cs.ID).First(&existing).Error; err == nil {
		cs.ProcedureLink = existing.ProcedureLink
		cs.ProcedureID = existing.ProcedureID
		cs.DeploymentLink = existing.DeploymentLink
		cs.DeploymentID = existing.DeploymentID
		// issueTime and executionTime are read-only extents derived from
		// commands; carry the stored values forward so Save can't clobber them.
		cs.IssueTime = common_shared.NonEmptyTimeRange(existing.IssueTime)
		cs.ExecutionTime = common_shared.NonEmptyTimeRange(existing.ExecutionTime)
		// Carry the schema registry forward, folding in the incoming schema.
		cs.Schemas = existing.Schemas
		if cs.Schema != nil {
			cs.Schemas = cs.Schemas.Upsert(*cs.Schema)
		}
	}
	normalizeControlStreamRefs(cs)
	r.deriveFOIFromSamplingFeature(cs)
	return r.db.Save(cs).Error
}

// populateSystemAssociations sets procedure and deployment from the parent system on create.
// Procedure comes from system.SystemKindID; deployment from system_deployments.
// FOI is derived from the user-supplied samplingFeature via deriveFOIFromSamplingFeature.
func (r *ControlStreamRepository) populateSystemAssociations(cs *domains.ControlStream) {
	if cs.SystemID == nil {
		return
	}
	systemID := *cs.SystemID

	// Procedure from TypeOfID FK
	var sys domains.System
	if err := r.db.Select("id", "type_of_id").Where("id = ?", systemID).First(&sys).Error; err == nil {
		if sys.TypeOfID != nil && *sys.TypeOfID != "" {
			kindID := *sys.TypeOfID
			cs.ProcedureLink = &common_shared.Link{Href: "procedures/" + kindID}
			cs.ProcedureID = &kindID
		}
	}

	// Deployment — first entry in system_deployments join table
	var depID string
	if err := r.db.Table("system_deployments").Select("deployment_id").Where("system_id = ?", systemID).Limit(1).Scan(&depID).Error; err == nil && depID != "" {
		cs.DeploymentLink = &common_shared.Link{Href: "deployments/" + depID}
		cs.DeploymentID = &depID
	}

	// FOI derived from user-provided samplingFeature
	r.deriveFOIFromSamplingFeature(cs)
}

// deriveFOIFromSamplingFeature sets FeatureOfInterest and FeatureOfInterestID on the control stream
// by looking up the sampledFeature link stored on the referenced SamplingFeature.
func (r *ControlStreamRepository) deriveFOIFromSamplingFeature(cs *domains.ControlStream) {
	if cs.SamplingFeatureID == nil {
		return
	}
	var sf domains.SamplingFeature
	if err := r.db.Select("id", "sampled_feature_link", "sampled_feature_id").
		Where("id = ?", *cs.SamplingFeatureID).First(&sf).Error; err == nil {
		cs.FeatureOfInterest = sf.SampledFeatureLink
		cs.FeatureOfInterestID = sf.SampledFeatureID
	}
}

// Delete deletes a control stream.
// If cascade is true, all commands associated with the control stream are deleted first.
func (r *ControlStreamRepository) Delete(id string, cascade bool) error {
	// Always clean join table first to unblock FK constraints.
	if err := r.db.Exec("DELETE FROM system_controlstreams WHERE control_stream_id = ?", id).Error; err != nil {
		return err
	}

	if !cascade {
		var cmdCount int64
		if err := r.db.Model(&domains.Command{}).Where("control_stream_id = ?", id).Count(&cmdCount).Error; err != nil {
			return err
		}
		if cmdCount > 0 {
			return ErrHasChildren
		}

		result := r.db.Delete(&domains.ControlStream{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("control_stream_id = ?", id).Delete(&domains.Command{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&domains.ControlStream{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// GetSchema retrieves the current schema and the per-format schema registry
// for a control stream.
func (r *ControlStreamRepository) GetSchema(id string) (*domains.ControlStreamSchema, domains.ControlStreamSchemas, error) {
	var cs domains.ControlStream
	err := r.db.Select("id", "schema", "schemas").Where("id = ?", id).First(&cs).Error
	if err != nil {
		return nil, nil, err
	}
	// Legacy rows predate the registry: expose the current schema through it.
	if len(cs.Schemas) == 0 && cs.Schema != nil {
		cs.Schemas = cs.Schemas.Upsert(*cs.Schema)
	}
	return cs.Schema, cs.Schemas, nil
}

// UpdateSchema upserts the schema into the control stream's per-format
// registry (replacing the entry with the same commandFormat, adding it
// otherwise) and makes it the current schema.
func (r *ControlStreamRepository) UpdateSchema(id string, schema *domains.ControlStreamSchema) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing domains.ControlStream
		if err := tx.Select("id", "schema", "schemas").Where("id = ?", id).First(&existing).Error; err != nil {
			return err
		}
		schemas := existing.Schemas
		// Legacy rows predate the registry: fold the stored schema in first.
		if len(schemas) == 0 && existing.Schema != nil {
			schemas = schemas.Upsert(*existing.Schema)
		}
		if schema != nil {
			schemas = schemas.Upsert(*schema)
		}
		return tx.Model(&domains.ControlStream{}).Where("id = ?", id).
			Updates(map[string]interface{}{"schema": schema, "schemas": schemas}).Error
	})
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

	if params.IssueTime != nil && !params.IssueTime.Latest {
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
		query = query.Where("feature_of_interest_id IN ?", params.FOI)
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
