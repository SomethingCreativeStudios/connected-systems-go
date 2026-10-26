package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mqtt"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/pubsub"
)

const asyncAPIContentType = "application/vnd.aai.asyncapi+json;version=2.6.0"

type AsyncAPIHandler struct{ cfg *config.Config }

func NewAsyncAPIHandler(cfg *config.Config) *AsyncAPIHandler { return &AsyncAPIHandler{cfg} }

func (h *AsyncAPIHandler) GetAsyncAPI(w http.ResponseWriter, r *http.Request) {
	cfg := h.cfg
	document := map[string]any{
		"asyncapi":                      "2.6.0",
		"info":                          map[string]any{"title": cfg.API.Title + " - MQTT Pub/Sub", "version": cfg.API.Version, "contact": map[string]string{"url": cfg.API.BaseURL}, "description": "Current OGC Connected Systems Part 3 draft; class identifiers and system/subsystem tokens remain provisional."},
		"channels":                      asyncAPIChannels(cfg),
		"x-ogc-provisional-conformance": pubSubConformance(cfg),
	}
	document["defaultContentType"] = mediatype.JSON
	document["tags"] = []map[string]string{{"name": "ConnectedSystems"}}
	if cfg.API.BaseURL != "" {
		document["id"] = strings.TrimRight(cfg.API.BaseURL, "/") + "/asyncapi"
	}
	if cfg.MQTT.Enabled {
		servers := map[string]any{}
		for i, address := range strings.Split(cfg.MQTT.PublicBrokers, ",") {
			parsed, err := url.Parse(strings.TrimSpace(address))
			if err != nil || parsed.Host == "" {
				continue
			}
			protocol := "mqtt"
			if parsed.Scheme == "mqtts" {
				protocol = "mqtts"
			}
			if parsed.Scheme == "ws" || parsed.Scheme == "wss" {
				protocol = parsed.Scheme
			}
			for _, version := range []string{"3.1.1", "5.0"} {
				security := []map[string]any{{"brokerCredentials": []string{}}}
				if cfg.MQTT.AllowAnonymous {
					security = append(security, map[string]any{})
				}
				server := map[string]any{"security": security, "url": parsed.Host + parsed.EscapedPath(), "protocol": protocol, "protocolVersion": version, "description": "Public broker address; authentication is configured by the deployment."}
				if protocol == "ws" || protocol == "wss" {
					server["protocolVersion"] = "13"
					server["bindings"] = map[string]any{"ws": map[string]any{"method": "GET", "bindingVersion": "0.1.0"}}
					server["x-mqtt-version"] = version
				}
				servers[fmt.Sprintf("broker%d-%s", i, strings.ReplaceAll(version, ".", "-"))] = server
			}
		}
		document["servers"] = servers
		document["x-ogc-http-api-root"] = cfg.API.BaseURL
		document["x-ogc-node-id"] = cfg.MQTT.NodeID
		document["x-ogc-identifier-encoding"] = "Encode each identifier byte outside URI unreserved A-Z a-z 0-9 - . _ ~ using uppercase percent escapes, including percent itself. Decode once; noncanonical aliases are rejected."
		document["components"] = map[string]any{"securitySchemes": map[string]any{"brokerCredentials": map[string]any{"type": "userPassword", "description": "Broker-authenticated username scopes idempotency. Anonymous local clients share one principal. Production requires credentials and deployment-provided TLS."}}}
		document["x-ogc-delivery"] = map[string]any{
			"publicationQoS": 1, "subscriptionQoS": []int{0, 1, 2}, "retained": false,
			"mqtt311PersistentClientExpiration": fmt.Sprintf("%dd", int(mqtt.MQTT311SessionExpiry.Hours()/24)), "mqtt5SessionExpiry": "Client negotiated",
			"maxQueuedMessages": mqtt.MaxQueuedMessages, "maxQueuedBytes": mqtt.MaxQueuedBytes, "maxPendingWrites": mqtt.MaxPendingWrites,
			"queueLimits":     "maxQueuedMessages and maxQueuedBytes bound each client's queued QoS 1/2 deliveries. A connected client also drops deliveries beyond maxPendingWrites pending socket writes, for example when its socket stops accepting data or a burst outpaces its writer.",
			"maxMessageBytes": cfg.MQTT.MaxMessageBytes, "maxPacketBytes": mqtt.MaxPacketBytes, "applicationReplay": false,
			"clientLastWill": "CONNECT rejected when a Will is configured", "persistence": "Sessions and queues survive API restarts when the deployment retains broker state",
			"ordering": "MQTT ordering only; no order across topics or representations", "publicationGuarantee": "Best effort from committed API operations; no application outbox or exactly-once execution guarantee",
			"allowAnonymous": cfg.MQTT.AllowAnonymous,
			"security":       "The embedded endpoint authenticates MQTT username/password against its configured accounts. Anonymous access is permitted only when allowAnonymous is true. mqtts/wss addresses provide TLS; mqtt/ws addresses are plaintext. Only the in-process server publisher may publish lifecycle events. All client data is validated before distribution.",
			"wildcards":      "MQTT + and # filters cover authorized present and future topics. No payload filtering, shared subscriptions, or client Last Will publication is offered in the CS API namespace.",
			"errors":         "Failed subscriptions use MQTT SUBACK. Rejected MQTT 5 writes use a PUBACK/PUBREC reason code with a Reason String unless the client disabled problem information; retained or oversized writes and protocol errors disconnect. MQTT 3.1.1 rejections disconnect. MQTT acknowledgement is not command execution status.",
		}
	}
	if pubSubResourceDataConfigured(cfg) {
		document["x-ogc-resource-data-types"] = pubsub.DataTypes()
		document["x-ogc-idempotency"] = map[string]any{
			"status": "Implementation extension; not part of baseline Part 3 conformance", "protocol": "MQTT 5.0", "userProperty": "idempotency-key", "maxUTF8Bytes": 128,
			"scope":         "Canonical HTTP API context and authenticated username; anonymous clients share one principal; client ID is excluded",
			"fingerprint":   "Canonical target collection, media type, and JSON with sorted object keys, insignificant whitespace removed, and number lexemes preserved",
			"retention":     "No automatic expiry; records survive resource deletion",
			"matchingRetry": "Original representation may be delivered again; no new resource, lifecycle event, history revision, alternate-representation publication or batch contribution",
			"conflict":      "Different input with the same key is rejected; a deleted original resource is never recreated",
			"qos":           "Creates accept QoS 0/1; QoS 2 requires MQTT 5 and a key",
			"retry":         "After an uncertain outcome, retry the same input with the same key. Keyless publications are separate creates and may duplicate resources. Command execution is a separate consumer responsibility.",
		}
	}
	if pubSubResourceEventsConfigured(cfg) {
		document["x-ogc-resource-event-types"] = supportedResourceEventTokens(cfg)
	}
	if pubSubBatchResourceEventsConfigured(cfg) {
		document["x-ogc-batch-resource-event-types"] = supportedBatchResourceEventTokens()
	}
	w.Header().Set("Content-Type", asyncAPIContentType)
	_ = json.NewEncoder(w).Encode(document)
}

