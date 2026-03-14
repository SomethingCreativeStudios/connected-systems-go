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
	SystemGeoSchema = "geojson/system-bundled.json"
	SystemSMLSchema = "sensorml/system-bundled.json"
)

func baseSystemPayload(name string) map[string]interface{} {
	uid := "urn:uuid:" + uuid.NewString()
	return map[string]interface{}{
		"type": "Feature",
		"properties": map[string]interface{}{
			"uid":         uid,
			"name":        name,
			"featureType": "http://www.w3.org/ns/sosa/Sensor",
			"assetType":   "Equipment",
		},
		"geometry": map[string]interface{}{
			"type":        "Point",
			"coordinates": []float64{-117.1625, 32.715},
		},
	}
}

func createSystemViaAPI(t *testing.T, endpoint string, payload map[string]interface{}) string {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, testServer.URL+endpoint, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/geo+json")
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	location := resp.Header.Get("Location")
	require.NotEmpty(t, location)
	id := parseID(location, "/systems/")
	require.NotEmpty(t, id)
	return id
}

func getFeatureCollectionIDs(t *testing.T, body []byte) []string {
	t.Helper()
	var collection map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &collection))

	features, ok := collection["features"].([]interface{})
	require.True(t, ok, "response must contain features array")

	ids := make([]string, 0, len(features))
	for _, f := range features {
		obj, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		if id, ok := obj["id"].(string); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func requireSchemaOrSkip(t *testing.T, body []byte, schema string) {
	t.Helper()
	err := validateAgainstSchema(t, body, schema)
	if err != nil && strings.Contains(err.Error(), "failed to compile schema") {
		t.Skipf("skipping schema assertion due invalid bundled schema refs: %v", err)
	}
	require.NoError(t, err)
}

// =============================================================================
// Conformance Class: /conf/system
// Requirement: /req/system/resources-endpoint
// Abstract Test: /conf/system/resources-endpoint (A.10)
// =============================================================================
func TestSystemConformance_ResourcesEndpoint(t *testing.T) {
	cleanupDB(t)

	createdID := createSystemViaAPI(t, "/systems", baseSystemPayload("System Resources Endpoint"))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	ids := getFeatureCollectionIDs(t, body)
	assert.Contains(t, ids, createdID)
}

// =============================================================================
// Conformance Class: /conf/system
// Requirement: /req/system/canonical-endpoint
// Abstract Test: /conf/system/canonical-endpoint (A.11)
// =============================================================================
func TestSystemConformance_CanonicalEndpoint(t *testing.T) {
	cleanupDB(t)
	createSystemViaAPI(t, "/systems", baseSystemPayload("System Canonical Endpoint"))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// =============================================================================
// Conformance Class: /conf/system
// Requirement: /req/system/canonical-url
// Abstract Test: /conf/system/canonical-url (A.9)
// =============================================================================
func TestSystemConformance_CanonicalURL(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("System Canonical URL"))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems/"+systemID, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var feature map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&feature))
	assert.Equal(t, systemID, feature["id"])
}

// =============================================================================
// Conformance Class: /conf/system
// Requirement: /req/system/location-time (+ /rec/system/location)
// Abstract Test: /conf/system/location and /conf/system/location-time (A.7/A.8)
// =============================================================================
func TestSystemConformance_LocationAndLocationTime(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("System Location"))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems/"+systemID, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var feature map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&feature))
	_, hasGeometry := feature["geometry"]
	assert.True(t, hasGeometry, "physical system should expose geometry when provided")
}

// =============================================================================
// Conformance Class: /conf/geojson and /conf/sensorml
// Requirements: /req/geojson/system-schema and /req/sensorml/system-schema
// Abstract Tests: /conf/geojson/system-schema (A.88), /conf/sensorml/system-schema (A.101)
// =============================================================================
func TestSystemSchema_GeoJSON(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("System Schema GeoJSON"))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems/"+systemID, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	requireSchemaOrSkip(t, body, SystemGeoSchema)
}

func TestSystemSchema_SensorML(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("System Schema SensorML"))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems/"+systemID, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/sml+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	requireSchemaOrSkip(t, body, SystemSMLSchema)
}

// =============================================================================
// Conformance Class: /conf/subsystem
// Requirement: /req/subsystem/collection
// Abstract Test: /conf/subsystem/collection (A.13)
// =============================================================================
func TestSubsystemConformance_Collection(t *testing.T) {
	cleanupDB(t)

	parentID := createSystemViaAPI(t, "/systems", baseSystemPayload("Subsystem Parent"))
	childID := createSystemViaAPI(t, "/systems/"+parentID+"/subsystems", baseSystemPayload("Subsystem Child"))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems/"+parentID+"/subsystems", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	ids := getFeatureCollectionIDs(t, body)
	assert.Contains(t, ids, childID)
}

