package config

import (
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
