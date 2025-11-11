package repository

import (
	"context"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
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
func (r *ProcedureRepository) Create(procedure *domains.Procedure) error {
	return r.db.Create(procedure).Error
}

// GetByID retrieves a procedure by ID
func (r *ProcedureRepository) GetByID(id string) (*domains.Procedure, error) {
	var procedure domains.Procedure
	err := r.db.Where("id = ?", id).First(&procedure).Error
	if err != nil {
		return nil, err
	}
	return &procedure, nil
}

// List retrieves procedures with filtering
func (r *ProcedureRepository) List(params *queryparams.ProceduresQueryParams) ([]*domains.Procedure, int64, error) {
	var procedures []*domains.Procedure
	var total int64

	query := r.db.Model(&domains.Procedure{})
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
func (r *ProcedureRepository) Update(procedure *domains.Procedure) error {
	return r.db.Save(procedure).Error
}

// Delete deletes a procedure
func (r *ProcedureRepository) Delete(id string) error {
	return r.db.Delete(&domains.Procedure{}, "id = ?", id).Error
}

func (r *ProcedureRepository) applyFilters(query *gorm.DB, params *queryparams.ProceduresQueryParams) *gorm.DB {

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

	if params.DateTime != nil {
		// Only add conditions if start/end are not nil
		if params.DateTime.Start != nil && params.DateTime.End != nil {
			query = query.Where("valid_time_start <= ? AND (valid_time_end IS NULL OR valid_time_end >= ?)", params.DateTime.End, params.DateTime.Start)
		} else if params.DateTime.Start != nil {
			query = query.Where("valid_time_end IS NULL OR valid_time_end >= ?", params.DateTime.Start)
		} else if params.DateTime.End != nil {
			query = query.Where("valid_time_start <= ?", params.DateTime.End)
		}
	}

	if len(params.ControlledProperty) > 0 {
		query = query.Joins("JOIN procedure_controlled_properties ON procedures.id = procedure_controlled_properties.procedure_id").
			Where("procedure_controlled_properties.property_id IN ?", params.ControlledProperty)
	}

	if len(params.ObservedProperty) > 0 {
		query = query.Joins("JOIN procedure_observed_properties ON procedures.id = procedure_observed_properties.procedure_id").
			Where("procedure_observed_properties.property_id IN ?", params.ObservedProperty)
	}

	return query
}

// GetByIDs returns procedures keyed by ID or unique identifier
func (r *ProcedureRepository) GetByIDs(ctx context.Context, ids []string) (map[string]*domains.Procedure, error) {
	result := make(map[string]*domains.Procedure)
	if len(ids) == 0 {
		return result, nil
	}

	var procedures []*domains.Procedure
	if err := r.db.WithContext(ctx).Where("id IN ? OR unique_identifier IN ?", ids, ids).Find(&procedures).Error; err != nil {
		return nil, err
	}

	for _, p := range procedures {
		if p == nil {
			continue
		}
		result[p.ID] = p
		if string(p.UniqueIdentifier) != "" {
			result[string(p.UniqueIdentifier)] = p
		}
	}

	return result, nil
}
