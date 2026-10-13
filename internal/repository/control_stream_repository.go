package repository

import (
	"context"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
func (r *ControlStreamRepository) Create(ctx context.Context, cs *domains.ControlStream) error {
	// issueTime and executionTime are read-only extents derived from commands;
	// a new control stream has none. live is server-owned (command channel
	// availability) and starts out false.
	cs.IssueTime = nil
	cs.ExecutionTime = nil
	cs.Live = nil
	// Seed the per-format schema registry with the initial schema.
	cs.Schemas = nil
	if cs.Schema != nil {
		cs.Schemas = cs.Schemas.Upsert(*cs.Schema)
	}
	cs.RecomputeSchemaMetadata()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.resolveReferences(ctx, tx, cs); err != nil {
			return err
		}
		if cs.SystemID != nil {
			if err := requireParent(tx, &domains.System{}, "system", *cs.SystemID); err != nil {
				return err
			}
		}
		if err := r.populateSystemAssociations(tx, cs); err != nil {
			return err
		}
		if err := r.resolveReferences(ctx, tx, cs); err != nil {
			return err
		}
		return tx.Create(cs).Error
	})
}

// GetByID retrieves a control stream by ID.
func (r *ControlStreamRepository) GetByID(ctx context.Context, id string) (*domains.ControlStream, error) {
	var cs domains.ControlStream
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&cs).Error
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

// List retrieves control streams with filtering.
func (r *ControlStreamRepository) List(ctx context.Context, params *queryparams.ControlStreamsQueryParams, systemID *string) ([]*domains.ControlStream, int64, error) {
	var controlStreams []*domains.ControlStream

	query := r.db.WithContext(ctx).Model(&domains.ControlStream{})
	query = r.applyFilters(query, params, systemID)

	if params.IssueTime != nil && params.IssueTime.Latest {
		err := query.Order("issue_time_start desc").Limit(1).Find(&controlStreams).Error
		return controlStreams, int64(len(controlStreams)), err
	}

	return listPage(query, &params.QueryParams, idOrder("control_streams"), idAnchor[*domains.ControlStream])
}

// Update updates a control stream.
// Writable links are replaced; system membership and schema registries are preserved.
func (r *ControlStreamRepository) Update(ctx context.Context, cs *domains.ControlStream) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return replaceByIDChecked(tx, cs, func(existing *domains.ControlStream) error {
			// system@link is read-only: a control stream cannot be reparented.
			cs.SystemID = existing.SystemID
			// issueTime and executionTime are read-only extents derived from
			// commands; carry the stored values forward.
			cs.IssueTime = common_shared.NonEmptyTimeRange(existing.IssueTime)
			cs.ExecutionTime = common_shared.NonEmptyTimeRange(existing.ExecutionTime)

			// live is server-owned; carry the stored value forward.
			cs.Live = existing.Live
			if err := r.resolveReferences(ctx, tx, cs); err != nil {
				return err
			}
			return mergeControlStreamSchema(tx, cs, existing)
		})
	})
}

// populateSystemAssociations sets procedure and deployment from the parent system on create.
// Procedure comes from systems.type_of_id; deployment is inferred only when unique.
// FOI is derived from the user-supplied samplingFeature via deriveFOIFromSamplingFeature.
func (r *ControlStreamRepository) populateSystemAssociations(db *gorm.DB, cs *domains.ControlStream) error {
	if cs.SystemID == nil {
		return nil
	}
	systemID := *cs.SystemID

	// Procedure from TypeOfID FK
	var sys domains.System
	if err := db.Select("id", "type_of_id").Where("id = ?", systemID).First(&sys).Error; err != nil {
		return err
	}
	{
		if cs.ProcedureLink == nil && sys.TypeOfID != nil && *sys.TypeOfID != "" {
			kindID := *sys.TypeOfID
			cs.ProcedureLink = &common_shared.Link{Href: "procedures/" + kindID}
			cs.ProcedureID = &kindID
		}
	}

	// Infer deployment only for an unambiguous single membership.
	var depIDs []string
	if err := db.Table("system_deployments").Where("system_id = ?", systemID).Limit(2).Pluck("deployment_id", &depIDs).Error; err != nil {
		return err
	}
	if cs.DeploymentLink == nil && len(depIDs) == 1 {
		depID := depIDs[0]
		cs.DeploymentLink = &common_shared.Link{Href: "deployments/" + depID}
		cs.DeploymentID = &depID
	}

	// FOI derived from user-provided samplingFeature
	return r.deriveFOIFromSamplingFeature(db, cs)
}

