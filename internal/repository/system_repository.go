package repository

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"gorm.io/gorm"
)

// SystemRepository handles System data access
type SystemRepository struct {
	db *gorm.DB
}

// NewSystemRepository creates a new SystemRepository
func NewSystemRepository(db *gorm.DB) *SystemRepository {
	return &SystemRepository{db: db}
}

// Build all necessary associations for a system
// Create creates a new system
func (r *SystemRepository) Create(ctx context.Context, system *domains.System) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if system.ParentSystemID != nil {
			if err := requireParent(tx, &domains.System{}, "system", *system.ParentSystemID); err != nil {
				return err
			}
		}
		return tx.Create(system).Error
	})
}

// GetByID retrieves a system by ID
func (r *SystemRepository) GetByID(ctx context.Context, id string) (*domains.System, error) {
	var system domains.System
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&system).Error
	if err != nil {
		return nil, err
	}
	return &system, nil
}

// GetByUID retrieves a system by unique identifier
func (r *SystemRepository) GetByUID(ctx context.Context, uid string) (*domains.System, error) {
	var system domains.System
	err := r.db.WithContext(ctx).Where("unique_identifier = ?", uid).First(&system).Error
	if err != nil {
		return nil, err
	}
	return &system, nil
}

// GetByIDs returns systems keyed by ID or unique identifier
func (r *SystemRepository) GetByIDs(ctx context.Context, ids []string) (map[string]*domains.System, error) {
	result := make(map[string]*domains.System)
	if len(ids) == 0 {
		return result, nil
	}

	var systems []*domains.System
	if err := r.db.WithContext(ctx).Where("id IN ? OR unique_identifier IN ?", ids, ids).Find(&systems).Error; err != nil {
		return nil, err
	}

	for _, s := range systems {
		if s == nil {
			continue
		}
		result[s.ID] = s
		if string(s.UniqueIdentifier) != "" {
			result[string(s.UniqueIdentifier)] = s
		}
	}

	return result, nil
}

// List retrieves systems with filtering
func (r *SystemRepository) List(ctx context.Context, params *queryparams.SystemQueryParams) ([]*domains.System, int64, error) {
	var systems []*domains.System

	query := r.db.WithContext(ctx).Model(&domains.System{})
	query = r.applyFilters(query, params)

	if params.Datetime != nil && params.Datetime.Latest {
		err := query.Order("systems.valid_time_start desc").Limit(1).Find(&systems).Error
		return systems, int64(len(systems)), err
	}

	return listPage(query, &params.QueryParams, idOrder("systems"), idAnchor[*domains.System])
}

// ListSubsystems retrieves direct or recursive subsystems through the database
// so the route can use the same deterministic cursor pagination as all other
// resource collections.
func (r *SystemRepository) ListSubsystems(ctx context.Context, parentID string, params *queryparams.SystemQueryParams) ([]*domains.System, int64, error) {
	var systems []*domains.System

	query := r.db.WithContext(ctx).Model(&domains.System{})
	if params.Recursive {
		query = query.Where(`systems.id IN (
			WITH RECURSIVE subtree(id) AS (
				SELECT id FROM systems WHERE parent_system_id = ?
				UNION
				SELECT systems.id FROM systems JOIN subtree ON systems.parent_system_id = subtree.id
			)
			SELECT id FROM subtree
		)`, parentID)
	} else {
		query = query.Where("systems.parent_system_id = ?", parentID)
	}
	query = applyCQLFilter(r.applyResourceFilters(query, params), &params.QueryParams)

	if params.Datetime != nil && params.Datetime.Latest {
		err := query.Order("systems.valid_time_start desc").Limit(1).Find(&systems).Error
		return systems, int64(len(systems)), err
	}

	return listPage(query, &params.QueryParams, idOrder("systems"), idAnchor[*domains.System])
}

// Update replaces an existing system. It returns ErrNotFound rather than
// creating the system, and keeps the parent, which is only ever set through
// /systems/{id}/subsystems.
func (r *SystemRepository) Update(ctx context.Context, systemId string, system *domains.System) error {
	system.ID = systemId
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return replaceByID(tx, system, func(existing *domains.System) {
			system.ParentSystemID = existing.ParentSystemID
		})
	})
}

