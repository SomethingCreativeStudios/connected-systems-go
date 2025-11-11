package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository/testutil"
)

func TestDeploymentRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewDeploymentRepository(db)

	tests := []struct {
		name       string
		deployment *domains.Deployment
		wantErr    bool
		checkFunc  func(t *testing.T, created *domains.Deployment)
	}{
		{
			name: "create fixed deployment",
			deployment: &domains.Deployment{
				CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:deployment1", Name: "Fixed Deployment"},
				DeploymentType: "Fixed",
				ValidTime: &common_shared.TimeRange{
					Start: testutil.PtrTime(time.Now()),
					End:   testutil.PtrTime(time.Now().Add(30 * 24 * time.Hour)),
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.Deployment) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Fixed Deployment", created.Name)
				require.Equal(t, "Fixed", created.DeploymentType)
				require.NotNil(t, created.ValidTime)
			},
		},
		{
			name: "create mobile deployment",
			deployment: &domains.Deployment{
				CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:deployment2", Name: "Mobile Deployment", Description: "Test mobile deployment"},
				DeploymentType: "Mobile",
				ValidTime: &common_shared.TimeRange{
					Start: testutil.PtrTime(time.Now()),
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *domains.Deployment) {
				require.NotEmpty(t, created.ID)
				require.Equal(t, "Mobile Deployment", created.Name)
				require.Equal(t, "Mobile", created.DeploymentType)
				require.Equal(t, "Test mobile deployment", created.Description)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(tt.deployment)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.deployment)
			}
		})
	}
}

func TestDeploymentRepository_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewDeploymentRepository(db)

	// Setup: create test deployment
	dep1 := &domains.Deployment{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:get1", Name: "Deployment 1"},
		DeploymentType: "Fixed",
	}
	require.NoError(t, repo.Create(dep1))

	tests := []struct {
		name     string
		id       string
		wantName string
		wantErr  bool
	}{
		{
			name:     "get existing deployment",
			id:       dep1.ID,
			wantName: "Deployment 1",
			wantErr:  false,
		},
		{
			name:     "get non-existent deployment",
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

func TestDeploymentRepository_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewDeploymentRepository(db)

	// Setup: create multiple test deployments
	dep1 := &domains.Deployment{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:dep1", Name: "Deployment Alpha"},
		DeploymentType: "Fixed",
	}
	require.NoError(t, repo.Create(dep1))

	dep2 := &domains.Deployment{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:dep2", Name: "Deployment Beta"},
		DeploymentType: "Mobile",
	}
	require.NoError(t, repo.Create(dep2))

	dep3 := &domains.Deployment{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:dep3", Name: "Deployment Gamma", Description: "Special deployment"},
		DeploymentType: "Fixed",
	}
	require.NoError(t, repo.Create(dep3))

	tests := []struct {
		name      string
		params    *queryparams.DeploymentsQueryParams
		wantCount int
		wantTotal int64
		checkFunc func(t *testing.T, deployments []*domains.Deployment)
	}{
		{
			name: "list all deployments",
			params: &queryparams.DeploymentsQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 10},
			},
			wantCount: 3,
			wantTotal: 3,
			checkFunc: func(t *testing.T, deployments []*domains.Deployment) {
				require.Len(t, deployments, 3)
			},
		},
		{
			name: "list with limit 2",
			params: &queryparams.DeploymentsQueryParams{
				QueryParams: queryparams.QueryParams{Limit: 2},
			},
			wantCount: 2,
			wantTotal: 3,
			checkFunc: func(t *testing.T, deployments []*domains.Deployment) {
				require.Len(t, deployments, 2)
			},
		},
		{
			name: "filter by specific IDs",
			params: &queryparams.DeploymentsQueryParams{
				QueryParams: queryparams.QueryParams{
					IDs:   []string{dep1.ID, dep2.ID},
					Limit: 10,
				},
			},
			wantCount: 2,
			wantTotal: 2,
			checkFunc: func(t *testing.T, deployments []*domains.Deployment) {
				require.Len(t, deployments, 2)
				ids := []string{deployments[0].ID, deployments[1].ID}
				require.Contains(t, ids, dep1.ID)
				require.Contains(t, ids, dep2.ID)
			},
		},
		{
			name: "query test - search by name",
			params: &queryparams.DeploymentsQueryParams{
				QueryParams: queryparams.QueryParams{
					Q:     []string{"Beta"},
					Limit: 10,
				},
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, deployments []*domains.Deployment) {
				require.Len(t, deployments, 1)
				require.Equal(t, "Deployment Beta", deployments[0].Name)
			},
		},
		{
			name: "query test - search by description",
			params: &queryparams.DeploymentsQueryParams{
				QueryParams: queryparams.QueryParams{
					Q:     []string{"Special"},
					Limit: 10,
				},
			},
			wantCount: 1,
			wantTotal: 1,
			checkFunc: func(t *testing.T, deployments []*domains.Deployment) {
				require.Len(t, deployments, 1)
				require.Equal(t, "Deployment Gamma", deployments[0].Name)
			},
		},
		{
			name: "empty result set",
			params: &queryparams.DeploymentsQueryParams{
				QueryParams: queryparams.QueryParams{
					IDs:   []string{"non-existent-id"},
					Limit: 10,
				},
			},
			wantCount: 0,
			wantTotal: 0,
			checkFunc: func(t *testing.T, deployments []*domains.Deployment) {
				require.Empty(t, deployments)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployments, total, err := repo.List(tt.params)
			require.NoError(t, err)
			require.Equal(t, tt.wantTotal, total, "total count mismatch")
			require.Len(t, deployments, tt.wantCount, "result count mismatch")
			if tt.checkFunc != nil {
				tt.checkFunc(t, deployments)
			}
		})
	}
}

func TestDeploymentRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewDeploymentRepository(db)

	// Setup: create a deployment
	original := &domains.Deployment{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:update1", Name: "Original Name"},
		DeploymentType: "Fixed",
	}
	require.NoError(t, repo.Create(original))

	tests := []struct {
		name       string
		updateFunc func(*domains.Deployment)
		checkFunc  func(t *testing.T, updated *domains.Deployment)
		wantErr    bool
	}{
		{
			name: "update name",
			updateFunc: func(d *domains.Deployment) {
				d.Name = "Updated Name"
			},
			checkFunc: func(t *testing.T, updated *domains.Deployment) {
				require.Equal(t, "Updated Name", updated.Name)
			},
			wantErr: false,
		},
		{
			name: "update description",
			updateFunc: func(d *domains.Deployment) {
				d.Description = "New description"
			},
			checkFunc: func(t *testing.T, updated *domains.Deployment) {
				require.Equal(t, "New description", updated.Description)
			},
			wantErr: false,
		},
		{
			name: "update deployment type",
			updateFunc: func(d *domains.Deployment) {
				d.DeploymentType = "Mobile"
			},
			checkFunc: func(t *testing.T, updated *domains.Deployment) {
				require.Equal(t, "Mobile", updated.DeploymentType)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get fresh copy for each test
			dep, err := repo.GetByID(original.ID)
			require.NoError(t, err)

			// Apply update
			tt.updateFunc(dep)

			// Save
			err = repo.Update(dep)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify
			updated, err := repo.GetByID(dep.ID)
			require.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, updated)
			}
		})
	}
}

func TestDeploymentRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewDeploymentRepository(db)

	tests := []struct {
		name      string
		setupFunc func() string
		checkFunc func(t *testing.T, id string)
	}{
		{
			name: "delete existing deployment",
			setupFunc: func() string {
				dep := &domains.Deployment{
					CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:del1", Name: "Delete Me"},
					DeploymentType: "Fixed",
				}
				require.NoError(t, repo.Create(dep))
				return dep.ID
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
