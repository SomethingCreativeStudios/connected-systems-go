package json_formatters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
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

// TestDatastreamJSONFormatter_SerializeAll_EnrichesInlineLinksFromDB verifies that
// SerializeAll populates Title and UID on inline @link properties when real repos are available.
func TestDatastreamJSONFormatter_SerializeAll_EnrichesInlineLinksFromDB(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	repos, cleanup := setupFormatterTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Seed linked resources
	sys := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sys:ds", Name: "Test System"},
		SystemType: domains.SystemTypeSensor,
	}
	require.NoError(t, repos.System.Create(sys))

	proc := &domains.Procedure{
		CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:proc:ds", Name: "Test Procedure"},
		ProcessType: "SimpleProcess",
	}
	require.NoError(t, repos.Procedure.Create(proc))

	dep := &domains.Deployment{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:dep:ds", Name: "Test Deployment"},
		DeploymentType: "Fixed",
	}
	require.NoError(t, repos.Deployment.Create(dep))

	sf := &domains.SamplingFeature{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sf:ds", Name: "Test SamplingFeature"},
		FeatureType: domains.SamplingFeatureTypeSample,
	}
	require.NoError(t, repos.SamplingFeature.Create(sf))

	// Create datastream with all link IDs
	ds := &domains.Datastream{
		Base:     domains.Base{ID: "ds-1"},
		SystemID: &sys.ID,
		ProcedureLink: &common_shared.Link{
			Href: "/procedures/" + proc.ID,
		},
		DeploymentLink: &common_shared.Link{
			Href: "/deployments/" + dep.ID,
		},
		FeatureOfInterest: &common_shared.Link{
			Href: "/features/feat-1",
		},
		SamplingFeatureLink: &common_shared.Link{
			Href: "/samplingFeatures/" + sf.ID,
		},
		ProcedureID:       &proc.ID,
		DeploymentID:      &dep.ID,
		SamplingFeatureID: &sf.ID,
	}

	formatter := NewDatastreamJSONFormatter(repos)
	results, err := formatter.SerializeAll(ctx, []*domains.Datastream{ds})
	require.NoError(t, err)
	require.Len(t, results, 1)

	r := results[0]

	// system@link enrichment
	require.NotNil(t, r.SystemLink)
	require.Equal(t, formaters.GeoJSONContentType, r.SystemLink.Type)
	require.Equal(t, "Test System", r.SystemLink.Title)
	require.NotNil(t, r.SystemLink.UID)
	require.Equal(t, "urn:test:sys:ds", *r.SystemLink.UID)

	// procedure@link enrichment
	require.NotNil(t, r.ProcedureLink)
	require.Equal(t, formaters.SensorMLContentType, r.ProcedureLink.Type)
	require.Equal(t, "Test Procedure", r.ProcedureLink.Title)
	require.NotNil(t, r.ProcedureLink.UID)
	require.Equal(t, "urn:test:proc:ds", *r.ProcedureLink.UID)

	// deployment@link enrichment
	require.NotNil(t, r.DeploymentLink)
	require.Equal(t, formaters.GeoJSONContentType, r.DeploymentLink.Type)
	require.Equal(t, "Test Deployment", r.DeploymentLink.Title)
	require.NotNil(t, r.DeploymentLink.UID)
	require.Equal(t, "urn:test:dep:ds", *r.DeploymentLink.UID)

	// samplingFeature@link enrichment
	require.NotNil(t, r.SamplingFeatureLink)
	require.Equal(t, formaters.GeoJSONContentType, r.SamplingFeatureLink.Type)
	require.Equal(t, "Test SamplingFeature", r.SamplingFeatureLink.Title)
	require.NotNil(t, r.SamplingFeatureLink.UID)
	require.Equal(t, "urn:test:sf:ds", *r.SamplingFeatureLink.UID)

	// featureOfInterest@link — Type is set but Title/UID are not enriched (server-derived, no FK)
	require.NotNil(t, r.FeatureOfInterest)
	require.Equal(t, formaters.GeoJSONContentType, r.FeatureOfInterest.Type)
	require.Empty(t, r.FeatureOfInterest.Title)
	require.Nil(t, r.FeatureOfInterest.UID)
}
