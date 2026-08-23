package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository/testutil"
)

func TestEnsureSpatialGeometryStorageNormalizesLegacySRID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	system := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:legacy-srid", Name: "Legacy SRID System"},
		SystemType: domains.SystemTypeSensor,
		Geometry:   testutil.MakePoint(-117.1625, 32.715),
	}
	repo := NewSystemRepository(db)
	require.NoError(t, repo.Create(system))
	require.NoError(t, db.Exec("UPDATE systems SET geometry = ST_SetSRID(geometry, 0) WHERE id = ?", system.ID).Error)

	var before int
	require.NoError(t, db.Raw("SELECT ST_SRID(geometry) FROM systems WHERE id = ?", system.ID).Scan(&before).Error)
	require.Zero(t, before)

	require.NoError(t, ensureSpatialGeometryStorage(db))

	var after int
	require.NoError(t, db.Raw("SELECT ST_SRID(geometry) FROM systems WHERE id = ?", system.ID).Scan(&after).Error)
	require.Equal(t, 4326, after)

	systems, total, err := repo.List(&queryparams.SystemQueryParams{
		QueryParams: queryparams.QueryParams{Limit: 10},
		Geom:        "POLYGON((-118 32,-117 32,-117 33,-118 33,-118 32))",
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, systems, 1)
	require.Equal(t, system.ID, systems[0].ID)
}
