package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository/testutil"
	"gorm.io/gorm"
)

// setupTestDB is a helper to set up a test database with PostGIS container
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	ctx := context.Background()

	container := testutil.StartPostGISContainer(ctx, t)

	db := testutil.OpenTestDB(t, container.DSN, testutil.OpenTestDBOptions{
		EnableLogging: false, // Set to true for debugging
		Models:        testutil.DefaultSystemModels(),
	})

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		container.Terminate(ctx)
	}

	return db, cleanup
}

func TestSystemRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewSystemRepository(db)

	tests := []struct {
		name      string
		system    *domains.System
		wantErr   bool
		checkFunc func(t *testing.T, created *domains.System)
	}{
		{
			name: "create sensor system",
			system: &domains.System{
				CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sensor1", Name: "Temperature Sensor"},
				SystemType: domains.SystemTypeSensor,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.System) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Temperature Sensor", created.Name)
				require.Equal(t, domains.SystemTypeSensor, created.SystemType)
			},
		},
		{
			name: "create subsystem",
			system: &domains.System{
				CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:subsystem1", Name: "Temperature Sensor"},
				SystemType:     domains.SystemTypeSensor,
				ParentSystemID: testutil.PtrStr("parent-system-id"),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.System) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Temperature Sensor", created.Name)
				require.Equal(t, domains.SystemTypeSensor, created.SystemType)
				require.Equal(t, "parent-system-id", *created.ParentSystemID)
			},
		},
		{
			name: "create platform system with geometry",
			system: &domains.System{
				CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:platform1", Name: "Weather Station", Description: "Test platform"},
				SystemType: domains.SystemTypePlatform,
				ValidTime: &common_shared.TimeRange{
					Start: testutil.PtrTime(time.Now()),
					End:   testutil.PtrTime(time.Now().Add(24 * time.Hour)),
				},
				AssetType: testutil.PtrStr("Platform"),
				Geometry:  testutil.MakePoint(-122.4194, 37.7749), // San Francisco
				Links: common_shared.Links{
					{Rel: "self", Href: "http://example.com/systems/1"},
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.System) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Weather Station", created.Name)
				require.Equal(t, domains.SystemTypePlatform, created.SystemType)
				require.NotNil(t, created.ValidTime)
				require.Equal(t, "Platform", *created.AssetType)
				require.NotNil(t, created.Geometry)
				require.NotNil(t, created.Geometry.T)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(tt.system)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.system)
			}
		})
	}
}

func TestSystemRepository_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewSystemRepository(db)

	// Setup: create test systems
	sys1 := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:get1", Name: "System 1"},
		SystemType: domains.SystemTypeSensor,
	}
	require.NoError(t, repo.Create(sys1))

	tests := []struct {
		name     string
		id       string
		wantName string
		wantErr  bool
	}{
		{
			name:     "get existing system",
			id:       sys1.ID,
			wantName: "System 1",
			wantErr:  false,
		},
		{
			name:     "get non-existent system",
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

func TestSystemRepository_GetByUniqueIdentifier(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewSystemRepository(db)

	// Setup: create test systems
	sys1 := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:get1", Name: "System 1"},
		SystemType: domains.SystemTypeSensor,
	}
	require.NoError(t, repo.Create(sys1))

	tests := []struct {
		name     string
		id       string
		wantName string
		wantErr  bool
	}{
		{
			name:     "get existing system",
			id:       string(sys1.UniqueIdentifier),
			wantName: "System 1",
			wantErr:  false,
		},
		{
			name:     "get non-existent system",
			id:       "non-existent-id",
			wantName: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByUID(tt.id)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantName, got.Name)
		})
	}
}