// Delete deletes a system
// systemChildren are the tables whose rows block a non-cascading system delete.
var systemChildren = []struct {
	model  any
	column string
}{
	{&domains.System{}, "parent_system_id"},
	{&domains.SamplingFeature{}, "parent_system_id"},
	{&domains.Datastream{}, "system_id"},
	{&domains.ControlStream{}, "system_id"},
}

// Delete deletes a system. Without cascade it refuses while the system still
// has children; the checks and the delete share one transaction.
func (r *SystemRepository) Delete(ctx context.Context, id string, cascade bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		exists, err := hasRows(tx, &domains.System{}, "id = ?", id)
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		if cascade {
			return r.deleteCascade(tx, id)
		}
		// Application-level child check (no DB FK constraints on these columns).
		for _, child := range systemChildren {
			has, err := hasRows(tx, child.model, child.column+" = ?", id)
			if err != nil {
				return err
			}
			if has {
				return ErrHasChildren
			}
		}
		return tx.Delete(&domains.System{}, "id = ?", id).Error
	})
}

func (r *SystemRepository) deleteCascade(tx *gorm.DB, systemID string) error {
	var childIDs []string
	if err := tx.Model(&domains.System{}).Where("parent_system_id = ?", systemID).Pluck("id", &childIDs).Error; err != nil {
		return err
	}

	for _, childID := range childIDs {
		if err := r.deleteCascade(tx, childID); err != nil {
			return err
		}
	}

	if err := tx.Where("parent_system_id = ?", systemID).Delete(&domains.SamplingFeature{}).Error; err != nil {
		return err
	}

	if err := r.deleteSystemDatastreams(tx, systemID); err != nil {
		return err
	}

	if err := r.deleteSystemControlStreams(tx, systemID); err != nil {
		return err
	}

	if err := tx.Where("system_id = ?", systemID).Delete(&domains.SystemHistoryRevision{}).Error; err != nil {
		return err
	}

	if err := tx.Where("system_id = ?", systemID).Delete(&domains.SystemEvent{}).Error; err != nil {
		return err
	}

	if err := r.removeSystemFromDeployments(tx, systemID); err != nil {
		return err
	}

	if err := tx.Exec("DELETE FROM system_deployments WHERE system_id = ?", systemID).Error; err != nil {
		return err
	}

	return tx.Delete(&domains.System{}, "id = ?", systemID).Error
}

func (r *SystemRepository) deleteSystemDatastreams(tx *gorm.DB, systemID string) error {
	var datastreamIDs []string
	if err := tx.Model(&domains.Datastream{}).Where("system_id = ?", systemID).Pluck("id", &datastreamIDs).Error; err != nil {
		return err
	}

	if len(datastreamIDs) > 0 {
		if err := tx.Where("datastream_id IN ?", datastreamIDs).Delete(&domains.Observation{}).Error; err != nil {
			return err
		}
	}

	return tx.Where("system_id = ?", systemID).Delete(&domains.Datastream{}).Error
}

func (r *SystemRepository) deleteSystemControlStreams(tx *gorm.DB, systemID string) error {
	var controlStreamIDs []string
	if err := tx.Model(&domains.ControlStream{}).Where("system_id = ?", systemID).Pluck("id", &controlStreamIDs).Error; err != nil {
		return err
	}

	if len(controlStreamIDs) > 0 {
		if err := deleteControlStreamCommands(tx, controlStreamIDs); err != nil {
			return err
		}
	}

	return tx.Where("system_id = ?", systemID).Delete(&domains.ControlStream{}).Error
}

