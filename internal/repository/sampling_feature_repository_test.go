package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository/testutil"
)

func TestSamplingFeatureRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewSamplingFeatureRepository(db)

	tests := []struct {
		name            string
		samplingFeature *domains.SamplingFeature
		wantErr         bool
		checkFunc       func(t *testing.T, created *domains.SamplingFeature)
	}{
		{
			name: "create point sampling feature",
			samplingFeature: &domains.SamplingFeature{
				CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:sf1", Name: "Sampling Point 1"},
				FeatureType: "Point",
				Geometry:    testutil.MakePoint(-122.4194, 37.7749),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.SamplingFeature) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Sampling Point 1", created.Name)
				require.Equal(t, "Point", created.FeatureType)
				require.NotNil(t, created.Geometry)
			},
		},
		{
			name: "create curve sampling feature",
			samplingFeature: &domains.SamplingFeature{
				CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:sf2", Name: "Sampling Curve", Description: "Test curve"},
				FeatureType: "Curve",
				Geometry:    testutil.MakeLineString([]float64{-122.0, 37.0, -123.0, 38.0}),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.SamplingFeature) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Sampling Curve", created.Name)
				require.Equal(t, "Curve", created.FeatureType)
				require.Equal(t, "Test curve", created.Description)
				require.NotNil(t, created.Geometry)
			},
		},
		{
			name: "create surface sampling feature",
			samplingFeature: &domains.SamplingFeature{
				CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:sf3", Name: "Sampling Surface"},
				FeatureType: "Surface",
				Geometry:    testutil.MakePolygon([]float64{-122.0, 37.0, -123.0, 37.0, -123.0, 38.0, -122.0, 38.0, -122.0, 37.0}),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.SamplingFeature) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Sampling Surface", created.Name)
				require.Equal(t, "Surface", created.FeatureType)
				require.NotNil(t, created.Geometry)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(tt.samplingFeature)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.samplingFeature)
			}
		})
	}
}

func TestSamplingFeatureRepository_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewSamplingFeatureRepository(db)

	// Setup: create test sampling feature
	sf1 := &domains.SamplingFeature{
		CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:get1", Name: "SamplingFeature 1"},
		FeatureType: "Point",
	}
	require.NoError(t, repo.Create(sf1))

	tests := []struct {
		name     string
		id       string
		wantName string
		wantErr  bool
	}{
		{
			name:     "get existing sampling feature",
			id:       sf1.ID,
			wantName: "SamplingFeature 1",
			wantErr:  false,
		},
		{
			name:     "get non-existent sampling feature",
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

func TestSamplingFeatureRepository_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewSamplingFeatureRepository(db)

	// Setup: create multiple test sampling features
	sf1 := &domains.SamplingFeature{
		CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:sf1", Name: "Point Feature 1"},
		FeatureType: "Point",
		Geometry:    testutil.MakePoint(-122.4194, 37.7749),
	}
	require.NoError(t, repo.Create(sf1))

	sf2 := &domains.SamplingFeature{
		CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:sf2", Name: "Curve Feature 1"},
		FeatureType: "Curve",
		Geometry:    testutil.MakeLineString([]float64{-122.0, 37.0, -123.0, 38.0}),
	}
	require.NoError(t, repo.Create(sf2))

	sf3 := &domains.SamplingFeature{
		CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:sf3", Name: "Surface Feature 1", Description: "Test surface"},
		FeatureType: "Surface",
		Geometry:    testutil.MakePolygon([]float64{-122.0, 37.0, -123.0, 37.0, -123.0, 38.0, -122.0, 38.0, -122.0, 37.0}),
	}
	require.NoError(t, repo.Create(sf3))

	tests := []struct {
		name      string
		params    *queryparams.SamplingFeatureQueryParams
		wantCount int
		wantTotal int64
		checkFunc func(t *testing.T, features []*domains.SamplingFeature)
	}{
		{
			name: "list all sampling features",
			params: &queryparams.SamplingFeatureQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
			},
			wantCount: 3,
			wantTotal: 3,
			checkFunc: func(t *testing.T, features []*domains.SamplingFeature) {
				require.Len(t, features, 3)
			},
		},
		{
			name: "list with limit 2",
			params: &queryparams.SamplingFeatureQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 2},
			},
			wantCount: 2,
			wantTotal: 3,
			checkFunc: func(t *testing.T, features []*domains.SamplingFeature) {
				require.Len(t, features, 2)
			},
		},
		{
			name: "filter by specific IDs",
			params: &queryparams.SamplingFeatureQueryParams{
				QueryParams: queryparams.QueryParams{
					IDs:   []string{sf1.ID, sf2.ID},
					Limit: 10,
				},
			},
			wantCount: 2,
			wantTotal: 2,
			checkFunc: func(t *testing.T, features []*domains.SamplingFeature) {
				require.Len(t, features, 2)
				ids := []string{features[0].ID, features[1].ID}
				require.Contains(t, ids, sf1.ID)
				require.Contains(t, ids, sf2.ID)
			},
		},
		{
			name: "query test - search by name",
			params: &queryparams.SamplingFeatureQueryParams{
				QueryParams: queryparams.QueryParams{
					Q:     []string{"Curve"},
					Limit: 10,
				},
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, features []*domains.SamplingFeature) {
				require.Len(t, features, 1)
				require.Equal(t, "Curve Feature 1", features[0].Name)
			},
		},
		{
			name: "query test - search by description",
			params: &queryparams.SamplingFeatureQueryParams{
				QueryParams: queryparams.QueryParams{
					Q:     []string{"surface"},
					Limit: 10,
				},
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, features []*domains.SamplingFeature) {
				require.Len(t, features, 1)
				require.Equal(t, "Surface Feature 1", features[0].Name)
			},
		},
		{
			name: "empty result set",
			params: &queryparams.SamplingFeatureQueryParams{
				QueryParams: queryparams.QueryParams{
					IDs:   []string{"non-existent-id"},
					Limit: 10,
				},
			},
			wantCount: 0,
			wantTotal: 0,
			checkFunc: func(t *testing.T, features []*domains.SamplingFeature) {
				require.Empty(t, features)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features, total, err := repo.List(tt.params)
			require.NoError(t, err)
			require.Equal(t, tt.wantTotal, total, "total count mismatch")
			require.Len(t, features, tt.wantCount, "result count mismatch")
			if tt.checkFunc != nil {
				tt.checkFunc(t, features)
			}
		})
	}
}

func TestSamplingFeatureRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewSamplingFeatureRepository(db)

	// Setup: create a sampling feature
	original := &domains.SamplingFeature{
		CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:update1", Name: "Original Name"},
		FeatureType: "Point",
	}
	require.NoError(t, repo.Create(original))

	tests := []struct {
		name       string
		updateFunc func(*domains.SamplingFeature)
		checkFunc  func(t *testing.T, updated *domains.SamplingFeature)
		wantErr    bool
	}{
		{
			name: "update name",
			updateFunc: func(sf *domains.SamplingFeature) {
				sf.Name = "Updated Name"
			},
			checkFunc: func(t *testing.T, updated *domains.SamplingFeature) {
				require.Equal(t, "Updated Name", updated.Name)
			},
			wantErr: false,
		},
		{
			name: "update description",
			updateFunc: func(sf *domains.SamplingFeature) {
				sf.Description = "New description"
			},
			checkFunc: func(t *testing.T, updated *domains.SamplingFeature) {
				require.Equal(t, "New description", updated.Description)
			},
			wantErr: false,
		},
		{
			name: "update geometry",
			updateFunc: func(sf *domains.SamplingFeature) {
				sf.Geometry = testutil.MakePoint(-118.2437, 34.0522)
			},
			checkFunc: func(t *testing.T, updated *domains.SamplingFeature) {
				require.NotNil(t, updated.Geometry)
				require.NotNil(t, updated.Geometry.T)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get fresh copy for each test
			sf, err := repo.GetByID(original.ID)
			require.NoError(t, err)

			// Apply update
			tt.updateFunc(sf)

			// Save
			err = repo.Update(sf)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify
			updated, err := repo.GetByID(sf.ID)
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, updated)
			}
		})
	}
}

func TestSamplingFeatureRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewSamplingFeatureRepository(db)

	tests := []struct {
		name      string
		setupFunc func() string
		checkFunc func(t *testing.T, id string)
	}{
		{
			name: "delete existing sampling feature",
			setupFunc: func() string {
				sf := &domains.SamplingFeature{
					CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:del1", Name: "Delete Me"},
					FeatureType: "Point",
				}
				require.NoError(t, repo.Create(sf))
				return sf.ID
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
