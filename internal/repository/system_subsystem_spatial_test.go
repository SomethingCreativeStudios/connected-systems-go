package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository/testutil"
)

func TestSystemRepositoryListSubsystemsAppliesSpatialFilters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := NewSystemRepository(db)

	parent := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:spatial-parent", Name: "Spatial Parent"},
		SystemType: domains.SystemTypeSystem,
	}
	require.NoError(t, repo.Create(parent))

	included := &domains.System{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:spatial-child-in", Name: "Spatial Child Inside"},
		SystemType:     domains.SystemTypeSensor,
		ParentSystemID: &parent.ID,
		Geometry:       testutil.MakePoint(-117.1625, 32.715),
	}
	require.NoError(t, repo.Create(included))

	excluded := &domains.System{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:spatial-child-out", Name: "Spatial Child Outside"},
		SystemType:     domains.SystemTypeSensor,
		ParentSystemID: &parent.ID,
		Geometry:       testutil.MakePoint(-73.9857, 40.7484),
	}
	require.NoError(t, repo.Create(excluded))

	systems, total, err := repo.ListSubsystems(parent.ID, &queryparams.SystemQueryParams{
		QueryParams: queryparams.QueryParams{Limit: 10},
		Bbox:        &common_shared.BoundingBox{MinX: -118, MinY: 32, MaxX: -117, MaxY: 33},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, systems, 1)
	require.Equal(t, included.ID, systems[0].ID)
}
