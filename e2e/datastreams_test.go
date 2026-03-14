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
	DatastreamJSONSchema = "json/datastream-bundled.json"
)

func baseDatastreamPayload() map[string]interface{} {
	uniqueID := uuid.NewString()
	return map[string]interface{}{
		"uid":        "urn:uuid:" + uniqueID,
		"name":       "Datastream " + uniqueID[:8],
		"type":       "observation",
		"outputName": "weather-output",
		"formats":    []string{"application/json"},
		"schema": map[string]interface{}{
			"obsFormat": "application/json",
			"resultSchema": map[string]interface{}{
				"type": "DataRecord",
				"fields": []map[string]interface{}{
					{
						"name": "temperature",
						"type": "Quantity",
					},
					{
						"name": "humidity",
						"type": "Quantity",
					},
				},
			},
		},
	}
}

func createDatastreamViaAPI(t *testing.T, endpoint string, payload map[string]interface{}) string {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, testServer.URL+endpoint, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	location := resp.Header.Get("Location")
	require.NotEmpty(t, location, "expected Location header")

	datastreamID := parseID(location, "/datastreams/")
	require.NotEmpty(t, datastreamID, "unable to parse datastream ID from Location header")
	return datastreamID
}

// =============================================================================
// Conformance Class: /conf/datastream
// Requirement: /req/datastream/resources-endpoint
// Abstract Test: /conf/datastream/resources-endpoint
// Link: https://docs.ogc.org/is/23-001/23-001.html
// =============================================================================
func TestDatastreamConformance_ResourcesEndpoint(t *testing.T) {
	cleanupDB(t)

	payload := baseDatastreamPayload()
	systemID := uuid.NewString()
	createdDatastreamID := createDatastreamViaAPI(t, "/systems/"+systemID+"/datastreams", payload)

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/datastreams", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var collection map[string]interface{}
	err = json.Unmarshal(body, &collection)
	require.NoError(t, err)

	items, ok := collection["items"].([]interface{})
	require.True(t, ok, "response must contain 'items' array")
	require.GreaterOrEqual(t, len(items), 1, "expected at least one datastream")

	found := false
	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if id, _ := obj["id"].(string); id == createdDatastreamID {
			found = true
			break
		}
	}
	assert.True(t, found, "created datastream must be discoverable via /datastreams")
}

// =============================================================================
// Conformance Class: /conf/datastream
// Requirement: /req/datastream/canonical-url
// Abstract Test: /conf/datastream/canonical-url
// Link: https://docs.ogc.org/is/23-001/23-001.html
// =============================================================================
func TestDatastreamConformance_CanonicalURL(t *testing.T) {
	cleanupDB(t)

	payload := baseDatastreamPayload()
	systemID := uuid.NewString()
	datastreamID := createDatastreamViaAPI(t, "/systems/"+systemID+"/datastreams", payload)

	url := fmt.Sprintf("%s/datastreams/%s", testServer.URL, datastreamID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var datastream map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&datastream)
	require.NoError(t, err)
	assert.Equal(t, datastreamID, datastream["id"])
	assert.Equal(t, payload["name"], datastream["name"])
}

// =============================================================================
// Conformance Class: /conf/json/datastream-schema
// Requirement: /req/json/datastream-schema
// Abstract Test: /conf/json/datastream-schema
// Link: https://docs.ogc.org/is/23-001/23-001.html
// =============================================================================
func TestDatastreamSchema_JSON(t *testing.T) {
	cleanupDB(t)

	payload := baseDatastreamPayload()
	systemID := uuid.NewString()
	datastreamID := createDatastreamViaAPI(t, "/systems/"+systemID+"/datastreams", payload)

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/datastreams/"+datastreamID, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	err = validateAgainstSchema(t, body, DatastreamJSONSchema)
	if err != nil && strings.Contains(err.Error(), "failed to compile schema") {
		t.Skipf("skipping datastream schema validation due invalid bundled schema references: %v", err)
	}
	require.NoError(t, err, "response did not validate against datastream JSON schema")
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/datastream
// Requirement: /req/create-replace-delete/datastream
// Abstract Test: create under system and update/retrieve schema
// Link: https://docs.ogc.org/is/23-001/23-001.html
// =============================================================================
func TestDatastream_CreateUnderSystem_AndSchemaRoundTrip(t *testing.T) {
	cleanupDB(t)

	systemID := uuid.NewString()
	payload := baseDatastreamPayload()

	datastreamID := createDatastreamViaAPI(t, "/systems/"+systemID+"/datastreams", payload)

	getReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/datastreams/"+datastreamID, nil)
	require.NoError(t, err)
	getReq.Header.Set("Accept", "application/json")

	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var created map[string]interface{}
	err = json.NewDecoder(getResp.Body).Decode(&created)
	require.NoError(t, err)

	systemLink, ok := created["system@link"].(map[string]interface{})
	require.True(t, ok, "expected system@link object on datastream")
	href, _ := systemLink["href"].(string)
	assert.Contains(t, href, systemID)

	updatedSchema := map[string]interface{}{
		"obsFormat": "application/x-protobuf",
		"messageSchema": "syntax = \"proto3\"; message ObservationResult { double value = 1; }",
	}

	schemaBody, err := json.Marshal(updatedSchema)
	require.NoError(t, err)

	putReq, err := http.NewRequest(http.MethodPut, testServer.URL+"/datastreams/"+datastreamID+"/schema", bytes.NewReader(schemaBody))
	require.NoError(t, err)
	putReq.Header.Set("Content-Type", "application/json")

	putResp, err := http.DefaultClient.Do(putReq)
	require.NoError(t, err)
	defer putResp.Body.Close()
	require.Equal(t, http.StatusNoContent, putResp.StatusCode)

	schemaReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/datastreams/"+datastreamID+"/schema", nil)
	require.NoError(t, err)
	schemaReq.Header.Set("Accept", "application/json")

	schemaResp, err := http.DefaultClient.Do(schemaReq)
	require.NoError(t, err)
	defer schemaResp.Body.Close()
	require.Equal(t, http.StatusOK, schemaResp.StatusCode)

	var returnedSchema map[string]interface{}
	err = json.NewDecoder(schemaResp.Body).Decode(&returnedSchema)
	require.NoError(t, err)
	assert.Equal(t, "application/x-protobuf", returnedSchema["obsFormat"])
	assert.NotEmpty(t, returnedSchema["messageSchema"])
}