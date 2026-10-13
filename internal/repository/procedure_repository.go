package repository

import (
	"context"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
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
func (r *ProcedureRepository) Create(ctx context.Context, procedure *domains.Procedure) error {
	return r.db.WithContext(ctx).Create(procedure).Error
}

// GetByID retrieves a procedure by ID
func (r *ProcedureRepository) GetByID(ctx context.Context, id string) (*domains.Procedure, error) {
	var procedure domains.Procedure
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&procedure).Error
	if err != nil {
		return nil, err
	}
	return &procedure, nil
}

// List retrieves procedures with filtering
func (r *ProcedureRepository) List(ctx context.Context, params *queryparams.ProceduresQueryParams) ([]*domains.Procedure, int64, error) {

	query := r.db.WithContext(ctx).Model(&domains.Procedure{})
	query = r.applyFilters(query, params)

	return listPage(query, &params.QueryParams, idOrder("procedures"), idAnchor[*domains.Procedure])
}

// ListBySystem retrieves procedures associated with a specific system.
func (r *ProcedureRepository) ListBySystem(ctx context.Context, systemID string, params *queryparams.ProceduresQueryParams) ([]*domains.Procedure, int64, error) {

	query := r.db.WithContext(ctx).Model(&domains.Procedure{}).
		Where("procedures.id = (SELECT type_of_id FROM systems WHERE id = ?)", systemID)

	query = r.applyFilters(query, params)

	return listPage(query, &params.QueryParams, idOrder("procedures"), idAnchor[*domains.Procedure])
}

// Update updates a procedure
func (r *ProcedureRepository) Update(ctx context.Context, procedure *domains.Procedure) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return replaceByID(tx, procedure, nil)
	})
}

// Delete deletes a procedure
func (r *ProcedureRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&domains.Procedure{}, "id = ?", id)
	if result.Error != nil {
		if isFKViolation(result.Error) {
			return ErrHasChildren
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ProcedureRepository) applyFilters(query *gorm.DB, params *queryparams.ProceduresQueryParams) *gorm.DB {

	if len(params.IDs) > 0 {
		query = query.Where("procedures.id IN ? OR procedures.unique_identifier IN ?", params.IDs, params.IDs)
	}

	query = applyKeywordFilter(query, params.Q, "procedures.name", "procedures.description")

	query = applyValidTimeFilter(query, "procedures", params.DateTime)

	// EXISTS rather than JOIN so a procedure matching several of the
	// requested properties is still returned once.
	if len(params.ControlledProperty) > 0 {
		query = query.Where(`EXISTS (SELECT 1 FROM procedure_controlled_properties pcp
			WHERE pcp.procedure_id = procedures.id AND pcp.property_id IN ?)`, params.ControlledProperty)
	}

	if len(params.ObservedProperty) > 0 {
		query = query.Where(`EXISTS (SELECT 1 FROM procedure_observed_properties pop
			WHERE pop.procedure_id = procedures.id AND pop.property_id IN ?)`, params.ObservedProperty)
	}

	return applyCQLFilter(query, &params.QueryParams)
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
