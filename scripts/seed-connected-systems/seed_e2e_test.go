package main

import (
	"context"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/api"
	serverconfig "github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"github.com/yourusername/connected-systems-go/internal/repository/testutil"
	"go.uber.org/zap"
)

func TestSeederE2EPostGIS(t *testing.T) {
	ctx := context.Background()
	container := testutil.StartPostGISContainer(ctx, t)
	t.Cleanup(func() { require.NoError(t, container.Terminate(ctx)) })

	db := testutil.OpenTestDB(t, container.DSN, testutil.OpenTestDBOptions{})
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, repository.AutoMigrate(db))
	repos := repository.NewRepositories(db)

	apiCfg := &serverconfig.Config{API: serverconfig.APIConfig{BaseURL: "http://placeholder"}}
	server := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(server.Close)
	apiCfg.API.BaseURL = server.URL
	server.Config.Handler = api.NewRouter(apiCfg, zap.NewNop(), repos, nil)

	cfg := validConfig("seed")
	cfg.Endpoint = server.URL
	cfg.Seed.SubsystemsPerSystem = IntRange{Min: 1, Max: 1}
	cfg.Seed.FeaturesPerCollection = IntRange{Min: 2, Max: 2}
	cfg.Seed.SamplingFeaturesPerSystem = IntRange{Min: 1, Max: 1}
	cfg.Seed.DatastreamsPerSystem = IntRange{Min: 4, Max: 4}
	cfg.Seed.ControlStreamsPerSystem = IntRange{Min: 2, Max: 2}
	cfg.Seed.StatusReportsPerCommand = IntRange{Min: 3, Max: 3}
	cfg.Seed.SubdeploymentsPerDeployment = IntRange{Min: 1, Max: 1}
	client, err := NewAPIClient(cfg)
	require.NoError(t, err)
	require.NoError(t, client.Preflight(ctx))

	report, err := NewSeeder(cfg, client, randForTest()).Run(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, report.Created["properties"])
	require.Equal(t, 1, report.Created["collections"])
	require.Equal(t, 2, report.Created["features"])
	require.Equal(t, 2, report.Created["systems"])
	require.Equal(t, 1, report.Created["subsystems"])
	require.Equal(t, 2, report.Created["sampling_features"])
	require.Equal(t, 8, report.Created["datastreams"])
	require.Equal(t, 8, report.Created["observations"])
	require.Equal(t, 4, report.Created["control_streams"])
	require.Equal(t, 4, report.Created["commands"])
	require.Equal(t, 12, report.Created["command_status_reports"])
	require.Equal(t, 4, report.Created["command_results"])
	require.Equal(t, 2, report.Created["deployments"])
	require.Equal(t, 1, report.Created["subdeployments"])
	require.Equal(t, 2, report.Created["system_events"])

	assertCount := func(model any, expected int64) {
		var count int64
		require.NoError(t, db.Model(model).Count(&count).Error)
		require.Equal(t, expected, count, "model %T", model)
	}
	assertCount(&domains.Property{}, 1)
	assertCount(&domains.Collection{}, 1)
	assertCount(&domains.Feature{}, 2)
	assertCount(&domains.Procedure{}, 1)
	assertCount(&domains.System{}, 2)
	assertCount(&domains.SamplingFeature{}, 2)
	assertCount(&domains.Datastream{}, 8)
	assertCount(&domains.Observation{}, 8)
	assertCount(&domains.ControlStream{}, 4)
	assertCount(&domains.Command{}, 4)
	assertCount(&domains.CommandStatusReport{}, 12)
	assertCount(&domains.CommandResult{}, 4)
	assertCount(&domains.Deployment{}, 2)
	assertCount(&domains.SystemEvent{}, 2)
	assertCount(&domains.SystemHistoryRevision{}, 3)

	var genericFeatures []domains.Feature
	require.NoError(t, db.Find(&genericFeatures).Error)
	require.Len(t, genericFeatures, 2)
	require.NotEmpty(t, genericFeatures[0].UniqueIdentifier)
	require.NotEmpty(t, genericFeatures[1].UniqueIdentifier)
	require.NotEqual(t, genericFeatures[0].UniqueIdentifier, genericFeatures[1].UniqueIdentifier)

	var procedure domains.Procedure
	require.NoError(t, db.First(&procedure).Error)
	require.Equal(t, sosaActuator, procedure.ProcedureType)
	require.Equal(t, "PhysicalComponent", procedure.ProcessType)

	var systems []domains.System
	require.NoError(t, db.Order("created_at").Find(&systems).Error)
	require.Len(t, systems, 2)
	require.NotNil(t, systems[0].TypeOfID)
	require.NotEmpty(t, *systems[0].TypeOfID)
	require.NotNil(t, systems[1].ParentSystemID)
	require.Equal(t, systems[0].ID, *systems[1].ParentSystemID)

	var datastream domains.Datastream
	require.NoError(t, db.First(&datastream).Error)
	require.NotNil(t, datastream.Schema)
	require.NotNil(t, datastream.ProcedureID)
}

func randForTest() *rand.Rand {
	return rand.New(rand.NewPCG(101, 202))
}