func asyncAPIChannels(cfg *config.Config) map[string]any {
	channels := map[string]any{}
	if cfg == nil || !cfg.MQTT.Enabled {
		return channels
	}
	for _, capability := range pubsub.Capabilities() {
		if pubSubResourceEventsConfigured(cfg) && (!capability.Batch || !pubSubBatchResourceEventsConfigured(cfg)) {
			name := cfg.MQTT.NodeID + "/" + capability.EventPath
			channels[name] = eventChannel(name, capability, false, capability.ParentCollection != "")
		}
		if pubSubBatchResourceEventsConfigured(cfg) && capability.Batch {
			name := cfg.MQTT.NodeID + "/" + capability.CollectionPath
			channels[name] = eventChannel(name, capability, true, true)
		}
		if pubSubResourceDataConfigured(cfg) && len(capability.MediaTypes) > 0 {
			for _, media := range capability.MediaTypes {
				name := pubsub.DataTopic(cfg.MQTT.NodeID, capability.CollectionPath, media, capability.MediaTypes[0])
				channel := channelWithParameters(name)
				if capability.ClientCreates {
					channel["publish"] = map[string]any{"summary": "Create one " + capability.ResourceType, "message": resourceMessage(capability.ResourceType, media, true)}
				}
				if capability.ServerPublishes {
					channel["subscribe"] = map[string]any{"summary": "Receive complete " + capability.ResourceType + " representations", "message": resourceMessage(capability.ResourceType, media, false)}
				}
				channel["bindings"] = map[string]any{"mqtt": map[string]any{"bindingVersion": "0.2.0"}}
				channel["x-ogc-default-representation"] = capability.MediaTypes[0]
				channel["x-ogc-create-operation"] = "POST " + capability.CreatePath("{parentId}")
				channel["x-ogc-client-publication-qos"] = map[string]any{"mqtt311": []int{0, 1}, "mqtt5WithoutKey": []int{0, 1}, "mqtt5WithKey": []int{0, 1, 2}}
				if capability.ResourceType == "observation" {
					channel["x-ogc-result-schema"] = "Follow the owning datastream's HTTP schema link"
				}
				if capability.ResourceType == "command" {
					channel["x-ogc-parameters-schema"] = "Follow the owning controlstream's HTTP schema link"
				}
				channels[name] = channel
			}
		}
	}
	for name, channel := range channels {
		item := channel.(map[string]any)
		for _, direction := range []string{"publish", "subscribe"} {
			if operation, ok := item[direction].(map[string]any); ok {
				sum := sha256.Sum256([]byte(direction + "/" + name))
				id := fmt.Sprintf("%s_%x", direction, sum[:8])
				operation["operationId"] = id
				operation["message"].(map[string]any)["messageId"] = id + "_message"
			}
		}
	}
	return channels
}

