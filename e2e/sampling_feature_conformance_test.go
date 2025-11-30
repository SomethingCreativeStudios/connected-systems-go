package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Schema path constant for sampling feature validation
const SamplingFeatureSchema = "geojson/samplingFeature-bundled.json"

// TestSamplingFeatureConformance implements OGC Connected Systems API conformance tests
// for the Sampling Features resource as defined in:
// - Clause 14: Sampling Features (https://docs.ogc.org/is/23-001/23-001.html#clause-sampling-features)
// - Annex A.8: Sampling Feature Conformance Tests (A.33-A.37)
// - Annex A.11: Create, Replace, Delete Conformance Tests (A.73)
// - Annex A.10: Advanced Filtering Conformance Tests
// - Annex A.14: GeoJSON Encoding Conformance Tests (A.94-A.95)
func TestSamplingFeatureConformance(t *testing.T) {
	cleanupDB(t)

	// Setup: Create parent system and test sampling features
	systemID, createdSFIDs := setupSamplingFeatureConformanceData(t)
	defer cleanupSamplingFeatureConformanceData(t, createdSFIDs)

	t.Run("Conformance Class: /conf/sf", func(t *testing.T) {
		testSamplingFeatureConformanceClass(t, systemID, createdSFIDs)
	})

	t.Run("Conformance Class: /conf/geojson/sf-schema", func(t *testing.T) {
		testGeoJSONSamplingFeatureSchema(t, createdSFIDs)
	})

	t.Run("Conformance Class: /conf/create-replace-delete/sampling-feature", func(t *testing.T) {
		testSamplingFeatureCRUD(t, systemID)
	})

	t.Run("Conformance Class: /conf/sf/system-link-relationships", func(t *testing.T) {
		testSamplingFeatureSystemLinks(t, systemID, createdSFIDs)
	})

	t.Run("Conformance Class: /conf/sf/sample-of-relationships", func(t *testing.T) {
		testSamplingFeatureSampleOfRelationships(t, systemID)
	})
}

// setupSamplingFeatureConformanceData creates parent system and test sampling features
func setupSamplingFeatureConformanceData(t *testing.T) (string, []string) {
	t.Helper()

	// Create a parent system to host sampling features
	sysPayload := map[string]interface{}{
		"type": "Feature",
		"properties": map[string]interface{}{
			"uid":         "urn:ogc:conf:system:sf-parent-001",
			"name":        "Parent System for SF Conformance Tests",
			"description": "System used as parent for sampling feature conformance tests",
		},
	}
	sysBody, _ := json.Marshal(sysPayload)
	sysResp, err := http.Post(testServer.URL+"/systems", "application/geo+json", bytes.NewReader(sysBody))
	require.NoError(t, err)
	defer sysResp.Body.Close()
	require.Equal(t, http.StatusCreated, sysResp.StatusCode, "failed to create parent system")

	var sysCreated map[string]interface{}
	err = json.NewDecoder(sysResp.Body).Decode(&sysCreated)
	require.NoError(t, err)
	systemID := sysCreated["id"].(string)

	// Test sampling features with different geometry types
	testSamplingFeatures := []map[string]interface{}{
		{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":         "urn:ogc:conf:sf:point-001",
				"name":        "Weather Station Sampling Point",
				"description": "Primary measurement location at weather station",
				"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
				"sampledFeature@link": map[string]interface{}{
					"href":  "http://example.org/features/central-park",
					"type":  "application/geo+json",
					"title": "Central Park Feature of Interest",
				},
			},
			"geometry": map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{-117.1625, 32.715, 125.0},
			},
		},
		{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":         "urn:ogc:conf:sf:curve-001",
				"name":        "River Cross-Section Transect",
				"description": "Sampling line across river for flow measurements",
				"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingCurve",
				"sampledFeature@link": map[string]interface{}{
					"href":  "http://example.org/features/colorado-river",
					"type":  "application/geo+json",
					"title": "Colorado River",
				},
			},
			"geometry": map[string]interface{}{
				"type": "LineString",
				"coordinates": [][]float64{
					{-114.5678, 36.1234, 320.5},
					{-114.5672, 36.1238, 318.2},
					{-114.5666, 36.1242, 319.8},
				},
			},
		},
		{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":         "urn:ogc:conf:sf:surface-001",
				"name":        "Satellite Image Footprint",
				"description": "Satellite image footprint representing sampled surface area",
				"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingSurface",
				"sampledFeature@link": map[string]interface{}{
					"href":  "http://example.org/features/agricultural-region",
					"type":  "application/geo+json",
					"title": "Agricultural Region 7A",
				},
			},
			"geometry": map[string]interface{}{
				"type": "Polygon",
				"coordinates": [][][]float64{
					{
						{-122.5, 37.7},
						{-122.5, 37.9},
						{-122.3, 37.9},
						{-122.3, 37.7},
						{-122.5, 37.7},
					},
				},
			},
		},
		{
			"type": "Feature",
			"properties": map[string]interface{}{
				"uid":         "urn:ogc:conf:sf:point-002",
				"name":        "Soil Sample Location",
				"description": "Soil sampling point for agricultural analysis",
				"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
				"sampledFeature@link": map[string]interface{}{
					"href":  "http://example.org/features/agricultural-region",
					"type":  "application/geo+json",
					"title": "Agricultural Region 7A",
				},
			},
			"geometry": map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{-122.4, 37.8},
			},
		},
	}

	var createdIDs []string
	for _, sf := range testSamplingFeatures {
		body, _ := json.Marshal(sf)
		req, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/samplingFeatures", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/geo+json")
		req.Header.Set("Accept", "application/geo+json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode, "failed to create sampling feature: %s", sf["properties"].(map[string]interface{})["name"])

		createdIDs = append(createdIDs, parseSamplingFeatureID(resp.Header.Get("Location")))
	}

	return systemID, createdIDs
}

