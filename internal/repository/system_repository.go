package repository

import (
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
			Rel:  "subsystems",
			Href: "/systems/" + systemID + "/subsystems",
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
	if cascade {
		// Delete subsystems first
		if err := r.db.Where("parent_system_id = ?", id).Delete(&domains.System{}).Error; err != nil {
			return err
		}
	}
	return r.db.Delete(&domains.System{}, "id = ?", id).Error
}

// Checks if a system has subsystems
func (r *SystemRepository) HasSubsystems(systemID string) (bool, error) {
	var count int64

	err := r.db.Model(&domains.System{}).Where("parent_system_id = ?", systemID).Count(&count).Error

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
