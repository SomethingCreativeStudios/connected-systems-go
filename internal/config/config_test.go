package config

import (
	"net/url"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestPubSubMessageClassesDefaultToEnabled(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.PubSub.ResourceData.Enabled)
	require.True(t, cfg.PubSub.ResourceEvents.Enabled)
	require.True(t, cfg.PubSub.BatchResourceEvents.Enabled)
	require.Equal(t, time.Minute, cfg.PubSub.BatchResourceEvents.Window)
	require.Equal(t, "http://localhost:8080", cfg.API.BaseURL)
}

func TestPubSubMessageClassesCanBeDisabledIndependently(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("PUBSUB_RESOURCE_EVENTS_ENABLED", "false")

	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.PubSub.ResourceData.Enabled)
	require.False(t, cfg.PubSub.ResourceEvents.Enabled)
	require.True(t, cfg.PubSub.BatchResourceEvents.Enabled)
}

func TestBatchResourceEventWindowCanBeConfiguredFromEnvironment(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("PUBSUB_BATCH_RESOURCE_EVENTS_WINDOW", "15s")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, 15*time.Second, cfg.PubSub.BatchResourceEvents.Window)
}

func TestBatchResourceEventWindowMustBePositive(t *testing.T) {
	for _, value := range []string{"0s", "-1s", "not-a-duration"} {
		t.Run(value, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			t.Setenv("PUBSUB_BATCH_RESOURCE_EVENTS_WINDOW", value)

			_, err := Load()
			require.Error(t, err)
		})
	}
}

// The DSN escapes credentials, so a password with URL-special characters
// still reaches the server unchanged.
func TestDatabaseDSN(t *testing.T) {
	db := DatabaseConfig{Host: "db.internal", Port: 5432, Name: "connected_systems", User: "cs", Password: "p@ss w/rd?#", SSLMode: "require"}
	require.Equal(t, "postgres://cs:p%40ss%20w%2Frd%3F%23@db.internal:5432/connected_systems?sslmode=require", db.DSN())

	parsed, err := url.Parse(db.DSN())
	require.NoError(t, err)
	password, _ := parsed.User.Password()
	require.Equal(t, "p@ss w/rd?#", password)
}

func TestDatabaseDefaults(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "disable", cfg.Database.SSLMode)
	require.True(t, cfg.Database.AutoMigrate)
}
