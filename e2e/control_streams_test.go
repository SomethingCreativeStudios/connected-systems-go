package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ControlStreamJSONSchema = "json/controlStream-bundled.json"
	CommandJSONSchema       = "json/command-bundled.json"
)

// ---------------------------------------------------------------------------
// Helper: base payloads
// ---------------------------------------------------------------------------

func baseControlStreamPayload() map[string]interface{} {
	uniqueID := uuid.NewString()
	return map[string]interface{}{
		"uid":       "urn:uuid:" + uniqueID,
		"name":      "ControlStream " + uniqueID[:8],
		"inputName": "thermostat-set-point",
		"async":     false,
		"formats":   []string{"application/json"},
		"schema": map[string]interface{}{
			"commandFormat": "application/json",
			"parametersSchema": map[string]interface{}{
				"type": "DataRecord",
				"fields": []map[string]interface{}{
					{
						"name":       "setPoint",
						"type":       "Quantity",
						"definition": "http://sensorml.com/ont/swe/property/Temperature",
						"label":      "Set Point",
						"uom":        map[string]interface{}{"code": "Cel"},
					},
				},
			},
		},
	}
}

func baseCommandPayload() map[string]interface{} {
	return map[string]interface{}{
		"parameters": map[string]interface{}{
			"setPoint": 22.5,
		},
	}
}

func createCommandViaAPI(t *testing.T, controlStreamID string, payload map[string]interface{}) string {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+controlStreamID+"/commands", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	id := parseID(resp.Header.Get("Location"), "/commands/")
	require.NotEmpty(t, id)
	return id
}

// createControlStreamViaAPI creates a control stream under a given system and returns the new ID.
func createControlStreamViaAPI(t *testing.T, systemID string, payload map[string]interface{}) string {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/controlstreams", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, "POST /systems/{id}/controlstreams must return 201")

	location := resp.Header.Get("Location")
	require.NotEmpty(t, location, "response must include a Location header")

	csID := parseID(location, "/controlstreams/")
	require.NotEmpty(t, csID, "unable to parse control stream ID from Location: %s", location)
	return csID
}

// createSystemForControlStreamTest is a thin helper that returns a bare system ID.
func createSystemForControlStreamTest(t *testing.T) string {
	t.Helper()
	sys := map[string]interface{}{
		"type": "Feature",
		"properties": map[string]interface{}{
			"uid":         "urn:uuid:" + uuid.NewString(),
			"name":        "Test System " + uuid.NewString()[:8],
			"featureType": "http://www.w3.org/ns/sosa/Sensor",
		},
	}
	body, _ := json.Marshal(sys)
	req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/systems", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/geo+json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	location := resp.Header.Get("Location")
	id := parseID(location, "/systems/")
	require.NotEmpty(t, id)
	return id
}

func requireControlStreamSchemaOrSkip(t *testing.T, body []byte, schema string) {
	t.Helper()
	err := validateAgainstSchema(t, body, schema)
	if err != nil && strings.Contains(err.Error(), "failed to compile schema") {
		t.Skipf("skipping schema assertion due invalid bundled schema refs: %v", err)
	}
	require.NoError(t, err)
}

// =============================================================================
// Conformance Class: /conf/control-stream
// Requirement: /req/control-stream/resources-endpoint
// Abstract Test: /ats/control-stream/resources-endpoint
// https://docs.ogc.org/is/23-002/23-002.html
// =============================================================================

