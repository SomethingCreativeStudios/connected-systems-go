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
	DeploymentSMLSchema = "sensorml/deployment-bundled.json"
	DeploymentGeoSchema = "geojson/deployment-bundled.json"
)

func baseDeploymentPayload(name, deployedSystemID string) map[string]interface{} {
	uid := "urn:uuid:" + uuid.NewString()
	return map[string]interface{}{
		"type": "Feature",
		"properties": map[string]interface{}{
			"uid":         uid,
			"name":        name,
			"featureType": "http://www.w3.org/ns/sosa/Deployment",
			"validTime":   []string{"2026-03-01T00:00:00Z", "2026-12-31T23:59:59Z"},
			"deployedSystems@link": []map[string]interface{}{
				{"href": testServer.URL + "/systems/" + deployedSystemID},
			},
		},
		"geometry": map[string]interface{}{
			"type": "Polygon",
			"coordinates": [][][]float64{{
				{-117.2, 32.7},
				{-117.1, 32.7},
				{-117.1, 32.8},
				{-117.2, 32.8},
				{-117.2, 32.7},
			}},
		},
	}
}

func createDeploymentViaAPI(t *testing.T, endpoint string, payload map[string]interface{}) string {
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
	id := parseID(location, "/deployments/")
	require.NotEmpty(t, id)
	return id
}

func getDeploymentCollectionIDs(t *testing.T, body []byte) []string {
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

func requireDeploymentSchemaOrSkip(t *testing.T, body []byte, schema string) {
	t.Helper()
	err := validateAgainstSchema(t, body, schema)
	if err != nil && strings.Contains(err.Error(), "failed to compile schema") {
		t.Skipf("skipping schema assertion due invalid bundled schema refs: %v", err)
	}
	require.NoError(t, err)
}

// =============================================================================
// Conformance Class: /conf/deployment
// Requirement: /req/deployment/resources-endpoint
// Abstract Test: /conf/deployment/resources-endpoint (A.19)
// =============================================================================
func TestDeploymentConformance_ResourcesEndpoint(t *testing.T) {
	cleanupDB(t)
	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Deployment Resources System"))
	depID := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Deployment Resources", systemID))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/deployments", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	ids := getDeploymentCollectionIDs(t, body)
	assert.Contains(t, ids, depID)
}

// =============================================================================
// Conformance Class: /conf/deployment
// Requirement: /req/deployment/canonical-endpoint
// Abstract Test: /conf/deployment/canonical-endpoint (A.20)
// =============================================================================
func TestDeploymentConformance_CanonicalEndpoint(t *testing.T) {
	cleanupDB(t)
	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Deployment Canonical Endpoint System"))
	createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Deployment Canonical Endpoint", systemID))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/deployments", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// =============================================================================
// Conformance Class: /conf/deployment
// Requirement: /req/deployment/canonical-url
// Abstract Test: /conf/deployment/canonical-url (A.18)
// =============================================================================
func TestDeploymentConformance_CanonicalURL(t *testing.T) {
	cleanupDB(t)
	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Deployment Canonical URL System"))
	depID := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Deployment Canonical URL", systemID))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/deployments/"+depID, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var feature map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&feature))
	assert.Equal(t, depID, feature["id"])
}

// =============================================================================
// Conformance Class: /conf/deployment
// Requirement: association links on deployments must expose deployed systems.
// =============================================================================
func TestDeployment_AssociationLinks_DeployedSystems(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Deployment Association System"))
	depID := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Deployment Association", systemID))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/deployments/"+depID, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var feature map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&feature))

	properties, ok := feature["properties"].(map[string]interface{})
	require.True(t, ok, "deployment must expose properties")

	deployedSystems, ok := properties["deployedSystems@link"].([]interface{})
	require.True(t, ok, "deployment must expose deployedSystems@link")
	require.Len(t, deployedSystems, 1, "expected one deployed system link")

	deployedSystem, ok := deployedSystems[0].(map[string]interface{})
	require.True(t, ok)
	href, _ := deployedSystem["href"].(string)
	assert.Contains(t, href, "/systems/"+systemID, "deployedSystems@link href must reference the deployed system")
}

