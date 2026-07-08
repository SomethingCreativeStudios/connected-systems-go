package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	generators "github.com/yourusername/connected-systems-go/internal/model/generators"
)

const (
	ObservationJSONSchema = "json/observation-bundled.json"
)

func seedDatastreamForObservationTests(t *testing.T) *domains.Datastream {
	t.Helper()

	datastream := generators.FakeDatastreamJSONRecord()
	systemID := "unknown"
	datastream.SystemID = &systemID
	require.NoError(t, testRepos.Datastream.Create(&datastream), "failed to seed datastream")

	return &datastream
}

func createObservationViaAPI(t *testing.T, datastreamID string, payload map[string]interface{}) string {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/datastreams/"+datastreamID+"/observations", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	location := resp.Header.Get("Location")
	require.NotEmpty(t, location, "expected Location header")

	obsID := parseID(location, "/observations/")
	require.NotEmpty(t, obsID, "unable to parse observation ID from Location header")
	return obsID
}

// =============================================================================
// Conformance Class: /conf/observation
// Requirement: /req/observation/resources-endpoint
// Abstract Test: /conf/observation/resources-endpoint
// Link: https://docs.ogc.org/is/23-001/23-001.html#_8f377cd0-0756-e87c-cceb-14668d78e846
// =============================================================================
func TestObservationConformance_ResourcesEndpoint(t *testing.T) {
	cleanupDB(t)

	datastream := seedDatastreamForObservationTests(t)

	payload := map[string]interface{}{
		"resultTime": "2026-03-13T10:00:00Z",
		"result": map[string]interface{}{
			"temperature": 21.4,
			"humidity":    57.9,
		},
	}
	createdObsID := createObservationViaAPI(t, datastream.ID, payload)

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/observations", nil)
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
	require.GreaterOrEqual(t, len(items), 1, "expected at least one observation")
	links, ok := collection["links"].([]interface{})
	require.True(t, ok, "response must contain 'links' array")
	require.GreaterOrEqual(t, len(links), 1)

	found := false
	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if id, _ := obj["id"].(string); id == createdObsID {
			found = true
			break
		}
	}
	assert.True(t, found, "created observation must be discoverable via /observations")
}

// =============================================================================
// Conformance Class: /conf/observation
// Requirement: /req/observation/canonical-url
// Abstract Test: /conf/observation/canonical-url
// Link: https://docs.ogc.org/is/23-001/23-001.html#_8f377cd0-0756-e87c-cceb-14668d78e846
// =============================================================================
func TestObservationConformance_CanonicalURL(t *testing.T) {
	cleanupDB(t)

	datastream := seedDatastreamForObservationTests(t)

	payload := map[string]interface{}{
		"resultTime": "2026-03-13T11:00:00Z",
		"result": map[string]interface{}{
			"temperature": 19.1,
			"humidity":    44.2,
		},
	}
	obsID := createObservationViaAPI(t, datastream.ID, payload)

	url := fmt.Sprintf("%s/observations/%s", testServer.URL, obsID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var obs map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&obs)
	require.NoError(t, err)
	assert.Equal(t, obsID, obs["id"])
	assert.Equal(t, datastream.ID, obs["datastream@id"])
}