func cleanupSamplingFeatureConformanceData(t *testing.T, ids []string) {
	t.Helper()
	for _, id := range ids {
		req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/samplingFeatures/"+id, nil)
		http.DefaultClient.Do(req)
	}
}

// =============================================================================
// Conformance Class: /conf/sf
// Requirement: /req/sf
// =============================================================================

func testSamplingFeatureConformanceClass(t *testing.T, systemID string, sfIDs []string) {
	tests := []struct {
		name        string
		testID      string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "/conf/sf/resources-endpoint",
			testID:      "A.34",
			description: "Verify that sampling feature resources can be retrieved from the /samplingFeatures endpoint",
			testFunc: func(t *testing.T) {
				// Test A.34: /conf/sf/resources-endpoint
				// Requirement: /req/sf/resources-endpoint
				// The server SHALL expose all served sampling feature resources at the path /samplingFeatures

				req, err := http.NewRequest(http.MethodGet, testServer.URL+"/samplingFeatures", nil)
				require.NoError(t, err)
				req.Header.Set("Accept", "application/geo+json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				// Response must be 200 OK
				assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /samplingFeatures must return 200 OK")

				// Response must be valid JSON
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				var collection map[string]interface{}
				err = json.Unmarshal(body, &collection)
				require.NoError(t, err, "Response must be valid JSON")

				// Must be a FeatureCollection
				assert.Equal(t, "FeatureCollection", collection["type"], "Response must be a FeatureCollection")

				// Must contain features array
				features, ok := collection["features"].([]interface{})
				assert.True(t, ok, "Response must contain 'features' array")
				assert.GreaterOrEqual(t, len(features), 1, "Collection must contain at least one sampling feature")
			},
		},
		{
			name:        "/conf/sf/canonical-endpoint",
			testID:      "A.35",
			description: "Verify that sampling feature resources can be accessed via the /samplingFeatures endpoint",
			testFunc: func(t *testing.T) {
				// Test A.35: /conf/sf/canonical-endpoint
				// Requirement: /req/sf/canonical-endpoint
				// Server SHALL expose the canonical URL at path /samplingFeatures

				// Verify the endpoint exists and returns proper content type
				req, err := http.NewRequest(http.MethodGet, testServer.URL+"/samplingFeatures", nil)
				require.NoError(t, err)
				req.Header.Set("Accept", "application/geo+json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				// Verify content type header contains expected media type
				contentType := resp.Header.Get("Content-Type")
				assert.NotEmpty(t, contentType, "Content-Type header must be set")
				// Check that it's either the requested format or at least a valid JSON type
				assert.True(t,
					contentType == "application/geo+json" ||
						contentType == "application/json" ||
						contentType == "application/json; charset=utf-8",
					"Content-Type must be a valid GeoJSON media type, got: %s", contentType)
			},
		},
		{
			name:        "/conf/sf/canonical-url",
			testID:      "A.33",
			description: "Verify that each sampling feature resource is accessible via its canonical URL",
			testFunc: func(t *testing.T) {
				// Test A.33: /conf/sf/canonical-url
				// Requirement: /req/sf/canonical-url
				// Each sampling feature resource SHALL be accessible at URL /samplingFeatures/{id}

				for _, id := range sfIDs {
					t.Run(fmt.Sprintf("samplingFeature-%s", id), func(t *testing.T) {
						url := fmt.Sprintf("%s/samplingFeatures/%s", testServer.URL, id)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/geo+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /samplingFeatures/{id} must return 200 OK")

						var sf map[string]interface{}
						err = json.NewDecoder(resp.Body).Decode(&sf)
						require.NoError(t, err)

						// Verify it's a GeoJSON Feature
						assert.Equal(t, "Feature", sf["type"], "Response must be a GeoJSON Feature")

						// Verify the returned sampling feature has the expected ID
						assert.Equal(t, id, sf["id"], "Returned sampling feature ID must match requested ID")
					})
				}
			},
		},
		{
			name:        "/conf/sf/ref-from-system",
			testID:      "A.37",
			description: "Verify that sampling features can be accessed as sub-resources of systems",
			testFunc: func(t *testing.T) {
				// Test A.37: /conf/sf/ref-from-system
				// Requirement: /req/sf/ref-from-system
				// Sampling features SHALL be accessible at /systems/{sysId}/samplingFeatures

				url := fmt.Sprintf("%s/systems/%s/samplingFeatures", testServer.URL, systemID)
				req, err := http.NewRequest(http.MethodGet, url, nil)
				require.NoError(t, err)
				req.Header.Set("Accept", "application/geo+json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /systems/{sysId}/samplingFeatures must return 200 OK")

				var collection map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&collection)
				require.NoError(t, err)

				// Must be a FeatureCollection
				assert.Equal(t, "FeatureCollection", collection["type"], "Response must be a FeatureCollection")

				features, ok := collection["features"].([]interface{})
				assert.True(t, ok, "Response must contain 'features' array")

				// Should contain all the sampling features created for this system
				assert.Equal(t, len(sfIDs), len(features), "Collection should contain all sampling features for the system")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Test %s: %s", tc.testID, tc.description)
			tc.testFunc(t)
		})
	}
}