func TestControlStream_ResourcesEndpoint(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/controlstreams", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var collection map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &collection))

	items, ok := collection["items"].([]interface{})
	require.True(t, ok, "response must contain 'items' array")
	require.GreaterOrEqual(t, len(items), 1)
	links, ok := collection["links"].([]interface{})
	require.True(t, ok, "response must contain 'links' array")
	require.GreaterOrEqual(t, len(links), 1)

	found := false
	for _, item := range items {
		if obj, ok := item.(map[string]interface{}); ok {
			if id, _ := obj["id"].(string); id == csID {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "created control stream must appear in /controlstreams listing")
}

// =============================================================================
// Conformance Class: /conf/control-stream
// Requirement: /req/control-stream/canonical-url
// =============================================================================

func TestControlStream_CanonicalURL(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	payload := baseControlStreamPayload()
	csID := createControlStreamViaAPI(t, systemID, payload)

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/controlstreams/"+csID, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var cs map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&cs))
	assert.Equal(t, csID, cs["id"])
	assert.Equal(t, payload["name"], cs["name"])
	assert.Equal(t, payload["inputName"], cs["inputName"])
}

// =============================================================================
// Conformance Class: /conf/control-stream
// Requirement: /req/control-stream/system-link
// The control stream must expose a system@link referencing the parent system.
// =============================================================================

func TestControlStream_SystemLink(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/controlstreams/"+csID, nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var cs map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&cs))

	systemLink, ok := cs["system@link"].(map[string]interface{})
	require.True(t, ok, "control stream must have a 'system@link' object")
	href, _ := systemLink["href"].(string)
	assert.Contains(t, href, systemID, "system@link href must reference the parent system ID")

	links, ok := cs["links"].([]interface{})
	require.True(t, ok, "control stream must expose a links array")

	foundCommandsLink := false
	for _, rawLink := range links {
		link, ok := rawLink.(map[string]interface{})
		if !ok {
			continue
		}
		rel, _ := link["rel"].(string)
		if rel == "ogc-rel:commands" || rel == "commands" {
			commandHref, _ := link["href"].(string)
			assert.Equal(t, testServer.URL+"/controlstreams/"+csID+"/commands", commandHref)
			foundCommandsLink = true
			break
		}
	}
	assert.True(t, foundCommandsLink, "control stream must expose a commands association link")
}

// =============================================================================
// Conformance Class: /conf/control-stream
// Requirement: /req/control-stream/system-sub-collection
// GET /systems/{id}/controlstreams must return this system's control streams.
// =============================================================================

func TestControlStream_SystemSubCollection(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/systems/"+systemID+"/controlstreams", nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var collection map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&collection))

	items, ok := collection["items"].([]interface{})
	require.True(t, ok)
	links, ok := collection["links"].([]interface{})
	require.True(t, ok, "response must contain 'links' array")
	require.GreaterOrEqual(t, len(links), 1)

	found := false
	for _, item := range items {
		if obj, ok := item.(map[string]interface{}); ok {
			if id, _ := obj["id"].(string); id == csID {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "control stream must appear in /systems/{id}/controlstreams")
}

// =============================================================================
// Conformance Class: /conf/json
// Requirement: /req/json/control-stream-schema
// Response body must validate against the bundled JSON schema.
// =============================================================================

func TestControlStreamSchema_JSON(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/controlstreams/"+csID, nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	requireControlStreamSchemaOrSkip(t, body, ControlStreamJSONSchema)
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/control-stream
// Requirement: /req/create-replace-delete/control-stream
// Server SHALL support full CRUD on control streams.
// =============================================================================

func TestControlStreamCRUD(t *testing.T) {
	cleanupDB(t)
	systemID := createSystemForControlStreamTest(t)

	// CREATE
	payload := baseControlStreamPayload()
	csID := createControlStreamViaAPI(t, systemID, payload)
	assert.NotEmpty(t, csID)

	// READ
	getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/controlstreams/"+csID, nil)
	getReq.Header.Set("Accept", "application/json")
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var fetched map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&fetched))
	assert.Equal(t, payload["name"], fetched["name"])

	// UPDATE (replace)
	updated := baseControlStreamPayload()
	updated["name"] = "Updated Control Stream Name"
	updBody, _ := json.Marshal(updated)
	putReq, _ := http.NewRequest(http.MethodPut, testServer.URL+"/controlstreams/"+csID, bytes.NewReader(updBody))
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := http.DefaultClient.Do(putReq)
	require.NoError(t, err)
	defer putResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, putResp.StatusCode)

	// Verify update
	getReq2, _ := http.NewRequest(http.MethodGet, testServer.URL+"/controlstreams/"+csID, nil)
	getReq2.Header.Set("Accept", "application/json")
	getResp2, err := http.DefaultClient.Do(getReq2)
	require.NoError(t, err)
	defer getResp2.Body.Close()
	var refetched map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp2.Body).Decode(&refetched))
	assert.Equal(t, "Updated Control Stream Name", refetched["name"])

	// DELETE
	delReq, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/controlstreams/"+csID, nil)
	delResp, err := http.DefaultClient.Do(delReq)
	require.NoError(t, err)
	defer delResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)

	// Verify deletion
	getReq3, _ := http.NewRequest(http.MethodGet, testServer.URL+"/controlstreams/"+csID, nil)
	getResp3, err := http.DefaultClient.Do(getReq3)
	require.NoError(t, err)
	defer getResp3.Body.Close()
	assert.Equal(t, http.StatusNotFound, getResp3.StatusCode)
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/control-stream
// Requirement: /req/create-replace-delete/control-stream
// DELETE /controlstreams/{id}?cascade=true must delete the control stream and its commands.
// =============================================================================
func TestControlStream_DeleteCascade_RemovesCommands(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	commandBody, err := json.Marshal(baseCommandPayload())
	require.NoError(t, err)
	createCommandReq, err := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+csID+"/commands", bytes.NewReader(commandBody))
	require.NoError(t, err)
	createCommandReq.Header.Set("Content-Type", "application/json")

	createCommandResp, err := http.DefaultClient.Do(createCommandReq)
	require.NoError(t, err)
	defer createCommandResp.Body.Close()
	require.Equal(t, http.StatusCreated, createCommandResp.StatusCode)

	commandID := parseID(createCommandResp.Header.Get("Location"), "/commands/")
	require.NotEmpty(t, commandID)

	deleteReq, err := http.NewRequest(http.MethodDelete, testServer.URL+"/controlstreams/"+csID+"?cascade=true", nil)
	require.NoError(t, err)
	deleteResp, err := http.DefaultClient.Do(deleteReq)
	require.NoError(t, err)
	defer deleteResp.Body.Close()
	require.Equal(t, http.StatusNoContent, deleteResp.StatusCode)

	getControlStreamResp := doGet(t, "/controlstreams/"+csID)
	defer getControlStreamResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, getControlStreamResp.StatusCode)

	getCommandResp := doGet(t, "/commands/"+commandID)
	defer getCommandResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, getCommandResp.StatusCode)
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/control-stream
// Requirement: /req/create-replace-delete/control-stream-schema
// Server SHALL support schema sub-resource on control streams.
// =============================================================================

