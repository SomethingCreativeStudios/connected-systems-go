package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/config"
)

const asyncAPIContentType = "application/vnd.aai.asyncapi+json;version=2.6.0"

var resourceEventTypes = []string{
	"system", "subsystem", "deployment", "procedure", "property", "samplingfeature",
	"datastream", "observation", "controlstream", "command", "commandstatus", "commandresult", "systemevent",
}

var resourceDataTypes = []string{"observation", "command", "commandstatus", "systemevent"}

var batchResourceEventTypes = []string{"observation", "command"}

// AsyncAPIHandler exposes the currently implemented MQTT publish/subscribe
// channels and their supported message types.
type AsyncAPIHandler struct {
	cfg *config.Config
}

func NewAsyncAPIHandler(cfg *config.Config) *AsyncAPIHandler {
	return &AsyncAPIHandler{cfg: cfg}
}

func (h *AsyncAPIHandler) GetAsyncAPI(w http.ResponseWriter, r *http.Request) {
	document := map[string]any{
		"asyncapi": "2.6.0",
		"info": map[string]any{
			"title":       h.cfg.API.Title + " - MQTT Pub/Sub",
			"version":     h.cfg.API.Version,
			"description": "OGC API - Connected Systems MQTT publish/subscribe channels implemented by this server.",
		},
		"channels": asyncAPIChannels(h.cfg),
	}

	if h.cfg.MQTT.Enabled {
		brokerURL, protocol := discoverableBroker(h.cfg.MQTT.Broker)
		document["servers"] = map[string]any{
			"mqtt": map[string]any{
				"url":         brokerURL,
				"protocol":    protocol,
				"description": "Connected Systems MQTT broker",
			},
		}
	}
	if pubSubResourceDataConfigured(h.cfg) {
		document["x-ogc-resource-data-types"] = resourceDataTypes
	}
	if pubSubResourceEventsConfigured(h.cfg) {
		document["x-ogc-resource-event-types"] = supportedResourceEventTokens(h.cfg)
	}
	if pubSubBatchResourceEventsConfigured(h.cfg) {
		document["x-ogc-batch-resource-event-types"] = supportedBatchResourceEventTokens()
	}

	w.Header().Set("Content-Type", asyncAPIContentType)
	if err := json.NewEncoder(w).Encode(document); err != nil {
		http.Error(w, "Failed to encode AsyncAPI document", http.StatusInternalServerError)
	}
}

func asyncAPIChannels(cfg *config.Config) map[string]any {
	channels := map[string]any{}
	if pubSubResourceDataConfigured(cfg) {
		channels["systems/{systemId}/events:data"] = resourceDataChannel(
			"System Event Resource Data Messages for one system", true, false,
			map[string]string{"systemId": "System resource ID"},
		)
		channels["systemEvents:data"] = resourceDataChannel(
			"System Event Resource Data Messages for all systems", true, false, nil,
		)
		channels["datastreams/{dataStreamId}/observations:data"] = resourceDataChannel(
			"Observation Resource Data Messages", true, true,
			map[string]string{"dataStreamId": "Datastream resource ID"},
		)
		channels["controlstreams/{controlStreamId}/commands:data"] = resourceDataChannel(
			"Command Resource Data Messages", true, false,
			map[string]string{"controlStreamId": "Control stream resource ID"},
		)
		channels["commands/{cmdId}/status:data"] = resourceDataChannel(
			"Command Status Resource Data Messages", true, true,
			map[string]string{
				"cmdId": "Command resource ID",
			},
		)
	}
	if pubSubResourceEventsConfigured(cfg) {
		for _, definition := range resourceEventChannelDefinitions(!pubSubBatchResourceEventsConfigured(cfg)) {
			channels[definition.name] = resourceEventChannel(definition.summary, definition.parameters)
		}
	}
	if pubSubBatchResourceEventsConfigured(cfg) {
		channels["datastreams/{dataStreamId}/observations:batch-events"] = batchResourceEventChannel(
			"Batch Resource Events for observations in one datastream",
			map[string]string{"dataStreamId": "Datastream resource ID"},
		)
		channels["controlstreams/{controlStreamId}/commands:batch-events"] = batchResourceEventChannel(
			"Batch Resource Events for commands in one control stream",
			map[string]string{"controlStreamId": "Control stream resource ID"},
		)
	}
	return channels
}

type asyncAPIChannelDefinition struct {
	name       string
	summary    string
	parameters map[string]string
}

