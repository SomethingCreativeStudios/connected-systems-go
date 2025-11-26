package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectionsAPI_E2E(t *testing.T) {
	cleanupDB(t)

	t.Run("POST /collections - create collection", func(t *testing.T) {
		payload := map[string]interface{}{
			"id":          "test-collection",
			"title":       "Test Collection",
			"description": "A collection for testing",
		}

		body, _ := json.Marshal(payload)
		resp, err := http.Post(testServer.URL+"/collections", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "test-collection", result["id"])
	})

	t.Run("GET /collections - list collections", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/collections")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Expect "collections" key with an array
		cols, ok := result["collections"].([]interface{})
		require.True(t, ok)
		require.GreaterOrEqual(t, len(cols), 1)
	})

	t.Run("GET /collections/{id} - retrieve collection", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/collections/test-collection")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "test-collection", result["id"])
	})
}