func TestControlStream_SchemaSubResource(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	// Initial schema GET
	schemaReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/controlstreams/"+csID+"/schema", nil)
	schemaResp, err := http.DefaultClient.Do(schemaReq)
	require.NoError(t, err)
	defer schemaResp.Body.Close()
	require.Equal(t, http.StatusOK, schemaResp.StatusCode)

	// Update schema
	newSchema := map[string]interface{}{
		"commandFormat": "application/json",
		"parametersSchema": map[string]interface{}{
			"type": "DataRecord",
			"fields": []map[string]interface{}{
				{"name": "targetTemperature", "type": "Quantity", "definition": "http://sensorml.com/ont/swe/property/Temperature", "label": "Target Temperature", "uom": map[string]interface{}{"code": "Cel"}},
				{"name": "mode", "type": "Text", "definition": "http://sensorml.com/ont/swe/property/OperatingMode", "label": "Mode"},
			},
		},
	}
	schemaBody, _ := json.Marshal(newSchema)

	putReq, _ := http.NewRequest(http.MethodPut, testServer.URL+"/controlstreams/"+csID+"/schema", bytes.NewReader(schemaBody))
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := http.DefaultClient.Do(putReq)
	require.NoError(t, err)
	defer putResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, putResp.StatusCode)

	// Verify the updated schema
	getSchemaReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/controlstreams/"+csID+"/schema", nil)
	getSchemaResp, err := http.DefaultClient.Do(getSchemaReq)
	require.NoError(t, err)
	defer getSchemaResp.Body.Close()
	require.Equal(t, http.StatusOK, getSchemaResp.StatusCode)

	var returnedSchema map[string]interface{}
	require.NoError(t, json.NewDecoder(getSchemaResp.Body).Decode(&returnedSchema))
	assert.Equal(t, "application/json", returnedSchema["commandFormat"])
}

// =============================================================================
// Conformance Class: /conf/command
// Requirement: /req/command/resources-endpoint
// GET /commands must return all existing commands.
// =============================================================================