func (r *SystemRepository) removeSystemFromDeployments(tx *gorm.DB, systemID string) error {
	needle, err := json.Marshal([]string{systemID})
	if err != nil {
		return err
	}

	var deployments []*domains.Deployment
	if err := tx.Where("system_ids @> ?::jsonb", string(needle)).Find(&deployments).Error; err != nil {
		return err
	}

	for _, deployment := range deployments {
		changed := false

		if deployment.SystemIds != nil {
			filteredSystemIDs := make(common_shared.StringArray, 0, len(*deployment.SystemIds))
			for _, id := range *deployment.SystemIds {
				if id != systemID {
					filteredSystemIDs = append(filteredSystemIDs, id)
				}
			}
			if len(filteredSystemIDs) != len(*deployment.SystemIds) {
				changed = true
				if len(filteredSystemIDs) == 0 {
					deployment.SystemIds = nil
				} else {
					deployment.SystemIds = &filteredSystemIDs
				}
			}
		}

		if len(deployment.DeployedSystems) > 0 {
			filteredDeployedSystems := make(domains.DeployedSystemItems, 0, len(deployment.DeployedSystems))
			for _, item := range deployment.DeployedSystems {
				itemSystemID := item.System.GetId("systems")
				if itemSystemID != nil && *itemSystemID == systemID {
					changed = true
					continue
				}
				filteredDeployedSystems = append(filteredDeployedSystems, item)
			}
			deployment.DeployedSystems = filteredDeployedSystems
		}

		if deployment.Platform != nil {
			platformSystemID := deployment.Platform.System.GetId("systems")
			if platformSystemID != nil && *platformSystemID == systemID {
				changed = true
				deployment.Platform = nil
				deployment.PlatformID = nil
			}
		}

		if changed {
			if err := tx.Save(deployment).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// HasSubsystems reports whether a system has subsystems.
func (r *SystemRepository) HasSubsystems(ctx context.Context, systemID string) (bool, error) {
	return hasRows(r.db.WithContext(ctx), &domains.System{}, "parent_system_id = ?", systemID)
}

// applyFilters applies query filters
func (r *SystemRepository) applyFilters(query *gorm.DB, params *queryparams.SystemQueryParams) *gorm.DB {
	if !params.Recursive {
		query = query.Where("systems.parent_system_id IS NULL")
	}
	return applyCQLFilter(r.applyResourceFilters(query, params), &params.QueryParams)
}

// applyResourceFilters applies filters that are common to canonical and
// nested System collections. Hierarchy scoping is handled by the caller.
// Columns are table qualified, and association filters are EXISTS subqueries
// so a system matching several children is still returned once.
func (r *SystemRepository) applyResourceFilters(query *gorm.DB, params *queryparams.SystemQueryParams) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("systems.id IN ? OR systems.unique_identifier IN ?", params.IDs, params.IDs)
	}

	query = applyKeywordFilter(query, params.Q, "systems.name", "systems.description")

	if len(params.Parent) > 0 {
		query = query.Where("systems.parent_system_id IN ?", params.Parent)
	}

	query = applyValidTimeFilter(query, "systems", params.Datetime)

	query = applySpatialIntersectionFilters(query, "systems.geometry", params.Bbox, params.Geom)

	if len(params.Procedure) > 0 {
		query = query.Where("systems.type_of_id IN ?", params.Procedure)
	}

	if len(params.FOI) > 0 {
		query = query.Where(`EXISTS (SELECT 1 FROM sampling_features sf
			WHERE sf.parent_system_id = systems.id AND (sf.sampled_feature_id IN ? OR sf.id IN ?))`, params.FOI, params.FOI)
	}

	if len(params.ObservedProperty) > 0 {
		match, args := ilikeAll("ds.observed_properties::text", params.ObservedProperty)
		query = query.Where(`EXISTS (SELECT 1 FROM datastreams ds
			WHERE ds.system_id = systems.id AND `+match+`)`, args...)
	}

	if len(params.ControlledProperty) > 0 {
		match, args := ilikeAll("cs.controlled_properties::text", params.ControlledProperty)
		query = query.Where(`EXISTS (SELECT 1 FROM control_streams cs
			WHERE cs.system_id = systems.id AND `+match+`)`, args...)
	}
	return query
}

// ilikeAll matches a single row that mentions every term, which keeps the
// semantics the JOIN-based property filters had: one datastream (or control
// stream) has to carry all requested properties.
func ilikeAll(column string, terms []string) (string, []any) {
	clauses := make([]string, len(terms))
	args := make([]any, len(terms))
	for i, term := range terms {
		clauses[i] = column + " ILIKE ?"
		args[i] = "%" + term + "%"
	}
	return strings.Join(clauses, " AND "), args
}
