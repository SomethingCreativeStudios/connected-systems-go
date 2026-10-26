package api

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/pubsub"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
)

func pubSubDiscoveryConfig() *config.Config {
	return &config.Config{API: config.APIConfig{BaseURL: "https://example.test/api", Title: "Connected Systems", Version: "1.0.0"}, MQTT: config.MQTTConfig{Enabled: true, NodeID: "node1", PublicBrokers: "mqtt://broker.example:1883,ws://broker.example:9001/mqtt", MaxMessageBytes: 10 << 20}, PubSub: config.PubSubConfig{ResourceData: config.PubSubFeatureConfig{Enabled: true}, ResourceEvents: config.PubSubFeatureConfig{Enabled: true}, BatchResourceEvents: config.BatchResourceEventsConfig{Enabled: true, Window: time.Minute}}}
}
func discovery(t *testing.T, cfg *config.Config) map[string]any {
	t.Helper()
	r := httptest.NewRecorder()
	NewAsyncAPIHandler(cfg).GetAsyncAPI(r, httptest.NewRequest("GET", "/asyncapi", nil))
	require.Equal(t, 200, r.Code)
	require.Equal(t, asyncAPIContentType, r.Header().Get("Content-Type"))
	var doc map[string]any
	require.NoError(t, json.Unmarshal(r.Body.Bytes(), &doc))
	return doc
}
func TestAsyncAPIDiscoveryEnumeratesEnabledImplementedChannels(t *testing.T) {
	cfg := pubSubDiscoveryConfig()
	doc := discovery(t, cfg)
	require.Equal(t, "2.6.0", doc["asyncapi"])
	require.Len(t, doc["servers"], 4)
	require.NotContains(t, doc, "private-broker")
	channels := doc["channels"].(map[string]any)
	require.Len(t, channels, 25)
	require.Contains(t, channels, "node1/systems/{id}")
	require.Contains(t, channels, "node1/{systemPath}/datastreams/{datastreamId}/observations")
	require.Contains(t, channels, "node1/systems:data/sml-json")
	require.NotContains(t, channels, "systems:events")
	require.ElementsMatch(t, []string{"system", "datastream", "controlstream", "observation", "command", "commandstatus", "commandresult", "systemevent"}, doc["x-ogc-resource-data-types"])
	for name, raw := range channels {
		require.NotContains(t, name, ":events")
		require.NotContains(t, name, ":batch-events")
		c := raw.(map[string]any)
		require.Contains(t, c, "subscribe")
		if _, ok := c["publish"]; ok {
			require.Contains(t, c, "x-ogc-client-publication-qos")
		} else {
			m := c["subscribe"].(map[string]any)["message"].(map[string]any)
			props := m["payload"].(map[string]any)["properties"].(map[string]any)
			require.Contains(t, props, "resourceparent")
			require.NotContains(t, props, "parentid")
		}
	}
	require.Contains(t, doc, "x-ogc-idempotency")
}
func TestPubSubConformanceTracksFeatures(t *testing.T) {
	cfg := pubSubDiscoveryConfig()
	require.Len(t, pubSubConformance(cfg), 5)
	r := httptest.NewRecorder()
	NewConformanceHandler(cfg, zap.NewNop()).GetConformance(r, httptest.NewRequest("GET", "/conformance", nil))
	var decl model.ConformanceDeclaration
	require.NoError(t, json.Unmarshal(r.Body.Bytes(), &decl))
	for _, class := range pubSubConformance(cfg) {
		require.Contains(t, decl.ConformsTo, class)
	}
	cfg.PubSub.BatchResourceEvents.Enabled = false
	require.Len(t, pubSubConformance(cfg), 4)
	doc := discovery(t, cfg)
	c := doc["channels"].(map[string]any)
	require.Contains(t, c, "node1/{systemPath}/datastreams/{datastreamId}/observations/{id}")
	require.NotContains(t, doc, "x-ogc-batch-resource-event-types")
	cfg.PubSub.ResourceEvents.Enabled = false
	doc = discovery(t, cfg)
	require.Len(t, doc["channels"], 11)
	require.Len(t, pubSubConformance(cfg), 3)
	cfg.MQTT.Enabled = false
	doc = discovery(t, cfg)
	require.Empty(t, doc["channels"])
	require.Empty(t, pubSubConformance(cfg))
	require.NotContains(t, doc, "x-ogc-idempotency")
}
func TestLandingPageLinksEnabledPubSubDiscovery(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		cfg := pubSubDiscoveryConfig()
		cfg.MQTT.Enabled = enabled
		r := httptest.NewRecorder()
		NewLandingHandler(cfg, zap.NewNop()).GetLandingPage(r, httptest.NewRequest("GET", "/", nil))
		var page model.LandingPage
		require.NoError(t, json.Unmarshal(r.Body.Bytes(), &page))
		found := false
		for _, link := range page.Links {
			if link.Href == cfg.API.BaseURL+"/asyncapi" {
				found = true
				require.Equal(t, "service-desc", link.Rel)
			}
		}
		require.Equal(t, enabled, found)
	}
}
func TestCheckedInAsyncAPIFixtureMatchesGeneratedDiscovery(t *testing.T) {
	doc := discovery(t, pubSubDiscoveryConfig())
	const path = "../../e2e/schemas/asyncapi/pubsub.yaml"
	if os.Getenv("UPDATE_ASYNCAPI_FIXTURE") == "1" {
		b, err := yaml.Marshal(doc)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, b, 0644))
	}
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	var fixture map[string]any
	require.NoError(t, yaml.Unmarshal(b, &fixture))
	normalized, err := json.Marshal(fixture)
	require.NoError(t, err)
	var actual map[string]any
	require.NoError(t, json.Unmarshal(normalized, &actual))
	require.Equal(t, doc, actual)
}
func TestEventTokenCatalog(t *testing.T) {
	require.Len(t, pubsub.EventTokens(true, false), 30)
	require.Len(t, pubsub.EventTokens(true, true), 6)
	require.Len(t, pubsub.EventTokens(false, false), 36)
}