func TestCommand_ResourcesEndpoint(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	// Create a command
	cmdBody, _ := json.Marshal(baseCommandPayload())
	postReq, _ := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+csID+"/commands", bytes.NewReader(cmdBody))
	postReq.Header.Set("Content-Type", "application/json")
	postResp, err := http.DefaultClient.Do(postReq)
	require.NoError(t, err)
	defer postResp.Body.Close()
	require.Equal(t, http.StatusCreated, postResp.StatusCode)

	cmdLocation := postResp.Header.Get("Location")
	cmdID := parseID(cmdLocation, "/commands/")
	require.NotEmpty(t, cmdID)

	// List all commands
	listReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/commands", nil)
	listReq.Header.Set("Accept", "application/json")
	listResp, err := http.DefaultClient.Do(listReq)
	require.NoError(t, err)
	defer listResp.Body.Close()
	assert.Equal(t, http.StatusOK, listResp.StatusCode)

	var collection map[string]interface{}
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&collection))

	items, ok := collection["items"].([]interface{})
	require.True(t, ok, "response must contain 'items' array")
	require.GreaterOrEqual(t, len(items), 1)
	links, ok := collection["links"].([]interface{})
	require.True(t, ok, "response must contain 'links' array")
	require.GreaterOrEqual(t, len(links), 1)

	found := false
	for _, item := range items {
		if obj, ok := item.(map[string]interface{}); ok {
			if id, _ := obj["id"].(string); id == cmdID {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "created command must appear in /commands listing")
}

// =============================================================================
// Conformance Class: /conf/command
// Requirement: /req/command/canonical-url
// =============================================================================

func TestCommand_CanonicalURL(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	cmdBody, _ := json.Marshal(baseCommandPayload())
	postReq, _ := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+csID+"/commands", bytes.NewReader(cmdBody))
	postReq.Header.Set("Content-Type", "application/json")
	postResp, err := http.DefaultClient.Do(postReq)
	require.NoError(t, err)
	defer postResp.Body.Close()
	require.Equal(t, http.StatusCreated, postResp.StatusCode)

	location := postResp.Header.Get("Location")
	cmdID := parseID(location, "/commands/")

	getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/commands/"+cmdID, nil)
	getReq.Header.Set("Accept", "application/json")
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	var cmd map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&cmd))
	assert.Equal(t, cmdID, cmd["id"])
	assert.Equal(t, csID, cmd["controlstream@id"])
}

// =============================================================================
// Conformance Class: /conf/command
// Requirement: /req/command/controlstream-sub-collection
// GET /controlstreams/{id}/commands must return commands for that stream.
// =============================================================================

func TestCommand_ControlStreamSubCollection(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	// Create two commands
	for range [2]struct{}{} {
		body, _ := json.Marshal(baseCommandPayload())
		req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+csID+"/commands", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
	}

	listReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/controlstreams/"+csID+"/commands", nil)
	listResp, err := http.DefaultClient.Do(listReq)
	require.NoError(t, err)
	defer listResp.Body.Close()
	assert.Equal(t, http.StatusOK, listResp.StatusCode)

	var collection map[string]interface{}
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&collection))

	items, ok := collection["items"].([]interface{})
	require.True(t, ok)
	links, ok := collection["links"].([]interface{})
	require.True(t, ok, "response must contain 'links' array")
	require.GreaterOrEqual(t, len(links), 1)
	assert.GreaterOrEqual(t, len(items), 2, "expected at least 2 commands for this control stream")
}

// =============================================================================
// Conformance Class: /conf/command
// Requirement: /req/command/issue-time
// issueTime MUST be set by the server on creation.
// =============================================================================

func TestCommand_IssueTimeSetByServer(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	// Payload intentionally omits issueTime
	payload := map[string]interface{}{
		"parameters": map[string]interface{}{"setPoint": 21.0},
	}
	body, _ := json.Marshal(payload)
	postReq, _ := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+csID+"/commands", bytes.NewReader(body))
	postReq.Header.Set("Content-Type", "application/json")
	postResp, err := http.DefaultClient.Do(postReq)
	require.NoError(t, err)
	defer postResp.Body.Close()
	require.Equal(t, http.StatusCreated, postResp.StatusCode)

	cmdID := parseID(postResp.Header.Get("Location"), "/commands/")

	getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/commands/"+cmdID, nil)
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()

	var cmd map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&cmd))
	assert.NotEmpty(t, cmd["issueTime"], "server must set issueTime on command creation")
}

// =============================================================================
// Conformance Class: /conf/command
// Requirement: /req/command/status
// Newly created commands must have PENDING status.
// =============================================================================

func TestCommand_InitialStatusIsPending(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	body, _ := json.Marshal(baseCommandPayload())
	req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+csID+"/commands", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	cmdID := parseID(resp.Header.Get("Location"), "/commands/")
	getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/commands/"+cmdID, nil)
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()

	var cmd map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&cmd))
	assert.Equal(t, "PENDING", cmd["currentStatus"], "newly created command must have PENDING status")
}

// =============================================================================
// Conformance Class: /conf/json
// Requirement: /req/json/command-schema
// GET /commands/{id} response must validate against bundled command schema.
// =============================================================================

