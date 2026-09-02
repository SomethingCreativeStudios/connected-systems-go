package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWriteContractValidationRejectsInvalidSensorMLAndAssetValues(t *testing.T) {
	cleanupDB(t)

	post := func(t *testing.T, endpoint, contentType string, payload any) (int, map[string]any) {
		t.Helper()
		body, err := json.Marshal(payload)
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPost, testServer.URL+endpoint, bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", contentType)
		response, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer response.Body.Close()
		var result map[string]any
		responseBody, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		if len(responseBody) > 0 {
			require.NoError(t, json.Unmarshal(responseBody, &result))
		}
		return response.StatusCode, result
	}

	assertDetailPath := func(t *testing.T, body map[string]any, path string) {
		t.Helper()
		details, ok := body["details"].([]any)
		require.True(t, ok, "expected validation details in %v", body)
		require.NotEmpty(t, details)
		first, ok := details[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, path, first["path"])
	}

	validProcedure := func() map[string]any {
		return map[string]any{
			"type":       "PhysicalComponent",
			"label":      "Procedure validation test",
			"uniqueId":   "urn:uuid:" + uuid.NewString(),
			"definition": "http://www.w3.org/ns/sosa/Sensor",
		}
	}

	t.Run("invalid SensorML process type", func(t *testing.T) {
		payload := validProcedure()
		payload["type"] = "Platform"
		status, body := post(t, "/procedures", "application/sml+json", payload)
		require.Equal(t, http.StatusBadRequest, status)
		assertDetailPath(t, body, "type")
	})

	t.Run("invalid SensorML procedure definition", func(t *testing.T) {
		payload := validProcedure()
		payload["definition"] = "https://example.test/not-a-procedure"
		status, body := post(t, "/procedures", "application/sml+json", payload)
		require.Equal(t, http.StatusBadRequest, status)
		assertDetailPath(t, body, "definition")
	})

	t.Run("Procedure GeoJSON explains the required representation", func(t *testing.T) {
		status, body := post(t, "/procedures", "application/geo+json", map[string]any{
			"type": "Feature",
			"properties": map[string]any{
				"uid":         "urn:uuid:" + uuid.NewString(),
				"name":        "Cannot represent process type",
				"featureType": "http://www.w3.org/ns/sosa/Procedure",
			},
		})
		require.Equal(t, http.StatusBadRequest, status)
		assertDetailPath(t, body, "Content-Type")
		require.Contains(t, body["error"], "application/geo+json")
	})

	t.Run("Procedure GeoJSON replacement leaves the record unchanged", func(t *testing.T) {
		createBody, err := json.Marshal(validProcedure())
		require.NoError(t, err)
		create, err := http.NewRequest(http.MethodPost, testServer.URL+"/procedures", bytes.NewReader(createBody))
		require.NoError(t, err)
		create.Header.Set("Content-Type", "application/sml+json")
		created, err := http.DefaultClient.Do(create)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, created.StatusCode)
		location := created.Header.Get("Location")
		created.Body.Close()
		require.NotEmpty(t, location)

		geoBody, err := json.Marshal(map[string]any{
			"type": "Feature",
			"properties": map[string]any{
				"uid": "urn:uuid:" + uuid.NewString(),
			},
		})
		require.NoError(t, err)
		replace, err := http.NewRequest(http.MethodPut, location, bytes.NewReader(geoBody))
		require.NoError(t, err)
		replace.Header.Set("Content-Type", "application/geo+json")
		replaced, err := http.DefaultClient.Do(replace)
		require.NoError(t, err)
		var response map[string]any
		require.NoError(t, json.NewDecoder(replaced.Body).Decode(&response))
		replaced.Body.Close()
		require.Equal(t, http.StatusBadRequest, replaced.StatusCode)
		assertDetailPath(t, response, "Content-Type")

		read, err := http.NewRequest(http.MethodGet, location, nil)
		require.NoError(t, err)
		read.Header.Set("Accept", "application/sml+json")
		current, err := http.DefaultClient.Do(read)
		require.NoError(t, err)
		defer current.Body.Close()
		require.Equal(t, http.StatusOK, current.StatusCode)
		var procedure map[string]any
		require.NoError(t, json.NewDecoder(current.Body).Decode(&procedure))
		require.Equal(t, "Procedure validation test", procedure["label"])
	})

	t.Run("invalid GeoJSON asset type", func(t *testing.T) {
		payload := baseSystemPayload("Invalid GeoJSON asset type")
		payload["properties"].(map[string]any)["assetType"] = "Platform"
		status, body := post(t, "/systems", "application/geo+json", payload)
		require.Equal(t, http.StatusBadRequest, status)
		assertDetailPath(t, body, "properties.assetType")
	})

	t.Run("invalid SensorML asset classifier", func(t *testing.T) {
		status, body := post(t, "/systems", "application/sml+json", map[string]any{
			"type":       "PhysicalSystem",
			"label":      "Invalid SML asset type",
			"uniqueId":   "urn:uuid:" + uuid.NewString(),
			"definition": "http://www.w3.org/ns/sosa/Platform",
			"classifiers": []any{map[string]any{
				"definition": "cs:AssetType",
				"value":      "Platform",
			}},
		})
		require.Equal(t, http.StatusBadRequest, status)
		assertDetailPath(t, body, "classifiers[0].value")
	})

	t.Run("valid SensorML procedure still persists", func(t *testing.T) {
		status, _ := post(t, "/procedures", "application/sml+json", validProcedure())
		require.Equal(t, http.StatusCreated, status)
	})
}
