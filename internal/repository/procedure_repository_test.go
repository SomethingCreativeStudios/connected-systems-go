package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
)

func TestProcedureRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProcedureRepository(db)

	tests := []struct {
		name      string
		procedure *domains.Procedure
		wantErr   bool
		checkFunc func(t *testing.T, created *domains.Procedure)
	}{
		{
			name: "create method procedure",
			procedure: &domains.Procedure{
				CommonSSN:     domains.CommonSSN{UniqueIdentifier: "urn:test:procedure1", Name: "Temperature Measurement Method"},
				ProcedureType: domains.ProcedureTypeActuating,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.Procedure) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Temperature Measurement Method", created.Name)
				require.Equal(t, domains.ProcedureTypeActuating, created.ProcedureType)
			},
		},
		{
			name: "create datasheet procedure",
			procedure: &domains.Procedure{
				CommonSSN:     domains.CommonSSN{UniqueIdentifier: "urn:test:procedure2", Name: "Sensor Datasheet", Description: "Manufacturer datasheet"},
				ProcedureType: "Datasheet",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.Procedure) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Sensor Datasheet", created.Name)
				require.Equal(t, "Datasheet", created.ProcedureType)
				require.Equal(t, "Manufacturer datasheet", created.Description)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(tt.procedure)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.procedure)
			}
		})
	}
}

func TestProcedureRepository_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProcedureRepository(db)

	// Setup: create test procedure
	proc1 := &domains.Procedure{
		CommonSSN:     domains.CommonSSN{UniqueIdentifier: "urn:test:get1", Name: "Procedure 1"},
		ProcedureType: "Method",
	}
	require.NoError(t, repo.Create(proc1))

	tests := []struct {
		name     string
		id       string
		wantName string
		wantErr  bool
	}{
		{
			name:     "get existing procedure",
			id:       proc1.ID,
			wantName: "Procedure 1",
			wantErr:  false,
		},
		{
			name:     "get non-existent procedure",
			id:       "non-existent-id",
			wantName: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByID(tt.id)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantName, got.Name)
		})
	}
}

func TestProcedureRepository_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProcedureRepository(db)

	// Setup: create multiple test procedures
	proc1 := &domains.Procedure{
		CommonSSN:     domains.CommonSSN{UniqueIdentifier: "urn:test:proc1", Name: "Temperature Method"},
		ProcedureType: domains.ProcedureTypeActuating,
	}
	require.NoError(t, repo.Create(proc1))

	proc2 := &domains.Procedure{
		CommonSSN:     domains.CommonSSN{UniqueIdentifier: "urn:test:proc2", Name: "Humidity Method"},
		ProcedureType: domains.ProcedureTypeObserving,
	}
	require.NoError(t, repo.Create(proc2))

	proc3 := &domains.Procedure{
		CommonSSN:     domains.CommonSSN{UniqueIdentifier: "urn:test:proc3", Name: "Sensor Datasheet", Description: "Complete specifications"},
		ProcedureType: "Datasheet",
	}
	require.NoError(t, repo.Create(proc3))

	// Create basic properties for testing
	property1 := &domains.Property{
		CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:property1", Name: "Temperature"},
	}
	require.NoError(t, repo.db.Create(property1).Error)

	property2 := &domains.Property{
		CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:property2", Name: "Humidity"},
	}
	require.NoError(t, repo.db.Create(property2).Error)

	// Make a value in the pivot table procedure_observed_properties
	require.NoError(t, repo.db.Exec("INSERT INTO procedure_observed_properties (procedure_id, property_id) VALUES (?, ?)", proc1.ID, property1.ID).Error)

	// Make a value in the pivot table procedure_controlled_properties
	require.NoError(t, repo.db.Exec("INSERT INTO procedure_controlled_properties (procedure_id, property_id) VALUES (?, ?)", proc2.ID, property2.ID).Error)

	tests := []struct {
		name      string
		params    *queryparams.ProceduresQueryParams
		wantCount int
		wantTotal int64
		checkFunc func(t *testing.T, procedures []*domains.Procedure)
	}{
		{
			name: "list all procedures",
			params: &queryparams.ProceduresQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
			},
			wantCount: 3,
			wantTotal: 3,
			checkFunc: func(t *testing.T, procedures []*domains.Procedure) {
				require.Len(t, procedures, 3)
			},
		},
		{
			name: "list with limit 2",
			params: &queryparams.ProceduresQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 2},
			},
			wantCount: 2,
			wantTotal: 3,
			checkFunc: func(t *testing.T, procedures []*domains.Procedure) {
				require.Len(t, procedures, 2)
			},
		},
		{
			name: "filter by specific IDs",
			params: &queryparams.ProceduresQueryParams{
				QueryParams: queryparams.QueryParams{
					IDs:   []string{proc1.ID, proc2.ID},
					Limit: 10,
				},
			},
			wantCount: 2,
			wantTotal: 2,
			checkFunc: func(t *testing.T, procedures []*domains.Procedure) {
				require.Len(t, procedures, 2)
				ids := []string{procedures[0].ID, procedures[1].ID}
				require.Contains(t, ids, proc1.ID)
				require.Contains(t, ids, proc2.ID)
			},
		},
		{
			name: "filter by controlled property",
			params: &queryparams.ProceduresQueryParams{
				QueryParams:        queryparams.QueryParams{Limit: 10},
				ControlledProperty: []string{property2.ID},
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, procedures []*domains.Procedure) {
				require.Len(t, procedures, 1)
				require.Equal(t, "Humidity Method", procedures[0].Name)
			},
		},
		{
			name: "query test - search by name",
			params: &queryparams.ProceduresQueryParams{
				QueryParams: queryparams.QueryParams{
					Q:     []string{"Humidity"},
					Limit: 10,
				},
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, procedures []*domains.Procedure) {
				require.Len(t, procedures, 1)
				require.Equal(t, "Humidity Method", procedures[0].Name)
			},
		},
		{
			name: "query test - search by description",
			params: &queryparams.ProceduresQueryParams{
				QueryParams: queryparams.QueryParams{
					Q:     []string{"specifications"},
					Limit: 10,
				},
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, procedures []*domains.Procedure) {
				require.Len(t, procedures, 1)
				require.Equal(t, "Sensor Datasheet", procedures[0].Name)
			},
		},
		{
			name: "empty result set",
			params: &queryparams.ProceduresQueryParams{
				QueryParams: queryparams.QueryParams{
					IDs:   []string{"non-existent-id"},
					Limit: 10,
				},
			},
			wantCount: 0,
			wantTotal: 0,
			checkFunc: func(t *testing.T, procedures []*domains.Procedure) {
				require.Empty(t, procedures)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			procedures, total, err := repo.List(tt.params)
			require.NoError(t, err)
			require.Equal(t, tt.wantTotal, total, "total count mismatch")
			require.Len(t, procedures, tt.wantCount, "result count mismatch")
			if tt.checkFunc != nil {
				tt.checkFunc(t, procedures)
			}
		})
	}
}

func TestProcedureRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProcedureRepository(db)

	// Setup: create a procedure
	original := &domains.Procedure{
		CommonSSN:     domains.CommonSSN{UniqueIdentifier: "urn:test:update1", Name: "Original Name"},
		ProcedureType: "Method",
	}
	require.NoError(t, repo.Create(original))

	tests := []struct {
		name       string
		updateFunc func(*domains.Procedure)
		checkFunc  func(t *testing.T, updated *domains.Procedure)
		wantErr    bool
	}{
		{
			name: "update name",
			updateFunc: func(p *domains.Procedure) {
				p.Name = "Updated Name"
			},
			checkFunc: func(t *testing.T, updated *domains.Procedure) {
				require.Equal(t, "Updated Name", updated.Name)
			},
			wantErr: false,
		},
		{
			name: "update description",
			updateFunc: func(p *domains.Procedure) {
				p.Description = "New description"
			},
			checkFunc: func(t *testing.T, updated *domains.Procedure) {
				require.Equal(t, "New description", updated.Description)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get fresh copy for each test
			proc, err := repo.GetByID(original.ID)
			require.NoError(t, err)

			// Apply update
			tt.updateFunc(proc)

			// Save
			err = repo.Update(proc)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify
			updated, err := repo.GetByID(proc.ID)
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, updated)
			}
		})
	}
}

func TestProcedureRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewProcedureRepository(db)

	tests := []struct {
		name      string
		setupFunc func() string
		checkFunc func(t *testing.T, id string)
	}{
		{
			name: "delete existing procedure",
			setupFunc: func() string {
				proc := &domains.Procedure{
					CommonSSN:     domains.CommonSSN{UniqueIdentifier: "urn:test:del1", Name: "Delete Me"},
					ProcedureType: "Method",
				}
				require.NoError(t, repo.Create(proc))
				return proc.ID
			},
			checkFunc: func(t *testing.T, id string) {
				_, err := repo.GetByID(id)
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.setupFunc()
			err := repo.Delete(id)
			require.NoError(t, err)
			tt.checkFunc(t, id)
		})
	}
}
