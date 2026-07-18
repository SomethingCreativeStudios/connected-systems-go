package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Datastream phenomenonTime/resultTime and control stream issueTime/
// executionTime are read-only extents computed server-side from the stream's
// observations/commands: start = earliest child, end = latest child.
// =============================================================================

func putJSONResource(t *testing.T, path string, payload map[string]interface{}) {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPut, testServer.URL+path, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Contains(t, []int{http.StatusOK, http.StatusNoContent}, resp.StatusCode, "PUT %s", path)
}

func deleteResourceViaAPI(t *testing.T, path string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, testServer.URL+path, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "DELETE %s", path)
}

// assertTimeRange asserts the value is a 2-element [start, end] RFC3339 array.
// Bounds are compared as instants, since the server may render them in a
// non-UTC offset after a database round-trip.
func assertTimeRange(t *testing.T, v interface{}, start, end, msg string) {
	t.Helper()
	arr, ok := v.([]interface{})
	require.True(t, ok, "%s: expected time range array, got %T (%v)", msg, v, v)
	require.Len(t, arr, 2, "%s: expected [start, end]", msg)
	assertTimeInstant(t, arr[0], start, msg+": start")
	assertTimeInstant(t, arr[1], end, msg+": end")
}

func assertTimeInstant(t *testing.T, v interface{}, expected, msg string) {
	t.Helper()
	s, ok := v.(string)
	require.True(t, ok, "%s: expected RFC3339 string, got %T (%v)", msg, v, v)
	got, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err, "%s: invalid RFC3339 time %q", msg, s)
	want, err := time.Parse(time.RFC3339, expected)
	require.NoError(t, err)
	assert.True(t, want.Equal(got), "%s: expected %s, got %s", msg, expected, s)
}

func TestDatastream_TimeRanges_ComputedFromObservations(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("DS Time Range System"))

	// Client-supplied phenomenonTime/resultTime must be ignored (readOnly).
	payload := baseDatastreamPayload()
	payload["phenomenonTime"] = []string{"1999-01-01T00:00:00Z", "1999-12-31T00:00:00Z"}
	payload["resultTime"] = []string{"1999-01-01T00:00:00Z", "1999-12-31T00:00:00Z"}
	dsID := createDatastreamViaAPI(t, "/systems/"+systemID+"/datastreams", payload)

	ds := getJSONResource(t, "/datastreams/"+dsID, "application/json")
	assert.Nil(t, ds["phenomenonTime"], "empty datastream must report null phenomenonTime")
	assert.Nil(t, ds["resultTime"], "empty datastream must report null resultTime")
	assert.Equal(t, []interface{}{"application/json"}, ds["formats"], "formats must be derived from the schema obsFormat")

	createObservationViaAPI(t, dsID, map[string]interface{}{
		"phenomenonTime": "2026-01-01T00:00:00Z",
		"resultTime":     "2026-01-01T01:00:00Z",
		"result":         map[string]interface{}{"temperature": 20.0, "humidity": 40.0},
	})
	// No phenomenonTime — defaults to resultTime. This is the newest observation.
	newestObsID := createObservationViaAPI(t, dsID, map[string]interface{}{
		"resultTime": "2026-03-01T00:00:00Z",
		"result":     map[string]interface{}{"temperature": 21.0, "humidity": 40.0},
	})
	createObservationViaAPI(t, dsID, map[string]interface{}{
		"phenomenonTime": "2025-06-01T00:00:00Z",
		"resultTime":     "2026-02-01T00:00:00Z",
		"result":         map[string]interface{}{"temperature": 22.0, "humidity": 40.0},
	})

	ds = getJSONResource(t, "/datastreams/"+dsID, "application/json")
	assertTimeRange(t, ds["phenomenonTime"], "2025-06-01T00:00:00Z", "2026-03-01T00:00:00Z", "phenomenonTime after 3 observations")
	assertTimeRange(t, ds["resultTime"], "2026-01-01T01:00:00Z", "2026-03-01T00:00:00Z", "resultTime after 3 observations")

	// The list endpoint must report the same computed extents.
	listed := getJSONResource(t, "/systems/"+systemID+"/datastreams", "application/json")
	items, ok := listed["items"].([]interface{})
	require.True(t, ok, "expected items array in datastream list")
	require.Len(t, items, 1)
	listedDS, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assertTimeRange(t, listedDS["phenomenonTime"], "2025-06-01T00:00:00Z", "2026-03-01T00:00:00Z", "phenomenonTime in list")
	assertTimeRange(t, listedDS["resultTime"], "2026-01-01T01:00:00Z", "2026-03-01T00:00:00Z", "resultTime in list")

	// PUT on the datastream (with bogus client-supplied ranges) must not
	// disturb the computed extents.
	updated := baseDatastreamPayload()
	updated["phenomenonTime"] = []string{"1999-01-01T00:00:00Z", "1999-12-31T00:00:00Z"}
	putJSONResource(t, "/datastreams/"+dsID, updated)

	ds = getJSONResource(t, "/datastreams/"+dsID, "application/json")
	assertTimeRange(t, ds["phenomenonTime"], "2025-06-01T00:00:00Z", "2026-03-01T00:00:00Z", "phenomenonTime after datastream PUT")
	assertTimeRange(t, ds["resultTime"], "2026-01-01T01:00:00Z", "2026-03-01T00:00:00Z", "resultTime after datastream PUT")

	// Deleting the newest observation must shrink both extents.
	deleteResourceViaAPI(t, "/observations/"+newestObsID)

	ds = getJSONResource(t, "/datastreams/"+dsID, "application/json")
	assertTimeRange(t, ds["phenomenonTime"], "2025-06-01T00:00:00Z", "2026-01-01T00:00:00Z", "phenomenonTime after observation delete")
	assertTimeRange(t, ds["resultTime"], "2026-01-01T01:00:00Z", "2026-02-01T00:00:00Z", "resultTime after observation delete")
}

