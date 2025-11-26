package e2e

import (
	"context"
	"net/http/httptest"
	"os"
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
