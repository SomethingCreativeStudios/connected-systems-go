package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeaturesAPI_E2E(t *testing.T) {
	cleanupDB(t)

	// First create a collection to hold features
	var collectionID string
	var featureLocation string
	t.Run("Setup - create collection", func(t *testing.T) {
		payload := map[string]interface{}{
			"id":          "test-features-collection",
			"title":       "Test Features Collection",
			"description": "Collection to host feature E2E tests",
		}

		body, _ := json.Marshal(payload)
		resp, err := http.Post(testServer.URL+"/collections", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		collectionID = result["id"].(string)
	})

	t.Run("POST /collections/{id}/items - create feature", func(t *testing.T) {
		payload := map[string]interface{}{
			"type": "Feature",
			"properties": map[string]interface{}{
				"name":        "Test Feature",
				"description": "A test feature",
			},
			"geometry": map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{-117.1625, 32.715},
			},
		}

		body, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, testServer.URL+"/collections/"+collectionID+"/items", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/geo+json")
		req.Header.Set("Accept", "application/geo+json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		featureLocation = resp.Header.Get("Location")
		require.NotEmpty(t, featureLocation)
	})

	t.Run("GET /collections/{id}/items - list features", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, testServer.URL+"/collections/"+collectionID+"/items", nil)
		require.NoError(t, err)
		req.Header.Set("Accept", "application/geo+json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/geo+json", resp.Header.Get("Content-Type"))
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "FeatureCollection", result["type"])
	})

	t.Run("GET /collections/{id}/items/{featureId} - retrieve feature", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, featureLocation, nil)
		require.NoError(t, err)
		req.Header.Set("Accept", "application/geo+json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/geo+json", resp.Header.Get("Content-Type"))

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "Feature", result["type"])
		assert.NotEmpty(t, result["id"])
	})
}
