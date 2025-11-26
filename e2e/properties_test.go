package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropertiesAPI_E2E(t *testing.T) {
	cleanupDB(t)

	t.Run("POST /properties - create observable property", func(t *testing.T) {
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":          "urn:test:property:prop-001",
				"name":         "Temperature",
				"description":  "Air temperature property",
				"featureType":  "http://www.w3.org/ns/sosa/ObservableProperty",
				"propertyType": "observable",
			},
		}

		body, _ := json.Marshal(payload)
		resp, err := http.Post(testServer.URL+"/properties", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.NotEmpty(t, result["id"])
	})

	t.Run("GET /properties - list properties", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/properties")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "FeatureCollection", result["type"])
	})

	t.Run("GET /properties/{id} - get specific property", func(t *testing.T) {
		// Create property
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":          "urn:test:property:prop-002",
				"name":         "Humidity",
				"propertyType": "observable",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/properties", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		var created map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&created)
		createResp.Body.Close()
		require.NoError(t, err)

		propID := created["id"].(string)

		// Get it
		resp, err := http.Get(testServer.URL + "/properties/" + propID)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("PUT /properties/{id} - update property", func(t *testing.T) {
		// Create property
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:property:prop-003",
				"name": "Property to Update",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/properties", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		var created map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&created)
		createResp.Body.Close()
		require.NoError(t, err)

		propID := created["id"].(string)

		// Update it
		updatePayload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:property:prop-003",
				"name": "Updated Property",
			},
		}

		updateBody, _ := json.Marshal(updatePayload)
		req, _ := http.NewRequest(http.MethodPut, testServer.URL+"/properties/"+propID, bytes.NewReader(updateBody))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("DELETE /properties/{id} - delete property", func(t *testing.T) {
		// Create property
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":  "urn:test:property:prop-004",
				"name": "Property to Delete",
			},
		}

		body, _ := json.Marshal(payload)
		createResp, err := http.Post(testServer.URL+"/properties", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		var created map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&created)
		createResp.Body.Close()
		require.NoError(t, err)

		propID := created["id"].(string)

		// Delete it
		req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/properties/"+propID, nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("POST /properties - create actuable property", func(t *testing.T) {
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":          "urn:test:property:prop-005",
				"name":         "Valve Position",
				"description":  "Actuable valve position property",
				"featureType":  "http://www.w3.org/ns/sosa/ActuableProperty",
				"propertyType": "actuable",
			},
		}

		body, _ := json.Marshal(payload)
		resp, err := http.Post(testServer.URL+"/properties", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})
}