// deriveFOIFromSamplingFeature sets FeatureOfInterest and FeatureOfInterestID on the control stream
// by looking up the sampledFeature link stored on the referenced SamplingFeature.
func (r *ControlStreamRepository) deriveFOIFromSamplingFeature(db *gorm.DB, cs *domains.ControlStream) error {
	if cs.FeatureOfInterest != nil || cs.SamplingFeatureID == nil {
		return nil
	}
	var sf domains.SamplingFeature
	if err := db.Select("id", "sampled_feature_link", "sampled_feature_id").
		Where("id = ?", *cs.SamplingFeatureID).First(&sf).Error; err != nil {
		return err
	}
	{
		cs.FeatureOfInterest = sf.SampledFeatureLink
		cs.FeatureOfInterestID = sf.SampledFeatureID
	}
	return nil
}

// Delete deletes a control stream.
// If cascade is true, all commands associated with the control stream are
// deleted first. The checks and every delete share one transaction, so a
// refused or failed delete leaves the control stream exactly as it was.
func (r *ControlStreamRepository) Delete(ctx context.Context, id string, cascade bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockControlStream(tx, id); err != nil {
			return err
		}
		if cascade {
			if err := deleteControlStreamCommands(tx, []string{id}); err != nil {
				return err
			}
		} else if has, err := hasRows(tx, &domains.Command{}, "control_stream_id = ?", id); err != nil {
			return err
		} else if has {
			return ErrHasChildren
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

// deleteControlStreamCommands removes the commands of the given control
// streams together with their status reports and results, which no foreign
// key would otherwise clean up.
func deleteControlStreamCommands(tx *gorm.DB, controlStreamIDs []string) error {
	if len(controlStreamIDs) == 0 {
		return nil
	}
	commandIDs := tx.Model(&domains.Command{}).Select("id").Where("control_stream_id IN ?", controlStreamIDs)
	if err := tx.Where("command_id IN (?)", commandIDs).Delete(&domains.CommandStatusReport{}).Error; err != nil {
		return err
	}
	if err := tx.Where("command_id IN (?)", commandIDs).Delete(&domains.CommandResult{}).Error; err != nil {
		return err
	}
	return tx.Where("control_stream_id IN ?", controlStreamIDs).Delete(&domains.Command{}).Error
}

// GetSchema retrieves the current schema and the per-format schema registry
// for a control stream.
func (r *ControlStreamRepository) GetSchema(ctx context.Context, id string) (*domains.ControlStreamSchema, domains.ControlStreamSchemas, error) {
	var cs domains.ControlStream
	err := r.db.WithContext(ctx).Select("id", "schema", "schemas").Where("id = ?", id).First(&cs).Error
	if err != nil {
		return nil, nil, err
	}
	// Legacy rows predate the registry: expose the current schema through it.
	if len(cs.Schemas) == 0 && cs.Schema != nil {
		cs.Schemas = cs.Schemas.Upsert(*cs.Schema)
	}
	if cs.Schema == nil && len(cs.Schemas) > 0 {
		cs.Schema = cs.Schemas.ForFormat("application/json")
		if cs.Schema == nil {
			cs.Schema = &cs.Schemas[0]
		}
	}
	return cs.Schema, cs.Schemas, nil
}

// UpdateSchema upserts the schema into the control stream's per-format
// registry (replacing the entry with the same commandFormat, adding it
// otherwise) and makes it the current schema.
func (r *ControlStreamRepository) UpdateSchema(ctx context.Context, id string, schema *domains.ControlStreamSchema) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domains.ControlStream
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&existing).Error; err != nil {
			return err
		}
		replacement := existing
		replacement.Schema = schema
		if err := mergeControlStreamSchema(tx, &replacement, &existing); err != nil {
			return err
		}
		return tx.Model(&replacement).Select("schema", "schemas", "formats", "controlled_properties").Updates(&replacement).Error
	})
}

func (r *ControlStreamRepository) applyFilters(query *gorm.DB, params *queryparams.ControlStreamsQueryParams, systemID *string) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ? OR unique_identifier IN ?", params.IDs, params.IDs)
	}

	query = applyKeywordFilter(query, params.Q, "name", "description", "input_name")

	query = applyRangeFilter(query, "control_streams.issue_time_start", "control_streams.issue_time_end", params.IssueTime)

	query = applyRangeFilter(query, "control_streams.execution_time_start", "control_streams.execution_time_end", params.ExecutionTime)

	if systemID != nil {
		query = query.Where("system_id IN ("+systemTreeSQL("?")+")", *systemID)
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

	return applyCQLFilter(query, &params.QueryParams)
}

func (r *ControlStreamRepository) resolveReferences(ctx context.Context, tx *gorm.DB, cs *domains.ControlStream) error {
	var err error
	cs.ProcedureID, cs.DeploymentID, cs.SamplingFeatureID, cs.FeatureOfInterestID, err = resolveStreamReferences(ctx, tx, cs.ProcedureLink, cs.DeploymentLink, cs.SamplingFeatureLink, cs.FeatureOfInterest)
	return err
}