func resourceMessage(kind, media string, create bool) map[string]any {
	properties := map[string]any{"id": map[string]any{"type": "string", "readOnly": true}}
	required := []string{"id"}
	switch kind {
	case "system":
		if media == mediatype.GeoJSON {
			properties["type"] = map[string]any{"type": "string", "const": "Feature"}
			properties["geometry"] = map[string]any{}
			properties["properties"] = map[string]any{"type": "object", "required": []string{"uid", "name", "featureType", "assetType"}}
			required = []string{"type", "properties"}
		} else {
			properties["type"] = map[string]any{"type": "string", "enum": []string{"PhysicalSystem", "PhysicalComponent", "AggregateProcess", "SimpleProcess"}}
			properties["uniqueId"] = map[string]any{"type": "string"}
			required = []string{"type", "uniqueId", "label", "definition"}
		}
	case "datastream", "controlstream":
		properties["name"] = map[string]any{"type": "string", "minLength": 1}
		properties["schema"] = map[string]any{"type": "object", "writeOnly": true}
		if create {
			required = []string{"name", "schema"}
		} else {
			required = []string{"id", "name"}
		}
	case "observation":
		properties["resultTime"] = map[string]any{"type": "string", "format": "date-time"}
		properties["result"] = map[string]any{}
		properties["result@link"] = map[string]any{"type": "object"}
		properties["datastream@id"] = map[string]any{"type": "string", "readOnly": true}
		required = []string{"resultTime"}
		if !create {
			required = append(required, "id", "datastream@id")
		}
	case "command":
		properties["parameters"] = map[string]any{}
		properties["issueTime"] = map[string]any{"type": "string", "format": "date-time", "readOnly": true}
		required = []string{"parameters"}
		if !create {
			required = append(required, "id", "issueTime", "controlstream@id")
		}
	case "commandstatus":
		properties["statusCode"] = map[string]any{"type": "string", "enum": []string{"PENDING", "ACCEPTED", "REJECTED", "SCHEDULED", "UPDATED", "CANCELED", "EXECUTING", "FAILED", "COMPLETED"}}
		required = []string{"statusCode", "reportTime"}
		if !create {
			required = append(required, "id", "command@id")
		}
	case "commandresult":
		properties["data"] = map[string]any{}
		required = nil
		if !create {
			required = []string{"id", "command@id"}
		}
	case "systemevent":
		properties["definition"] = map[string]any{"type": "string"}
		properties["time"] = map[string]any{}
		required = []string{"definition", "label", "time"}
		if !create {
			required = append(required, "id")
		}
	}
	if kind == "system" && !create {
		required = append(required, "id")
	}
	payload := map[string]any{"type": "object", "properties": properties, "description": "Native CS API " + kind + " representation. Create operations apply the associated HTTP create contract, including strict fields and parent/schema validation."}
	if len(required) > 0 {
		payload["required"] = required
	}
	return map[string]any{"name": kind, "contentType": media, "payload": payload}
}

