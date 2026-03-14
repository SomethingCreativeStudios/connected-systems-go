package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	SystemEventJSONSchema = "json/systemEvent-bundled.json"
)

func baseSystemEventPayload(label string) map[string]interface{} {
	return map[string]interface{}{
		"definition":  "https://example.org/event/calibration",
		"label":       label,
		"description": "Calibration performed",
		"time":        time.Now().UTC().Format(time.RFC3339),
	}
}

func createSystemEventViaAPI(t *testing.T, systemID string, payload map[string]interface{}) string {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/events", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	location := resp.Header.Get("Location")
	require.NotEmpty(t, location)
	eventID := parseID(location, "/events/")
	require.NotEmpty(t, eventID)
	return eventID
}

func requireSystemEventSchemaOrSkip(t *testing.T, body []byte, schema string) {
	t.Helper()
	err := validateAgainstSchema(t, body, schema)
	if err != nil && strings.Contains(err.Error(), "failed to compile schema") {
		t.Skipf("skipping schema assertion due invalid bundled schema refs: %v", err)
	}
	require.NoError(t, err)
}

func baseSystemWithValidTimePayload(name string, start, end string) map[string]interface{} {
	uid := "urn:uuid:" + uuid.NewString()
	return map[string]interface{}{
		"type": "Feature",
		"properties": map[string]interface{}{
			"uid":         uid,
			"name":        name,
			"featureType": "http://www.w3.org/ns/sosa/Sensor",
			"assetType":   "Equipment",
			"validTime":   []string{start, end},
		},
		"geometry": map[string]interface{}{
			"type":        "Point",
			"coordinates": []float64{-117.1625, 32.715},
		},
	}
}

// =============================================================================
// Conformance Class: /conf/system-event
// Requirement: /req/system-event/resources-endpoint
// =============================================================================

func TestSystemEvent_ResourcesEndpoint(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Event Parent"))
	eventID := createSystemEventViaAPI(t, systemID, baseSystemEventPayload("Calibration Event"))

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/systemEvents", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var collection map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&collection))

	items, ok := collection["items"].([]interface{})
	require.True(t, ok)

	found := false
	for _, item := range items {
		obj, _ := item.(map[string]interface{})
		if id, _ := obj["id"].(string); id == eventID {
			found = true
			break
		}
	}
	assert.True(t, found, "created event must appear in /systemEvents")
}

// =============================================================================
// Conformance Class: /conf/system-event
// Requirement: /req/system-event/canonical-url
// =============================================================================

func TestSystemEvent_CanonicalURL(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Event Canonical Parent"))
	payload := baseSystemEventPayload("Canonical Event")
	eventID := createSystemEventViaAPI(t, systemID, payload)

	resp := doGet(t, "/systems/"+systemID+"/events/"+eventID)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var event map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&event))
	assert.Equal(t, eventID, event["id"])
	assert.Equal(t, payload["label"], event["label"])
}

// =============================================================================
// Conformance Class: /conf/json
// Requirement: /req/json/system-event-schema
// =============================================================================

func TestSystemEventSchema_JSON(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Event Schema Parent"))
	eventID := createSystemEventViaAPI(t, systemID, baseSystemEventPayload("Schema Event"))

	resp := doGet(t, "/systems/"+systemID+"/events/"+eventID)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	requireSystemEventSchemaOrSkip(t, body, SystemEventJSONSchema)
}

// =============================================================================
// Conformance Class: /conf/system-event
// Requirement: /req/system-event/system-sub-collection
// =============================================================================

func TestSystemEvent_SystemSubCollection(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("System Event Parent"))
	eventID := createSystemEventViaAPI(t, systemID, baseSystemEventPayload("System Scoped Event"))

	resp := doGet(t, "/systems/"+systemID+"/events")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var collection map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&collection))
	items, ok := collection["items"].([]interface{})
	require.True(t, ok)

	found := false
	for _, item := range items {
		obj, _ := item.(map[string]interface{})
		if id, _ := obj["id"].(string); id == eventID {
			found = true
		}
	}
	assert.True(t, found, "created event must appear in /systems/{id}/events")
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/system-event
// Requirement: /req/create-replace-delete/system-event
// =============================================================================

func TestSystemEventCRUD(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Event CRUD Parent"))
	eventID := createSystemEventViaAPI(t, systemID, baseSystemEventPayload("Create Event"))

	updated := baseSystemEventPayload("Updated Event")
	updated["definition"] = "https://example.org/event/maintenance"
	body, _ := json.Marshal(updated)

	putReq, _ := http.NewRequest(http.MethodPut, testServer.URL+"/systems/"+systemID+"/events/"+eventID, bytes.NewReader(body))
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := http.DefaultClient.Do(putReq)
	require.NoError(t, err)
	defer putResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, putResp.StatusCode)

	getResp := doGet(t, "/systems/"+systemID+"/events/"+eventID)
	defer getResp.Body.Close()
	var got map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&got))
	assert.Equal(t, "Updated Event", got["label"])

	delReq, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/systems/"+systemID+"/events/"+eventID, nil)
	delResp, err := http.DefaultClient.Do(delReq)
	require.NoError(t, err)
	defer delResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)

	getAfterDelete := doGet(t, "/systems/"+systemID+"/events/"+eventID)
	defer getAfterDelete.Body.Close()
	assert.Equal(t, http.StatusNotFound, getAfterDelete.StatusCode)
}

