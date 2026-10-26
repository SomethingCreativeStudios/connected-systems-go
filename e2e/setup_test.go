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

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/api"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository/testutil"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	testServer    *httptest.Server
	testDB        *gorm.DB
	testContainer *testutil.PostGISContainer
	testRepos     *repository.Repositories
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Set up logger
	logger, _ := zap.NewDevelopment()

	// Start the same TimescaleDB/PostGIS image configured for Compose.
	testContainer = testutil.StartPostGISContainer(ctx, &testing.T{})

	// Open the test database, then run the production migration path so E2E
	// requests exercise the observations hypertable rather than a plain table.
	testDB = testutil.OpenTestDB(&testing.T{}, testContainer.DSN, testutil.OpenTestDBOptions{
		EnableLogging: false,
	})
	if err := repository.Migrate(context.Background(), testDB); err != nil {
		panic(fmt.Sprintf("migrate E2E database: %v", err))
	}

	// Initialize repositories
	testRepos = repository.NewRepositories(testDB)

	// Set up config — BaseURL will be set after we know the test server URL.
	cfg := &config.Config{
		API: config.APIConfig{
			Title:   "Test API",
			Version: "1.0.0",
		},
	}

	// Create router with a placeholder handler so we can reserve a URL,
	// then update config and rebuild the router with the correct BaseURL.
	placeholderRouter := api.NewRouter(cfg, logger, testRepos, nil)
	testServer = httptest.NewServer(placeholderRouter)
	cfg.API.BaseURL = testServer.URL

	// Rebuild the router with the correct BaseURL so association links
	// and Location headers use the real test server address.
	router := api.NewRouter(cfg, logger, testRepos, nil)
	testServer.Config.Handler = router

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
	testDB.Exec("TRUNCATE TABLE mqtt_create_operations, observations, datastreams, command_status_reports, command_results, commands, control_streams, system_events, system_history_revisions, systems, deployments, procedures, sampling_features, properties, features, collections CASCADE")
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

func FollowLocation(initialResp *http.Response, acceptType string) (*map[string]any, error) {
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

	var p map[string]any
	err = json.NewDecoder(resp.Body).Decode(&p)
	if err != nil {
		return nil, err
	}

	return &p, nil
}
