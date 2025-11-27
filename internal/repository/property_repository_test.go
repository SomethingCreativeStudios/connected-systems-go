package repository

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository/testutil"
)

func TestPropertyRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPropertyRepository(db)

	tests := []struct {
		name      string
		property  *domains.Property
		wantErr   bool
		checkFunc func(t *testing.T, created *domains.Property)
	}{
		{
			name: "create observable property",
			property: &domains.Property{
				CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:property1", Name: "Temperature"},
				PropertyType: "ObservableProperty",
				ObjectType:   testutil.PtrStr("sensor"),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.Property) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Temperature", created.Name)
				require.Equal(t, "ObservableProperty", created.PropertyType)
				require.NotNil(t, created.ObjectType)
				require.Equal(t, "sensor", *created.ObjectType)
			},
		},
		{
			name: "create actuable property",
			property: &domains.Property{
				CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:property2", Name: "Valve Position", Description: "Control valve position"},
				PropertyType: "ActuableProperty",
				ObjectType:   testutil.PtrStr("actuator"),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.Property) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Valve Position", created.Name)
				require.Equal(t, "ActuableProperty", created.PropertyType)
				require.Equal(t, "Control valve position", created.Description)
			},
		},
		{
			name: "create property with definition",
			property: &domains.Property{
				CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:property3", Name: "Humidity"},
				PropertyType: "ObservableProperty",
				Definition:   "http://vocab.example.org/humidity",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.Property) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Humidity", created.Name)
				require.Equal(t, "http://vocab.example.org/humidity", created.Definition)
			},
		},
		{
			name: "create property with qualifiers",
			property: &domains.Property{
				CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:property4", Name: "Boolean Qualifier"},
				PropertyType: "ObservableProperty",
				Qualifiers: []common_shared.ComponentWrapper{
					{
						Type:       "Boolean",
						Label:      "Some basic boolean value",
						Definition: "http://vocab.example.org/humidity",
						Value:      json.RawMessage(`true`),
						Updatable:  testutil.PtrBool(true),
						Optional:   testutil.PtrBool(false),
					},
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.Property) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Boolean Qualifier", created.Name)
				require.Equal(t, 1, len(created.Qualifiers))

				qualifier := created.Qualifiers[0]
				require.Equal(t, "Boolean", qualifier.Type)
				require.Equal(t, "Some basic boolean value", qualifier.Label)
				require.Equal(t, "http://vocab.example.org/humidity", qualifier.Definition)
				require.Equal(t, json.RawMessage(`true`), qualifier.Value)

				require.NotNil(t, qualifier.Updatable)
				require.True(t, *qualifier.Updatable)
				require.NotNil(t, qualifier.Optional)
				require.False(t, *qualifier.Optional)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(tt.property)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.property)
			}
		})
	}
}