// =============================================================================
// Conformance Class: /conf/geojson/sf-schema
// Requirement: /req/geojson/sf-schema
// =============================================================================

func testGeoJSONSamplingFeatureSchema(t *testing.T, sfIDs []string) {
	validator := GetSchemaValidator()

	tests := []struct {
		name        string
		testID      string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "/conf/geojson/sf-schema/single-resource",
			testID:      "A.94",
			description: "Verify single sampling feature resource conforms to samplingFeature.json schema",
			testFunc: func(t *testing.T) {
				// Test A.94: /conf/geojson/sf-schema (single resource)
				// Requirement: /req/geojson/sf-schema
				// Sampling Feature resources SHALL validate against samplingFeature.json schema

				for _, id := range sfIDs {
					t.Run(fmt.Sprintf("samplingFeature-%s", id), func(t *testing.T) {
						url := fmt.Sprintf("%s/samplingFeatures/%s", testServer.URL, id)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/geo+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						require.Equal(t, http.StatusOK, resp.StatusCode)

						body, err := io.ReadAll(resp.Body)
						require.NoError(t, err)

						// Validate against sampling feature schema
						err = validator.ValidateJSON(SamplingFeatureSchema, body)
						assert.NoError(t, err, "Sampling feature must validate against samplingFeature.json schema")
					})
				}
			},
		},
		{
			name:        "/conf/geojson/sf-schema/collection-items",
			testID:      "A.94",
			description: "Verify sampling feature collection items conform to samplingFeature.json schema",
			testFunc: func(t *testing.T) {
				// Test A.94: /conf/geojson/sf-schema (collection items)
				// Each item in sampling feature collection SHALL validate against samplingFeature.json schema

				req, err := http.NewRequest(http.MethodGet, testServer.URL+"/samplingFeatures", nil)
				require.NoError(t, err)
				req.Header.Set("Accept", "application/geo+json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				var collection map[string]interface{}
				err = json.Unmarshal(body, &collection)
				require.NoError(t, err)

				features, ok := collection["features"].([]interface{})
				require.True(t, ok, "Collection must have features array")

				for i, item := range features {
					t.Run(fmt.Sprintf("item-%d", i), func(t *testing.T) {
						itemBytes, err := json.Marshal(item)
						require.NoError(t, err)

						err = validator.ValidateJSON(SamplingFeatureSchema, itemBytes)
						assert.NoError(t, err, "Collection item must validate against samplingFeature.json schema")
					})
				}
			},
		},
		{
			name:        "/conf/geojson/sf-mappings",
			testID:      "A.95",
			description: "Verify sampling feature resource has required GeoJSON mappings per Table 46/47",
			testFunc: func(t *testing.T) {
				// Test A.95: /conf/geojson/sf-mappings
				// Requirement: /req/geojson/sf-mappings
				// Sampling Feature resources SHALL have mappings per Table 46/47 of the spec

				// Table 46/47 defines required sampling feature attributes:
				// - type: "Feature" (GeoJSON)
				// - id: local identifier
				// - geometry: GeoJSON geometry (Point, LineString, Polygon, etc.)
				// - properties.uid: unique identifier (URI)
				// - properties.name: human readable name
				// - properties.featureType: type of sampling feature
				// - properties.sampledFeature@link: link to sampled feature (required)
				// - links: array with parentSystem link

				for _, id := range sfIDs {
					t.Run(fmt.Sprintf("samplingFeature-%s", id), func(t *testing.T) {
						url := fmt.Sprintf("%s/samplingFeatures/%s", testServer.URL, id)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/geo+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						require.Equal(t, http.StatusOK, resp.StatusCode)

						var sf map[string]interface{}
						err = json.NewDecoder(resp.Body).Decode(&sf)
						require.NoError(t, err)

						// Verify GeoJSON Feature structure
						assert.Equal(t, "Feature", sf["type"], "Sampling feature must have type 'Feature'")
						assert.NotEmpty(t, sf["id"], "Sampling feature must have id")
						assert.NotNil(t, sf["geometry"], "Sampling feature must have geometry")

						// Verify properties
						props, ok := sf["properties"].(map[string]interface{})
						require.True(t, ok, "Sampling feature must have properties object")

						assert.NotEmpty(t, props["uid"], "Properties must have uid")
						assert.NotEmpty(t, props["name"], "Properties must have name")
						assert.NotEmpty(t, props["featureType"], "Properties must have featureType")

						// Verify sampledFeature@link (required per spec)
						sampledFeatureLink, ok := props["sampledFeature@link"]
						assert.True(t, ok, "Properties must have sampledFeature@link")
						if ok {
							link, ok := sampledFeatureLink.(map[string]interface{})
							assert.True(t, ok, "sampledFeature@link must be an object")
							if ok {
								assert.NotEmpty(t, link["href"], "sampledFeature@link must have href")
							}
						}

						// Verify geometry has valid type
						geometry, ok := sf["geometry"].(map[string]interface{})
						require.True(t, ok, "Geometry must be an object")
						geomType, ok := geometry["type"].(string)
						require.True(t, ok, "Geometry must have type")
						validGeomTypes := []string{"Point", "LineString", "Polygon", "MultiPoint", "MultiLineString", "MultiPolygon"}
						assert.Contains(t, validGeomTypes, geomType, "Geometry type must be valid GeoJSON type")
					})
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Test %s: %s", tc.testID, tc.description)
			tc.testFunc(t)
		})
	}
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/sampling-feature
// Requirement: /req/create-replace-delete/sampling-feature
// =============================================================================

func testSamplingFeatureCRUD(t *testing.T, systemID string) {
	tests := []struct {
		name        string
		testID      string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "/conf/create-replace-delete/sampling-feature/create",
			testID:      "A.73",
			description: "Verify sampling feature creation via POST",
			testFunc: func(t *testing.T) {
				// Test A.73: Create sampling feature
				// Requirement: Server SHALL support POST to /systems/{sysId}/samplingFeatures

				payload := map[string]interface{}{
					"type": "Feature",
					"properties": map[string]interface{}{
						"uid":         "urn:ogc:conf:sf:crud-create-001",
						"name":        "CRUD Test Point",
						"description": "Sampling feature created for CRUD test",
						"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
						"sampledFeature@link": map[string]interface{}{
							"href":  "http://example.org/features/test-foi",
							"type":  "application/geo+json",
							"title": "Test Feature of Interest",
						},
					},
					"geometry": map[string]interface{}{
						"type":        "Point",
						"coordinates": []float64{-118.0, 34.0},
					},
				}

				body, _ := json.Marshal(payload)
				req, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/samplingFeatures", bytes.NewReader(body))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/geo+json")
				req.Header.Set("Accept", "application/geo+json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				// Must return 201 Created
				assert.Equal(t, http.StatusCreated, resp.StatusCode, "POST must return 201 Created")

				// Must return Location header
				location := resp.Header.Get("Location")
				assert.NotEmpty(t, location, "Response must include Location header")

				var id = parseSamplingFeatureID(location)

				// Must return the created resource with an ID
				assert.NotEmpty(t, id, "Response must include id")

				// Cleanup
				req, _ = http.NewRequest(http.MethodDelete, testServer.URL+"/samplingFeatures/"+id, nil)
				http.DefaultClient.Do(req)
			},
		},
		{
			name:        "/conf/create-replace-delete/sampling-feature/replace",
			testID:      "A.73",
			description: "Verify sampling feature replacement via PUT",
			testFunc: func(t *testing.T) {
				// Test A.73: Replace sampling feature
				// Requirement: Server SHALL support PUT to /samplingFeatures/{id}

				// Create a sampling feature first
				createPayload := map[string]interface{}{
					"type": "Feature",
					"properties": map[string]interface{}{
						"uid":         "urn:ogc:conf:sf:crud-replace-001",
						"name":        "Original Name",
						"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
						"sampledFeature@link": map[string]interface{}{
							"href":  "http://example.org/features/test-foi",
							"type":  "application/geo+json",
							"title": "Test Feature of Interest",
						},
					},
					"geometry": map[string]interface{}{
						"type":        "Point",
						"coordinates": []float64{-118.0, 34.0},
					},
				}

				createBody, _ := json.Marshal(createPayload)
				createReq, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/samplingFeatures", bytes.NewReader(createBody))
				require.NoError(t, err)
				createReq.Header.Set("Content-Type", "application/geo+json")

				createResp, err := http.DefaultClient.Do(createReq)
				require.NoError(t, err)
				require.Equal(t, http.StatusCreated, createResp.StatusCode)

				sfID := parseSamplingFeatureID(createResp.Header.Get("Location"))
				createResp.Body.Close()

				defer func() {
					req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/samplingFeatures/"+sfID, nil)
					http.DefaultClient.Do(req)
				}()

				// Replace the sampling feature
				replacePayload := map[string]interface{}{
					"type": "Feature",
					"properties": map[string]interface{}{
						"uid":         "urn:ogc:conf:sf:crud-replace-001",
						"name":        "Updated Name",
						"description": "Updated description",
						"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
						"sampledFeature@link": map[string]interface{}{
							"href":  "http://example.org/features/updated-foi",
							"type":  "application/geo+json",
							"title": "Updated Feature of Interest",
						},
					},
					"geometry": map[string]interface{}{
						"type":        "Point",
						"coordinates": []float64{-119.0, 35.0},
					},
				}

				replaceBody, _ := json.Marshal(replacePayload)
				replaceReq, err := http.NewRequest(http.MethodPut, testServer.URL+"/samplingFeatures/"+sfID, bytes.NewReader(replaceBody))
				require.NoError(t, err)
				replaceReq.Header.Set("Content-Type", "application/geo+json")
				replaceReq.Header.Set("Accept", "application/geo+json")

				replaceResp, err := http.DefaultClient.Do(replaceReq)
				require.NoError(t, err)
				defer replaceResp.Body.Close()

				// Must return 200 OK
				assert.Equal(t, http.StatusOK, replaceResp.StatusCode, "PUT must return 200 OK")

				// Verify the update
				getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/samplingFeatures/"+sfID, nil)
				getReq.Header.Set("Accept", "application/geo+json")
				getResp, err := http.DefaultClient.Do(getReq)
				require.NoError(t, err)
				defer getResp.Body.Close()

				var updated map[string]interface{}
				err = json.NewDecoder(getResp.Body).Decode(&updated)
				require.NoError(t, err)

				props := updated["properties"].(map[string]interface{})
				assert.Equal(t, "Updated Name", props["name"], "Name must be updated")
			},
		},
		{
			name:        "/conf/create-replace-delete/sampling-feature/delete",
			testID:      "A.73",
			description: "Verify sampling feature deletion via DELETE",
			testFunc: func(t *testing.T) {
				// Test A.73: Delete sampling feature
				// Requirement: Server SHALL support DELETE to /samplingFeatures/{id}

				// Create a sampling feature first
				createPayload := map[string]interface{}{
					"type": "Feature",
					"properties": map[string]interface{}{
						"uid":         "urn:ogc:conf:sf:crud-delete-001",
						"name":        "To Be Deleted",
						"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
						"sampledFeature@link": map[string]interface{}{
							"href":  "http://example.org/features/test-foi",
							"type":  "application/geo+json",
							"title": "Test Feature of Interest",
						},
					},
					"geometry": map[string]interface{}{
						"type":        "Point",
						"coordinates": []float64{-118.0, 34.0},
					},
				}

				createBody, _ := json.Marshal(createPayload)
				createReq, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/samplingFeatures", bytes.NewReader(createBody))
				require.NoError(t, err)
				createReq.Header.Set("Content-Type", "application/geo+json")

				createResp, err := http.DefaultClient.Do(createReq)
				require.NoError(t, err)
				require.Equal(t, http.StatusCreated, createResp.StatusCode)

				sfID := parseSamplingFeatureID(createResp.Header.Get("Location"))
				createResp.Body.Close()

				// Delete the sampling feature
				deleteReq, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/samplingFeatures/"+sfID, nil)
				deleteResp, err := http.DefaultClient.Do(deleteReq)
				require.NoError(t, err)
				defer deleteResp.Body.Close()

				// Must return 204 No Content
				assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode, "DELETE must return 204 No Content")

				// Verify the sampling feature is gone
				getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/samplingFeatures/"+sfID, nil)
				getResp, err := http.DefaultClient.Do(getReq)
				require.NoError(t, err)
				defer getResp.Body.Close()

				assert.Equal(t, http.StatusNotFound, getResp.StatusCode, "Deleted sampling feature must return 404")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Test %s: %s", tc.testID, tc.description)
			tc.testFunc(t)
		})
	}
}

