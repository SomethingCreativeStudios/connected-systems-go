package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Schema path constant for sampling feature validation
const SamplingFeatureSchema = "geojson/samplingFeature-bundled.json"

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

		location := resp.Header.Get("Location")
		resp.Body.Close()
		id := parseID(location, "/samplingFeatures/")
		require.NotEmpty(t, id, "unable to parse id from Location header: %s", location)
		createdIDs = append(createdIDs, id)
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
// Annex A.8: Sampling Feature Conformance Tests (A.33-A.37)
// =============================================================================
func TestSamplingFeatureConformance(t *testing.T) {
	cleanupDB(t)

	// Setup: Create parent system and test sampling features
	systemID, createdSFIDs := setupSamplingFeatureConformanceData(t)
	defer cleanupSamplingFeatureConformanceData(t, createdSFIDs)

	tests := map[string]struct {
		testID      string
		description string
		testFunc    func(t *testing.T)
	}{
		"/conf/sf/resources-endpoint": {
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
		"/conf/sf/canonical-endpoint": {
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
		"/conf/sf/canonical-url": {
			testID:      "A.33",
			description: "Verify that each sampling feature resource is accessible via its canonical URL",
			testFunc: func(t *testing.T) {
				// Test A.33: /conf/sf/canonical-url
				// Requirement: /req/sf/canonical-url
				// Each sampling feature resource SHALL be accessible at URL /samplingFeatures/{id}

				for _, id := range createdSFIDs {
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
		"/conf/sf/ref-from-system": {
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
				assert.Equal(t, len(createdSFIDs), len(features), "Collection should contain all sampling features for the system")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Logf("Test %s: %s", tc.testID, tc.description)
			tc.testFunc(t)
		})
	}
}

// =============================================================================
// Conformance Class: /conf/geojson/sf-schema
// Requirement: /req/geojson/sf-schema
// =============================================================================
func TestGeoJSONSamplingFeatureSchema(t *testing.T) {
	cleanupDB(t)

	// Setup: Create parent system and test sampling features
	_, createdSFIDs := setupSamplingFeatureConformanceData(t)
	defer cleanupSamplingFeatureConformanceData(t, createdSFIDs)

	validator := GetSchemaValidator()

	tests := map[string]struct {
		testID      string
		description string
		testFunc    func(t *testing.T)
	}{
		"/conf/geojson/sf-schema/single-resource": {
			testID:      "A.94",
			description: "Verify single sampling feature resource conforms to samplingFeature.json schema",
			testFunc: func(t *testing.T) {
				// Test A.94: /conf/geojson/sf-schema (single resource)
				// Requirement: /req/geojson/sf-schema
				// Sampling Feature resources SHALL validate against samplingFeature.json schema

				for _, id := range createdSFIDs {
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
		"/conf/geojson/sf-schema/collection-items": {
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
		"/conf/geojson/sf-mappings": {
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

				for _, id := range createdSFIDs {
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

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.testFunc(t)
		})
	}
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/sampling-feature
// Requirement: /req/create-replace-delete/sampling-feature
// A.73: Create
// =============================================================================

func TestSamplingFeatureCRUD_Create(t *testing.T) {
	cleanupDB(t)

	// Setup: Create parent system and test sampling features
	systemID, createdSFIDs := setupSamplingFeatureConformanceData(t)
	defer cleanupSamplingFeatureConformanceData(t, createdSFIDs)

	testCases := map[string]struct {
		createPayload map[string]interface{}
		validateFunc  func(created map[string]interface{})
	}{
		"Create Sampling Feature": {
			createPayload: map[string]interface{}{
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
			},
			validateFunc: func(created map[string]interface{}) {
				// Validate created resource
				assert.Equal(t, "Feature", created["type"], "Created resource must be a GeoJSON Feature")

				props, ok := created["properties"].(map[string]interface{})
				require.True(t, ok, "Created resource must have properties object")

				assert.Equal(t, "urn:ogc:conf:sf:crud-create-001", props["uid"], "Created resource must have correct uid")
				assert.Equal(t, "CRUD Test Point", props["name"], "Created resource must have correct name")
				assert.Equal(t, "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint", props["featureType"], "Created resource must have correct featureType")

				sampledFeatureLink, ok := props["sampledFeature@link"].(map[string]interface{})
				require.True(t, ok, "Created resource must have sampledFeature@link")
				assert.Equal(t, "http://example.org/features/test-foi", sampledFeatureLink["href"], "sampledFeature@link must have correct href")

				geometry, ok := created["geometry"].(map[string]interface{})
				require.True(t, ok, "Created resource must have geometry object")
				assert.Equal(t, "Point", geometry["type"], "Geometry type must be Point")

				coords, ok := geometry["coordinates"].([]interface{})
				require.True(t, ok, "Geometry must have coordinates array")
				assert.Equal(t, -118.0, coords[0], "Geometry longitude must match")
				assert.Equal(t, 34.0, coords[1], "Geometry latitude must match")
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			body, _ := json.Marshal(tc.createPayload)
			req, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/samplingFeatures", bytes.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/geo+json")
			req.Header.Set("Accept", "application/geo+json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Must return 201 Created
			assert.Equal(t, http.StatusCreated, resp.StatusCode, "POST must return 201 Created")

			createdResource, err := FollowLocation(resp, "application/geo+json")
			require.NoError(t, err)

			// Validate created resource
			tc.validateFunc(*createdResource)
		})
	}
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/sampling-feature
// Requirement: /req/create-replace-delete/sampling-feature
// A.73: Replace
// =============================================================================

func testSamplingFeatureCRUD_Replace(t *testing.T, systemID string) {
	cleanupDB(t)

	testcases := map[string]struct {
		createPayload  map[string]interface{}
		replacePayload map[string]interface{}
		validateFunc   func(updated map[string]interface{})
	}{
		"Replace Sampling Feature": {
			// Create a sampling feature first
			createPayload: map[string]interface{}{
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
			},
			replacePayload: map[string]interface{}{
				"type": "Feature",
				"properties": map[string]interface{}{
					"uid":         "urn:ogc:conf:sf:crud-replace-001",
					"name":        "Updated Name",
					"description": "Updated description",
					"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
					"sampledFeature@link": map[string]interface{}{
						"href":  "http://example.org/features/test-foi",
						"type":  "application/geo+json",
						"title": "Test Feature of Interest",
					},
				},
				"geometry": map[string]interface{}{
					"type":        "Point",
					"coordinates": []float64{-119.0, 35.0},
				},
			},
			validateFunc: func(updated map[string]interface{}) {
				props, ok := updated["properties"].(map[string]interface{})
				require.True(t, ok, "Updated resource must have properties object")

				assert.Equal(t, "urn:ogc:conf:sf:crud-replace-001", props["uid"], "Updated resource must have correct uid")
				assert.Equal(t, "Updated Name", props["name"], "Updated resource must have correct name")
				assert.Equal(t, "Updated description", props["description"], "Updated resource must have correct description")
				assert.Equal(t, "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint", props["featureType"], "Updated resource must have correct featureType")

				sampledFeatureLink, ok := props["sampledFeature@link"].(map[string]interface{})
				require.True(t, ok, "Updated resource must have sampledFeature@link")
				assert.Equal(t, "http://example.org/features/updated-foi", sampledFeatureLink["href"], "sampledFeature@link must have correct href")

				geometry, ok := updated["geometry"].(map[string]interface{})
				require.True(t, ok, "Updated resource must have geometry object")
				assert.Equal(t, "Point", geometry["type"], "Geometry type must be Point")

				coords, ok := geometry["coordinates"].([]interface{})
				require.True(t, ok, "Geometry must have coordinates array")
				assert.Equal(t, -119.0, coords[0], "Geometry longitude must match")
				assert.Equal(t, 35.0, coords[1], "Geometry latitude must match")
			},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			// Create the sampling feature
			createBody, _ := json.Marshal(tc.createPayload)
			createReq, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/samplingFeatures", bytes.NewReader(createBody))
			require.NoError(t, err)
			createReq.Header.Set("Content-Type", "application/geo+json")

			createResp, err := http.DefaultClient.Do(createReq)
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, createResp.StatusCode)

			created, err := FollowLocation(createResp, "application/geo+json")
			require.NoError(t, err)
			sfID := (*created)["id"].(string)
			createResp.Body.Close()

			defer func() {
				req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/samplingFeatures/"+sfID, nil)
				http.DefaultClient.Do(req)
			}()

			// Replace the sampling feature
			replaceBody, _ := json.Marshal(tc.replacePayload)
			replaceReq, err := http.NewRequest(http.MethodPut, testServer.URL+"/samplingFeatures/"+sfID, bytes.NewReader(replaceBody))
			require.NoError(t, err)
			replaceReq.Header.Set("Content-Type", "application/geo+json")
			replaceReq.Header.Set("Accept", "application/geo+json")

			replaceResp, err := http.DefaultClient.Do(replaceReq)
			require.NoError(t, err)
			replaceResp.Body.Close()

			// Must return 204 No Content
			assert.Equal(t, http.StatusNoContent, replaceResp.StatusCode, "PUT must return 204 No Content")

			// Verify the update by fetching the resource
			getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/samplingFeatures/"+sfID, nil)
			getReq.Header.Set("Accept", "application/geo+json")
			getResp, err := http.DefaultClient.Do(getReq)
			require.NoError(t, err)
			defer getResp.Body.Close()

			var updated map[string]interface{}
			err = json.NewDecoder(getResp.Body).Decode(&updated)
			require.NoError(t, err)

			// Validate updated resource
			tc.validateFunc(updated)
		})
	}
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/sampling-feature
// Requirement: /req/create-replace-delete/sampling-feature
// A.73: Delete
// =============================================================================

func TestSamplingFeatureCRUD_Delete(t *testing.T) {
	cleanupDB(t)

	// Setup: Create parent system
	_, samplingFeatureIds := setupSamplingFeatureConformanceData(t)
	defer cleanupSamplingFeatureConformanceData(t, samplingFeatureIds)

	testcases := map[string]struct {
		sampledFeatureID string
	}{
		"Delete Sampling Feature": {
			sampledFeatureID: samplingFeatureIds[0],
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			// Delete the sampling feature
			deleteReq, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/samplingFeatures/"+tc.sampledFeatureID, nil)
			deleteResp, err := http.DefaultClient.Do(deleteReq)
			require.NoError(t, err)
			defer deleteResp.Body.Close()

			// Must return 204 No Content
			assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode, "DELETE must return 204 No Content")

			// Verify the sampling feature is gone
			getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/samplingFeatures/"+tc.sampledFeatureID, nil)
			getResp, err := http.DefaultClient.Do(getReq)
			require.NoError(t, err)
			defer getResp.Body.Close()

			assert.Equal(t, http.StatusNotFound, getResp.StatusCode, "Deleted sampling feature must return 404")
		})
	}
}

// =============================================================================
// Conformance Class: System Link Relationships
// Tests for parentSystem link in sampling features
// =============================================================================

func TestSamplingFeatureSystemLinks(t *testing.T) {
	cleanupDB(t)

	tests := map[string]struct {
		createPayload map[string]interface{}
		description   string
	}{
		"simple-parent-system": {
			createPayload: map[string]interface{}{
				"type": "Feature",
				"properties": map[string]interface{}{
					"uid":         "urn:ogc:conf:sf:link-test-001",
					"name":        "Link Test Point",
					"description": "Sampling feature to test parentSystem link",
					"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
					"sampledFeature@link": map[string]interface{}{
						"href":  "http://example.org/features/link-test-foi",
						"type":  "application/geo+json",
						"title": "Link Test Feature of Interest",
					},
				},
				"geometry": map[string]interface{}{
					"type":        "Point",
					"coordinates": []float64{-117.0, 33.0},
				},
			},
			description: "Verify each sampling feature has a parentSystem link",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			body, _ := json.Marshal(tc.createPayload)
			req, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/some-system-id-2/samplingFeatures", bytes.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/geo+json")
			req.Header.Set("Accept", "application/geo+json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode, "POST must return 201 Created")

			createdResource, err := FollowLocation(resp, "application/geo+json")
			require.NoError(t, err)

			links, ok := (*createdResource)["links"].([]interface{})
			require.True(t, ok, "Created resource must have links array")

			var foundParentSystemLink bool
			for _, linkIface := range links {
				link, ok := linkIface.(map[string]interface{})
				if !ok {
					continue
				}
				if rel, ok := link["rel"].(string); ok && rel == "parentSystem" {
					foundParentSystemLink = true
					break
				}
			}

			assert.True(t, foundParentSystemLink, "Created sampling feature must have a parentSystem link")
		})
	}
}

// =============================================================================
// Conformance Class: SampleOf Relationships
// Tests for sampleOf associations between sampling features
// =============================================================================

func TestSamplingFeatureSampleOfRelationships(t *testing.T) {
	testcases := map[string]struct {
		parentFeature map[string]interface{}
		childFeature  map[string]interface{}
		description   string
	}{
		"point-sample-of-surface": {
			parentFeature: map[string]interface{}{
				"type": "Feature",
				"properties": map[string]interface{}{
					"uid":         "urn:ogc:conf:sf:sampleof-parent-001",
					"name":        "Parent Surface Feature",
					"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingSurface",
					"sampledFeature@link": map[string]interface{}{
						"href":  "http://example.org/features/parent-foi",
						"type":  "application/geo+json",
						"title": "Parent Feature of Interest",
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
			},
			childFeature: map[string]interface{}{
				"type": "Feature",
				"properties": map[string]interface{}{
					"uid":         "urn:ogc:conf:sf:sampleof-child-001",
					"name":        "Child Point Feature",
					"featureType": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
					"sampledFeature@link": map[string]interface{}{
						"href":  "http://example.org/features/child-foi",
						"type":  "application/geo+json",
						"title": "Child Feature of Interest",
					},
				},
				"geometry": map[string]interface{}{
					"type":        "Point",
					"coordinates": []float64{-119.5, 35.5},
				},
			},
			description: "Verify sampleOf relationship between child point and parent surface sampling features",
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			// Create parent feature
			parentBody, _ := json.Marshal(tc.parentFeature)
			parentReq, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/some-parent-id/samplingFeatures", bytes.NewReader(parentBody))
			require.NoError(t, err)
			parentReq.Header.Set("Content-Type", "application/geo+json")
			parentReq.Header.Set("Accept", "application/geo+json")

			parentResp, err := http.DefaultClient.Do(parentReq)
			require.NoError(t, err)
			defer parentResp.Body.Close()

			require.Equal(t, http.StatusCreated, parentResp.StatusCode, "POST must return 201 Created for parent feature")

			parentResource, err := FollowLocation(parentResp, "application/geo+json")
			require.NoError(t, err)
			parentID := (*parentResource)["id"].(string)

			// Create child feature with sampleOf link to parent
			childFeatureWithLink := tc.childFeature
			sampleOfLink := map[string]interface{}{
				"href":  fmt.Sprintf("samplingFeatures/%s", parentID),
				"type":  "application/geo+json",
				"title": "Parent Surface Feature",
				"rel":   "sampleOf",
			}
			childFeatureWithLink["links"] = []map[string]interface{}{sampleOfLink}

			childBody, _ := json.Marshal(childFeatureWithLink)
			childReq, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/some-parent-id/samplingFeatures", bytes.NewReader(childBody))
			require.NoError(t, err)
			childReq.Header.Set("Content-Type", "application/geo+json")
			childReq.Header.Set("Accept", "application/geo+json")

			childResp, err := http.DefaultClient.Do(childReq)
			require.NoError(t, err)
			defer childResp.Body.Close()

			require.Equal(t, http.StatusCreated, childResp.StatusCode, "POST must return 201 Created for child feature")

			childResource, err := FollowLocation(childResp, "application/geo+json")
			require.NoError(t, err)

			// Verify sampleOf@link in child feature
			childLinks, ok := (*childResource)["links"].([]interface{})
			require.True(t, ok, "Child resource must have links array")

			var sampleOfLinkRetrieved map[string]interface{}
			for _, link := range childLinks {
				linkMap, ok := link.(map[string]interface{})
				if ok && linkMap["rel"] == "sampleOf" {
					sampleOfLinkRetrieved = linkMap
					break
				}
			}

			require.NotNil(t, sampleOfLinkRetrieved, "Child resource must have sampleOf@link")
			assert.Equal(t, fmt.Sprintf("samplingFeatures/%s", parentID), sampleOfLinkRetrieved["href"], "sampleOf@link href must match parent feature ID")
		})
	}
}
