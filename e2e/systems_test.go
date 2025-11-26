package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemsAPI_E2E(t *testing.T) {
	cleanupDB(t)

	t.Run("POST /systems - create sensor", func(t *testing.T) {
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:sensor:temp-001",
				"name": "Temperature Sensor",
			},
			"geometry": map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{-117.1625, 32.715},
			},
		}

		body, _ := json.Marshal(payload)
		resp, err := http.Post(testServer.URL+"/systems", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.NotEmpty(t, result["id"])
	})

	t.Run("GET /systems - list systems", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/systems")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "FeatureCollection", result["type"])
	})

	t.Run("GET /systems/{id} - get specific system", func(t *testing.T) {
		// Create a system
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:sensor:temp-002",
				"name": "Another Temperature Sensor",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/systems", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		var created map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&created)
		createResp.Body.Close()
		require.NoError(t, err)

		systemID := created["id"].(string)

		resp, err := http.Get(testServer.URL + "/systems/" + systemID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var got map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&got)
		require.NoError(t, err)
		assert.Equal(t, systemID, got["id"])
	})

	t.Run("PUT /systems/{id} - update system", func(t *testing.T) {
		// Create a system
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:sensor:temp-003",
				"name": "Sensor to Update",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/systems", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		var created map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&created)
		createResp.Body.Close()
		require.NoError(t, err)

		systemID := created["id"].(string)

		updatePayload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:sensor:temp-003",
				"name": "Updated Sensor Name",
			},
		}

		updateBody, _ := json.Marshal(updatePayload)
		req, _ := http.NewRequest(http.MethodPut, testServer.URL+"/systems/"+systemID, bytes.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("DELETE /systems/{id} - delete system", func(t *testing.T) {
		// Create a system
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:sensor:temp-004",
				"name": "Sensor to Delete",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/systems", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		var created map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&created)
		createResp.Body.Close()
		require.NoError(t, err)

		systemID := created["id"].(string)

		req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/systems/"+systemID, nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Accept either 204 No Content or 200 OK depending on implementation
		require.True(t, resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK, "unexpected status: %d", resp.StatusCode)

		// Verify it's deleted
		getResp, _ := http.Get(testServer.URL + "/systems/" + systemID)
		assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
		getResp.Body.Close()
	})
}
