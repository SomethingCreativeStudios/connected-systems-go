package api

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
)

func pubSubDiscoveryConfig() *config.Config {
	return &config.Config{
		API: config.APIConfig{
			BaseURL:     "https://example.test/api",
			Title:       "Connected Systems",
			Description: "Test API",
			Version:     "1.0.0",
		},
		MQTT: config.MQTTConfig{
			Enabled: true,
			Broker:  "tcp://user:secret@broker.example:1883/mqtt",
		},
		PubSub: config.PubSubConfig{
			ResourceData:        config.PubSubFeatureConfig{Enabled: true},
			ResourceEvents:      config.PubSubFeatureConfig{Enabled: true},
			BatchResourceEvents: config.BatchResourceEventsConfig{Enabled: true, Window: time.Minute},
		},
	}
}

func TestAsyncAPIDiscoveryEnumeratesEnabledImplementedChannels(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/asyncapi", nil)
	NewAsyncAPIHandler(pubSubDiscoveryConfig()).GetAsyncAPI(recorder, request)

	require.Equal(t, 200, recorder.Code)
	require.Equal(t, asyncAPIContentType, recorder.Header().Get("Content-Type"))
	var document map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &document))
	require.Equal(t, "2.6.0", document["asyncapi"])

	channels := document["channels"].(map[string]any)
	require.Len(t, channels, 31)
	require.Contains(t, channels, "datastreams/{dataStreamId}/observations:data")
	require.Contains(t, channels, "systems:events")
	require.Contains(t, channels, "systems/{systemId}:events")
	require.Contains(t, channels, "datastreams/{dataStreamId}/observations:batch-events")
	require.Contains(t, channels, "controlstreams/{controlStreamId}/commands:batch-events")
	require.NotContains(t, channels, "datastreams/{dataStreamId}/observations:events")
	observationChannel := channels["datastreams/{dataStreamId}/observations:data"].(map[string]any)
	require.Contains(t, observationChannel, "subscribe")
	require.Contains(t, observationChannel, "publish")
	resourceEventChannel := channels["systems:events"].(map[string]any)
	require.Contains(t, resourceEventChannel, "subscribe")
	require.NotContains(t, resourceEventChannel, "publish")
	resourceEventMessage := resourceEventChannel["subscribe"].(map[string]any)["message"].(map[string]any)
	resourceEventPayload := resourceEventMessage["payload"].(map[string]any)
	resourceEventProperties := resourceEventPayload["properties"].(map[string]any)
	require.Equal(t, "application/json", resourceEventProperties["datacontenttype"].(map[string]any)["const"])
	resourceEventData := resourceEventProperties["data"].(map[string]any)
	require.Equal(t, true, resourceEventData["additionalProperties"])
	resourceEventSummaryProperties := resourceEventData["properties"].(map[string]any)
	require.ElementsMatch(t, []string{"name", "description", "uniqueId"}, mapKeys(resourceEventSummaryProperties))
	require.Equal(t, "uri", resourceEventSummaryProperties["uniqueId"].(map[string]any)["format"])
	batchChannel := channels["datastreams/{dataStreamId}/observations:batch-events"].(map[string]any)
	batchMessage := batchChannel["subscribe"].(map[string]any)["message"].(map[string]any)
	require.Equal(t, "application/cloudevents+json", batchMessage["contentType"])
	batchPayload := batchMessage["payload"].(map[string]any)
	require.ElementsMatch(t, []any{
		"specversion", "type", "source", "subject", "id", "parentId", "time", "datacontenttype", "data",
	}, batchPayload["required"])
	batchData := batchPayload["properties"].(map[string]any)["data"].(map[string]any)
	require.ElementsMatch(t, []any{"timerange", "count"}, batchData["required"])
	timerange := batchData["properties"].(map[string]any)["timerange"].(map[string]any)
	require.Equal(t, float64(2), timerange["minItems"])
	require.Equal(t, float64(2), timerange["maxItems"])

	eventTypes := document["x-ogc-resource-event-types"].([]any)
	require.Len(t, eventTypes, 33)
	require.NotContains(t, eventTypes, "observation.create")
	require.NotContains(t, eventTypes, "command.create")
	require.Contains(t, eventTypes, "systemevent.delete")
	batchEventTypes := document["x-ogc-batch-resource-event-types"].([]any)
	require.ElementsMatch(t, []any{
		"observation.create", "observation.update", "observation.delete",
		"command.create", "command.update", "command.delete",
	}, batchEventTypes)
	dataTypes := document["x-ogc-resource-data-types"].([]any)
	require.ElementsMatch(t, []any{"observation", "command", "commandstatus", "systemevent"}, dataTypes)

	servers := document["servers"].(map[string]any)
	server := servers["mqtt"].(map[string]any)
	require.Equal(t, "broker.example:1883/mqtt", server["url"])
}

