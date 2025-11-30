package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSamplingFeaturesAPI_E2E(t *testing.T) {
	cleanupDB(t)

	// Create a parent system to host sampling features
	sysPayload := map[string]interface{}{
		"type": "Feature",
		"properties": map[string]interface{}{
			"uid":  "urn:test:sensor:sf-parent-001",
			"name": "Parent System for Sampling Features",
		},
	}
	sysBody, _ := json.Marshal(sysPayload)
	sysResp, err := http.Post(testServer.URL+"/systems", "application/json", bytes.NewReader(sysBody))
	require.NoError(t, err)
	defer sysResp.Body.Close()
	var sysCreated map[string]interface{}
	err = json.NewDecoder(sysResp.Body).Decode(&sysCreated)
	require.NoError(t, err)
	systemID := sysCreated["id"].(string)

	t.Run("POST /samplingFeatures - create sampling feature", func(t *testing.T) {
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":         "urn:test:samplingfeature:sf-001",
				"name":        "Sampling Point 1",
				"description": "Test sampling feature",
				"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
			},
			"geometry": map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{-117.1625, 32.715},
			},
		}

		body, _ := json.Marshal(payload)
		resp, err := http.Post(testServer.URL+"/systems/"+systemID+"/samplingFeatures", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		location := resp.Header.Get("Location")
		assert.NotEmpty(t, location, "Location header should be set")

		// Cleanup - delete created sampling feature
		createdID := parseSamplingFeatureID(location)

		assert.NotEmpty(t, createdID, "Created sampling feature should not be empty")

		found, err := fetchById(createdID)
		require.NoError(t, err)
		assert.NotNil(t, found, "Fetched sampling feature should not be nil")

		assert.Equal(t, (*found)["id"], createdID)

	})

	t.Run("GET /samplingFeatures - list sampling features", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/samplingFeatures")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "FeatureCollection", result["type"])
	})

	t.Run("GET /samplingFeatures/{id} - get specific sampling feature", func(t *testing.T) {
		// Create sampling feature
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":         "urn:test:samplingfeature:sf-002",
				"name":        "Sampling Curve 1",
				"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingCurve",
			},
			"geometry": map[string]interface{}{
				"type":        "LineString",
				"coordinates": [][]float64{{-117.0, 32.0}, {-117.1, 32.1}},
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/systems/"+systemID+"/samplingFeatures", "application/json", bytes.NewReader(body))
		createResp.Body.Close()

		locationId := parseSamplingFeatureID(createResp.Header.Get("Location"))

		sfID, err := fetchById(locationId)
		require.NoError(t, err)
		assert.NotEmpty(t, sfID, "Created sampling feature should not be empty")

		// Get it
		resp, err := http.Get(testServer.URL + "/samplingFeatures/" + locationId)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("PUT /samplingFeatures/{id} - update sampling feature", func(t *testing.T) {
		// Create sampling feature
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:samplingfeature:sf-003",
				"name": "SF to Update",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/systems/"+systemID+"/samplingFeatures", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		createResp.Body.Close()

		sfID := parseSamplingFeatureID(createResp.Header.Get("Location"))

		// Update it
		updatePayload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:samplingfeature:sf-003",
				"name": "Updated SF",
			},
		}

		updateBody, _ := json.Marshal(updatePayload)
		req, _ := http.NewRequest(http.MethodPut, testServer.URL+"/samplingFeatures/"+sfID, bytes.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		// Update now returns 204 No Content; verify status and fetch resource to confirm changes
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Fetch and verify update
		fetched, err := fetchById(sfID)
		require.NoError(t, err)
		require.NotNil(t, fetched)
		props := (*fetched)["properties"].(map[string]interface{})
		assert.Equal(t, "Updated SF", props["name"])
	})

	t.Run("DELETE /samplingFeatures/{id} - delete sampling feature", func(t *testing.T) {
		// Create sampling feature
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:samplingfeature:sf-004",
				"name": "SF to Delete",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/systems/"+systemID+"/samplingFeatures", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		createResp.Body.Close()

		sfID := parseSamplingFeatureID(createResp.Header.Get("Location"))
		created, err := fetchById(sfID)
		require.NoError(t, err)
		assert.NotEmpty(t, created, "Created sampling feature should not be empty")

		// verify parentSystem link is present and references our system
		linksIface, ok := (*created)["links"].([]interface{})
		require.True(t, ok, "expected links to be an array")
		foundParent := false
		for _, li := range linksIface {
			lm := li.(map[string]interface{})
			if lm["rel"] == "parentSystem" {
				require.Equal(t, "systems/"+systemID, lm["href"].(string))
				foundParent = true
			}
		}
		require.True(t, foundParent, "parentSystem link not found in created sampling feature")

		// Delete it
		req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/samplingFeatures/"+sfID, nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}