// =============================================================================
// Conformance Class: System Link Relationships
// Tests for parentSystem link in sampling features
// =============================================================================

func testSamplingFeatureSystemLinks(t *testing.T, systemID string, sfIDs []string) {
	tests := []struct {
		name        string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "parentSystem-link-present",
			description: "Verify each sampling feature has a parentSystem link",
			testFunc: func(t *testing.T) {
				// Per OGC spec Table 18, parentSystem is a required association
				// The link should be included in the links array with rel="parentSystem"
				// or rel="ogc-rel:parentSystem"

				for _, id := range sfIDs {
					t.Run(fmt.Sprintf("samplingFeature-%s", id), func(t *testing.T) {
						url := fmt.Sprintf("%s/samplingFeatures/%s", testServer.URL, id)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/geo+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						require.Equal(t, http.StatusOK, resp.StatusCode)

						var sf map[string]interface{}
						err = json.NewDecoder(resp.Body).Decode(&sf)
						require.NoError(t, err)

						// Check for links array
						linksRaw, ok := sf["links"]
						require.True(t, ok, "Sampling feature must have links array")

						links, ok := linksRaw.([]interface{})
						require.True(t, ok, "Links must be an array")

						// Find parentSystem link
						var foundParentSystem bool
						var parentSystemHref string
						for _, linkRaw := range links {
							link, ok := linkRaw.(map[string]interface{})
							if !ok {
								continue
							}
							rel, _ := link["rel"].(string)
							if rel == "parentSystem" || rel == "ogc-rel:parentSystem" {
								foundParentSystem = true
								parentSystemHref, _ = link["href"].(string)
								break
							}
						}

						assert.True(t, foundParentSystem, "Sampling feature must have parentSystem link")
						assert.Contains(t, parentSystemHref, systemID, "parentSystem link must reference the parent system")
					})
				}
			},
		},
		{
			name:        "parentSystem-link-correct-format",
			description: "Verify parentSystem link has correct format and properties",
			testFunc: func(t *testing.T) {
				// Verify the parentSystem link follows the Link schema
				// Required: href
				// Optional: rel, type, title, uid

				if len(sfIDs) == 0 {
					t.Skip("No sampling features to test")
				}

				url := fmt.Sprintf("%s/samplingFeatures/%s", testServer.URL, sfIDs[0])
				req, err := http.NewRequest(http.MethodGet, url, nil)
				require.NoError(t, err)
				req.Header.Set("Accept", "application/geo+json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode)

				var sf map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&sf)
				require.NoError(t, err)

				links := sf["links"].([]interface{})
				for _, linkRaw := range links {
					link := linkRaw.(map[string]interface{})
					rel, _ := link["rel"].(string)
					if rel == "parentSystem" || rel == "ogc-rel:parentSystem" {
						// Verify required href
						href, ok := link["href"].(string)
						assert.True(t, ok, "parentSystem link must have href")
						assert.NotEmpty(t, href, "parentSystem link href must not be empty")

						// rel should be set
						assert.NotEmpty(t, rel, "parentSystem link must have rel")
						break
					}
				}
			},
		},
		{
			name:        "parentSystem-link-resolves",
			description: "Verify parentSystem link href resolves to valid system",
			testFunc: func(t *testing.T) {
				// The parentSystem link href should be resolvable and return the parent system

				if len(sfIDs) == 0 {
					t.Skip("No sampling features to test")
				}

				url := fmt.Sprintf("%s/samplingFeatures/%s", testServer.URL, sfIDs[0])
				req, err := http.NewRequest(http.MethodGet, url, nil)
				require.NoError(t, err)
				req.Header.Set("Accept", "application/geo+json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusOK, resp.StatusCode)

				var sf map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&sf)
				require.NoError(t, err)

				links := sf["links"].([]interface{})
				for _, linkRaw := range links {
					link := linkRaw.(map[string]interface{})
					rel, _ := link["rel"].(string)
					if rel == "parentSystem" || rel == "ogc-rel:parentSystem" {
						href := link["href"].(string)

						// Resolve the link (it might be relative)
						resolvedURL := testServer.URL + "/" + href
						if href[0] == '/' {
							resolvedURL = testServer.URL + href
						}

						sysReq, err := http.NewRequest(http.MethodGet, resolvedURL, nil)
						require.NoError(t, err)
						sysReq.Header.Set("Accept", "application/geo+json")

						sysResp, err := http.DefaultClient.Do(sysReq)
						require.NoError(t, err)
						defer sysResp.Body.Close()

						assert.Equal(t, http.StatusOK, sysResp.StatusCode, "parentSystem link must resolve to valid system")

						var system map[string]interface{}
						err = json.NewDecoder(sysResp.Body).Decode(&system)
						require.NoError(t, err)

						assert.Equal(t, "Feature", system["type"], "Resolved system must be a GeoJSON Feature")
						break
					}
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Log(tc.description)
			tc.testFunc(t)
		})
	}
}

