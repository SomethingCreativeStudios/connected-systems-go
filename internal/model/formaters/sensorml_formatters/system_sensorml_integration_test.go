package sensorml_formatters

import (
	"context"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository/testutil"
	"github.com/stretchr/testify/require"
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

	repos, cleanup := setupFormatterTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Seed parent system
	parentSys := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sys:parent", Name: "Parent System"},
		SystemType: domains.SystemTypePlatform,
	}
	require.NoError(t, repos.System.Create(t.Context(), parentSys))

	// Seed procedure (system kind)
	kindProc := &domains.Procedure{
		CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:proc:kind", Name: "System Kind Procedure"},
		ProcessType: "SimpleProcess",
	}
	require.NoError(t, repos.Procedure.Create(t.Context(), kindProc))

	// Create system with parent and kind
	system := &domains.System{
		Base:           domains.Base{ID: "sys-1"},
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:sys:1", Name: "Test System"},
		SystemType:     domains.SystemTypeSensor,
		ParentSystemID: &parentSys.ID,
		TypeOfID:       &kindProc.ID,
	}

	formatter := NewSystemSensorMLFormatter(&formaters.LinkResolver{BaseURL: "http://example.test", Systems: repos.System, Procedures: repos.Procedure, Deployments: repos.Deployment, SamplingFeatures: repos.SamplingFeature, SystemChildren: repos.System, DeploymentChildren: repos.Deployment, ProcedureChildren: repos.Procedure})
	features, err := formatter.SerializeAll(ctx, []*domains.System{system})
	require.NoError(t, err)
	require.Len(t, features, 1)

	f := features[0]

	// attachedTo enrichment
	require.NotNil(t, f.AttachedTo)
	require.Equal(t, mediatype.GeoJSON, f.AttachedTo.Type)
	require.Equal(t, "Parent System", f.AttachedTo.Title)
	require.NotNil(t, f.AttachedTo.UID)
	require.Equal(t, "urn:test:sys:parent", *f.AttachedTo.UID)

	// typeOf enrichment
	require.NotNil(t, f.TypeOf)
	require.Equal(t, mediatype.SensorML, f.TypeOf.Type)
	require.Equal(t, "System Kind Procedure", f.TypeOf.Title)
	require.NotNil(t, f.TypeOf.UID)
	require.Equal(t, "urn:test:proc:kind", *f.TypeOf.UID)
}