// =============================================================================
// Conformance Class: /conf/system-event
// Requirement: /req/system-event/filtering
// =============================================================================

func TestSystemEvent_Filtering(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Event Filter Parent"))
	payloadA := baseSystemEventPayload("Calibration Event")
	payloadA["definition"] = "https://example.org/event/calibration"
	createSystemEventViaAPI(t, systemID, payloadA)

	payloadB := baseSystemEventPayload("Maintenance Event")
	payloadB["definition"] = "https://example.org/event/maintenance"
	createSystemEventViaAPI(t, systemID, payloadB)

	resp := doGet(t, "/systemEvents?eventType=https://example.org/event/calibration")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var collection map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&collection))
	items, ok := collection["items"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(items), 1)

	for _, item := range items {
		obj, _ := item.(map[string]interface{})
		assert.Equal(t, "https://example.org/event/calibration", obj["definition"])
	}
}

func TestSystemEvent_FilteringNoMatches(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Event No-Match Parent"))
	createSystemEventViaAPI(t, systemID, baseSystemEventPayload("Calibration Event"))

	resp := doGet(t, "/systemEvents?eventType=https://example.org/event/nonexistent")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var collection map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&collection))
	items, ok := collection["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 0)
}

func TestSystemEvent_CreateArrayPayload(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Event Array Parent"))

	payload := []map[string]interface{}{
		baseSystemEventPayload("Batch Event 1"),
		baseSystemEventPayload("Batch Event 2"),
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/events", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotEmpty(t, resp.Header.Get("Location"))

	listResp := doGet(t, "/systems/"+systemID+"/events")
	defer listResp.Body.Close()
	require.Equal(t, http.StatusOK, listResp.StatusCode)

	var collection map[string]interface{}
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&collection))
	items, ok := collection["items"].([]interface{})
	require.True(t, ok)
	require.GreaterOrEqual(t, len(items), 2)

	labels := map[string]bool{}
	for _, item := range items {
		obj, _ := item.(map[string]interface{})
		if label, ok := obj["label"].(string); ok {
			labels[label] = true
		}
	}
	assert.True(t, labels["Batch Event 1"])
	assert.True(t, labels["Batch Event 2"])
}

func TestSystemEvent_PaginationLinks(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Event Pagination Parent"))
	createSystemEventViaAPI(t, systemID, baseSystemEventPayload("Page Event 1"))
	createSystemEventViaAPI(t, systemID, baseSystemEventPayload("Page Event 2"))
	createSystemEventViaAPI(t, systemID, baseSystemEventPayload("Page Event 3"))

	resp := doGet(t, "/systemEvents?limit=1")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var collection map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&collection))
	links, ok := collection["links"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, links)

	byRel := map[string]string{}
	for _, l := range links {
		obj, _ := l.(map[string]interface{})
		rel, _ := obj["rel"].(string)
		href, _ := obj["href"].(string)
		if rel != "" && href != "" {
			byRel[rel] = href
		}
	}

	assert.NotEmpty(t, byRel["self"])
	assert.NotEmpty(t, byRel["next"])
	assert.True(t, strings.Contains(byRel["next"], "offset=1"), "next link must advance offset")
}

// =============================================================================
// Conformance Class: /conf/system-history
// Requirement: /req/system-history/resources-endpoint
// =============================================================================

func TestSystemHistory_ListAndGet(t *testing.T) {
	cleanupDB(t)

	start := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	end := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	systemPayload := baseSystemWithValidTimePayload("History Parent", start, end)
	systemID := createSystemViaAPI(t, "/systems", systemPayload)

	updatedPayload := baseSystemWithValidTimePayload("History Parent Updated", start, end)
	updBody, _ := json.Marshal(updatedPayload)
	putReq, _ := http.NewRequest(http.MethodPut, testServer.URL+"/systems/"+systemID, bytes.NewReader(updBody))
	putReq.Header.Set("Content-Type", "application/geo+json")
	putResp, err := http.DefaultClient.Do(putReq)
	require.NoError(t, err)
	defer putResp.Body.Close()
	require.Equal(t, http.StatusNoContent, putResp.StatusCode)

	historyResp := doGet(t, "/systems/"+systemID+"/history")
	defer historyResp.Body.Close()
	require.Equal(t, http.StatusOK, historyResp.StatusCode)

	body, err := io.ReadAll(historyResp.Body)
	require.NoError(t, err)
	revIDs := getFeatureCollectionIDs(t, body)
	require.GreaterOrEqual(t, len(revIDs), 2)

	revResp := doGet(t, "/systems/"+systemID+"/history/"+revIDs[0])
	defer revResp.Body.Close()
	require.Equal(t, http.StatusOK, revResp.StatusCode)
}