// =============================================================================
// Conformance Class: SampleOf Relationships
// Tests for sampleOf associations between sampling features
// =============================================================================

func testSamplingFeatureSampleOfRelationships(t *testing.T, systemID string) {
	tests := []struct {
		name        string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "sampleOf-link-chain",
			description: "Verify sampleOf relationships can be established between sampling features",
			testFunc: func(t *testing.T) {
				// Per OGC spec Table 18, sampleOf is an optional association
				// that links to other SamplingFeature resources via sub-sampling

				// Create a primary sampling feature (the one being sampled from)
				primarySF := map[string]interface{}{
					"type": "Feature",
					"properties": map[string]interface{}{
						"uid":         "urn:ogc:conf:sf:primary-sample-001",
						"name":        "Primary Sampling Surface",
						"description": "Large area sampling surface",
						"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingSurface",
						"sampledFeature@link": map[string]interface{}{
							"href":  "http://example.org/features/large-region",
							"type":  "application/geo+json",
							"title": "Large Regional Area",
						},
					},
					"geometry": map[string]interface{}{
						"type": "Polygon",
						"coordinates": [][][]float64{
							{
								{-120.0, 35.0},
								{-120.0, 36.0},
								{-119.0, 36.0},
								{-119.0, 35.0},
								{-120.0, 35.0},
							},
						},
					},
				}

				body, _ := json.Marshal(primarySF)
				req, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/samplingFeatures", bytes.NewReader(body))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/geo+json")
				req.Header.Set("Accept", "application/geo+json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusCreated, resp.StatusCode)

				resp.Body.Close()

				primaryCreated, err := fetchById(parseSamplingFeatureID(resp.Header.Get("Location")))
				require.NoError(t, err)

				primaryID := (*primaryCreated)["id"].(string)
				defer func() {
					req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/samplingFeatures/"+primaryID, nil)
					http.DefaultClient.Do(req)
				}()

				// Create a sub-sampling feature that references the primary via sampleOf
				subSF := map[string]interface{}{
					"type": "Feature",
					"properties": map[string]interface{}{
						"uid":         "urn:ogc:conf:sf:sub-sample-001",
						"name":        "Sub-sampling Point",
						"description": "Point sample taken from larger surface",
						"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
						"sampledFeature@link": map[string]interface{}{
							"href":  "http://example.org/features/large-region",
							"type":  "application/geo+json",
							"title": "Large Regional Area",
						},
					},
					"geometry": map[string]interface{}{
						"type":        "Point",
						"coordinates": []float64{-119.5, 35.5},
					},
					"links": []map[string]interface{}{
						{
							"rel":   "sampleOf",
							"href":  "samplingFeatures/" + primaryID,
							"title": "Primary Sampling Surface",
						},
					},
				}

				subBody, _ := json.Marshal(subSF)
				subReq, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/samplingFeatures", bytes.NewReader(subBody))
				require.NoError(t, err)
				subReq.Header.Set("Content-Type", "application/geo+json")
				subReq.Header.Set("Accept", "application/geo+json")

				subResp, err := http.DefaultClient.Do(subReq)
				require.NoError(t, err)

				if subResp.StatusCode != http.StatusCreated {
					// Read error body for debugging
					errBody, _ := io.ReadAll(subResp.Body)
					subResp.Body.Close()
					t.Logf("Failed to create sub-sampling feature: %s", string(errBody))
					t.Skip("sampleOf link creation not supported")
					return
				}
				subCreated, err := fetchById(parseSamplingFeatureID(subResp.Header.Get("Location")))
				require.NoError(t, err)
				subResp.Body.Close()

				subID := (*subCreated)["id"].(string)
				defer func() {
					req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/samplingFeatures/"+subID, nil)
					http.DefaultClient.Do(req)
				}()

				// Retrieve the sub-sampling feature and verify sampleOf link
				getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/samplingFeatures/"+subID, nil)
				getReq.Header.Set("Accept", "application/geo+json")
				getResp, err := http.DefaultClient.Do(getReq)
				require.NoError(t, err)
				defer getResp.Body.Close()

				var retrieved map[string]interface{}
				err = json.NewDecoder(getResp.Body).Decode(&retrieved)
				require.NoError(t, err)

				// Check for sampleOf link in links array
				linksRaw, ok := retrieved["links"]
				if !ok {
					t.Skip("sampleOf links not returned in response")
					return
				}

				links, ok := linksRaw.([]interface{})
				require.True(t, ok, "Links must be an array")

				var foundSampleOf bool
				for _, linkRaw := range links {
					link, ok := linkRaw.(map[string]interface{})
					if !ok {
						continue
					}
					rel, _ := link["rel"].(string)
					if rel == "sampleOf" || rel == "ogc-rel:sampleOf" {
						foundSampleOf = true
						href, _ := link["href"].(string)
						assert.Contains(t, href, primaryID, "sampleOf link must reference primary sampling feature")
						break
					}
				}

				assert.True(t, foundSampleOf, "Sub-sampling feature must have sampleOf link to primary feature -- For this test")
			},
		},
		{
			name:        "sampleOf-link-format",
			description: "Verify sampleOf link follows correct format when present",
			testFunc: func(t *testing.T) {
				// This test verifies that if sampleOf links are supported,
				// they follow the correct Link schema format

				// Create a sampling feature and check its link format
				sf := map[string]interface{}{
					"type": "Feature",
					"properties": map[string]interface{}{
						"uid":         "urn:ogc:conf:sf:link-format-test",
						"name":        "Link Format Test",
						"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
						"sampledFeature@link": map[string]interface{}{
							"href":  "http://example.org/features/test",
							"type":  "application/geo+json",
							"title": "Test Feature",
						},
					},
					"geometry": map[string]interface{}{
						"type":        "Point",
						"coordinates": []float64{-118.0, 34.0},
					},
				}

				body, _ := json.Marshal(sf)
				req, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/samplingFeatures", bytes.NewReader(body))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/geo+json")
				req.Header.Set("Accept", "application/geo+json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusCreated, resp.StatusCode)

				resp.Body.Close()

				created, err := fetchById(parseSamplingFeatureID(resp.Header.Get("Location")))
				require.NoError(t, err)

				sfID := (*created)["id"].(string)

				defer func() {
					req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/samplingFeatures/"+sfID, nil)
					http.DefaultClient.Do(req)
				}()

				// Verify links array exists and follows Link schema
				linksRaw, ok := (*created)["links"]
				if !ok {
					t.Log("No links array returned")
					return
				}

				links, ok := linksRaw.([]interface{})
				require.True(t, ok, "Links must be an array")

				for _, linkRaw := range links {
					link, ok := linkRaw.(map[string]interface{})
					require.True(t, ok, "Each link must be an object")

					// href is required per Link schema
					href, ok := link["href"].(string)
					assert.True(t, ok, "Link must have href")
					assert.NotEmpty(t, href, "Link href must not be empty")

					// rel should be present
					if rel, ok := link["rel"].(string); ok {
						assert.NotEmpty(t, rel, "Link rel should not be empty if present")
					}
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Log(tc.description)
			tc.testFunc(t)
		})
	}
}

func parseSamplingFeatureID(locationHeader string) string {
	parts := strings.Split(locationHeader, "/samplingFeatures/")

	if len(parts) == 2 {
		return parts[1]
	}

	return ""
}

func fetchById(samplingFeatureID string) (*map[string]interface{}, error) {
	url := fmt.Sprintf("%s/samplingFeatures/%s", testServer.URL, samplingFeatureID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch sampling feature: status %d", resp.StatusCode)
	}

	var sf map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&sf)
	if err != nil {
		return nil, err
	}

	return &sf, nil
}