func TestPropertyRepository_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPropertyRepository(db)

	// Setup: create test property
	prop1 := &domains.Property{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:get1", Name: "Property 1"},
		PropertyType: "ObservableProperty",
	}
	require.NoError(t, repo.Create(prop1))

	tests := []struct {
		name     string
		id       string
		wantName string
		wantErr  bool
	}{
		{
			name:     "get existing property",
			id:       prop1.ID,
			wantName: "Property 1",
			wantErr:  false,
		},
		{
			name:     "get non-existent property",
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

func TestPropertyRepository_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPropertyRepository(db)

	// Setup: create multiple test properties
	prop1 := &domains.Property{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:prop1", Name: "Temperature"},
		PropertyType: "ObservableProperty",
		ObjectType:   testutil.PtrStr("sensor"),
	}
	require.NoError(t, repo.Create(prop1))

	prop2 := &domains.Property{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:prop2", Name: "Humidity"},
		PropertyType: "ObservableProperty",
		ObjectType:   testutil.PtrStr("sensor"),
	}
	require.NoError(t, repo.Create(prop2))

	prop3 := &domains.Property{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:prop3", Name: "Valve State", Description: "Binary valve control"},
		PropertyType: "ActuableProperty",
		ObjectType:   testutil.PtrStr("actuator"),
	}
	require.NoError(t, repo.Create(prop3))

	tests := []struct {
		name      string
		params    *queryparams.PropertiesQueryParams
		wantCount int
		wantTotal int64
		checkFunc func(t *testing.T, properties []*domains.Property)
	}{
		{
			name: "list all properties",
			params: &queryparams.PropertiesQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
			},
			wantCount: 3,
			wantTotal: 3,
			checkFunc: func(t *testing.T, properties []*domains.Property) {
				require.Len(t, properties, 3)
			},
		},
		{
			name: "list with limit 2",
			params: &queryparams.PropertiesQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 2},
			},
			wantCount: 2,
			wantTotal: 3,
			checkFunc: func(t *testing.T, properties []*domains.Property) {
				require.Len(t, properties, 2)
			},
		},
		{
			name: "filter by specific IDs",
			params: &queryparams.PropertiesQueryParams{
				QueryParams: queryparams.QueryParams{
					IDs:   []string{"urn:test:prop1", "urn:test:prop2"},
					Limit: 10,
				},
			},
			wantCount: 2,
			wantTotal: 2,
			checkFunc: func(t *testing.T, properties []*domains.Property) {
				require.Len(t, properties, 2)
				names := []string{properties[0].Name, properties[1].Name}
				require.Contains(t, names, "Temperature")
				require.Contains(t, names, "Humidity")
			},
		},
		{
			name: "filter by object type - sensor",
			params: &queryparams.PropertiesQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
				ObjectType:  []string{"sensor"},
			},
			wantCount: 2,
			wantTotal: 2,
			checkFunc: func(t *testing.T, properties []*domains.Property) {
				require.Len(t, properties, 2)
				for _, prop := range properties {
					require.NotNil(t, prop.ObjectType)
					require.Equal(t, "sensor", *prop.ObjectType)
				}
			},
		},
		{
			name: "filter by object type - actuator",
			params: &queryparams.PropertiesQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
				ObjectType:  []string{"actuator"},
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, properties []*domains.Property) {
				require.Len(t, properties, 1)
				require.Equal(t, "Valve State", properties[0].Name)
			},
		},
		{
			name: "query test - search by name",
			params: &queryparams.PropertiesQueryParams{
				QueryParams: queryparams.QueryParams{
					Q:     []string{"Humidity"},
					Limit: 10,
				},
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, properties []*domains.Property) {
				require.Len(t, properties, 1)
				require.Equal(t, "Humidity", properties[0].Name)
			},
		},
		{
			name: "query test - search by property type",
			params: &queryparams.PropertiesQueryParams{
				QueryParams: queryparams.QueryParams{
					Q:     []string{"ActuableProperty"},
					Limit: 10,
				},
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, properties []*domains.Property) {
				require.Len(t, properties, 1)
				require.Equal(t, "Valve State", properties[0].Name)
			},
		},
		{
			name: "empty result set",
			params: &queryparams.PropertiesQueryParams{
				QueryParams: queryparams.QueryParams{
					IDs:   []string{"non-existent-id"},
					Limit: 10,
				},
			},
			wantCount: 0,
			wantTotal: 0,
			checkFunc: func(t *testing.T, properties []*domains.Property) {
				require.Empty(t, properties)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			properties, total, err := repo.List(tt.params)
			require.NoError(t, err)
			require.Equal(t, tt.wantTotal, total, "total count mismatch")
			require.Len(t, properties, tt.wantCount, "result count mismatch")
			if tt.checkFunc != nil {
				tt.checkFunc(t, properties)
			}
		})
	}
}

func TestPropertyRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPropertyRepository(db)

	// Setup: create a property
	original := &domains.Property{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:update1", Name: "Original Name"},
		PropertyType: "ObservableProperty",
	}
	require.NoError(t, repo.Create(original))

	tests := []struct {
		name       string
		updateFunc func(*domains.Property)
		checkFunc  func(t *testing.T, updated *domains.Property)
		wantErr    bool
	}{
		{
			name: "update name",
			updateFunc: func(p *domains.Property) {
				p.Name = "Updated Name"
			},
			checkFunc: func(t *testing.T, updated *domains.Property) {
				require.Equal(t, "Updated Name", updated.Name)
			},
			wantErr: false,
		},
		{
			name: "update description",
			updateFunc: func(p *domains.Property) {
				p.Description = "New description"
			},
			checkFunc: func(t *testing.T, updated *domains.Property) {
				require.Equal(t, "New description", updated.Description)
			},
			wantErr: false,
		},
		{
			name: "update object type",
			updateFunc: func(p *domains.Property) {
				p.ObjectType = testutil.PtrStr("sensor")
			},
			checkFunc: func(t *testing.T, updated *domains.Property) {
				require.NotNil(t, updated.ObjectType)
				require.Equal(t, "sensor", *updated.ObjectType)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get fresh copy for each test
			prop, err := repo.GetByID(original.ID)
			require.NoError(t, err)

			// Apply update
			tt.updateFunc(prop)

			// Save
			err = repo.Update(prop)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify
			updated, err := repo.GetByID(prop.ID)
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, updated)
			}
		})
	}
}

func TestPropertyRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewPropertyRepository(db)

	tests := []struct {
		name      string
		setupFunc func() string
		checkFunc func(t *testing.T, id string)
	}{
		{
			name: "delete existing property",
			setupFunc: func() string {
				prop := &domains.Property{
					CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:del1", Name: "Delete Me"},
					PropertyType: "ObservableProperty",
				}
				require.NoError(t, repo.Create(prop))
				return prop.ID
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