func TestSystemRepository_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewSystemRepository(db)

	// Setup: create multiple test systems with different attributes
	sensor1 := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sensor1", Name: "Temperature Sensor 1"},
		SystemType: domains.SystemTypeSensor,
		Geometry:   testutil.MakePoint(-122.4194, 37.7749), // San Francisco
		// Always valid
		ValidTime: &common_shared.TimeRange{Start: testutil.PtrTime(time.Now())},
	}
	require.NoError(t, repo.Create(sensor1))

	sensor2 := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sensor2", Name: "Humidity Sensor"},
		SystemType: domains.SystemTypeSensor,
		Geometry:   testutil.MakePoint(-118.2437, 34.0522), // Los Angeles
		// Always valid
		ValidTime: &common_shared.TimeRange{Start: testutil.PtrTime(time.Now())},
	}
	require.NoError(t, repo.Create(sensor2))

	platform1 := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:platform1", Name: "Weather Station"},
		SystemType: domains.SystemTypePlatform,
		Geometry:   testutil.MakePoint(-122.3321, 47.6062), // Seattle
		// Only valid for november 2025 to december 2025
		ValidTime: &common_shared.TimeRange{Start: testutil.PtrTime(time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)), End: testutil.PtrTime(time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC))},
	}
	require.NoError(t, repo.Create(platform1))

	actuator1 := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:actuator1", Name: "Valve Controller"},
		SystemType: domains.SystemTypeActuator,
		// Valid whenever but only until end of 2024
		ValidTime: &common_shared.TimeRange{End: testutil.PtrTime(time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC))},
	}
	require.NoError(t, repo.Create(actuator1))

	// Create a child system
	childSensor := &domains.System{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:child1", Name: "Child Sensor"},
		SystemType:     domains.SystemTypeSensor,
		ParentSystemID: &platform1.ID,
		// No valid time
	}
	require.NoError(t, repo.Create(childSensor))

	tests := []struct {
		name      string
		params    *queryparams.SystemQueryParams
		wantCount int
		wantTotal int64
		checkFunc func(t *testing.T, systems []*domains.System)
	}{
		{
			name: "list all systems with default limit",
			params: &queryparams.SystemQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
				Recursive:   true,
			},
			wantCount: 5,
			wantTotal: 5,
			checkFunc: func(t *testing.T, systems []*domains.System) {
				require.Len(t, systems, 5)
			},
		},
		{
			name: "list with limit 2",
			params: &queryparams.SystemQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 2},
				Recursive:   true,
			},
			wantCount: 2,
			wantTotal: 5,
			checkFunc: func(t *testing.T, systems []*domains.System) {
				require.Len(t, systems, 2)
			},
		},
		{
			name: "list with limit and offset",
			params: &queryparams.SystemQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 2, Offset: 2},
				Recursive:   true,
			},
			wantCount: 2,
			wantTotal: 5,
			checkFunc: func(t *testing.T, systems []*domains.System) {
				require.Len(t, systems, 2)
			},
		},
		{
			name: "filter by specific IDs",
			params: &queryparams.SystemQueryParams{
				QueryParams: queryparams.QueryParams{
					IDs:   []string{sensor1.ID, sensor2.ID},
					Limit: 10,
				},
			},
			wantCount: 2,
			wantTotal: 2,
			checkFunc: func(t *testing.T, systems []*domains.System) {
				require.Len(t, systems, 2)
				ids := []string{systems[0].ID, systems[1].ID}
				require.Contains(t, ids, sensor1.ID)
				require.Contains(t, ids, sensor2.ID)
			},
		},
		{
			name: "filter by parent system",
			params: &queryparams.SystemQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
				Parent:      []string{platform1.ID},
				Recursive:   true,
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, systems []*domains.System) {
				require.Len(t, systems, 1)
				require.Equal(t, childSensor.ID, systems[0].ID)
				require.Equal(t, "Child Sensor", systems[0].Name)
			},
		},
		{
			name: "in bbox filter",
			params: &queryparams.SystemQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
				Bbox:        testutil.TestBoundingBoxLA(),
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, systems []*domains.System) {
				require.Len(t, systems, 1)
				require.Equal(t, sensor2.ID, systems[0].ID)
				require.Equal(t, "Humidity Sensor", systems[0].Name)
				require.Equal(t, sensor2.Geometry, systems[0].Geometry)
			},
		},
		{
			name: "Query test",
			params: &queryparams.SystemQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
				Q:           []string{"Humidity", "Controller"},
			},
			wantCount: 2,
			wantTotal: 2,
			checkFunc: func(t *testing.T, systems []*domains.System) {
				require.Len(t, systems, 2)

				names := []string{systems[0].Name, systems[1].Name}

				require.Contains(t, names, "Humidity Sensor")
				require.Contains(t, names, "Valve Controller")
			},
		},
		{
			name: "Datetime test",
			params: &queryparams.SystemQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
				// Current time falls within platform1 valid times but only that one
				Datetime: &common_shared.TimeRange{
					Start: testutil.PtrTime(time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC)),
					End:   testutil.PtrTime(time.Date(2025, 11, 4, 0, 0, 0, 0, time.UTC)),
				},
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, systems []*domains.System) {
				require.Len(t, systems, 1)

				names := []string{systems[0].Name}

				require.Contains(t, names, "Weather Station")
			},
		},
		{
			name: "Geom test",
			params: &queryparams.SystemQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
				// WKT format of geometry around Seattle
				Geom: "POINT(-122.3321 47.6062)",
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, systems []*domains.System) {
				require.Len(t, systems, 1)

				names := []string{systems[0].Name}

				require.Contains(t, names, "Weather Station")
			},
		},
		{
			name: "in bbox filter multiple results",
			params: &queryparams.SystemQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
				Bbox:        testutil.TestBoundingBoxLA_SF(),
			},
			wantCount: 2,
			wantTotal: 2,
			checkFunc: func(t *testing.T, systems []*domains.System) {
				require.Len(t, systems, 2)
			},
		},
		{
			name: "empty result set",
			params: &queryparams.SystemQueryParams{
				QueryParams: queryparams.QueryParams{
					IDs:   []string{"non-existent-id"},
					Limit: 10,
				},
			},
			wantCount: 0,
			wantTotal: 0,
			checkFunc: func(t *testing.T, systems []*domains.System) {
				require.Empty(t, systems)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			systems, total, err := repo.List(tt.params)
			require.NoError(t, err)
			require.Equal(t, tt.wantTotal, total, "total count mismatch")
			require.Len(t, systems, tt.wantCount, "result count mismatch")
			if tt.checkFunc != nil {
				tt.checkFunc(t, systems)
			}
		})
	}
}

