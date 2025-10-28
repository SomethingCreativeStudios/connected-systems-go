package repository

import (
	"github.com/yourusername/connected-systems-go/internal/model"
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
func (r *SystemRepository) BuildSystemAssociations(systemID string) model.Links {

	links := model.Links{}

	if has, err := r.HasSubsystems(systemID); err == nil && has {
		links = append(links, model.Link{
			Rel:  "subsystems",
			Href: "/systems/" + systemID + "/subsystems",
		})
	}

	return links
}

// Create creates a new system
func (r *SystemRepository) Create(system *model.System) error {
	return r.db.Create(system).Error
}

// GetByID retrieves a system by ID
func (r *SystemRepository) GetByID(id string) (*model.System, error) {
	var system model.System
	err := r.db.Where("id = ?", id).First(&system).Error
	if err != nil {
		return nil, err
	}
	return &system, nil
}

// GetByUID retrieves a system by unique identifier
func (r *SystemRepository) GetByUID(uid string) (*model.System, error) {
	var system model.System
	err := r.db.Where("unique_identifier = ?", uid).First(&system).Error
	if err != nil {
		return nil, err
	}
	return &system, nil
}

// List retrieves systems with filtering
func (r *SystemRepository) List(params *QueryParams) ([]*model.System, int64, error) {
	var systems []*model.System
	var total int64

	query := r.db.Model(&model.System{})

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
func (r *SystemRepository) GetSubsystems(parentID string, recursive bool) ([]*model.System, error) {
	var systems []*model.System

	query := r.db.Where("parent_system_id = ?", parentID)

	if err := query.Find(&systems).Error; err != nil {
		return nil, err
	}

	// If recursive, get subsystems of subsystems
	if recursive {
		var allSystems []*model.System
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
func (r *SystemRepository) Update(system *model.System) error {
	return r.db.Save(system).Error
}

// Delete deletes a system
func (r *SystemRepository) Delete(id string, cascade bool) error {
	if cascade {
		// Delete subsystems first
		if err := r.db.Where("parent_system_id = ?", id).Delete(&model.System{}).Error; err != nil {
			return err
		}
	}
	return r.db.Delete(&model.System{}, "id = ?", id).Error
}

// applyFilters applies query filters
func (r *SystemRepository) applyFilters(query *gorm.DB, params *QueryParams) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ? OR unique_identifier IN ?", params.IDs, params.IDs)
	}

	if params.Q != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+params.Q+"%", "%"+params.Q+"%")
	}

	if len(params.Parent) > 0 {
		query = query.Where("parent_system_id IN ?", params.Parent)
	}

	// Add more filters as needed (bbox, datetime, etc.)

	return query
}

// Checks if a system has subsystems
func (r *SystemRepository) HasSubsystems(systemID string) (bool, error) {
	var count int64

	err := r.db.Model(&model.System{}).Where("parent_system_id = ?", systemID).Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