func TestPubSubDraftDoesNotOverstateFormalConformance(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/conformance", nil)
	NewConformanceHandler(pubSubDiscoveryConfig(), zap.NewNop()).GetConformance(recorder, request)

	var declaration model.ConformanceDeclaration
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &declaration))
	for _, uri := range declaration.ConformsTo {
		require.NotContains(t, uri, "ogcapi-connectedsystems-3")
	}
}

func TestPubSubDiscoveryIsHiddenWhenMQTTIsDisabled(t *testing.T) {
	cfg := pubSubDiscoveryConfig()
	cfg.MQTT.Enabled = false

	recorder := httptest.NewRecorder()
	NewAsyncAPIHandler(cfg).GetAsyncAPI(recorder, httptest.NewRequest("GET", "/asyncapi", nil))
	var document map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &document))
	require.Empty(t, document["channels"].(map[string]any))
	require.NotContains(t, document, "x-ogc-resource-event-types")
	require.NotContains(t, document, "x-ogc-batch-resource-event-types")
	require.NotContains(t, document, "x-ogc-resource-data-types")

	recorder = httptest.NewRecorder()
	NewLandingHandler(cfg, zap.NewNop()).GetLandingPage(recorder, httptest.NewRequest("GET", "/", nil))
	var landing model.LandingPage
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &landing))
	for _, link := range landing.Links {
		require.NotEqual(t, cfg.API.BaseURL+"/asyncapi", link.Href)
	}
}

func TestPubSubDiscoveryTracksIndependentClassSwitches(t *testing.T) {
	cfg := pubSubDiscoveryConfig()
	cfg.PubSub.ResourceEvents.Enabled = false

	recorder := httptest.NewRecorder()
	NewAsyncAPIHandler(cfg).GetAsyncAPI(recorder, httptest.NewRequest("GET", "/asyncapi", nil))
	var document map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &document))
	channels := document["channels"].(map[string]any)
	require.Len(t, channels, 7)
	require.NotContains(t, channels, "systems:events")
	require.NotContains(t, document, "x-ogc-resource-event-types")
	require.Contains(t, channels, "datastreams/{dataStreamId}/observations:batch-events")
	require.Contains(t, document, "x-ogc-batch-resource-event-types")
	require.Contains(t, document, "x-ogc-resource-data-types")
}

func TestPubSubDiscoveryFallsBackToIndividualEventsWhenBatchingIsDisabled(t *testing.T) {
	cfg := pubSubDiscoveryConfig()
	cfg.PubSub.BatchResourceEvents.Enabled = false

	recorder := httptest.NewRecorder()
	NewAsyncAPIHandler(cfg).GetAsyncAPI(recorder, httptest.NewRequest("GET", "/asyncapi", nil))
	var document map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &document))
	channels := document["channels"].(map[string]any)
	require.Len(t, channels, 33)
	require.NotContains(t, channels, "datastreams/{dataStreamId}/observations:batch-events")
	require.Contains(t, channels, "datastreams/{dataStreamId}/observations:events")
	require.Contains(t, channels, "observations/{observationId}:events")
	require.Contains(t, channels, "controlstreams/{controlStreamId}/commands:events")
	require.NotContains(t, document, "x-ogc-batch-resource-event-types")
	eventTypes := document["x-ogc-resource-event-types"].([]any)
	require.Len(t, eventTypes, 39)
	require.Contains(t, eventTypes, "observation.create")
	require.Contains(t, eventTypes, "command.delete")
}