// =============================================================================
// Conformance Class: /conf/json/observation-schema
// Requirement: /req/json/observation-schema
// Abstract Test: /conf/json/observation-schema
// Link: https://docs.ogc.org/is/23-001/23-001.html#_8f377cd0-0756-e87c-cceb-14668d78e846
// =============================================================================
func TestObservationSchema_JSON(t *testing.T) {
	cleanupDB(t)

	datastream := seedDatastreamForObservationTests(t)

	payload := map[string]interface{}{
		"resultTime": "2026-03-13T12:00:00Z",
		"result": map[string]interface{}{
			"temperature": 22.6,
			"humidity":    41.3,
		},
	}
	obsID := createObservationViaAPI(t, datastream.ID, payload)

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/observations/"+obsID, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	err = validateAgainstSchema(t, body, ObservationJSONSchema)
	require.NoError(t, err, "response did not validate against observation JSON schema")
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/observation
// Requirement: /req/create-replace-delete/observation
// Abstract Test: create under datastream must satisfy parent datastream schema
// Link: https://docs.ogc.org/is/23-001/23-001.html#_8f377cd0-0756-e87c-cceb-14668d78e846
// =============================================================================
func TestObservation_Create_ValidatesParentDatastreamSchema(t *testing.T) {
	cleanupDB(t)

	datastream := seedDatastreamForObservationTests(t)

	validPayload := map[string]interface{}{
		"resultTime": "2026-03-13T13:00:00Z",
		"result": map[string]interface{}{
			"temperature": 20.2,
			"humidity":    58.8,
		},
	}
	createObservationViaAPI(t, datastream.ID, validPayload)

	invalidPayload := map[string]interface{}{
		"resultTime": "2026-03-13T13:05:00Z",
		"result": map[string]interface{}{
			"temperature": "not-a-number",
		},
	}

	body, err := json.Marshal(invalidPayload)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/datastreams/"+datastream.ID+"/observations", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// helper: POST to observations endpoint, return status + body
func postObservation(t *testing.T, datastreamID string, payload map[string]interface{}) (int, map[string]interface{}) {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/datastreams/"+datastreamID+"/observations", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result
}

// =============================================================================
// Issue #19 — numeric phenomenonTime must be rejected (not silently dropped)
// =============================================================================
func TestObservation_Create_NumericPhenomenonTime_Returns400(t *testing.T) {
	cleanupDB(t)
	ds := seedDatastreamForObservationTests(t)

	status, body := postObservation(t, ds.ID, map[string]interface{}{
		"phenomenonTime": 1773100000,
		"resultTime":     "2026-01-01T00:00:00Z",
		"result":         map[string]interface{}{"temperature": 21.0, "humidity": 55.0},
	})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body["error"], "phenomenonTime")
}

// =============================================================================
// Issue #20 — numeric resultTime must give an accurate type error, not "required"
// =============================================================================
func TestObservation_Create_NumericResultTime_Returns400WithTypeError(t *testing.T) {
	cleanupDB(t)
	ds := seedDatastreamForObservationTests(t)

	status, body := postObservation(t, ds.ID, map[string]interface{}{
		"resultTime": 1773100000,
		"result":     map[string]interface{}{"temperature": 21.0, "humidity": 55.0},
	})
	assert.Equal(t, http.StatusBadRequest, status)
	errMsg, _ := body["error"].(string)
	assert.Contains(t, errMsg, "resultTime")
	assert.NotContains(t, errMsg, "required", "error message should not say 'required' when field is present but wrong type")
}

// =============================================================================
// Regression — resultTime truly absent must still say "required"
// =============================================================================
func TestObservation_Create_MissingResultTime_Returns400Required(t *testing.T) {
	cleanupDB(t)
	ds := seedDatastreamForObservationTests(t)

	status, body := postObservation(t, ds.ID, map[string]interface{}{
		"result": map[string]interface{}{"temperature": 21.0, "humidity": 55.0},
	})
	assert.Equal(t, http.StatusBadRequest, status)
	errMsg, _ := body["error"].(string)
	assert.Contains(t, errMsg, "required")
}

// =============================================================================
// Regression — valid ISO 8601 phenomenonTime must still be accepted
// =============================================================================
func TestObservation_Create_ValidPhenomenonTime_Returns201(t *testing.T) {
	cleanupDB(t)
	ds := seedDatastreamForObservationTests(t)

	status, _ := postObservation(t, ds.ID, map[string]interface{}{
		"phenomenonTime": "2026-01-01T00:00:00Z",
		"resultTime":     "2026-01-01T00:00:00Z",
		"result":         map[string]interface{}{"temperature": 21.0, "humidity": 55.0},
	})
	assert.Equal(t, http.StatusCreated, status)
}

// =============================================================================
// Issue #11 — garbage resultTime query param must return 400
// =============================================================================
func TestObservation_List_GarbageResultTimeParam_Returns400(t *testing.T) {
	cleanupDB(t)
	seedDatastreamForObservationTests(t)

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/observations?resultTime=frobnicate", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// =============================================================================
// ?resultTime=latest must return only the single most-recent observation
// =============================================================================
func TestObservation_List_LatestResultTime_ReturnsMostRecent(t *testing.T) {
	cleanupDB(t)
	ds := seedDatastreamForObservationTests(t)

	// Older observation
	createObservationViaAPI(t, ds.ID, map[string]interface{}{
		"resultTime": "2025-01-01T00:00:00Z",
		"result":     map[string]interface{}{"temperature": 10.0, "humidity": 40.0},
	})
	// Newer observation
	newerID := createObservationViaAPI(t, ds.ID, map[string]interface{}{
		"resultTime": "2026-01-01T00:00:00Z",
		"result":     map[string]interface{}{"temperature": 20.0, "humidity": 60.0},
	})

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/observations?resultTime=latest", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	items, ok := result["items"].([]interface{})
	require.True(t, ok)
	links, ok := result["links"].([]interface{})
	require.True(t, ok, "response must contain 'links' array")
	require.GreaterOrEqual(t, len(links), 1)
	require.Equal(t, 1, len(items), "expected exactly one observation for ?resultTime=latest")

	obs, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, newerID, obs["id"], "expected the most recent observation to be returned")
}

// =============================================================================
// Issue #11 — valid resultTime range must filter correctly (positive test)
// =============================================================================
func TestObservation_List_ValidResultTimeRange_Filters(t *testing.T) {
	cleanupDB(t)
	ds := seedDatastreamForObservationTests(t)

	// Create one observation in 2025 and one in 2027
	createObservationViaAPI(t, ds.ID, map[string]interface{}{
		"resultTime": "2025-06-01T00:00:00Z",
		"result":     map[string]interface{}{"temperature": 10.0, "humidity": 40.0},
	})
	createObservationViaAPI(t, ds.ID, map[string]interface{}{
		"resultTime": "2027-06-01T00:00:00Z",
		"result":     map[string]interface{}{"temperature": 20.0, "humidity": 60.0},
	})

	// Query only the 2025 window
	url := testServer.URL + "/observations?resultTime=2025-01-01T00:00:00Z/2025-12-31T23:59:59Z"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	items, ok := result["items"].([]interface{})
	require.True(t, ok)
	links, ok := result["links"].([]interface{})
	require.True(t, ok, "response must contain 'links' array")
	require.GreaterOrEqual(t, len(links), 1)
	assert.Equal(t, 1, len(items), "expected only the 2025 observation to be returned")
}