// =============================================================================
// Conformance Class: /conf/subsystem
// Requirements: /req/subsystem/recursive-search-systems and /req/subsystem/recursive-search-subsystems
// Abstract Tests: /conf/subsystem/recursive-search-systems (A.15), /conf/subsystem/recursive-search-subsystems (A.16)
// =============================================================================
func TestSubsystemConformance_RecursiveSearchSystems(t *testing.T) {
	cleanupDB(t)

	parentID := createSystemViaAPI(t, "/systems", baseSystemPayload("Recursive Parent"))
	childID := createSystemViaAPI(t, "/systems/"+parentID+"/subsystems", baseSystemPayload("Recursive Child"))
	grandchildID := createSystemViaAPI(t, "/systems/"+childID+"/subsystems", baseSystemPayload("Recursive Grandchild"))

	nonRecReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems?recursive=false", nil)
	require.NoError(t, err)
	nonRecReq.Header.Set("Accept", "application/geo+json")
	nonRecResp, err := http.DefaultClient.Do(nonRecReq)
	require.NoError(t, err)
	defer nonRecResp.Body.Close()
	require.Equal(t, http.StatusOK, nonRecResp.StatusCode)
	nonRecBody, err := io.ReadAll(nonRecResp.Body)
	require.NoError(t, err)
	nonRecIDs := getFeatureCollectionIDs(t, nonRecBody)
	assert.Contains(t, nonRecIDs, parentID)
	assert.NotContains(t, nonRecIDs, childID)
	assert.NotContains(t, nonRecIDs, grandchildID)

	recReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems?recursive=true", nil)
	require.NoError(t, err)
	recReq.Header.Set("Accept", "application/geo+json")
	recResp, err := http.DefaultClient.Do(recReq)
	require.NoError(t, err)
	defer recResp.Body.Close()
	require.Equal(t, http.StatusOK, recResp.StatusCode)
	recBody, err := io.ReadAll(recResp.Body)
	require.NoError(t, err)
	recIDs := getFeatureCollectionIDs(t, recBody)
	assert.Contains(t, recIDs, parentID)
	assert.Contains(t, recIDs, childID)
	assert.Contains(t, recIDs, grandchildID)
}

func TestSubsystemConformance_RecursiveSearchSubsystems(t *testing.T) {
	cleanupDB(t)

	parentID := createSystemViaAPI(t, "/systems", baseSystemPayload("Subtree Parent"))
	childID := createSystemViaAPI(t, "/systems/"+parentID+"/subsystems", baseSystemPayload("Subtree Child"))
	grandchildID := createSystemViaAPI(t, "/systems/"+childID+"/subsystems", baseSystemPayload("Subtree Grandchild"))

	nonRecReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems/"+parentID+"/subsystems?recursive=false", nil)
	require.NoError(t, err)
	nonRecReq.Header.Set("Accept", "application/geo+json")
	nonRecResp, err := http.DefaultClient.Do(nonRecReq)
	require.NoError(t, err)
	defer nonRecResp.Body.Close()
	require.Equal(t, http.StatusOK, nonRecResp.StatusCode)
	nonRecBody, err := io.ReadAll(nonRecResp.Body)
	require.NoError(t, err)
	nonRecIDs := getFeatureCollectionIDs(t, nonRecBody)
	assert.Contains(t, nonRecIDs, childID)
	assert.NotContains(t, nonRecIDs, grandchildID)

	recReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems/"+parentID+"/subsystems?recursive=true", nil)
	require.NoError(t, err)
	recReq.Header.Set("Accept", "application/geo+json")
	recResp, err := http.DefaultClient.Do(recReq)
	require.NoError(t, err)
	defer recResp.Body.Close()
	require.Equal(t, http.StatusOK, recResp.StatusCode)
	recBody, err := io.ReadAll(recResp.Body)
	require.NoError(t, err)
	recIDs := getFeatureCollectionIDs(t, recBody)
	assert.Contains(t, recIDs, childID)
	assert.Contains(t, recIDs, grandchildID)
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete
// Requirements: /req/create-replace-delete/system and /req/create-replace-delete/subsystem
// Abstract Tests: /conf/create-replace-delete/system (A.67), /conf/create-replace-delete/subsystem (A.69)
// =============================================================================
func TestSystemCRUD_CreateReplaceDelete(t *testing.T) {
	cleanupDB(t)

	payload := baseSystemPayload("CRUD System")
	systemID := createSystemViaAPI(t, "/systems", payload)

	updated := baseSystemPayload("CRUD System Updated")
	updatedProps := updated["properties"].(map[string]interface{})
	updatedProps["uid"] = payload["properties"].(map[string]interface{})["uid"]

	updateBody, err := json.Marshal(updated)
	require.NoError(t, err)

	putReq, err := http.NewRequest(http.MethodPut, testServer.URL+"/systems/"+systemID, bytes.NewReader(updateBody))
	require.NoError(t, err)
	putReq.Header.Set("Content-Type", "application/geo+json")
	putResp, err := http.DefaultClient.Do(putReq)
	require.NoError(t, err)
	defer putResp.Body.Close()
	require.Equal(t, http.StatusNoContent, putResp.StatusCode)

	getReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems/"+systemID, nil)
	require.NoError(t, err)
	getReq.Header.Set("Accept", "application/geo+json")
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var got map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&got))
	props, ok := got["properties"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "CRUD System Updated", props["name"])

	delReq, err := http.NewRequest(http.MethodDelete, testServer.URL+"/systems/"+systemID, nil)
	require.NoError(t, err)
	delResp, err := http.DefaultClient.Do(delReq)
	require.NoError(t, err)
	defer delResp.Body.Close()
	require.Equal(t, http.StatusNoContent, delResp.StatusCode)

	verifyReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems/"+systemID, nil)
	require.NoError(t, err)
	verifyResp, err := http.DefaultClient.Do(verifyReq)
	require.NoError(t, err)
	defer verifyResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, verifyResp.StatusCode)
}

func TestSubsystemCRUD_CreateAndCanonicalRead(t *testing.T) {
	cleanupDB(t)

	parentID := createSystemViaAPI(t, "/systems", baseSystemPayload("Subsystem CRUD Parent"))
	childID := createSystemViaAPI(t, "/systems/"+parentID+"/subsystems", baseSystemPayload("Subsystem CRUD Child"))

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/systems/%s", testServer.URL, childID), nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var child map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&child))
	assert.Equal(t, childID, child["id"])
}
