package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/api"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/model/formaters/geojson_formatters"
	"github.com/yourusername/connected-systems-go/internal/model/formaters/sensorml_formatters"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"github.com/yourusername/connected-systems-go/internal/repository/testutil"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	testServer    *httptest.Server
	testDB        *gorm.DB
	testContainer *testutil.PostGISContainer
	testRepos     *repository.Repositories
	// Serializer collections available for tests
	testSystemFormatters          *formaters.MultiFormatFormatterCollection[*domains.System]
	testDeploymentFormatters      *formaters.MultiFormatFormatterCollection[*domains.Deployment]
	testProcedureFormatters       *formaters.MultiFormatFormatterCollection[*domains.Procedure]
	testSamplingFeatureFormatters *formaters.MultiFormatFormatterCollection[*domains.SamplingFeature]
	testPropertyFormatters        *formaters.MultiFormatFormatterCollection[*domains.Property]
	testFeatureFormatters         *formaters.MultiFormatFormatterCollection[*domains.Feature]
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Set up logger
	logger, _ := zap.NewDevelopment()

	// Start PostGIS container
	testContainer = testutil.StartPostGISContainer(ctx, &testing.T{})

	// Open test database
	testDB = testutil.OpenTestDB(&testing.T{}, testContainer.DSN, testutil.OpenTestDBOptions{
		EnableLogging: false,
		Models:        testutil.AllModels(),
	})

	// Initialize repositories
	testRepos = repository.NewRepositories(testDB)

	// Build formatter/serializer collections (mirrors router.go)
	// Systems
	sysCol := formaters.NewMultiFormatFormatterCollection[*domains.System]("application/geo+json")
	sysGeo := geojson_formatters.NewSystemGeoJSONFormatter(testRepos)
	formaters.RegisterFormatterTyped(sysCol, "application/geo+json", sysGeo)
	sysSML := sensorml_formatters.NewSystemSensorMLFormatter(testRepos)
	formaters.RegisterFormatterTyped(sysCol, "application/sml+json", sysSML)
	formaters.RegisterFormatterTypedDefault(sysCol, sysGeo, "application/geo+json")
	testSystemFormatters = sysCol

	// Deployments
	depCol := formaters.NewMultiFormatFormatterCollection[*domains.Deployment]("application/geo+json")
	depGeo := geojson_formatters.NewDeploymentGeoJSONFormatter(testRepos)
	formaters.RegisterFormatterTyped(depCol, "application/geo+json", depGeo)
	depSML := sensorml_formatters.NewDeploymentSensorMLFormatter(testRepos)
	formaters.RegisterFormatterTyped(depCol, "application/sml+json", depSML)
	formaters.RegisterFormatterTypedDefault(depCol, depGeo, "application/geo+json")
	testDeploymentFormatters = depCol

	// Procedures
	procCol := formaters.NewMultiFormatFormatterCollection[*domains.Procedure]("application/geo+json")
	procGeo := geojson_formatters.NewProcedureGeoJSONFormatter(testRepos)
	formaters.RegisterFormatterTyped(procCol, "application/geo+json", procGeo)
	procSML := sensorml_formatters.NewProcedureSensorMLFormatter(testRepos)
	formaters.RegisterFormatterTyped(procCol, "application/sml+json", procSML)
	formaters.RegisterFormatterTypedDefault(procCol, procGeo, "application/geo+json")
	testProcedureFormatters = procCol

	// Sampling features
	sfCol := formaters.NewMultiFormatFormatterCollection[*domains.SamplingFeature]("application/geo+json")
	sfGeo := geojson_formatters.NewSamplingFeatureGeoJSONFormatter(testRepos)
	formaters.RegisterFormatterTyped(sfCol, "application/geo+json", sfGeo)
	sfSML := sensorml_formatters.NewSamplingFeatureSensorMLFormatter(testRepos)
	formaters.RegisterFormatterTyped(sfCol, "application/sml+json", sfSML)
	formaters.RegisterFormatterTypedDefault(sfCol, sfGeo, "application/geo+json")
	testSamplingFeatureFormatters = sfCol

	// Properties
	propCol := formaters.NewMultiFormatFormatterCollection[*domains.Property]("application/sml+json")
	propGeo := geojson_formatters.NewPropertyGeoJSONFormatter(testRepos)
	formaters.RegisterFormatterTyped(propCol, "application/geo+json", propGeo)
	propSML := sensorml_formatters.NewPropertySensorMLFormatter(testRepos)
	formaters.RegisterFormatterTyped(propCol, "application/sml+json", propSML)
	formaters.RegisterFormatterTypedDefault(propCol, propSML, "application/sml+json")
	testPropertyFormatters = propCol

	// Features
	featCol := formaters.NewMultiFormatFormatterCollection[*domains.Feature]("application/geo+json")
	featGeo := geojson_formatters.NewFeatureGeoJSONFormatter(testRepos)
	formaters.RegisterFormatterTyped(featCol, "application/geo+json", featGeo)
	formaters.RegisterFormatterTypedDefault(featCol, featGeo, "application/geo+json")
	testFeatureFormatters = featCol

	// Set up config
	cfg := &config.Config{
		API: config.APIConfig{
			BaseURL: "http://localhost:8080",
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	// Set up router
	router := api.NewRouter(cfg, logger, testRepos)

	// Start test server
	testServer = httptest.NewServer(router)

	// Update config BaseURL to the actual test server URL so handlers build correct Location headers
	cfg.API.BaseURL = testServer.URL

	// Run tests
	exitCode := m.Run()

	// Cleanup
	testServer.Close()
	sqlDB, _ := testDB.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
	testContainer.Terminate(ctx)

	// Exit
	os.Exit(exitCode)
}

// cleanupDB truncates all tables to ensure test isolation
func cleanupDB(t *testing.T) {
	t.Helper()
	testDB.Exec("TRUNCATE TABLE observations, datastreams, commands, control_streams, system_events, system_history_revisions, systems, deployments, procedures, sampling_features, properties, features, collections CASCADE")
}

func parseID(locationHeader string, prefix string) string {
	parts := strings.Split(locationHeader, prefix)

	if len(parts) == 2 {
		// strip any trailing slash or query
		id := parts[1]
		id = strings.TrimPrefix(id, "/")
		if idx := strings.IndexAny(id, "?#/"); idx != -1 {
			id = id[:idx]
		}
		return id
	}

	return ""
}

func FollowLocation(initialResp *http.Response, acceptType string) (*map[string]interface{}, error) {
	url := initialResp.Header.Get("Location")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", acceptType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to follow location: %s status %d", url, resp.StatusCode)
	}

	var p map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&p)
	if err != nil {
		return nil, err
	}

	return &p, nil
}