func TestSystemRepository_DeeplyNestedSystems(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewSystemRepository(db)

	// Setup: create multiple test systems with different attributes
	sensor1 := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sensor1", Name: "Temperature Sensor 1"},
		SystemType: domains.SystemTypeSensor,
		Geometry:   testutil.MakePoint(-122.4194, 37.7749), // San Francisco
		// Always valid
		ValidTime: &common_shared.TimeRange{Start: testutil.PtrTime(time.Now())},
	}
	require.NoError(t, repo.Create(sensor1))

	sensor2 := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sensor2", Name: "Humidity Sensor"},
		SystemType: domains.SystemTypeSensor,
		Geometry:   testutil.MakePoint(-118.2437, 34.0522), // Los Angeles
		// Always valid
		ValidTime: &common_shared.TimeRange{Start: testutil.PtrTime(time.Now())},
	}
	require.NoError(t, repo.Create(sensor2))

	platform1 := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:platform1", Name: "Weather Station"},
		SystemType: domains.SystemTypePlatform,
		Geometry:   testutil.MakePoint(-122.3321, 47.6062), // Seattle
		// Only valid for november 2025 to december 2025
		ValidTime: &common_shared.TimeRange{Start: testutil.PtrTime(time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)), End: testutil.PtrTime(time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC))},
	}
	require.NoError(t, repo.Create(platform1))

	// Create a child system
	childSensor := &domains.System{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:child1", Name: "Child Sensor"},
		SystemType:     domains.SystemTypeSensor,
		ParentSystemID: &platform1.ID,
		// No valid time
	}

	require.NoError(t, repo.Create(childSensor))

	grandchildSensor := &domains.System{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:grandchild1", Name: "Grandchild Sensor"},
		SystemType:     domains.SystemTypeSensor,
		ParentSystemID: &childSensor.ID,
		// No valid time
	}

	require.NoError(t, repo.Create(grandchildSensor))

	tests := []struct {
		name      string
		params    *queryparams.SystemQueryParams
		wantCount int
		wantTotal int64
		checkFunc func(t *testing.T, systems []*domains.System)
	}{
		{
			name: "Finding the grandchild system via recursive listing",
			params: &queryparams.SystemQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
				Q:           []string{"Grandchild"},
				Recursive:   true,
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, systems []*domains.System) {
				require.Len(t, systems, 1)
			},
		},
		{
			name: "Finding the grandchild system but non-recursively (so should not find it)",
			params: &queryparams.SystemQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
				Q:           []string{"Grandchild"},
				Recursive:   false,
			},
			wantCount: 0,
			wantTotal: 0,
			checkFunc: func(t *testing.T, systems []*domains.System) {
				require.Len(t, systems, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			systems, total, err := repo.List(tt.params)
			require.NoError(t, err)
			require.Equal(t, tt.wantTotal, total, "total count mismatch")
			require.Len(t, systems, tt.wantCount, "result count mismatch")
			if tt.checkFunc != nil {
				tt.checkFunc(t, systems)
			}
		})
	}
}

