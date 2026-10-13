package repository

import (
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/stretchr/testify/require"
)

// Control streams and features have no required uid. Leaving it out must not
// collide with every other resource that also left it out.
func TestCreate_WithoutUIDDoesNotCollide(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db)

	system := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:uid-system", Name: "System"},
		SystemType: domains.SystemTypeSensor,
	}
	require.NoError(t, repos.System.Create(t.Context(), system))

	for _, name := range []string{"First", "Second"} {
		require.NoError(t, repos.ControlStream.Create(t.Context(), &domains.ControlStream{
			CommonSSN: domains.CommonSSN{Name: name},
			SystemID:  &system.ID,
		}))
		require.NoError(t, repos.Feature.Create(t.Context(), &domains.Feature{
			CommonSSN:    domains.CommonSSN{Name: name},
			CollectionID: "collection1",
		}))
	}

	stored, err := repos.Feature.GetByIDs(t.Context(), nil)
	require.NoError(t, err)
	require.Empty(t, stored)
}