func TestCommandSchema_JSON(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	body, _ := json.Marshal(baseCommandPayload())
	req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+csID+"/commands", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	cmdID := parseID(resp.Header.Get("Location"), "/commands/")
	getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/commands/"+cmdID, nil)
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()

	responseBody, err := io.ReadAll(getResp.Body)
	require.NoError(t, err)

	err = validateAgainstSchema(t, responseBody, CommandJSONSchema)
	if err != nil && strings.Contains(err.Error(), "failed to compile schema") {
		t.Skipf("skipping command schema validation due invalid bundled schema refs: %v", err)
	}
	require.NoError(t, err, "GET /commands/{id} response must validate against command JSON schema")
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/command
// Requirement: /req/create-replace-delete/command
// Server SHALL support CRUD on commands.
// =============================================================================

func TestCommandCRUD(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	// CREATE
	body, _ := json.Marshal(baseCommandPayload())
	postReq, _ := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+csID+"/commands", bytes.NewReader(body))
	postReq.Header.Set("Content-Type", "application/json")
	postResp, err := http.DefaultClient.Do(postReq)
	require.NoError(t, err)
	defer postResp.Body.Close()
	require.Equal(t, http.StatusCreated, postResp.StatusCode)
	cmdID := parseID(postResp.Header.Get("Location"), "/commands/")

	// READ
	getResp := doGet(t, "/commands/"+cmdID)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var cmd map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&cmd))
	assert.Equal(t, csID, cmd["controlstream@id"])

	// UPDATE: parameters are writable; client-supplied currentStatus is
	// read-only and must be ignored (only status reports change it).
	updated := map[string]interface{}{
		"currentStatus": "ACCEPTED",
		"parameters":    map[string]interface{}{"setPoint": 25.0},
	}
	updBody, _ := json.Marshal(updated)
	putReq, _ := http.NewRequest(http.MethodPut, testServer.URL+"/commands/"+cmdID, bytes.NewReader(updBody))
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := http.DefaultClient.Do(putReq)
	require.NoError(t, err)
	defer putResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, putResp.StatusCode)

	// Verify update: parameters changed, currentStatus untouched (read-only).
	getResp2 := doGet(t, "/commands/"+cmdID)
	defer getResp2.Body.Close()
	var refetched map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp2.Body).Decode(&refetched))
	assert.Equal(t, "PENDING", refetched["currentStatus"], "currentStatus is read-only; only status reports may change it")
	params, ok := refetched["parameters"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 25.0, params["setPoint"])

	// DELETE
	delReq, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/commands/"+cmdID, nil)
	delResp, err := http.DefaultClient.Do(delReq)
	require.NoError(t, err)
	defer delResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)

	// Verify deletion
	getResp3 := doGet(t, "/commands/"+cmdID)
	defer getResp3.Body.Close()
	assert.Equal(t, http.StatusNotFound, getResp3.StatusCode)
}

func TestCommandStatusReports_CRUDAndEnvelope(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())
	cmdID := createCommandViaAPI(t, csID, baseCommandPayload())

	payload := map[string]interface{}{
		"reportTime":        "2026-04-01T12:00:00Z",
		"statusCode":        "EXECUTING",
		"percentCompletion": 50.0,
		"message":           "Command is running",
	}
	body, _ := json.Marshal(payload)
	postReq, _ := http.NewRequest(http.MethodPost, testServer.URL+"/commands/"+cmdID+"/status", bytes.NewReader(body))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Accept", "application/json")
	postResp, err := http.DefaultClient.Do(postReq)
	require.NoError(t, err)
	defer postResp.Body.Close()
	require.Equal(t, http.StatusCreated, postResp.StatusCode)

	statusID := parseID(postResp.Header.Get("Location"), "/status/")
	require.NotEmpty(t, statusID)

	getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/commands/"+cmdID+"/status/"+statusID, nil)
	getReq.Header.Set("Accept", "application/json")
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var status map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&status))
	assert.Equal(t, statusID, status["id"])
	assert.Equal(t, cmdID, status["command@id"])
	assert.Equal(t, "EXECUTING", status["statusCode"])
	assert.Equal(t, 50.0, status["percentCompletion"])

	cmdResp := doGet(t, "/commands/"+cmdID)
	defer cmdResp.Body.Close()
	var cmd map[string]interface{}
	require.NoError(t, json.NewDecoder(cmdResp.Body).Decode(&cmd))
	assert.Equal(t, "EXECUTING", cmd["currentStatus"], "status report must update command currentStatus")

	listReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/commands/"+cmdID+"/status?statusCode=EXECUTING&reportTime=2026-04-01T00:00:00Z/2026-04-02T00:00:00Z", nil)
	listReq.Header.Set("Accept", "application/json")
	listResp, err := http.DefaultClient.Do(listReq)
	require.NoError(t, err)
	defer listResp.Body.Close()
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	listBody, err := io.ReadAll(listResp.Body)
	require.NoError(t, err)
	items := getJSONItems(t, listBody)
	require.Len(t, items, 1)

	update := map[string]interface{}{
		"statusCode":        "COMPLETED",
		"percentCompletion": 100.0,
		"message":           "Command finished",
	}
	updateBody, _ := json.Marshal(update)
	putReq, _ := http.NewRequest(http.MethodPut, testServer.URL+"/commands/"+cmdID+"/status/"+statusID, bytes.NewReader(updateBody))
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := http.DefaultClient.Do(putReq)
	require.NoError(t, err)
	defer putResp.Body.Close()
	require.Equal(t, http.StatusNoContent, putResp.StatusCode)

	refetch := doGet(t, "/commands/"+cmdID+"/status/"+statusID)
	defer refetch.Body.Close()
	var updatedStatus map[string]interface{}
	require.NoError(t, json.NewDecoder(refetch.Body).Decode(&updatedStatus))
	assert.Equal(t, "COMPLETED", updatedStatus["statusCode"])

	cmdResp2 := doGet(t, "/commands/"+cmdID)
	defer cmdResp2.Body.Close()
	var updatedCmd map[string]interface{}
	require.NoError(t, json.NewDecoder(cmdResp2.Body).Decode(&updatedCmd))
	assert.Equal(t, "COMPLETED", updatedCmd["currentStatus"])

	delReq, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/commands/"+cmdID+"/status/"+statusID, nil)
	delResp, err := http.DefaultClient.Do(delReq)
	require.NoError(t, err)
	defer delResp.Body.Close()
	require.Equal(t, http.StatusNoContent, delResp.StatusCode)

	missing := doGet(t, "/commands/"+cmdID+"/status/"+statusID)
	defer missing.Body.Close()
	assert.Equal(t, http.StatusNotFound, missing.StatusCode)
}