func TestSystemRepository_Hierarchy(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewSystemRepository(db)

	tests := []struct {
		name      string
		setupFunc func() (parent *domains.System, children []*domains.System)
		testFunc  func(t *testing.T, parent *domains.System, children []*domains.System)
	}{
		{
			name: "parent with single child",
			setupFunc: func() (parent *domains.System, children []*domains.System) {
				parent = &domains.System{
					CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:parent1", Name: "Parent Platform"},
					SystemType: domains.SystemTypePlatform,
				}
				require.NoError(t, repo.Create(parent))

				child := &domains.System{
					CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:child1", Name: "Child Sensor"},
					SystemType:     domains.SystemTypeSensor,
					ParentSystemID: &parent.ID,
				}
				require.NoError(t, repo.Create(child))
				return parent, []*domains.System{child}
			},
			testFunc: func(t *testing.T, parent *domains.System, children []*domains.System) {
				has, err := repo.HasSubsystems(parent.ID)
				require.NoError(t, err)
				require.True(t, has)

				subs, err := repo.GetSubsystems(parent.ID, false)
				require.NoError(t, err)
				require.Len(t, subs, 1)
				require.Equal(t, children[0].ID, subs[0].ID)
			},
		},
		{
			name: "parent with multiple children",
			setupFunc: func() (parent *domains.System, children []*domains.System) {
				parent = &domains.System{
					CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:parent2", Name: "Multi Parent"},
					SystemType: domains.SystemTypePlatform,
				}
				require.NoError(t, repo.Create(parent))

				for i := 0; i < 3; i++ {
					child := &domains.System{
						CommonSSN:      domains.CommonSSN{UniqueIdentifier: domains.UniqueID("urn:test:child" + string(rune(i+2))), Name: "Child " + string(rune(i+1))},
						SystemType:     domains.SystemTypeSensor,
						ParentSystemID: &parent.ID,
					}
					require.NoError(t, repo.Create(child))
					children = append(children, child)
				}
				return parent, children
			},
			testFunc: func(t *testing.T, parent *domains.System, children []*domains.System) {
				subs, err := repo.GetSubsystems(parent.ID, false)
				require.NoError(t, err)
				require.Len(t, subs, 3)
			},
		},
		{
			name: "system with no children",
			setupFunc: func() (parent *domains.System, children []*domains.System) {
				parent = &domains.System{
					CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:lone", Name: "Lone System"},
					SystemType: domains.SystemTypeSensor,
				}
				require.NoError(t, repo.Create(parent))
				return parent, nil
			},
			testFunc: func(t *testing.T, parent *domains.System, children []*domains.System) {
				has, err := repo.HasSubsystems(parent.ID)
				require.NoError(t, err)
				require.False(t, has)

				subs, err := repo.GetSubsystems(parent.ID, false)
				require.NoError(t, err)
				require.Len(t, subs, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent, children := tt.setupFunc()
			tt.testFunc(t, parent, children)
		})
	}
}

func TestSystemRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewSystemRepository(db)

	tests := []struct {
		name      string
		setupFunc func() (parentID string, childIDs []string)
		cascade   bool
		checkFunc func(t *testing.T, parentID string, childIDs []string)
	}{
		{
			name: "delete with cascade removes children",
			setupFunc: func() (parentID string, childIDs []string) {
				parent := &domains.System{
					CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:delparent1", Name: "Delete Parent"},
					SystemType: domains.SystemTypePlatform,
				}
				require.NoError(t, repo.Create(parent))

				child := &domains.System{
					CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:delchild1", Name: "Delete Child"},
					SystemType:     domains.SystemTypeSensor,
					ParentSystemID: &parent.ID,
				}
				require.NoError(t, repo.Create(child))
				return parent.ID, []string{child.ID}
			},
			cascade: true,
			checkFunc: func(t *testing.T, parentID string, childIDs []string) {
				// Parent should be gone
				_, err := repo.GetByID(parentID)
				require.Error(t, err)

				// Children should also be gone
				for _, childID := range childIDs {
					_, err := repo.GetByID(childID)
					require.Error(t, err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parentID, childIDs := tt.setupFunc()
			err := repo.Delete(parentID, tt.cascade)
			require.NoError(t, err)
			tt.checkFunc(t, parentID, childIDs)
		})
	}
}

func TestSystemRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewSystemRepository(db)

	// Setup: create multiple test systems with different attributes
	sensor1 := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sensor1", Name: "Temperature Sensor 1"},
		SystemType: domains.SystemTypeSensor,
		Geometry:   testutil.MakePoint(-122.4194, 37.7749), // San Francisco
		// Always valid
		ValidTime: &common_shared.TimeRange{Start: testutil.PtrTime(time.Now())},
	}
	require.NoError(t, repo.Create(sensor1))

	tests := []struct {
		name      string
		systemId  string
		system    *domains.System
		wantErr   bool
		checkFunc func(t *testing.T, updated *domains.System)
	}{
		{
			name:     "update sensor system",
			systemId: sensor1.ID,
			system: &domains.System{
				CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sensor1", Name: "Temperature Sensor 1"},
				SystemType: domains.SystemTypeSensor,
				Geometry:   testutil.MakePoint(-122.3321, 47.6062), // Seattle
				// Always valid
				ValidTime: &common_shared.TimeRange{Start: testutil.PtrTime(time.Now())},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, updated *domains.System) {
				require.NotEmpty(t, updated.ID)
				require.Equal(t, "Temperature Sensor 1", updated.Name)
				require.Equal(t, domains.SystemTypeSensor, updated.SystemType)
				require.Equal(t, updated.Geometry, testutil.MakePoint(-122.3321, 47.6062))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Update(tt.systemId, tt.system)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.system)
			}
		})
	}
}

func TestSystemRepository_HasSubsystems(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewSystemRepository(db)

	platform1 := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:platform1", Name: "Platform 1"},
		SystemType: domains.SystemTypePlatform,
	}
	require.NoError(t, repo.Create(platform1))

	actuator1 := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:actuator1", Name: "Valve Controller"},
		SystemType: domains.SystemTypeActuator,
		// Valid whenever but only until end of 2024
		ValidTime: &common_shared.TimeRange{End: testutil.PtrTime(time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC))},
	}
	require.NoError(t, repo.Create(actuator1))

	// Create a child system
	childSensor := &domains.System{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:child1", Name: "Child Sensor"},
		SystemType:     domains.SystemTypeSensor,
		ParentSystemID: &actuator1.ID,
		// No valid time
	}
	require.NoError(t, repo.Create(childSensor))

	tests := []struct {
		name     string
		parentID string
		wantHas  bool
	}{
		{
			name:     "has subsystems",
			parentID: actuator1.ID,
			wantHas:  true,
		},
		{
			name:     "has no subsystems",
			parentID: platform1.ID,
			wantHas:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has, err := repo.HasSubsystems(tt.parentID)
			require.NoError(t, err)
			require.Equal(t, tt.wantHas, has)
		})
	}
}
