package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository/testutil"
)

func TestFeatureRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewFeatureRepository(db)

	tests := []struct {
		name      string
		feature   *domains.Feature
		wantErr   bool
		checkFunc func(t *testing.T, created *domains.Feature)
	}{
		{
			name: "create point feature",
			feature: &domains.Feature{
				CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:feature1", Name: "Feature Point 1"},
				Geometry:     testutil.MakePoint(-122.4194, 37.7749),
				CollectionID: "collection1",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.Feature) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Feature Point 1", created.Name)
				require.NotNil(t, created.Geometry)
			},
		},
		{
			name: "create linestring feature",
			feature: &domains.Feature{
				CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:feature2", Name: "Feature Line", Description: "Test line feature"},
				Geometry:     testutil.MakeLineString([]float64{-122.0, 37.0, -121.5, 37.5}),
				CollectionID: "collection1",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.Feature) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Feature Line", created.Name)
				require.Equal(t, "Test line feature", created.Description)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(tt.feature)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.feature)
			}
		})
	}
}

func TestFeatureRepository_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewFeatureRepository(db)

	// Setup: create test feature
	feat1 := &domains.Feature{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:get1", Name: "Feature 1"},
		CollectionID: "collection1",
	}
	require.NoError(t, repo.Create(feat1))

	tests := []struct {
		name     string
		id       string
		wantName string
		wantErr  bool
	}{
		{
			name:     "get existing feature",
			id:       feat1.ID,
			wantName: "Feature 1",
			wantErr:  false,
		},
		{
			name:     "get non-existent feature",
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

func TestFeatureRepository_GetByCollectionAndID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewFeatureRepository(db)

	// Setup: create test features in different collections
	feat1 := &domains.Feature{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:feat1", Name: "Feature 1"},
		CollectionID: "collection1",
	}
	require.NoError(t, repo.Create(feat1))

	feat2 := &domains.Feature{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:feat2", Name: "Feature 2"},
		CollectionID: "collection2",
	}
	require.NoError(t, repo.Create(feat2))

	tests := []struct {
		name         string
		collectionID string
		featureID    string
		wantName     string
		wantErr      bool
	}{
		{
			name:         "get feature from correct collection",
			collectionID: "collection1",
			featureID:    feat1.ID,
			wantName:     "Feature 1",
			wantErr:      false,
		},
		{
			name:         "get feature from wrong collection",
			collectionID: "collection2",
			featureID:    feat1.ID,
			wantName:     "",
			wantErr:      true,
		},
		{
			name:         "get non-existent feature",
			collectionID: "collection1",
			featureID:    "non-existent-id",
			wantName:     "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByCollectionAndID(tt.collectionID, tt.featureID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantName, got.Name)
		})
	}
}