func eventChannel(name string, capability pubsub.Capability, batch, hasParent bool) map[string]any {
	kind := capability.ResourceType
	var eventTypes []string
	for _, op := range capability.Operations {
		eventTypes = append(eventTypes, "org.ogc.api.csapi."+kind+"."+string(op))
	}
	channel := channelWithParameters(name)
	required := []string{"specversion", "type", "source", "subject", "id", "time"}
	if hasParent {
		required = append(required, "resourceparent")
	}
	properties := map[string]any{
		"specversion": map[string]any{"type": "string", "const": "1.0"}, "type": map[string]any{"type": "string", "enum": eventTypes},
		"source": map[string]any{"type": "string", "format": "uri"}, "subject": map[string]any{"type": "string", "format": "uri"}, "resourceparent": map[string]any{"type": "string", "format": "uri"}, "id": map[string]any{"type": "string", "minLength": 1}, "time": map[string]any{"type": "string", "format": "date-time"}, "datacontenttype": map[string]any{"type": "string"}, "data": map[string]any{"type": "object"},
	}
	if batch {
		required = append(required, "data", "datacontenttype")
		properties["data"] = map[string]any{"type": "object", "required": []string{"timerange", "count"}, "properties": map[string]any{"timerange": map[string]any{"type": "array", "minItems": 2, "maxItems": 2, "items": map[string]any{"type": "string", "format": "date-time"}}, "count": map[string]any{"type": "integer", "minimum": 0}}}
	}
	channel["subscribe"] = map[string]any{"message": map[string]any{"name": kind + "Event", "contentType": "application/cloudevents+json", "payload": map[string]any{"type": "object", "required": required, "properties": properties}}}
	return channel
}

var channelParameter = regexp.MustCompile(`\{([^}]+)\}`)

func channelWithParameters(name string) map[string]any {
	channel := map[string]any{}
	parameters := map[string]any{}
	for _, match := range channelParameter.FindAllStringSubmatch(name, -1) {
		pattern := `^(?:[A-Za-z0-9._~-]|%[0-9A-F]{2})+$`
		if match[1] == "systemPath" {
			pattern = `^systems/[^/]+(?:/subsystems/[^/]+)*$`
		}
		if match[1] == "deploymentPath" {
			pattern = `^deployments/[^/]+(?:/subdeployments/[^/]+)*$`
		}
		parameters[match[1]] = map[string]any{"description": "Canonical MQTT path component; IDs use reversible percent encoding", "schema": map[string]any{"type": "string", "pattern": pattern}}
	}
	if len(parameters) > 0 {
		channel["parameters"] = parameters
	}
	return channel
}

func supportedResourceEventTokens(cfg *config.Config) []string {
	return pubsub.EventTokens(pubSubBatchResourceEventsConfigured(cfg), false)
}
func supportedBatchResourceEventTokens() []string { return pubsub.EventTokens(true, true) }
func pubSubResourceDataConfigured(cfg *config.Config) bool {
	return cfg != nil && cfg.MQTT.Enabled && cfg.PubSub.ResourceData.Enabled
}
func pubSubResourceEventsConfigured(cfg *config.Config) bool {
	return cfg != nil && cfg.MQTT.Enabled && cfg.PubSub.ResourceEvents.Enabled
}
func pubSubBatchResourceEventsConfigured(cfg *config.Config) bool {
	return pubSubResourceEventsConfigured(cfg) && cfg.PubSub.BatchResourceEvents.Enabled
}
func pubSubConformance(cfg *config.Config) []string {
	if cfg == nil || !cfg.MQTT.Enabled {
		return nil
	}
	return pubsub.ConformanceClasses(cfg.PubSub)
}
