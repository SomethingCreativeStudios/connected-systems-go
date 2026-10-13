package geojson_formatters

import (
	"context"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
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

// TestDeploymentGeoJSONFormatter_SerializeAll_EnrichesInlineLinksFromDB verifies that
// SerializeAll populates Title and UID on platform@link and deployedSystems@link.
func TestDeploymentGeoJSONFormatter_SerializeAll_EnrichesInlineLinksFromDB(t *testing.T) {

	repos, cleanup := setupFormatterTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Seed systems for platform and deployedSystems
	platformSys := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sys:platform", Name: "Platform System"},
		SystemType: domains.SystemTypePlatform,
	}
	require.NoError(t, repos.System.Create(t.Context(), platformSys))

	deployedSys := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sys:deployed", Name: "Deployed System"},
		SystemType: domains.SystemTypeSensor,
	}
	require.NoError(t, repos.System.Create(t.Context(), deployedSys))

	// Create deployment with platform and deployedSystems
	deployment := &domains.Deployment{
		Base: domains.Base{ID: "dep-1"},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: "urn:test:dep:1",
			Name:             "Test Deployment",
		},
		DeploymentType: "Fixed",
		Platform: &domains.DeployedSystemItem{
			System: common_shared.Link{Href: "/systems/" + platformSys.ID},
		},
		DeployedSystems: []domains.DeployedSystemItem{
			{System: common_shared.Link{Href: "/systems/" + deployedSys.ID}},
		},
	}

	formatter := NewDeploymentGeoJSONFormatter(&formaters.LinkResolver{BaseURL: "http://example.test", Systems: repos.System, Procedures: repos.Procedure, Deployments: repos.Deployment, SamplingFeatures: repos.SamplingFeature, SystemChildren: repos.System, DeploymentChildren: repos.Deployment, ProcedureChildren: repos.Procedure})
	features, err := formatter.SerializeAll(ctx, []*domains.Deployment{deployment})
	require.NoError(t, err)
	require.Len(t, features, 1)

	f := features[0]

	// platform@link enrichment
	require.NotNil(t, f.Properties.Platform)
	require.Equal(t, mediatype.GeoJSON, f.Properties.Platform.Type)
	require.Equal(t, "Platform System", f.Properties.Platform.Title)
	require.NotNil(t, f.Properties.Platform.UID)
	require.Equal(t, "urn:test:sys:platform", *f.Properties.Platform.UID)

	// deployedSystems@link enrichment
	require.Len(t, f.Properties.DeployedSystems, 1)
	ds := f.Properties.DeployedSystems[0]
	require.Equal(t, mediatype.GeoJSON, ds.Type)
	require.Equal(t, "Deployed System", ds.Title)
	require.NotNil(t, ds.UID)
	require.Equal(t, "urn:test:sys:deployed", *ds.UID)
}
