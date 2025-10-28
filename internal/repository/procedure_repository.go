package repository

import (
	"github.com/yourusername/connected-systems-go/internal/model"
	"gorm.io/gorm"
)

// ProcedureRepository handles Procedure data access
type ProcedureRepository struct {
	db *gorm.DB
}

// NewProcedureRepository creates a new ProcedureRepository
func NewProcedureRepository(db *gorm.DB) *ProcedureRepository {
	return &ProcedureRepository{db: db}
}

// Create creates a new procedure
func (r *ProcedureRepository) Create(procedure *model.Procedure) error {
	return r.db.Create(procedure).Error
}

// GetByID retrieves a procedure by ID
func (r *ProcedureRepository) GetByID(id string) (*model.Procedure, error) {
	var procedure model.Procedure
	err := r.db.Where("id = ?", id).First(&procedure).Error
	if err != nil {
		return nil, err
	}
	return &procedure, nil
}

// List retrieves procedures with filtering
func (r *ProcedureRepository) List(params *QueryParams) ([]*model.Procedure, int64, error) {
	var procedures []*model.Procedure
	var total int64

	query := r.db.Model(&model.Procedure{})
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

	err := query.Find(&procedures).Error
	return procedures, total, err
}

// Update updates a procedure
func (r *ProcedureRepository) Update(procedure *model.Procedure) error {
	return r.db.Save(procedure).Error
}

// Delete deletes a procedure
func (r *ProcedureRepository) Delete(id string) error {
	return r.db.Delete(&model.Procedure{}, "id = ?", id).Error
}

func (r *ProcedureRepository) applyFilters(query *gorm.DB, params *QueryParams) *gorm.DB {
	if len(params.IDs) > 0 {
		query = query.Where("id IN ? OR unique_identifier IN ?", params.IDs, params.IDs)
	}
	if params.Q != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+params.Q+"%", "%"+params.Q+"%")
	}
	return query
}
