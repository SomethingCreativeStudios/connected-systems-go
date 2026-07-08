package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Inline @link.href absolutization tests (Issue #10)
// Verifies that all 11 inline-link sites across Datastream, ControlStream,
// Command, and Observation emit absolute URIs.
// =============================================================================

// ---------------------------------------------------------------------------
// Datastream: 4 sibling inline links
// ---------------------------------------------------------------------------

func TestDatastream_AllInlineLinksAreAbsolute(t *testing.T) {
	cleanupDB(t)

	systemID := uuid.NewString()
	payload := baseDatastreamPayload()
	payload["procedure@link"] = map[string]interface{}{
		"href": "/procedures/proc-1",
	}
	payload["deployment@link"] = map[string]interface{}{
		"href": "/deployments/dep-1",
	}
	payload["featureOfInterest@link"] = map[string]interface{}{
		"href": "/features/foi-1",
	}
	payload["samplingFeature@link"] = map[string]interface{}{
		"href": "/samplingFeatures/sf-1",
	}

	dsID := createDatastreamViaAPI(t, "/systems/"+systemID+"/datastreams", payload)

	req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/datastreams/"+dsID, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var ds map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&ds))

	assertLinkHrefIsAbsolute(t, ds, "procedure@link")
	assertLinkHrefIsAbsolute(t, ds, "deployment@link")
	assertLinkHrefIsAbsolute(t, ds, "featureOfInterest@link")
	assertLinkHrefIsAbsolute(t, ds, "samplingFeature@link")
}

// ---------------------------------------------------------------------------
// ControlStream: 4 sibling inline links
// ---------------------------------------------------------------------------

func TestControlStream_AllInlineLinksAreAbsolute(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	payload := baseControlStreamPayload()
	payload["procedure@link"] = map[string]interface{}{
		"href": "/procedures/proc-1",
	}
	payload["deployment@link"] = map[string]interface{}{
		"href": "/deployments/dep-1",
	}
	payload["featureOfInterest@link"] = map[string]interface{}{
		"href": "/features/foi-1",
	}
	payload["samplingFeature@link"] = map[string]interface{}{
		"href": "/samplingFeatures/sf-1",
	}

	csID := createControlStreamViaAPI(t, systemID, payload)

	req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/controlstreams/"+csID, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var cs map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&cs))

	assertLinkHrefIsAbsolute(t, cs, "procedure@link")
	assertLinkHrefIsAbsolute(t, cs, "deployment@link")
	assertLinkHrefIsAbsolute(t, cs, "featureOfInterest@link")
	assertLinkHrefIsAbsolute(t, cs, "samplingFeature@link")
}

// ---------------------------------------------------------------------------
// Command: procedure@link
// ---------------------------------------------------------------------------

func TestCommand_ProcedureLinkIsAbsolute(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	cmdPayload := baseCommandPayload()
	cmdPayload["procedure@link"] = map[string]interface{}{
		"href": "/procedures/proc-1",
	}

	body, err := json.Marshal(cmdPayload)
	require.NoError(t, err)

	req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/controlstreams/"+csID+"/commands", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	cmdID := parseID(resp.Header.Get("Location"), "/commands/")
	require.NotEmpty(t, cmdID)

	getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/commands/"+cmdID, nil)
	getReq.Header.Set("Accept", "application/json")
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var cmd map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&cmd))

	assertLinkHrefIsAbsolute(t, cmd, "procedure@link")
}

// ---------------------------------------------------------------------------
// Observation: procedure@link + result@link
// ---------------------------------------------------------------------------

func TestObservation_ProcedureAndResultLinksAreAbsolute(t *testing.T) {
	cleanupDB(t)

	datastream := seedDatastreamForObservationTests(t)

	payload := map[string]interface{}{
		"resultTime": "2026-03-13T10:00:00Z",
		"result": map[string]interface{}{
			"temperature": 21.4,
		},
		"procedure@link": map[string]interface{}{
			"href": "/procedures/proc-1",
		},
		"result@link": map[string]interface{}{
			"href": "/results/result-1",
		},
	}

	obsID := createObservationViaAPI(t, datastream.ID, payload)

	getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/observations/"+obsID, nil)
	getReq.Header.Set("Accept", "application/json")
	getResp, err := http.DefaultClient.Do(getReq)
	require.NoError(t, err)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var obs map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&obs))

	assertLinkHrefIsAbsolute(t, obs, "procedure@link")
	assertLinkHrefIsAbsolute(t, obs, "result@link")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// assertLinkHrefIsAbsolute checks that a JSON object has a key whose value is
// a Link object with an absolute href.
func assertLinkHrefIsAbsolute(t *testing.T, obj map[string]interface{}, key string) {
	t.Helper()

	raw, ok := obj[key]
	if !ok {
		t.Errorf("expected key %q to exist in response", key)
		return
	}

	link, ok := raw.(map[string]interface{})
	if !ok {
		t.Errorf("expected %q to be a Link object, got %T", key, raw)
		return
	}

	href, ok := link["href"].(string)
	if !ok {
		t.Errorf("expected %q.href to be a string, got %T", key, link["href"])
		return
	}

	if href == "" {
		t.Errorf("expected %q.href to be non-empty", key)
		return
	}

	if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
		t.Errorf("expected %q.href to be absolute (start with http:// or https://), got %q", key, href)
	}
}