func TestLandingPageLinksDiscoveryWhenOnlyBatchEventsAreEnabled(t *testing.T) {
	cfg := pubSubDiscoveryConfig()
	cfg.PubSub.ResourceData.Enabled = false
	cfg.PubSub.ResourceEvents.Enabled = false

	recorder := httptest.NewRecorder()
	NewLandingHandler(cfg, zap.NewNop()).GetLandingPage(recorder, httptest.NewRequest("GET", "/", nil))
	var landing model.LandingPage
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &landing))
	for _, link := range landing.Links {
		if link.Href == cfg.API.BaseURL+"/asyncapi" {
			return
		}
	}
	t.Fatal("expected AsyncAPI link for independently enabled batch events")
}

func TestLandingPageLinksEnabledPubSubDiscovery(t *testing.T) {
	cfg := pubSubDiscoveryConfig()
	recorder := httptest.NewRecorder()
	NewLandingHandler(cfg, zap.NewNop()).GetLandingPage(recorder, httptest.NewRequest("GET", "/", nil))

	var landing model.LandingPage
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &landing))
	var found bool
	for _, link := range landing.Links {
		if link.Href == cfg.API.BaseURL+"/asyncapi" {
			found = true
			require.Equal(t, "service-desc", link.Rel)
			require.Equal(t, asyncAPIContentType, link.Type)
		}
	}
	require.True(t, found)
}

func TestCheckedInAsyncAPIFixtureMatchesGeneratedDiscovery(t *testing.T) {
	fixtureBytes, err := os.ReadFile("../../e2e/schemas/asyncapi/pubsub.yaml")
	require.NoError(t, err)
	var fixture map[string]any
	require.NoError(t, yaml.Unmarshal(fixtureBytes, &fixture))

	recorder := httptest.NewRecorder()
	NewAsyncAPIHandler(pubSubDiscoveryConfig()).GetAsyncAPI(recorder, httptest.NewRequest("GET", "/asyncapi", nil))
	var generated map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &generated))

	require.ElementsMatch(t, mapKeys(fixture["channels"].(map[string]any)), mapKeys(generated["channels"].(map[string]any)))
	require.ElementsMatch(t, fixture["x-ogc-resource-data-types"], generated["x-ogc-resource-data-types"])
	require.ElementsMatch(t, fixture["x-ogc-resource-event-types"], generated["x-ogc-resource-event-types"])
	require.ElementsMatch(t, fixture["x-ogc-batch-resource-event-types"], generated["x-ogc-batch-resource-event-types"])

	fixtureMessages := fixture["components"].(map[string]any)["messages"].(map[string]any)
	fixtureResourceEventPayload := fixtureMessages["resourceEvent"].(map[string]any)["payload"].(map[string]any)
	fixtureResourceEventData := fixtureResourceEventPayload["properties"].(map[string]any)["data"]
	fixtureResourceEventDataJSON, err := json.Marshal(fixtureResourceEventData)
	require.NoError(t, err)
	var normalizedFixtureResourceEventData any
	require.NoError(t, json.Unmarshal(fixtureResourceEventDataJSON, &normalizedFixtureResourceEventData))
	generatedChannels := generated["channels"].(map[string]any)
	generatedResourceEvent := generatedChannels["systems:events"].(map[string]any)
	generatedResourceEventPayload := generatedResourceEvent["subscribe"].(map[string]any)["message"].(map[string]any)["payload"].(map[string]any)
	generatedResourceEventData := generatedResourceEventPayload["properties"].(map[string]any)["data"]
	require.Equal(t, normalizedFixtureResourceEventData, generatedResourceEventData)
}

func mapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
