package repository

import (
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/stretchr/testify/require"
)

// PUT replaces a resource in place: it must never create a missing resource,
// and it must keep created_at and the parent, which clients cannot send.
func TestUpdate_ReplacesWithoutUpsert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db)

	t.Run("missing system is not created", func(t *testing.T) {
		err := repos.System.Update(t.Context(), "does-not-exist", &domains.System{
			CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:ghost", Name: "Ghost"},
			SystemType: domains.SystemTypeSensor,
		})
		require.ErrorIs(t, err, ErrNotFound)

		var count int64
		require.NoError(t, db.Model(&domains.System{}).Where("id = ?", "does-not-exist").Count(&count).Error)
		require.Zero(t, count)
	})

	t.Run("subsystem keeps parent and created_at", func(t *testing.T) {
		parent := &domains.System{
			CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:parent", Name: "Parent"},
			SystemType: domains.SystemTypePlatform,
		}
		require.NoError(t, repos.System.Create(t.Context(), parent))
		child := &domains.System{
			CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:child", Name: "Child"},
			SystemType:     domains.SystemTypeSensor,
			ParentSystemID: &parent.ID,
		}
		require.NoError(t, repos.System.Create(t.Context(), child))
		before, err := repos.System.GetByID(t.Context(), child.ID)
		require.NoError(t, err)

		require.NoError(t, repos.System.Update(t.Context(), child.ID, &domains.System{
			CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:child", Name: "Renamed"},
			SystemType: domains.SystemTypeSensor,
		}))

		after, err := repos.System.GetByID(t.Context(), child.ID)
		require.NoError(t, err)
		require.Equal(t, "Renamed", after.Name)
		require.NotNil(t, after.ParentSystemID)
		require.Equal(t, parent.ID, *after.ParentSystemID)
		require.True(t, before.CreatedAt.Equal(after.CreatedAt), "created_at changed from %v to %v", before.CreatedAt, after.CreatedAt)
	})

	t.Run("sampling feature keeps parent system", func(t *testing.T) {
		system := &domains.System{
			CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sf-owner", Name: "Owner"},
			SystemType: domains.SystemTypeSensor,
		}
		require.NoError(t, repos.System.Create(t.Context(), system))
		sf := &domains.SamplingFeature{
			CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:sf", Name: "Station"},
			ParentSystemID: &system.ID,
		}
		require.NoError(t, repos.SamplingFeature.Create(t.Context(), sf))

		replacement := &domains.SamplingFeature{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:sf", Name: "Station 2"}}
		replacement.ID = sf.ID
		require.NoError(t, repos.SamplingFeature.Update(t.Context(), replacement))

		after, err := repos.SamplingFeature.GetByID(t.Context(), sf.ID)
		require.NoError(t, err)
		require.Equal(t, "Station 2", after.Name)
		require.NotNil(t, after.ParentSystemID)
		require.Equal(t, system.ID, *after.ParentSystemID)
	})

	t.Run("subdeployment keeps parent", func(t *testing.T) {
		parent := &domains.Deployment{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:dep-parent", Name: "Campaign"}}
		require.NoError(t, repos.Deployment.Create(t.Context(), parent))
		child := &domains.Deployment{
			CommonSSN:          domains.CommonSSN{UniqueIdentifier: "urn:test:dep-child", Name: "Leg 1"},
			ParentDeploymentID: &parent.ID,
		}
		require.NoError(t, repos.Deployment.Create(t.Context(), child))

		replacement := &domains.Deployment{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:dep-child", Name: "Leg 1b"}}
		replacement.ID = child.ID
		require.NoError(t, repos.Deployment.Update(t.Context(), replacement))

		var after domains.Deployment
		require.NoError(t, db.First(&after, "id = ?", child.ID).Error)
		require.Equal(t, "Leg 1b", after.Name)
		require.NotNil(t, after.ParentDeploymentID)
		require.Equal(t, parent.ID, *after.ParentDeploymentID)
	})

	t.Run("missing procedure, property and sampling feature are not created", func(t *testing.T) {
		procedure := &domains.Procedure{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:p", Name: "P"}}
		procedure.ID = "missing-procedure"
		require.ErrorIs(t, repos.Procedure.Update(t.Context(), procedure), ErrNotFound)

		property := &domains.Property{}
		property.ID = "missing-property"
		require.ErrorIs(t, repos.Property.Update(t.Context(), property), ErrNotFound)

		sf := &domains.SamplingFeature{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:sf-missing", Name: "SF"}}
		sf.ID = "missing-sf"
		require.ErrorIs(t, repos.SamplingFeature.Update(t.Context(), sf), ErrNotFound)
	})
}

// There are no foreign keys on parent columns, so nested creates check the
// parent themselves rather than storing an orphan.
func TestCreate_RequiresExistingParent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db)
	missing := "missing-parent"

	require.ErrorIs(t, repos.System.Create(t.Context(), &domains.System{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:orphan:system", Name: "Orphan"},
		SystemType:     domains.SystemTypeSensor,
		ParentSystemID: &missing,
	}), ErrParentNotFound)
	require.ErrorIs(t, repos.SamplingFeature.Create(t.Context(), &domains.SamplingFeature{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:orphan:sf", Name: "Orphan"},
		ParentSystemID: &missing,
	}), ErrParentNotFound)
	require.ErrorIs(t, repos.Datastream.Create(t.Context(), &domains.Datastream{Name: "Orphan", SystemID: &missing}), ErrParentNotFound)
	require.ErrorIs(t, repos.ControlStream.Create(t.Context(), &domains.ControlStream{
		CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:orphan:cs", Name: "Orphan"},
		SystemID:  &missing,
	}), ErrParentNotFound)
	require.ErrorIs(t, repos.Deployment.Create(t.Context(), &domains.Deployment{
		CommonSSN:          domains.CommonSSN{UniqueIdentifier: "urn:test:orphan:dep", Name: "Orphan"},
		ParentDeploymentID: &missing,
	}), ErrParentNotFound)

	for _, table := range []string{"systems", "sampling_features", "datastreams", "control_streams", "deployments"} {
		var count int64
		require.NoError(t, db.Table(table).Count(&count).Error)
		require.Zero(t, count, "%s should be empty", table)
	}
}
