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

	depWithSubdep := &domains.Deployment{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:get2", Name: "Deployment 2"},
		DeploymentType: "Mobile",
		SystemIds:      &common_shared.StringArray{"urn:test:get1"},
		Links: []common_shared.Link{
			{
				Href: "systems/urn:test:get1",
				Rel:  "deployedSystems",
				UID:  testutil.PtrStr("urn:test:get1"),
			},
		},
	}
	require.NoError(t, repo.Create(depWithSubdep))

	subdep := &domains.Deployment{
		CommonSSN:          domains.CommonSSN{UniqueIdentifier: "urn:test:get3", Name: "Subdeployment of Deployment 2"},
		DeploymentType:     "Fixed",
		ParentDeploymentID: &depWithSubdep.ID,
		SystemIds:          &common_shared.StringArray{"urn:test:subSystem1"},
		Links: []common_shared.Link{
			{
				Href: "systems/urn:test:subSystem1",
				Rel:  "deployedSystems",
				UID:  testutil.PtrStr("urn:test:subSystem1"),
			},
			{
				Href: "features/some-feature",
				Rel:  "samplingFeatures",
				UID:  testutil.PtrStr("some-feature"),
			},
			{
				Href: "features/some-foi",
				Rel:  "featuresOfInterest",
				UID:  testutil.PtrStr("some-foi"),
			},
		},
	}
	require.NoError(t, repo.Create(subdep))

	// Manually create parent-child relationship for testing

	tests := []struct {
		name     string
		id       string
		wantName string
		isParent bool
		wantErr  bool
	}{
		{
			name:     "get existing deployment",
			id:       dep1.ID,
			wantName: "Deployment 1",
			wantErr:  false,
			isParent: false,
		},
		{
			name:     "get parent deployment with subdeployment associations",
			id:       depWithSubdep.ID,
			wantName: "Deployment 2",
			isParent: true,
			wantErr:  false,
		},
		{
			name:     "get non-existent deployment",
			id:       "non-existent-id",
			wantName: "",
			isParent: false,
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

			if tt.isParent {
				foundSamplingFeature := false
				foundFOI := false
				foundDeployedSystem := false

				for _, link := range got.Links {
					if link.Rel == "samplingFeatures" && link.UID != nil && *link.UID == "some-feature" {
						foundSamplingFeature = true
					}
					if link.Rel == "featuresOfInterest" && link.UID != nil && *link.UID == "some-foi" {
						foundFOI = true
					}
					if link.Rel == "deployedSystems" && link.UID != nil && *link.UID == "urn:test:subSystem1" {
						foundDeployedSystem = true
					}
				}
				require.True(t, foundSamplingFeature, "Expected samplingFeatures link not found")
				require.True(t, foundFOI, "Expected featuresOfInterest link not found")
				require.True(t, foundDeployedSystem, "Expected deployedSystems link not found")
			}
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
			deployments, total, err := repo.List(tt.params, nil)
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
		setupFunc func() []string
		checkFunc func(t *testing.T, id []string)
	}{
		{
			name: "delete existing deployment",
			setupFunc: func() []string {
				dep := &domains.Deployment{
					CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:del1", Name: "Delete Me"},
					DeploymentType: "Fixed",
				}
				require.NoError(t, repo.Create(dep))
				return []string{dep.ID}
			},
			checkFunc: func(t *testing.T, id []string) {
				_, err := repo.GetByID(id[0])
				require.Error(t, err)
			},
		},
		{
			name: "delete deployment with sub-deployments - Reparent to parent",
			setupFunc: func() []string {
				dep := &domains.Deployment{
					CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:del1", Name: "Delete Me"},
					DeploymentType: "Fixed",
				}
				require.NoError(t, repo.Create(dep))

				child := &domains.Deployment{
					CommonSSN:          domains.CommonSSN{UniqueIdentifier: "urn:test:del2", Name: "Child Deployment"},
					DeploymentType:     "Mobile",
					ParentDeploymentID: &dep.ID,
				}
				require.NoError(t, repo.Create(child))

				grandChild := &domains.Deployment{
					CommonSSN:          domains.CommonSSN{UniqueIdentifier: "urn:test:del3", Name: "Grandchild Deployment"},
					DeploymentType:     "Fixed",
					ParentDeploymentID: &child.ID,
				}
				require.NoError(t, repo.Create(grandChild))

				// Delete child, grandChild should be reparented to dep
				return []string{dep.ID, child.ID, grandChild.ID}
			},
			checkFunc: func(t *testing.T, id []string) {
				parentId := id[0]
				childId := id[1]
				grandChildId := id[2]

				// Verify parent is deleted
				_, err := repo.GetByID(parentId)
				require.Error(t, err)

				// Verify child is now orphaned (no parent)
				child, err := repo.GetByID(childId)
				require.NoError(t, err)
				require.Nil(t, child.ParentDeploymentID)

				// Verify grandChild is did not change parent (still child)
				grandChild, err := repo.GetByID(grandChildId)
				require.NoError(t, err)
				require.NotNil(t, grandChild.ParentDeploymentID)
				require.Equal(t, childId, *grandChild.ParentDeploymentID)
			},
		},
		{
			name: "delete child deployment - Reparent to parent",
			setupFunc: func() []string {
				dep := &domains.Deployment{
					CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:del1", Name: "Delete Me"},
					DeploymentType: "Fixed",
				}
				require.NoError(t, repo.Create(dep))

				child := &domains.Deployment{
					CommonSSN:          domains.CommonSSN{UniqueIdentifier: "urn:test:del2", Name: "Child Deployment"},
					DeploymentType:     "Mobile",
					ParentDeploymentID: &dep.ID,
				}
				require.NoError(t, repo.Create(child))

				grandChild := &domains.Deployment{
					CommonSSN:          domains.CommonSSN{UniqueIdentifier: "urn:test:del3", Name: "Grandchild Deployment"},
					DeploymentType:     "Fixed",
					ParentDeploymentID: &child.ID,
				}
				require.NoError(t, repo.Create(grandChild))

				// Delete child, grandChild should be reparented to dep
				return []string{child.ID, grandChild.ID, dep.ID}
			},
			checkFunc: func(t *testing.T, id []string) {
				childId := id[0]
				grandChildId := id[1]
				parentId := id[2]

				// Verify child is deleted
				_, err := repo.GetByID(childId)
				require.Error(t, err)

				// Verify grandChild is now child of parent
				grandChild, err := repo.GetByID(grandChildId)
				require.NoError(t, err)
				require.NotNil(t, grandChild.ParentDeploymentID)
				require.Equal(t, parentId, *grandChild.ParentDeploymentID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.setupFunc()
			err := repo.Delete(id[0])
			require.NoError(t, err)
			tt.checkFunc(t, id)

			repo.DeleteAll()
		})
	}
}