// =============================================================================
// Conformance Class: /conf/system-history
// Requirement: /req/system-history/update-delete
// =============================================================================

func TestSystemHistory_UpdateDeleteRevision(t *testing.T) {
	cleanupDB(t)

	start := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	end := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	systemID := createSystemViaAPI(t, "/systems", baseSystemWithValidTimePayload("History Update Parent", start, end))

	historyResp := doGet(t, "/systems/"+systemID+"/history")
	defer historyResp.Body.Close()
	body, err := io.ReadAll(historyResp.Body)
	require.NoError(t, err)
	revIDs := getFeatureCollectionIDs(t, body)
	require.NotEmpty(t, revIDs)
	revID := revIDs[0]

	goodUpdate := baseSystemWithValidTimePayload("Revision Updated", start, end)
	goodBody, _ := json.Marshal(goodUpdate)
	goodPutReq, _ := http.NewRequest(http.MethodPut, testServer.URL+"/systems/"+systemID+"/history/"+revID, bytes.NewReader(goodBody))
	goodPutReq.Header.Set("Content-Type", "application/geo+json")
	goodPutResp, err := http.DefaultClient.Do(goodPutReq)
	require.NoError(t, err)
	defer goodPutResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, goodPutResp.StatusCode)

	badUpdate := baseSystemWithValidTimePayload("Revision Invalid Update", time.Now().UTC().Add(-1*time.Hour).Format(time.RFC3339), end)
	badBody, _ := json.Marshal(badUpdate)
	badPutReq, _ := http.NewRequest(http.MethodPut, testServer.URL+"/systems/"+systemID+"/history/"+revID, bytes.NewReader(badBody))
	badPutReq.Header.Set("Content-Type", "application/geo+json")
	badPutResp, err := http.DefaultClient.Do(badPutReq)
	require.NoError(t, err)
	defer badPutResp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, badPutResp.StatusCode)

	delReq, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/systems/"+systemID+"/history/"+revID, nil)
	delResp, err := http.DefaultClient.Do(delReq)
	require.NoError(t, err)
	defer delResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)

	getAfterDelete := doGet(t, "/systems/"+systemID+"/history/"+revID)
	defer getAfterDelete.Body.Close()
	assert.Equal(t, http.StatusNotFound, getAfterDelete.StatusCode)
}

// =============================================================================
// Conformance Class: /conf/system-history
// Requirement: /req/system-history/filter-validtime
// =============================================================================

func TestSystemHistory_FilterByValidTime(t *testing.T) {
	cleanupDB(t)

	start := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	end := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	systemID := createSystemViaAPI(t, "/systems", baseSystemWithValidTimePayload("History Filter Parent", start, end))

	resp := doGet(t, "/systems/"+systemID+"/history?validTime="+start+"/"+end)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	revIDs := getFeatureCollectionIDs(t, body)
	assert.NotEmpty(t, revIDs)
}

func TestSystemHistory_FilterByKeywordNoMatches(t *testing.T) {
	cleanupDB(t)

	start := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	end := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	systemID := createSystemViaAPI(t, "/systems", baseSystemWithValidTimePayload("History Keyword Parent", start, end))

	resp := doGet(t, "/systems/"+systemID+"/history?keyword=definitely-no-snapshot-match")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	revIDs := getFeatureCollectionIDs(t, body)
	assert.Len(t, revIDs, 0)
}

func TestSystemHistory_PaginationLinks(t *testing.T) {
	cleanupDB(t)

	start := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	end := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	systemID := createSystemViaAPI(t, "/systems", baseSystemWithValidTimePayload("History Pagination Parent", start, end))

	for i := 0; i < 3; i++ {
		payload := baseSystemWithValidTimePayload("History Pagination Update", start, end)
		updBody, _ := json.Marshal(payload)
		putReq, _ := http.NewRequest(http.MethodPut, testServer.URL+"/systems/"+systemID, bytes.NewReader(updBody))
		putReq.Header.Set("Content-Type", "application/geo+json")
		putResp, err := http.DefaultClient.Do(putReq)
		require.NoError(t, err)
		putResp.Body.Close()
		require.Equal(t, http.StatusNoContent, putResp.StatusCode)
	}

	resp := doGet(t, "/systems/"+systemID+"/history?limit=1")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var collection map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&collection))
	links, ok := collection["links"].([]interface{})
	require.True(t, ok)
	require.NotEmpty(t, links)

	byRel := map[string]string{}
	for _, l := range links {
		obj, _ := l.(map[string]interface{})
		rel, _ := obj["rel"].(string)
		href, _ := obj["href"].(string)
		if rel != "" && href != "" {
			byRel[rel] = href
		}
	}

	assert.NotEmpty(t, byRel["self"])
	assert.NotEmpty(t, byRel["next"])
	assert.True(t, strings.Contains(byRel["next"], "offset=1"), "next link must advance offset")
}