func TestCommandResults_CRUDAndEnvelope(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())
	cmdID := createCommandViaAPI(t, csID, baseCommandPayload())

	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"accepted": true,
			"value":    22.5,
		},
	}
	body, _ := json.Marshal(payload)
	postReq, _ := http.NewRequest(http.MethodPost, testServer.URL+"/commands/"+cmdID+"/result", bytes.NewReader(body))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Accept", "application/json")
	postResp, err := http.DefaultClient.Do(postReq)
	require.NoError(t, err)
	defer postResp.Body.Close()
	require.Equal(t, http.StatusCreated, postResp.StatusCode)

	resultID := parseID(postResp.Header.Get("Location"), "/result/")
	require.NotEmpty(t, resultID)

	getResp := doGet(t, "/commands/"+cmdID+"/result/"+resultID)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&result))
	assert.Equal(t, resultID, result["id"])
	assert.Equal(t, cmdID, result["command@id"])
	data, ok := result["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, data["accepted"])

	listResp := doGet(t, "/commands/"+cmdID+"/result")
	defer listResp.Body.Close()
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	listBody, err := io.ReadAll(listResp.Body)
	require.NoError(t, err)
	items := getJSONItems(t, listBody)
	require.Len(t, items, 1)

	update := map[string]interface{}{
		"external@link": map[string]interface{}{
			"href":  "https://example.org/results/command-output.json",
			"type":  "application/json",
			"title": "External result",
		},
	}
	updateBody, _ := json.Marshal(update)
	putReq, _ := http.NewRequest(http.MethodPut, testServer.URL+"/commands/"+cmdID+"/result/"+resultID, bytes.NewReader(updateBody))
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := http.DefaultClient.Do(putReq)
	require.NoError(t, err)
	defer putResp.Body.Close()
	require.Equal(t, http.StatusNoContent, putResp.StatusCode)

	refetch := doGet(t, "/commands/"+cmdID+"/result/"+resultID)
	defer refetch.Body.Close()
	var updatedResult map[string]interface{}
	require.NoError(t, json.NewDecoder(refetch.Body).Decode(&updatedResult))
	require.NotContains(t, updatedResult, "data")
	externalLink, ok := updatedResult["external@link"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "https://example.org/results/command-output.json", externalLink["href"])

	delReq, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/commands/"+cmdID+"/result/"+resultID, nil)
	delResp, err := http.DefaultClient.Do(delReq)
	require.NoError(t, err)
	defer delResp.Body.Close()
	require.Equal(t, http.StatusNoContent, delResp.StatusCode)

	missing := doGet(t, "/commands/"+cmdID+"/result/"+resultID)
	defer missing.Body.Close()
	assert.Equal(t, http.StatusNotFound, missing.StatusCode)
}

