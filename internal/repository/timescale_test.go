package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/repository/testutil"
)

func TestEnsureObservationHypertableRejectsLegacyPrimaryKey(t *testing.T) {
	ctx := context.Background()
	container := testutil.StartPostGISContainer(ctx, t)
	defer container.Terminate(ctx)

	db := testutil.OpenTestDB(t, container.DSN, testutil.OpenTestDBOptions{})
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	require.NoError(t, db.Exec(`
		CREATE TABLE observations (
			id varchar(255) PRIMARY KEY,
			result_time timestamptz NOT NULL
		)
	`).Error)

	err = ensureObservationHypertable(db)
	require.Error(t, err)
	require.ErrorContains(t, err, "recreate the fresh development database")
}