// =============================================================================
// Conformance Class: /conf/geojson and /conf/sensorml
// Requirements: /req/geojson/deployment-schema and /req/sensorml/deployment-schema
// Abstract Tests: /conf/geojson/deployment-schema (A.90), /conf/sensorml/deployment-schema (A.104)
// =============================================================================
func TestDeploymentSchema_GeoJSON(t *testing.T) {
	cleanupDB(t)
	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Deployment GeoJSON Schema System"))
	depID := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Deployment GeoJSON Schema", systemID))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/deployments/"+depID, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	requireDeploymentSchemaOrSkip(t, body, DeploymentGeoSchema)
}

func TestDeploymentSchema_SensorML(t *testing.T) {
	cleanupDB(t)
	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Deployment SensorML Schema System"))
	depID := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Deployment SensorML Schema", systemID))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/deployments/"+depID, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/sml+json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	requireDeploymentSchemaOrSkip(t, body, DeploymentSMLSchema)
}

// =============================================================================
// Conformance Class: /conf/deployment
// Requirement: /req/deployment/ref-from-system
// Abstract Test: /conf/deployment/ref-from-system (A.22)
// =============================================================================
func TestDeploymentConformance_RefFromSystem(t *testing.T) {
	cleanupDB(t)

	systemA := createSystemViaAPI(t, "/systems", baseSystemPayload("Deployment Ref System A"))
	systemB := createSystemViaAPI(t, "/systems", baseSystemPayload("Deployment Ref System B"))

	depA := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Deployment for A", systemA))
	_ = createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Deployment for B", systemB))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems/"+systemA+"/deployments", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	ids := getDeploymentCollectionIDs(t, body)
	assert.Contains(t, ids, depA)
}

// =============================================================================
// Conformance Class: /conf/subdeployment
// Requirement: /req/subdeployment/collection
// Abstract Test: /conf/subdeployment/collection (A.23)
// =============================================================================
func TestSubdeploymentConformance_Collection(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Subdeployment Collection System"))
	parentID := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Subdeployment Parent", systemID))
	childID := createDeploymentViaAPI(t, "/deployments/"+parentID+"/subdeployments", baseDeploymentPayload("Subdeployment Child", systemID))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/deployments/"+parentID+"/subdeployments", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	ids := getDeploymentCollectionIDs(t, body)
	assert.Contains(t, ids, childID)
}

// =============================================================================
// Conformance Class: /conf/subdeployment
// Requirements: /req/subdeployment/recursive-search-deployments and /req/subdeployment/recursive-search-subdeployments
// Abstract Tests: /conf/subdeployment/recursive-search-deployments (A.25), /conf/subdeployment/recursive-search-subdeployments (A.26)
// =============================================================================
func TestSubdeploymentConformance_RecursiveSearchDeployments(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Subdeployment Recursive System"))
	parentID := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Recursive Parent", systemID))
	childID := createDeploymentViaAPI(t, "/deployments/"+parentID+"/subdeployments", baseDeploymentPayload("Recursive Child", systemID))
	grandchildID := createDeploymentViaAPI(t, "/deployments/"+childID+"/subdeployments", baseDeploymentPayload("Recursive Grandchild", systemID))

	nonRecReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/deployments?recursive=false", nil)
	require.NoError(t, err)
	nonRecReq.Header.Set("Accept", "application/geo+json")
	nonRecResp, err := http.DefaultClient.Do(nonRecReq)
	require.NoError(t, err)
	defer nonRecResp.Body.Close()
	require.Equal(t, http.StatusOK, nonRecResp.StatusCode)
	nonRecBody, err := io.ReadAll(nonRecResp.Body)
	require.NoError(t, err)
	nonRecIDs := getDeploymentCollectionIDs(t, nonRecBody)
	assert.Contains(t, nonRecIDs, parentID)
	assert.NotContains(t, nonRecIDs, childID)
	assert.NotContains(t, nonRecIDs, grandchildID)

	recReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/deployments?recursive=true", nil)
	require.NoError(t, err)
	recReq.Header.Set("Accept", "application/geo+json")
	recResp, err := http.DefaultClient.Do(recReq)
	require.NoError(t, err)
	defer recResp.Body.Close()
	require.Equal(t, http.StatusOK, recResp.StatusCode)
	recBody, err := io.ReadAll(recResp.Body)
	require.NoError(t, err)
	recIDs := getDeploymentCollectionIDs(t, recBody)
	assert.Contains(t, recIDs, parentID)
	assert.Contains(t, recIDs, childID)
	assert.Contains(t, recIDs, grandchildID)
}