func TestCommandStatusAndResults_CursorPagination(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	controlStreamID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())
	commandID := createCommandViaAPI(t, controlStreamID, baseCommandPayload())

	for range 3 {
		createCommandStatusReportViaAPI(t, commandID, map[string]interface{}{
			"reportTime": "2026-04-01T12:00:00Z", // equal timestamps exercise the ID tie-breaker
			"statusCode": "EXECUTING",
		})
	}
	assertJSONCursorTraversal(t, "/commands/"+commandID+"/status?limit=1&statusCode=EXECUTING", 3)

	for range 3 {
		createCommandResultViaAPI(t, commandID, map[string]interface{}{
			"data": map[string]interface{}{"accepted": true},
		})
	}
	assertJSONCursorTraversal(t, "/commands/"+commandID+"/result?limit=1", 3)
}

func createCommandStatusReportViaAPI(t *testing.T, commandID string, payload map[string]interface{}) string {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/commands/"+commandID+"/status", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	return parseID(resp.Header.Get("Location"), "/status/")
}

func createCommandResultViaAPI(t *testing.T, commandID string, payload map[string]interface{}) string {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/commands/"+commandID+"/result", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	return parseID(resp.Header.Get("Location"), "/result/")
}

func assertJSONCursorTraversal(t *testing.T, path string, expectedCount int) {
	t.Helper()

	readPage := func(href string) ([]string, map[string]string) {
		t.Helper()
		if !strings.HasPrefix(href, "http") {
			href = testServer.URL + href
		}
		resp, err := http.Get(href)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var collection map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&collection))
		items, ok := collection["items"].([]interface{})
		require.True(t, ok)
		ids := make([]string, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.(map[string]interface{})["id"].(string))
		}

		links := map[string]string{}
		for _, raw := range collection["links"].([]interface{}) {
			link := raw.(map[string]interface{})
			links[link["rel"].(string)] = link["href"].(string)
		}
		return ids, links
	}

	firstIDs, links := readPage(path)
	require.Len(t, firstIDs, 1)
	seen := map[string]bool{firstIDs[0]: true}
	firstID := firstIDs[0]
	pageCount := 1
	for links["next"] != "" {
		nextURL := links["next"]
		assert.Contains(t, nextURL, "cursor=")
		assert.NotContains(t, nextURL, "offset=")
		if queryStart := strings.Index(path, "?"); queryStart >= 0 {
			for _, parameter := range strings.Split(path[queryStart+1:], "&") {
				if !strings.HasPrefix(parameter, "limit=") {
					assert.Contains(t, nextURL, parameter, "next links must preserve active filters")
				}
			}
		}
		ids, nextLinks := readPage(nextURL)
		require.Len(t, ids, 1)
		assert.False(t, seen[ids[0]], "cursor pages must not repeat an item")
		seen[ids[0]] = true
		pageCount++

		if pageCount == 2 {
			prevURL := nextLinks["prev"]
			require.NotEmpty(t, prevURL)
			prevIDs, _ := readPage(prevURL)
			require.Equal(t, []string{firstID}, prevIDs)
		}
		links = nextLinks
	}
	assert.Len(t, seen, expectedCount)
}

func TestCommandResult_RejectsMultipleVariants(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())
	cmdID := createCommandViaAPI(t, csID, baseCommandPayload())

	payload := map[string]interface{}{
		"data": map[string]interface{}{"value": 1},
		"external@link": map[string]interface{}{
			"href": "https://example.org/results/result.json",
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/commands/"+cmdID+"/result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	var validation map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&validation))
	require.NotEmpty(t, validation["details"])

	list := doGet(t, "/commands/"+cmdID+"/result")
	defer list.Body.Close()
	var results map[string]any
	require.NoError(t, json.NewDecoder(list.Body).Decode(&results))
	require.Empty(t, results["items"])
}

func TestCommandStatus_RejectsInvalidStatusCode(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())
	cmdID := createCommandViaAPI(t, csID, baseCommandPayload())
	body, err := json.Marshal(map[string]any{"statusCode": "NOT_A_STATUS"})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/commands/"+cmdID+"/status", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	var validation map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&validation))
	details := validation["details"].([]any)
	require.Equal(t, "statusCode", details[0].(map[string]any)["path"])

	list := doGet(t, "/commands/"+cmdID+"/status")
	defer list.Body.Close()
	var statuses map[string]any
	require.NoError(t, json.NewDecoder(list.Body).Decode(&statuses))
	require.Empty(t, statuses["items"])
}

// =============================================================================
// Conformance Class: /conf/command
// Requirement: /req/command/controlstream-parent-404
// POST to unknown control stream must return 404.
// =============================================================================

