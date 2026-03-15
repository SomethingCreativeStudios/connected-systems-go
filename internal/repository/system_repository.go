package repository

import (
	"encoding/json"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
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
func (r *SystemRepository) BuildSystemAssociations(systemID string) common_shared.Links {

	links := common_shared.Links{}

	if has, err := r.HasSubsystems(systemID); err == nil && has {
		links = append(links, common_shared.Link{
			Rel:  common_shared.OGCRel("subsystems"),
			Href: "/systems/" + systemID + "/subsystems",
		})
	}

	if has, err := r.HasSamplingFeatures(systemID); err == nil && has {
		links = append(links, common_shared.Link{
			Rel:  common_shared.OGCRel("samplingFeatures"),
			Href: "/systems/" + systemID + "/samplingFeatures",
		})
	}

	if has, err := r.HasDatastreams(systemID); err == nil && has {
		links = append(links, common_shared.Link{
			Rel:  common_shared.OGCRel("datastreams"),
			Href: "/systems/" + systemID + "/datastreams",
		})
	}

	if has, err := r.HasControlStreams(systemID); err == nil && has {
		links = append(links, common_shared.Link{
			Rel:  common_shared.OGCRel("controlstreams"),
			Href: "/systems/" + systemID + "/controlstreams",
		})
	}

	if has, err := r.HasDeployments(systemID); err == nil && has {
		links = append(links, common_shared.Link{
			Rel:  common_shared.OGCRel("deployments"),
			Href: "/systems/" + systemID + "/deployments",
		})
	}

	return links
}

// Create creates a new system
func (r *SystemRepository) Create(system *domains.System) error {
	return r.db.Create(system).Error
}

// GetByID retrieves a system by ID
func (r *SystemRepository) GetByID(id string) (*domains.System, error) {
	var system domains.System
	err := r.db.Where("id = ?", id).First(&system).Error
	if err != nil {
		return nil, err
	}
	return &system, nil
}

// GetByUID retrieves a system by unique identifier
func (r *SystemRepository) GetByUID(uid string) (*domains.System, error) {
	var system domains.System
	err := r.db.Where("unique_identifier = ?", uid).First(&system).Error
	if err != nil {
		return nil, err
	}
	return &system, nil
}

// List retrieves systems with filtering
func (r *SystemRepository) List(params *queryparams.SystemQueryParams) ([]*domains.System, int64, error) {
	var systems []*domains.System
	var total int64

	query := r.db.Model(&domains.System{})

	// Apply filters
	query = r.applyFilters(query, params)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	err := query.Debug().Find(&systems).Error
	return systems, total, err
}

// GetSubsystems retrieves subsystems of a parent system
func (r *SystemRepository) GetSubsystems(parentID string, recursive bool) ([]*domains.System, error) {
	var systems []*domains.System

	query := r.db.Where("parent_system_id = ?", parentID)

	if err := query.Find(&systems).Error; err != nil {
		return nil, err
	}

	// If recursive, get subsystems of subsystems
	if recursive {
		var allSystems []*domains.System
		allSystems = append(allSystems, systems...)

		for _, sys := range systems {
			children, err := r.GetSubsystems(sys.ID, true)
			if err != nil {
				return nil, err
			}
			allSystems = append(allSystems, children...)
		}
		return allSystems, nil
	}

	return systems, nil
}

// Update updates a system
func (r *SystemRepository) Update(systemId string, system *domains.System) error {
	system.ID = systemId
	return r.db.Save(system).Error
}

// Delete deletes a system
func (r *SystemRepository) Delete(id string, cascade bool) error {
	if !cascade {
		return r.db.Delete(&domains.System{}, "id = ?", id).Error
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		return r.deleteCascade(tx, id)
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

	if err := r.removeSystemFromDeployments(tx, systemID); err != nil {
		return err
	}

	if err := tx.Exec("DELETE FROM system_deployments WHERE system_id = ?", systemID).Error; err != nil {
		return err
	}
	if err := tx.Exec("DELETE FROM system_procedures WHERE system_id = ?", systemID).Error; err != nil {
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
		if err := tx.Where("control_stream_id IN ?", controlStreamIDs).Delete(&domains.Command{}).Error; err != nil {
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

// Checks if a system has subsystems
func (r *SystemRepository) HasSubsystems(systemID string) (bool, error) {
	return r.hasAssociatedRecords(&domains.System{}, "parent_system_id = ?", systemID)
}

func (r *SystemRepository) HasSamplingFeatures(systemID string) (bool, error) {
	return r.hasAssociatedRecords(&domains.SamplingFeature{}, "parent_system_id = ?", systemID)
}

func (r *SystemRepository) HasDatastreams(systemID string) (bool, error) {
	return r.hasAssociatedRecords(&domains.Datastream{}, "system_id = ?", systemID)
}

func (r *SystemRepository) HasControlStreams(systemID string) (bool, error) {
	return r.hasAssociatedRecords(&domains.ControlStream{}, "system_id = ?", systemID)
}

func (r *SystemRepository) HasDeployments(systemID string) (bool, error) {
	var count int64
	err := r.db.Model(&domains.Deployment{}).
		Where("system_ids @> ?::jsonb", "[\""+systemID+"\"]").
		Or("platform_id = ?", systemID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *SystemRepository) hasAssociatedRecords(model interface{}, query string, args ...interface{}) (bool, error) {
	var count int64
	err := r.db.Model(model).Where(query, args...).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// applyFilters applies query filters
func (r *SystemRepository) applyFilters(query *gorm.DB, params *queryparams.SystemQueryParams) *gorm.DB {
	if !params.Recursive {
		query = query.Where("parent_system_id IS NULL")
	}

	if len(params.IDs) > 0 {
		query = query.Where("id IN ? OR unique_identifier IN ?", params.IDs, params.IDs)
	}

	if len(params.Q) > 0 {
		var clauses []string
		var args []interface{}
		for _, term := range params.Q {
			clauses = append(clauses, "name ILIKE ?")
			args = append(args, "%"+term+"%")
			clauses = append(clauses, "description ILIKE ?")
			args = append(args, "%"+term+"%")
		}
		query = query.Where(strings.Join(clauses, " OR "), args...)
	}

	if len(params.Parent) > 0 {
		query = query.Where("parent_system_id IN ?", params.Parent)
	}

	if params.Datetime != nil {
		// Only add conditions if start/end are not nil
		if params.Datetime.Start != nil && params.Datetime.End != nil {
			query = query.Where("valid_time_start <= ? AND (valid_time_end IS NULL OR valid_time_end >= ?)", params.Datetime.End, params.Datetime.Start)
		} else if params.Datetime.Start != nil {
			query = query.Where("valid_time_end IS NULL OR valid_time_end >= ?", params.Datetime.Start)
		} else if params.Datetime.End != nil {
			query = query.Where("valid_time_start <= ?", params.Datetime.End)
		}
	}

	if params.Bbox != nil {
		query = query.Where("ST_Intersects(geometry, ST_MakeEnvelope(?, ?, ?, ?, 4326))", params.Bbox.MinX, params.Bbox.MinY, params.Bbox.MaxX, params.Bbox.MaxY)
	}

	if params.Geom != "" {
		query = query.Where("ST_Intersects(geometry, ST_GeomFromText(?, 4326))", params.Geom)
	}

	if len(params.Procedure) > 0 {
		query = query.Joins("JOIN system_procedures ON systems.id = system_procedures.system_id").
			Where("system_procedures.procedure_id IN ?", params.Procedure)
	}

	if len(params.FOI) > 0 {
		query = query.Joins("JOIN sampling_features ON systems.id = sampling_features.parent_system_id").
			Where("sampling_features.id IN ?", params.FOI)
	}

	if len(params.ObservedProperty) > 0 {
		query = query.Joins("JOIN system_observed_properties ON systems.id = system_observed_properties.system_id").
			Where("system_observed_properties.observed_property_id IN ?", params.ObservedProperty)
	}

	if len(params.ControlledProperty) > 0 {
		query = query.Joins("JOIN system_controlled_properties ON systems.id = system_controlled_properties.system_id").
			Where("system_controlled_properties.controlled_property_id IN ?", params.ControlledProperty)
	}
	return query
}