func TestControlStream_TimeRanges_ComputedFromCommands(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)

	// Client-supplied issueTime/executionTime must be ignored (readOnly).
	payload := baseControlStreamPayload()
	payload["issueTime"] = []string{"1999-01-01T00:00:00Z", "1999-12-31T00:00:00Z"}
	payload["executionTime"] = []string{"1999-01-01T00:00:00Z", "1999-12-31T00:00:00Z"}
	csID := createControlStreamViaAPI(t, systemID, payload)

	cs := getJSONResource(t, "/controlstreams/"+csID, "application/json")
	assert.Nil(t, cs["issueTime"], "empty control stream must report null issueTime")
	assert.Nil(t, cs["executionTime"], "empty control stream must report null executionTime")
	assert.Equal(t, []interface{}{"application/json"}, cs["formats"], "formats must be derived from the schema commandFormat")

	boundedCmdID := createCommandViaAPI(t, csID, map[string]interface{}{
		"parameters":    map[string]interface{}{"setPoint": 20.0},
		"issueTime":     "2026-01-01T00:00:00Z",
		"executionTime": []string{"2026-01-02T00:00:00Z", "2026-01-03T00:00:00Z"},
	})
	// No executionTime — contributes only to issueTime. This is the newest issue time.
	newestCmdID := createCommandViaAPI(t, csID, map[string]interface{}{
		"parameters": map[string]interface{}{"setPoint": 21.0},
		"issueTime":  "2026-02-01T00:00:00Z",
	})
	// Open-ended executionTime counts as a point at its start.
	createCommandViaAPI(t, csID, map[string]interface{}{
		"parameters":    map[string]interface{}{"setPoint": 22.0},
		"issueTime":     "2025-12-01T00:00:00Z",
		"executionTime": []interface{}{"2026-03-01T00:00:00Z", nil},
	})

	cs = getJSONResource(t, "/controlstreams/"+csID, "application/json")
	assertTimeRange(t, cs["issueTime"], "2025-12-01T00:00:00Z", "2026-02-01T00:00:00Z", "issueTime after 3 commands")
	assertTimeRange(t, cs["executionTime"], "2026-01-02T00:00:00Z", "2026-03-01T00:00:00Z", "executionTime after 3 commands")

	// The list endpoint must report the same computed extents.
	listed := getJSONResource(t, "/systems/"+systemID+"/controlstreams", "application/json")
	items, ok := listed["items"].([]interface{})
	require.True(t, ok, "expected items array in control stream list")
	require.Len(t, items, 1)
	listedCS, ok := items[0].(map[string]interface{})
	require.True(t, ok)
	assertTimeRange(t, listedCS["issueTime"], "2025-12-01T00:00:00Z", "2026-02-01T00:00:00Z", "issueTime in list")
	assertTimeRange(t, listedCS["executionTime"], "2026-01-02T00:00:00Z", "2026-03-01T00:00:00Z", "executionTime in list")

	// PUT a command with a wider executionTime — the extent must be recomputed.
	putJSONResource(t, "/commands/"+boundedCmdID, map[string]interface{}{
		"parameters":    map[string]interface{}{"setPoint": 20.0},
		"issueTime":     "2026-01-01T00:00:00Z",
		"executionTime": []string{"2026-01-02T00:00:00Z", "2026-04-01T00:00:00Z"},
	})

	cs = getJSONResource(t, "/controlstreams/"+csID, "application/json")
	assertTimeRange(t, cs["executionTime"], "2026-01-02T00:00:00Z", "2026-04-01T00:00:00Z", "executionTime after command PUT")

	// PUT on the control stream (with bogus client-supplied ranges) must not
	// disturb the computed extents.
	updated := baseControlStreamPayload()
	updated["issueTime"] = []string{"1999-01-01T00:00:00Z", "1999-12-31T00:00:00Z"}
	putJSONResource(t, "/controlstreams/"+csID, updated)

	cs = getJSONResource(t, "/controlstreams/"+csID, "application/json")
	assertTimeRange(t, cs["issueTime"], "2025-12-01T00:00:00Z", "2026-02-01T00:00:00Z", "issueTime after control stream PUT")
	assertTimeRange(t, cs["executionTime"], "2026-01-02T00:00:00Z", "2026-04-01T00:00:00Z", "executionTime after control stream PUT")

	// Deleting the command with the newest issueTime must shrink the extent.
	deleteResourceViaAPI(t, "/commands/"+newestCmdID)

	cs = getJSONResource(t, "/controlstreams/"+csID, "application/json")
	assertTimeRange(t, cs["issueTime"], "2025-12-01T00:00:00Z", "2026-01-01T00:00:00Z", "issueTime after command delete")
}
