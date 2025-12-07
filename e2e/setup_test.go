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
	testDB.Exec("TRUNCATE TABLE systems, deployments, procedures, sampling_features, properties, features, collections CASCADE")
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