func TestCommand_PostToNonexistentControlStream(t *testing.T) {
	cleanupDB(t)

	body, _ := json.Marshal(baseCommandPayload())
	req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+uuid.NewString()+"/commands", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// =============================================================================
// Conformance Class: /conf/command
// Requirement: /req/command/filter-by-controlstream
// GET /commands?controlStream={id} must filter results.
// =============================================================================

func TestCommand_FilterByControlStream(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID1 := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())
	csID2 := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	// Two commands in cs1, one in cs2
	for range [2]struct{}{} {
		b, _ := json.Marshal(baseCommandPayload())
		r, _ := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+csID1+"/commands", bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		resp, _ := http.DefaultClient.Do(r)
		resp.Body.Close()
	}
	b, _ := json.Marshal(baseCommandPayload())
	r, _ := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+csID2+"/commands", bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(r)
	resp.Body.Close()

	// Filter by csID1
	url := fmt.Sprintf("%s/commands?controlStream=%s", testServer.URL, csID1)
	listReq, _ := http.NewRequest(http.MethodGet, url, nil)
	listResp, err := http.DefaultClient.Do(listReq)
	require.NoError(t, err)
	defer listResp.Body.Close()
	assert.Equal(t, http.StatusOK, listResp.StatusCode)

	var collection map[string]interface{}
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&collection))
	items, ok := collection["items"].([]interface{})
	require.True(t, ok)
	links, ok := collection["links"].([]interface{})
	require.True(t, ok, "response must contain 'links' array")
	require.GreaterOrEqual(t, len(links), 1)
	assert.Equal(t, 2, len(items), "filter by controlStream should return exactly 2 commands")

	for _, item := range items {
		obj, _ := item.(map[string]interface{})
		assert.Equal(t, csID1, obj["controlstream@id"], "all returned commands must belong to csID1")
	}
}

// =============================================================================
// Error handling: GET /controlstreams/{unknown} must return 404
// =============================================================================

func TestControlStream_GetNonexistent(t *testing.T) {
	cleanupDB(t)
	resp := doGet(t, "/controlstreams/"+uuid.NewString())
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// =============================================================================
// Error handling: GET /commands/{unknown} must return 404
// =============================================================================

func TestCommand_GetNonexistent(t *testing.T) {
	cleanupDB(t)
	resp := doGet(t, "/commands/"+uuid.NewString())
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// =============================================================================
// ?issueTime=latest must return only the single control stream with the
// most-recent issue_time_start value regardless of insertion order.
// =============================================================================

func TestControlStream_List_LatestIssueTime_ReturnsMostRecent(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)

	// issueTime is a read-only extent computed from commands, so seed each
	// control stream with one command carrying the desired issue time.
	csWithIssueTime := func(issueTime string) string {
		csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())
		createCommandViaAPI(t, csID, map[string]interface{}{
			"parameters": map[string]interface{}{"setPoint": 21.0},
			"issueTime":  issueTime,
		})
		return csID
	}

	// Create 3 control streams with issueTime starts in non-monotonic order.
	csWithIssueTime("2025-01-01T00:00:00Z")
	newestID := csWithIssueTime("2027-01-01T00:00:00Z")
	csWithIssueTime("2024-01-01T00:00:00Z")

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/controlstreams?issueTime=latest", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	items := getJSONItems(t, body)
	require.Equal(t, 1, len(items), "expected exactly one control stream for ?issueTime=latest")

	cs, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, newestID, cs["id"], "expected the control stream with the most-recent issueTime start")
}

// =============================================================================
// ?issueTime=latest must return only the single command with the most-recent
// issue_time value regardless of insertion order.
// =============================================================================

func TestCommand_List_LatestIssueTime_ReturnsMostRecent(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	postCommand := func(issueTime string) string {
		t.Helper()
		payload := map[string]interface{}{
			"parameters": map[string]interface{}{"setPoint": 21.0},
			"issueTime":  issueTime,
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+csID+"/commands", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		return parseID(resp.Header.Get("Location"), "/commands/")
	}

	// Create 3 commands with issueTime in non-monotonic order.
	postCommand("2025-01-01T00:00:00Z")
	newestID := postCommand("2027-01-01T00:00:00Z")
	postCommand("2024-01-01T00:00:00Z")

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/commands?issueTime=latest", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	items := getJSONItems(t, body)
	require.Equal(t, 1, len(items), "expected exactly one command for ?issueTime=latest")

	cmd, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, newestID, cmd["id"], "expected the command with the most-recent issueTime")
}

// ---------------------------------------------------------------------------
// doGet is a tiny helper to do a GET request on the test server.
// ---------------------------------------------------------------------------

func doGet(t *testing.T, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, testServer.URL+path, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}