func resourceEventChannelDefinitions(includeObservationAndCommand bool) []asyncAPIChannelDefinition {
	definitions := []asyncAPIChannelDefinition{
		{name: "systems:events", summary: "Resource Events for the Systems collection"},
		{name: "systems/{systemId}:events", summary: "Resource Events for one System or Subsystem", parameters: map[string]string{"systemId": "System resource ID"}},
		{name: "systems/{systemId}/subsystems:events", summary: "Resource Events for one System's Subsystems", parameters: map[string]string{"systemId": "Parent System resource ID"}},
		{name: "deployments:events", summary: "Resource Events for the Deployments collection"},
		{name: "deployments/{deploymentId}:events", summary: "Resource Events for one Deployment", parameters: map[string]string{"deploymentId": "Deployment resource ID"}},
		{name: "procedures:events", summary: "Resource Events for the Procedures collection"},
		{name: "procedures/{procedureId}:events", summary: "Resource Events for one Procedure", parameters: map[string]string{"procedureId": "Procedure resource ID"}},
		{name: "properties:events", summary: "Resource Events for the Properties collection"},
		{name: "properties/{propertyId}:events", summary: "Resource Events for one Property", parameters: map[string]string{"propertyId": "Property resource ID"}},
		{name: "samplingFeatures:events", summary: "Resource Events for the Sampling Features collection"},
		{name: "systems/{systemId}/samplingFeatures:events", summary: "Resource Events for one System's Sampling Features", parameters: map[string]string{"systemId": "System resource ID"}},
		{name: "samplingFeatures/{samplingFeatureId}:events", summary: "Resource Events for one Sampling Feature", parameters: map[string]string{"samplingFeatureId": "Sampling Feature resource ID"}},
		{name: "datastreams:events", summary: "Resource Events for the Datastreams collection"},
		{name: "systems/{systemId}/datastreams:events", summary: "Resource Events for one System's Datastreams", parameters: map[string]string{"systemId": "System resource ID"}},
		{name: "datastreams/{dataStreamId}:events", summary: "Resource Events for one Datastream", parameters: map[string]string{"dataStreamId": "Datastream resource ID"}},
		{name: "controlstreams:events", summary: "Resource Events for the Control Streams collection"},
		{name: "systems/{systemId}/controlstreams:events", summary: "Resource Events for one System's Control Streams", parameters: map[string]string{"systemId": "System resource ID"}},
		{name: "controlstreams/{controlStreamId}:events", summary: "Resource Events for one Control Stream", parameters: map[string]string{"controlStreamId": "Control Stream resource ID"}},
		{name: "commands/{cmdId}/status:events", summary: "Resource Events for one Command's Status collection", parameters: map[string]string{"cmdId": "Command resource ID"}},
		{name: "commands/{cmdId}/status/{statusId}:events", summary: "Resource Events for one Command Status", parameters: map[string]string{"cmdId": "Command resource ID", "statusId": "Command Status resource ID"}},
		{name: "commands/{cmdId}/result:events", summary: "Resource Events for one Command's Result collection", parameters: map[string]string{"cmdId": "Command resource ID"}},
		{name: "commands/{cmdId}/result/{resultId}:events", summary: "Resource Events for one Command Result", parameters: map[string]string{"cmdId": "Command resource ID", "resultId": "Command Result resource ID"}},
		{name: "systems/{systemId}/events:events", summary: "Resource Events for one System's System Event collection", parameters: map[string]string{"systemId": "System resource ID"}},
		{name: "systems/{systemId}/events/{eventId}:events", summary: "Resource Events for one System Event", parameters: map[string]string{"systemId": "System resource ID", "eventId": "System Event resource ID"}},
	}
	if includeObservationAndCommand {
		definitions = append(definitions,
			asyncAPIChannelDefinition{name: "datastreams/{dataStreamId}/observations:events", summary: "Resource Events for one Datastream's Observations", parameters: map[string]string{"dataStreamId": "Datastream resource ID"}},
			asyncAPIChannelDefinition{name: "observations/{observationId}:events", summary: "Resource Events for one Observation", parameters: map[string]string{"observationId": "Observation resource ID"}},
			asyncAPIChannelDefinition{name: "controlstreams/{controlStreamId}/commands:events", summary: "Resource Events for one Control Stream's Commands", parameters: map[string]string{"controlStreamId": "Control Stream resource ID"}},
			asyncAPIChannelDefinition{name: "commands/{cmdId}:events", summary: "Resource Events for one Command", parameters: map[string]string{"cmdId": "Command resource ID"}},
		)
	}
	return definitions
}

func resourceDataChannel(summary string, serverPublishes, serverAccepts bool, parameters map[string]string) map[string]any {
	channel := channelWithParameters(parameters)
	message := map[string]any{
		"name":        "resourceData",
		"title":       "Resource Data Message",
		"contentType": "application/json",
		"payload": map[string]any{
			"type":                 "object",
			"description":          "A complete native representation of one supported Connected Systems resource.",
			"additionalProperties": true,
		},
	}
	if serverPublishes {
		channel["subscribe"] = map[string]any{"summary": summary, "message": message}
	}
	if serverAccepts {
		channel["publish"] = map[string]any{"summary": "Publish " + summary, "message": message}
	}
	return channel
}

