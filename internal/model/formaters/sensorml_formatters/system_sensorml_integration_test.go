package sensorml_formatters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"github.com/yourusername/connected-systems-go/internal/repository/testutil"
)

// setupFormatterTestDB spins up a PostGIS container, migrates models, and returns repos + cleanup.
func setupFormatterTestDB(t *testing.T) (*repository.Repositories, func()) {
	t.Helper()
	ctx := context.Background()

	container := testutil.StartPostGISContainer(ctx, t)

	db := testutil.OpenTestDB(t, container.DSN, testutil.OpenTestDBOptions{
		EnableLogging: false,
		Models:        testutil.AllModels(),
	})

	repos := repository.NewRepositories(db)

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		container.Terminate(ctx)
	}

	return repos, cleanup
}

// TestSystemSensorMLFormatter_SerializeAll_EnrichesInlineLinksFromDB verifies that
// SerializeAll populates Title and UID on attachedTo and typeOf.
func TestSystemSensorMLFormatter_SerializeAll_EnrichesInlineLinksFromDB(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	repos, cleanup := setupFormatterTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Seed parent system
	parentSys := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sys:parent", Name: "Parent System"},
		SystemType: domains.SystemTypePlatform,
	}
	require.NoError(t, repos.System.Create(parentSys))

	// Seed procedure (system kind)
	kindProc := &domains.Procedure{
		CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:proc:kind", Name: "System Kind Procedure"},
		ProcessType: "SimpleProcess",
	}
	require.NoError(t, repos.Procedure.Create(kindProc))

	// Create system with parent and kind
	system := &domains.System{
		Base:           domains.Base{ID: "sys-1"},
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:sys:1", Name: "Test System"},
		SystemType:     domains.SystemTypeSensor,
		ParentSystemID: &parentSys.ID,
		TypeOfID:       &kindProc.ID,
	}

	formatter := NewSystemSensorMLFormatter(repos)
	features, err := formatter.SerializeAll(ctx, []*domains.System{system})
	require.NoError(t, err)
	require.Len(t, features, 1)

	f := features[0]

	// attachedTo enrichment
	require.NotNil(t, f.AttachedTo)
	require.Equal(t, formaters.GeoJSONContentType, f.AttachedTo.Type)
	require.Equal(t, "Parent System", f.AttachedTo.Title)
	require.NotNil(t, f.AttachedTo.UID)
	require.Equal(t, "urn:test:sys:parent", *f.AttachedTo.UID)

	// typeOf enrichment
	require.NotNil(t, f.TypeOf)
	require.Equal(t, formaters.SensorMLContentType, f.TypeOf.Type)
	require.Equal(t, "System Kind Procedure", f.TypeOf.Title)
	require.NotNil(t, f.TypeOf.UID)
	require.Equal(t, "urn:test:proc:kind", *f.TypeOf.UID)
}
