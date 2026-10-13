package repository

import (
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/stretchr/testify/require"
)

// One query answers, for a whole page, which nested collections have members.
func TestChildCollections(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db)

	procedure := &domains.Procedure{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:proc", Name: "Procedure"}}
	require.NoError(t, repos.Procedure.Create(t.Context(), procedure))
	unused := &domains.Procedure{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:unused", Name: "Unused"}}
	require.NoError(t, repos.Procedure.Create(t.Context(), unused))

	parent := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:parent", Name: "Parent"},
		SystemType: domains.SystemTypeSensor,
		TypeOfID:   &procedure.ID,
	}
	require.NoError(t, repos.System.Create(t.Context(), parent))
	require.NoError(t, repos.System.Create(t.Context(), &domains.System{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:child", Name: "Child"},
		SystemType:     domains.SystemTypeSensor,
		ParentSystemID: &parent.ID,
	}))
	lone := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:lone", Name: "Lone"},
		SystemType: domains.SystemTypeSensor,
	}
	require.NoError(t, repos.System.Create(t.Context(), lone))
	require.NoError(t, repos.Datastream.Create(t.Context(), &domains.Datastream{Name: "Temperature", SystemID: &parent.ID}))

	deployment := &domains.Deployment{
		CommonSSN:       domains.CommonSSN{UniqueIdentifier: "urn:test:deployment", Name: "Deployment"},
		DeployedSystems: []domains.DeployedSystemItem{{System: common_shared.Link{Href: "/systems/" + parent.ID}}},
		SystemIds:       &common_shared.StringArray{parent.ID},
	}
	require.NoError(t, repos.Deployment.Create(t.Context(), deployment))
	require.NoError(t, repos.Deployment.Create(t.Context(), &domains.Deployment{
		CommonSSN:          domains.CommonSSN{UniqueIdentifier: "urn:test:subdeployment", Name: "Subdeployment"},
		ParentDeploymentID: &deployment.ID,
	}))

	systems, err := repos.System.ChildCollections(t.Context(), []string{parent.ID, lone.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"subsystems", "datastreams", "deployments", "procedures"}, systems[parent.ID])
	require.Empty(t, systems[lone.ID])

	deployments, err := repos.Deployment.ChildCollections(t.Context(), []string{deployment.ID})
	require.NoError(t, err)
	require.Equal(t, []string{"subdeployments", "datastreams"}, deployments[deployment.ID])

	procedures, err := repos.Procedure.ChildCollections(t.Context(), []string{procedure.ID, unused.ID})
	require.NoError(t, err)
	require.Equal(t, []string{"implementingSystems"}, procedures[procedure.ID])
	require.Empty(t, procedures[unused.ID])
}
