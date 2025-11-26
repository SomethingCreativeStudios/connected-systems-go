package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploymentsAPI_E2E(t *testing.T) {
	cleanupDB(t)

	t.Run("POST /deployments - create deployment", func(t *testing.T) {
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":          "urn:test:deployment:deploy-001",
				"name":         "Weather Station Deployment",
				"description":  "Test deployment",
				"featureType":  "http://www.w3.org/ns/ssn/Deployment",
				"deployedType": "fixed",
			},
			"geometry": map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{-117.1625, 32.715},
			},
		}

		body, _ := json.Marshal(payload)
		resp, err := http.Post(testServer.URL+"/deployments", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.NotEmpty(t, result["id"])
	})

	t.Run("GET /deployments - list deployments", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/deployments")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "FeatureCollection", result["type"])
	})

	t.Run("GET /deployments/{id} - get specific deployment", func(t *testing.T) {
		// Create deployment
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":          "urn:test:deployment:deploy-002",
				"name":         "Mobile Deployment",
				"deployedType": "mobile",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/deployments", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		var created map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&created)
		createResp.Body.Close()
		require.NoError(t, err)

		deploymentID := created["id"].(string)

		// Get it
		resp, err := http.Get(testServer.URL + "/deployments/" + deploymentID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("PUT /deployments/{id} - update deployment", func(t *testing.T) {
		// Create deployment
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:deployment:deploy-003",
				"name": "Deployment to Update",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/deployments", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		var created map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&created)
		createResp.Body.Close()
		require.NoError(t, err)

		deploymentID := created["id"].(string)

		// Update it
		updatePayload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:deployment:deploy-003",
				"name": "Updated Deployment",
			},
		}

		updateBody, _ := json.Marshal(updatePayload)
		req, _ := http.NewRequest(http.MethodPut, testServer.URL+"/deployments/"+deploymentID, bytes.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("DELETE /deployments/{id} - delete deployment", func(t *testing.T) {
		// Create deployment
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:deployment:deploy-004",
				"name": "Deployment to Delete",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/deployments", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		var created map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&created)
		createResp.Body.Close()
		require.NoError(t, err)

		deploymentID := created["id"].(string)

		// Delete it
		req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/deployments/"+deploymentID, nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}
