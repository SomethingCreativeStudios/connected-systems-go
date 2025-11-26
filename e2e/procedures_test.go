package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProceduresAPI_E2E(t *testing.T) {
	cleanupDB(t)

	t.Run("POST /procedures - create procedure", func(t *testing.T) {
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":           "urn:test:procedure:proc-001",
				"name":          "Temperature Measurement Method",
				"description":   "Standard procedure for temperature measurement",
				"featureType":   "http://www.w3.org/ns/ssn/Procedure",
				"procedureType": "method",
			},
		}

		body, _ := json.Marshal(payload)
		resp, err := http.Post(testServer.URL+"/procedures", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.NotEmpty(t, result["id"])
	})

	t.Run("GET /procedures - list procedures", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/procedures")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "FeatureCollection", result["type"])
	})

	t.Run("GET /procedures/{id} - get specific procedure", func(t *testing.T) {
		// Create procedure
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":           "urn:test:procedure:proc-002",
				"name":          "Datasheet Procedure",
				"procedureType": "datasheet",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/procedures", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		var created map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&created)
		createResp.Body.Close()
		require.NoError(t, err)

		procedureID := created["id"].(string)

		// Get it
		resp, err := http.Get(testServer.URL + "/procedures/" + procedureID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("PUT /procedures/{id} - update procedure", func(t *testing.T) {
		// Create procedure
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:procedure:proc-003",
				"name": "Procedure to Update",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/procedures", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		var created map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&created)
		createResp.Body.Close()
		require.NoError(t, err)

		procedureID := created["id"].(string)

		// Update it
		updatePayload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:procedure:proc-003",
				"name": "Updated Procedure",
			},
		}

		updateBody, _ := json.Marshal(updatePayload)
		req, _ := http.NewRequest(http.MethodPut, testServer.URL+"/procedures/"+procedureID, bytes.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("DELETE /procedures/{id} - delete procedure", func(t *testing.T) {
		// Create procedure
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:procedure:proc-004",
				"name": "Procedure to Delete",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/procedures", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		var created map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&created)
		createResp.Body.Close()
		require.NoError(t, err)

		procedureID := created["id"].(string)

		// Delete it
		req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/procedures/"+procedureID, nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}
