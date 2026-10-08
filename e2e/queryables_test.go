package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFeaturesPart3CQLAndCollectionFacade(t *testing.T) {
	cleanupDB(t)
	payload := baseSystemPayload("CQL harbor sensor")
	payload["properties"].(map[string]interface{})["validTime"] = []string{"2026-01-01T00:00:00Z", "2026-01-03T00:00:00Z"}
	createdID := createSystemViaAPI(t, "/systems", payload)

	get := func(filter, language string) *http.Response {
		t.Helper()
		requestURL := testServer.URL + "/collections/systems/items?filter=" + url.QueryEscape(filter)
		if language != "" {
			requestURL += "&filter-lang=" + language
		}
		req, err := http.NewRequest(http.MethodGet, requestURL, nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		return resp
	}

	t.Run("CQL2 text is served directly by the canonical collection facade", func(t *testing.T) {
		resp := get("name = 'CQL harbor sensor'", "")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.NotEqual(t, http.StatusTemporaryRedirect, resp.StatusCode)
		require.Contains(t, resp.Header.Get("Link"), "/collections/systems/queryables")
		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.EqualValues(t, 1, body["numberReturned"])
	})

	t.Run("CQL2 JSON has equivalent results", func(t *testing.T) {
		resp := get(`{"op":"=","args":[{"property":"name"},"CQL harbor sensor"]}`, "cql2-json")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.EqualValues(t, 1, body["numberReturned"])
	})

	t.Run("spatial and temporal CQL2 predicates use CRS84 and interval columns", func(t *testing.T) {
		for _, filter := range []string{
			"S_INTERSECTS(geometry,POINT(-117.1625 32.715))",
			"T_INTERSECTS(validTime,INTERVAL('2026-01-02T00:00:00Z','2026-01-02T12:00:00Z'))",
		} {
			resp := get(filter, "")
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode, filter)
			var body map[string]any
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
			require.EqualValues(t, 1, body["numberReturned"], filter)
		}
	})

	t.Run("CQL is ANDed with legacy filters", func(t *testing.T) {
		requestURL := testServer.URL + "/systems?id=not-" + createdID + "&filter=" + url.QueryEscape("name = 'CQL harbor sensor'")
		resp, err := http.Get(requestURL)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		require.EqualValues(t, 0, body["numberReturned"])
	})

	t.Run("queryables schema and HEAD are available", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/collections/systems/queryables")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Contains(t, resp.Header.Get("Content-Type"), "application/schema+json")
		var schema map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&schema))
		require.Equal(t, false, schema["additionalProperties"])

		headResponse, err := http.Head(testServer.URL + "/collections/systems/items")
		require.NoError(t, err)
		defer headResponse.Body.Close()
		require.Equal(t, http.StatusOK, headResponse.StatusCode)
		body, err := io.ReadAll(headResponse.Body)
		require.NoError(t, err)
		require.Empty(t, body)
	})

	t.Run("bad CQL returns 400", func(t *testing.T) {
		resp := get("notDeclared = 'x'", "")
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestCustomCollectionQueryableCatalog(t *testing.T) {
	cleanupDB(t)
	catalog := map[string]any{
		"properties": map[string]any{
			"siteCode":   map[string]any{"type": "string"},
			"capturedAt": map[string]any{"type": "string", "format": "date-time"},
			"tags":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"additionalProperties": false,
	}
	collection := map[string]any{"id": "custom-queryables", "title": "Custom queryables", "queryables": catalog}
	body, err := json.Marshal(collection)
	require.NoError(t, err)
	resp, err := http.Post(testServer.URL+"/collections", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	require.Equal(t, "custom-queryables", created["id"])
	require.NotContains(t, created, "queryables")

	t.Run("collection metadata omits the catalog", func(t *testing.T) {
		for _, path := range []string{"/collections", "/collections/custom-queryables"} {
			response, err := http.Get(testServer.URL + path)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, response.StatusCode)
			var metadata map[string]any
			err = json.NewDecoder(response.Body).Decode(&metadata)
			response.Body.Close()
			require.NoError(t, err)
			require.NotContains(t, metadata, "queryables")
			if path == "/collections" {
				found := false
				for _, raw := range metadata["collections"].([]any) {
					collection := raw.(map[string]any)
					require.NotContains(t, collection, "queryables")
					found = found || collection["id"] == "custom-queryables"
				}
				require.True(t, found)
			} else {
				require.Equal(t, "custom-queryables", metadata["id"])
			}
		}
	})

	feature := map[string]any{
		"type":       "Feature",
		"properties": map[string]any{"name": "Custom feature", "siteCode": "NORTH-1", "capturedAt": "2026-01-02T03:04:05Z", "tags": []string{"port", "wind"}},
		"geometry":   map[string]any{"type": "Point", "coordinates": []float64{-70, 40}},
	}
	body, err = json.Marshal(feature)
	require.NoError(t, err)
	request, err := http.NewRequest(http.MethodPost, testServer.URL+"/collections/custom-queryables/items", bytes.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/geo+json")
	resp, err = http.DefaultClient.Do(request)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	response, err := http.Get(testServer.URL + "/collections/custom-queryables/items?filter=" + url.QueryEscape("siteCode = 'NORTH-1'"))
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	queryablesURL, _, found := strings.Cut(strings.TrimPrefix(response.Header.Get("Link"), "<"), ">")
	require.True(t, found)
	require.Equal(t, testServer.URL+"/collections/custom-queryables/queryables", queryablesURL)
	queryablesResponse, err := http.Get(queryablesURL)
	require.NoError(t, err)
	defer queryablesResponse.Body.Close()
	require.Equal(t, http.StatusOK, queryablesResponse.StatusCode)
	require.Contains(t, queryablesResponse.Header.Get("Content-Type"), "application/schema+json")
	var schema map[string]any
	require.NoError(t, json.NewDecoder(queryablesResponse.Body).Decode(&schema))
	require.Equal(t, queryablesURL, schema["$id"])
	require.Equal(t, false, schema["additionalProperties"])
	for name, declaration := range catalog["properties"].(map[string]any) {
		require.Equal(t, declaration, schema["properties"].(map[string]any)[name])
	}
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		request, err := http.NewRequest(method, testServer.URL+"/collections/custom-queryables/items", nil)
		require.NoError(t, err)
		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		response.Body.Close()
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Contains(t, response.Header.Get("Link"), "<"+queryablesURL+">")
	}
	var listed map[string]any
	require.NoError(t, json.NewDecoder(response.Body).Decode(&listed))
	require.EqualValues(t, 1, listed["numberReturned"])

	invalid := map[string]any{"id": "unsafe-queryables", "queryables": map[string]any{"properties": map[string]any{"payload": map[string]any{"type": "object"}}, "additionalProperties": false}}
	body, err = json.Marshal(invalid)
	require.NoError(t, err)
	badResponse, err := http.Post(testServer.URL+"/collections", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer badResponse.Body.Close()
	require.Equal(t, http.StatusBadRequest, badResponse.StatusCode)
}