func TestFeatureRepository_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewFeatureRepository(db)

	// Setup: create multiple test features
	feat1 := &domains.Feature{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:f1", Name: "Feature Alpha"},
		Geometry:     testutil.MakePoint(-122.4194, 37.7749),
		CollectionID: "collection1",
	}
	require.NoError(t, repo.Create(feat1))

	feat2 := &domains.Feature{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:f2", Name: "Feature Beta"},
		Geometry:     testutil.MakeLineString([]float64{-122.0, 37.0, -121.5, 37.5}),
		CollectionID: "collection1",
	}
	require.NoError(t, repo.Create(feat2))

	feat3 := &domains.Feature{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:f3", Name: "Feature Gamma", Description: "Special feature"},
		CollectionID: "collection2",
	}
	require.NoError(t, repo.Create(feat3))

	tests := []struct {
		name      string
		params    *queryparams.FeatureQueryParams
		wantCount int
		wantTotal int64
		checkFunc func(t *testing.T, features []*domains.Feature)
	}{
		{
			name: "list all features",
			params: &queryparams.FeatureQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
			},
			wantCount: 3,
			wantTotal: 3,
			checkFunc: func(t *testing.T, features []*domains.Feature) {
				require.Len(t, features, 3)
			},
		},
		{
			name: "list with limit 2",
			params: &queryparams.FeatureQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 2},
			},
			wantCount: 2,
			wantTotal: 3,
			checkFunc: func(t *testing.T, features []*domains.Feature) {
				require.Len(t, features, 2)
			},
		},
		{
			name: "filter by specific IDs",
			params: &queryparams.FeatureQueryParams{
				QueryParams: queryparams.QueryParams{
					IDs:   []string{feat1.ID, feat2.ID},
					Limit: 10,
				},
			},
			wantCount: 2,
			wantTotal: 2,
			checkFunc: func(t *testing.T, features []*domains.Feature) {
				require.Len(t, features, 2)
				ids := []string{features[0].ID, features[1].ID}
				require.Contains(t, ids, feat1.ID)
				require.Contains(t, ids, feat2.ID)
			},
		},
		{
			name: "query test - search by name",
			params: &queryparams.FeatureQueryParams{
				QueryParams: queryparams.QueryParams{
					Q:     []string{"Beta"},
					Limit: 10,
				},
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, features []*domains.Feature) {
				require.Len(t, features, 1)
				require.Equal(t, "Feature Beta", features[0].Name)
			},
		},
		{
			name: "empty result set",
			params: &queryparams.FeatureQueryParams{
				QueryParams: queryparams.QueryParams{
					IDs:   []string{"non-existent-id"},
					Limit: 10,
				},
			},
			wantCount: 0,
			wantTotal: 0,
			checkFunc: func(t *testing.T, features []*domains.Feature) {
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

func TestFeatureRepository_ListByCollection(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewFeatureRepository(db)

	// Setup: create features in different collections
	feat1 := &domains.Feature{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:f1", Name: "Collection1 Feature 1"},
		CollectionID: "collection1",
	}
	require.NoError(t, repo.Create(feat1))

	feat2 := &domains.Feature{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:f2", Name: "Collection1 Feature 2"},
		CollectionID: "collection1",
	}
	require.NoError(t, repo.Create(feat2))

	feat3 := &domains.Feature{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:f3", Name: "Collection2 Feature 1"},
		CollectionID: "collection2",
	}
	require.NoError(t, repo.Create(feat3))

	tests := []struct {
		name         string
		collectionID string
		params       *queryparams.FeatureQueryParams
		wantCount    int
		wantTotal    int64
		checkFunc    func(t *testing.T, features []*domains.Feature)
	}{
		{
			name:         "list features in collection1",
			collectionID: "collection1",
			params: &queryparams.FeatureQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
			},
			wantCount: 2,
			wantTotal: 2,
			checkFunc: func(t *testing.T, features []*domains.Feature) {
				require.Len(t, features, 2)
				for _, f := range features {
					require.Equal(t, "collection1", f.CollectionID)
				}
			},
		},
		{
			name:         "list features in collection2",
			collectionID: "collection2",
			params: &queryparams.FeatureQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, features []*domains.Feature) {
				require.Len(t, features, 1)
				require.Equal(t, "Collection2 Feature 1", features[0].Name)
			},
		},
		{
			name:         "list features in non-existent collection",
			collectionID: "non-existent",
			params: &queryparams.FeatureQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
			},
			wantCount: 0,
			wantTotal: 0,
			checkFunc: func(t *testing.T, features []*domains.Feature) {
				require.Empty(t, features)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features, total, err := repo.ListByCollection(tt.collectionID, tt.params)
			require.NoError(t, err)
			require.Equal(t, tt.wantTotal, total, "total count mismatch")
			require.Len(t, features, tt.wantCount, "result count mismatch")
			if tt.checkFunc != nil {
				tt.checkFunc(t, features)
			}
		})
	}
}

func TestFeatureRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewFeatureRepository(db)

	// Setup: create a feature
	original := &domains.Feature{
		CommonSSN:    domains.CommonSSN{UniqueIdentifier: "urn:test:update1", Name: "Original Name"},
		CollectionID: "collection1",
	}
	require.NoError(t, repo.Create(original))

	tests := []struct {
		name       string
		updateFunc func(*domains.Feature)
		checkFunc  func(t *testing.T, updated *domains.Feature)
		wantErr    bool
	}{
		{
			name: "update name",
			updateFunc: func(f *domains.Feature) {
				f.Name = "Updated Name"
			},
			checkFunc: func(t *testing.T, updated *domains.Feature) {
				require.Equal(t, "Updated Name", updated.Name)
			},
			wantErr: false,
		},
		{
			name: "update description",
			updateFunc: func(f *domains.Feature) {
				f.Description = "New description"
			},
			checkFunc: func(t *testing.T, updated *domains.Feature) {
				require.Equal(t, "New description", updated.Description)
			},
			wantErr: false,
		},
		{
			name: "update geometry",
			updateFunc: func(f *domains.Feature) {
				f.Geometry = testutil.MakePoint(-118.2437, 34.0522)
			},
			checkFunc: func(t *testing.T, updated *domains.Feature) {
				require.NotNil(t, updated.Geometry)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get fresh copy for each test
			feat, err := repo.GetByID(original.ID)
			require.NoError(t, err)

			// Apply update
			tt.updateFunc(feat)

			// Save
			err = repo.Update(feat)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify
			updated, err := repo.GetByID(feat.ID)
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, updated)
			}
		})
	}
}

func TestFeatureRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewFeatureRepository(db)

	tests := []struct {
		name      string
		setupFunc func() string
		checkFunc func(t *testing.T, id string)
	}{
		{
			name: "delete existing feature",
			setupFunc: func() string {
				feat := &domains.Feature{
					Base:         domains.Base{},
					CollectionID: "collection1",
				}
				require.NoError(t, repo.Create(feat))
				return feat.ID
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