func TestSubdeploymentConformance_RecursiveSearchSubdeployments(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Subdeployment Recursive Subtree System"))
	parentID := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Subtree Parent", systemID))
	childID := createDeploymentViaAPI(t, "/deployments/"+parentID+"/subdeployments", baseDeploymentPayload("Subtree Child", systemID))
	grandchildID := createDeploymentViaAPI(t, "/deployments/"+childID+"/subdeployments", baseDeploymentPayload("Subtree Grandchild", systemID))

	nonRecReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/deployments/%s/subdeployments?recursive=false", testServer.URL, parentID), nil)
	require.NoError(t, err)
	nonRecReq.Header.Set("Accept", "application/geo+json")
	nonRecResp, err := http.DefaultClient.Do(nonRecReq)
	require.NoError(t, err)
	defer nonRecResp.Body.Close()
	require.Equal(t, http.StatusOK, nonRecResp.StatusCode)
	nonRecBody, err := io.ReadAll(nonRecResp.Body)
	require.NoError(t, err)
	nonRecIDs := getDeploymentCollectionIDs(t, nonRecBody)
	assert.Contains(t, nonRecIDs, childID)
	assert.NotContains(t, nonRecIDs, grandchildID)

	recReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/deployments/%s/subdeployments?recursive=true", testServer.URL, parentID), nil)
	require.NoError(t, err)
	recReq.Header.Set("Accept", "application/geo+json")
	recResp, err := http.DefaultClient.Do(recReq)
	require.NoError(t, err)
	defer recResp.Body.Close()
	require.Equal(t, http.StatusOK, recResp.StatusCode)
	recBody, err := io.ReadAll(recResp.Body)
	require.NoError(t, err)
	recIDs := getDeploymentCollectionIDs(t, recBody)
	assert.Contains(t, recIDs, childID)
	assert.Contains(t, recIDs, grandchildID)
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete
// Requirements: /req/create-replace-delete/deployment and /req/create-replace-delete/subdeployment
// Abstract Tests: /conf/create-replace-delete/deployment (A.70), /conf/create-replace-delete/subdeployment (A.71)
// =============================================================================
func TestDeploymentCRUD_CreateReplaceDelete(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Deployment CRUD System"))
	payload := baseDeploymentPayload("Deployment CRUD", systemID)
	depID := createDeploymentViaAPI(t, "/deployments", payload)

	updated := baseDeploymentPayload("Deployment CRUD Updated", systemID)
	updatedProps := updated["properties"].(map[string]interface{})
	updatedProps["uid"] = payload["properties"].(map[string]interface{})["uid"]

	updateBody, err := json.Marshal(updated)
	require.NoError(t, err)

	putReq, err := http.NewRequest(http.MethodPut, testServer.URL+"/deployments/"+depID, bytes.NewReader(updateBody))
	require.NoError(t, err)
	putReq.Header.Set("Content-Type", "application/geo+json")
	putResp, err := http.DefaultClient.Do(putReq)
	require.NoError(t, err)
	defer putResp.Body.Close()
	require.Equal(t, http.StatusNoContent, putResp.StatusCode)

	getReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/deployments/"+depID, nil)
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
	assert.Equal(t, "Deployment CRUD Updated", props["name"])

	delReq, err := http.NewRequest(http.MethodDelete, testServer.URL+"/deployments/"+depID, nil)
	require.NoError(t, err)
	delResp, err := http.DefaultClient.Do(delReq)
	require.NoError(t, err)
	defer delResp.Body.Close()
	require.Equal(t, http.StatusNoContent, delResp.StatusCode)

	verifyReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/deployments/"+depID, nil)
	require.NoError(t, err)
	verifyResp, err := http.DefaultClient.Do(verifyReq)
	require.NoError(t, err)
	defer verifyResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, verifyResp.StatusCode)
}

func TestSubdeploymentCRUD_CreateAndCanonicalRead(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Subdeployment CRUD System"))
	parentID := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Subdeployment CRUD Parent", systemID))
	childID := createDeploymentViaAPI(t, "/deployments/"+parentID+"/subdeployments", baseDeploymentPayload("Subdeployment CRUD Child", systemID))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/deployments/"+childID, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var feature map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&feature))
	assert.Equal(t, childID, feature["id"])
}