func resourceEventChannel(summary string, parameters map[string]string) map[string]any {
	channel := channelWithParameters(parameters)
	channel["subscribe"] = map[string]any{
		"summary": summary,
		"message": map[string]any{
			"name":        "resourceEvent",
			"title":       "Resource Event",
			"contentType": "application/cloudevents+json",
			"payload": map[string]any{
				"type":     "object",
				"required": []string{"specversion", "type", "source", "subject", "id", "time"},
				"properties": map[string]any{
					"specversion":     map[string]any{"type": "string", "const": "1.0"},
					"type":            map[string]any{"type": "string", "pattern": `^org\.ogc\.api\.consys\.[a-z]+\.(create|update|delete)$`},
					"source":          map[string]any{"type": "string", "format": "uri"},
					"subject":         map[string]any{"type": "string", "format": "uri"},
					"id":              map[string]any{"type": "string", "minLength": 1},
					"parentId":        map[string]any{"type": "string", "format": "uri"},
					"time":            map[string]any{"type": "string", "format": "date-time"},
					"datacontenttype": map[string]any{"type": "string", "const": "application/json"},
					"data": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name":        map[string]any{"type": "string", "minLength": 1},
							"description": map[string]any{"type": "string", "minLength": 1},
							"uniqueId":    map[string]any{"type": "string", "minLength": 1, "format": "uri"},
						},
						"additionalProperties": true,
					},
				},
			},
		},
	}
	return channel
}

func batchResourceEventChannel(summary string, parameters map[string]string) map[string]any {
	channel := channelWithParameters(parameters)
	channel["subscribe"] = map[string]any{
		"summary": summary,
		"message": map[string]any{
			"name":        "batchResourceEvent",
			"title":       "Batch Resource Event",
			"contentType": "application/cloudevents+json",
			"payload": map[string]any{
				"type": "object",
				"required": []string{
					"specversion", "type", "source", "subject", "id", "parentId", "time", "datacontenttype", "data",
				},
				"properties": map[string]any{
					"specversion":     map[string]any{"type": "string", "const": "1.0"},
					"type":            map[string]any{"type": "string", "pattern": `^org\.ogc\.api\.consys\.(observation|command)\.(create|update|delete)$`},
					"source":          map[string]any{"type": "string", "format": "uri"},
					"subject":         map[string]any{"type": "string", "format": "uri"},
					"id":              map[string]any{"type": "string", "minLength": 1},
					"parentId":        map[string]any{"type": "string", "format": "uri"},
					"time":            map[string]any{"type": "string", "format": "date-time"},
					"datacontenttype": map[string]any{"type": "string", "const": "application/json"},
					"data": map[string]any{
						"type":     "object",
						"required": []string{"timerange", "count"},
						"properties": map[string]any{
							"timerange": map[string]any{
								"type":     "array",
								"minItems": 2,
								"maxItems": 2,
								"items":    map[string]any{"type": "string", "format": "date-time"},
							},
							"count": map[string]any{"type": "integer", "minimum": 0},
						},
					},
				},
			},
		},
	}
	return channel
}

func channelWithParameters(parameters map[string]string) map[string]any {
	channel := map[string]any{}
	if len(parameters) == 0 {
		return channel
	}
	values := make(map[string]any, len(parameters))
	for name, description := range parameters {
		values[name] = map[string]any{
			"description": description,
			"schema":      map[string]any{"type": "string", "minLength": 1},
		}
	}
	channel["parameters"] = values
	return channel
}

func supportedResourceEventTokens(cfg *config.Config) []string {
	tokens := make([]string, 0, len(resourceEventTypes)*3)
	for _, resourceType := range resourceEventTypes {
		if pubSubBatchResourceEventsConfigured(cfg) && (resourceType == "observation" || resourceType == "command") {
			continue
		}
		for _, operation := range []string{"create", "update", "delete"} {
			tokens = append(tokens, resourceType+"."+operation)
		}
	}
	return tokens
}

func supportedBatchResourceEventTokens() []string {
	tokens := make([]string, 0, len(batchResourceEventTypes)*3)
	for _, resourceType := range batchResourceEventTypes {
		for _, operation := range []string{"create", "update", "delete"} {
			tokens = append(tokens, resourceType+"."+operation)
		}
	}
	return tokens
}

func pubSubResourceDataConfigured(cfg *config.Config) bool {
	return cfg != nil && cfg.MQTT.Enabled && cfg.PubSub.ResourceData.Enabled
}

func pubSubResourceEventsConfigured(cfg *config.Config) bool {
	return cfg != nil && cfg.MQTT.Enabled && cfg.PubSub.ResourceEvents.Enabled
}

func pubSubBatchResourceEventsConfigured(cfg *config.Config) bool {
	return cfg != nil && cfg.MQTT.Enabled && cfg.PubSub.BatchResourceEvents.Enabled
}

func discoverableBroker(raw string) (string, string) {
	protocol := "mqtt"
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" {
		return raw, protocol
	}
	if parsed.Scheme == "ssl" || parsed.Scheme == "tls" || parsed.Scheme == "mqtts" {
		protocol = "mqtts"
	}
	parsed.User = nil
	return parsed.Host + parsed.EscapedPath(), protocol
}
