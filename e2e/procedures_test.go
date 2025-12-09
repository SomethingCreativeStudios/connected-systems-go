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

// Schema path constants for procedure validation
const (
	ProcedureSMLSchema = "sensorml/procedure-bundled.json"
	ProcedureGeoSchema = "geojson/procedure-bundled.json"
)

// setupProcedureConformanceData creates test procedures for conformance testing
func setupProcedureConformanceData(t *testing.T) []string {
	t.Helper()

	testProcedures := []map[string]interface{}{
		{
			"type":        "SimpleProcess",
			"uniqueId":    "urn:ogc:conf:procedure:simple-001",
			"label":       "Simple Measurement Procedure",
			"description": "A simple procedure for measuring temperature",
			"definition":  "http://www.w3.org/ns/sosa/Procedure",
			"inputs": []map[string]interface{}{
				{
					"name":           "input1",
					"id":             "string",
					"label":          "string",
					"description":    "string",
					"type":           "Boolean",
					"updatable":      false,
					"optional":       false,
					"definition":     "http://example.com",
					"referenceFrame": "../dictionary",
					"axisID":         "string",
					"value":          true,
				},
			},
		},
		{
			"type":        "SimpleProcess",
			"uniqueId":    "urn:ogc:conf:procedure:simple-002",
			"label":       "Another Procedure",
			"description": "Another simple procedure",
			"definition":  "http://www.w3.org/ns/sosa/Procedure",
		},
	}

	var createdIDs []string
	for _, proc := range testProcedures {
		body, _ := json.Marshal(proc)
		req, err := http.NewRequest(http.MethodPost, testServer.URL+"/procedures", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/sml+json")
		req.Header.Set("Accept", "application/sml+json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode, "failed to create procedure: %s", proc["label"])

		location := resp.Header.Get("Location")
		resp.Body.Close()
		require.NotEmpty(t, location, "expected Location header on create")
		id := parseID(location, "/procedures/")
		require.NotEmpty(t, id, "unable to parse id from Location header: %s", location)
		createdIDs = append(createdIDs, id)
	}

	return createdIDs
}

func cleanupProcedureConformanceData(t *testing.T, ids []string) {
	t.Helper()
	for _, id := range ids {
		req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/procedures/"+id, nil)
		http.DefaultClient.Do(req)
	}
}

// =============================================================================
// Conformance Class: /conf/procedure
// Requirement: /req/procedure
// Annex A.9: Procedure Conformance Tests
// =============================================================================
func TestProcedureConformance(t *testing.T) {
	cleanupDB(t)

	// Setup: Create test procedures for conformance tests
	createdProcedureIDs := setupProcedureConformanceData(t)
	defer cleanupProcedureConformanceData(t, createdProcedureIDs)

	tests := map[string]struct {
		testID      string
		description string
		testFunc    func(t *testing.T)
	}{
		"/conf/procedure/resources-endpoint": {
			testID:      "A.33",
			description: "Verify that procedure resources can be retrieved from the /procedures endpoint",
			testFunc: func(t *testing.T) {
				// Test A.33: /conf/procedure/resources-endpoint
				// Requirement: /req/procedure/resources-endpoint
				// The server SHALL expose all served procedure resources at the path /procedures

				req, err := http.NewRequest(http.MethodGet, testServer.URL+"/procedures", nil)
				require.NoError(t, err)
				req.Header.Set("Accept", "application/sml+json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				// Response must be 200 OK
				assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /procedures must return 200 OK")

				// Response must be valid JSON
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				var collection map[string]interface{}
				err = json.Unmarshal(body, &collection)
				require.NoError(t, err, "Response must be valid JSON")

				// Must contain features array (OGC FeatureCollection uses "features")
				features, ok := collection["features"].([]interface{})
				assert.True(t, ok, "Response must contain 'features' array")
				assert.GreaterOrEqual(t, len(features), 1, "Collection must contain at least one procedure")
			},
		},
		"/conf/procedure/canonical-endpoint": {
			testID:      "A.34",
			description: "Verify that procedure resources can be accessed via the /procedures endpoint",
			testFunc: func(t *testing.T) {
				// Test A.34: /conf/procedure/canonical-endpoint
				// Requirement: /req/procedure/canonical-endpoint
				// Server SHALL expose the canonical URL at path /procedures

				// Verify the endpoint exists and returns proper content type
				req, err := http.NewRequest(http.MethodGet, testServer.URL+"/procedures", nil)
				require.NoError(t, err)
				req.Header.Set("Accept", "application/sml+json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				// Verify content type header contains expected media type
				contentType := resp.Header.Get("Content-Type")
				assert.NotEmpty(t, contentType, "Content-Type header must be set")
				assert.True(t,
					contentType == "application/sml+json" ||
						contentType == "application/json" ||
						contentType == "application/json; charset=utf-8",
					"Content-Type must be a valid JSON media type, got: %s", contentType)
			},
		},
		"/conf/procedure/canonical-url": {
			testID:      "A.32",
			description: "Verify that each procedure resource is accessible via its canonical URL",
			testFunc: func(t *testing.T) {
				// Test A.32: /conf/procedure/canonical-url
				// Requirement: /req/procedure/canonical-url
				// Each procedure resource SHALL be accessible at URL /procedures/{id}

				for _, id := range createdProcedureIDs {
					t.Run(fmt.Sprintf("procedure-%s", id), func(t *testing.T) {
						url := fmt.Sprintf("%s/procedures/%s", testServer.URL, id)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/sml+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /procedures/{id} must return 200 OK")

						// Verify ID in response matches
						var proc map[string]interface{}
						err = json.NewDecoder(resp.Body).Decode(&proc)
						require.NoError(t, err)
						assert.Equal(t, id, proc["id"], "Returned procedure ID must match requested ID")
					})
				}
			},
		},
		"/conf/sensorml/procedure-schema": {
			testID:      "A.35",
			description: "Verify that procedure resources validate against the SensorML schema",
			testFunc: func(t *testing.T) {
				// Test A.35: /conf/sensorml/procedure-schema
				// Requirement: /req/sensorml/procedure-schema
				// Procedure resources SHALL validate against the SensorML schema
				// See: https://docs.ogc.org/is/23-001/23-001.html#_conf_sensorml_procedure-schema

				for _, id := range createdProcedureIDs {
					t.Run(fmt.Sprintf("validate-sml-%s", id), func(t *testing.T) {
						url := fmt.Sprintf("%s/procedures/%s", testServer.URL, id)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/sml+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						body, err := io.ReadAll(resp.Body)
						require.NoError(t, err)

						// Validate against SensorML schema
						err = validateAgainstSchema(t, body, ProcedureSMLSchema)
						assert.NoError(t, err, "Procedure resource must validate against SensorML schema")
					})

					t.Run(fmt.Sprintf("validate-geo-%s", id), func(t *testing.T) {
						url := fmt.Sprintf("%s/procedures/%s", testServer.URL, id)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/geo+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						body, err := io.ReadAll(resp.Body)
						require.NoError(t, err)

						// Validate against GeoJSON schema
						err = validateAgainstSchema(t, body, ProcedureGeoSchema)
						assert.NoError(t, err, "Procedure resource must validate against GeoJSON schema")
					})
				}
			},
		},
		"/conf/sensorml/procedure-sml-class": {
			testID:      "A.36",
			description: "Verify that procedure resources are valid SensorML classes",
			testFunc: func(t *testing.T) {
				// Test A.36: /conf/sensorml/procedure-sml-class
				// Requirement: /req/sensorml/procedure-sml-class
				// Procedure resources SHALL be of type SimpleProcess, AggregateProcess, PhysicalSystem, or PhysicalComponent
				// See: https://docs.ogc.org/is/23-001/23-001.html#_conf_sensorml_procedure-sml-class

				validTypes := []string{"SimpleProcess", "AggregateProcess", "PhysicalSystem", "PhysicalComponent"}

				for _, id := range createdProcedureIDs {
					t.Run(fmt.Sprintf("check-type-%s", id), func(t *testing.T) {
						url := fmt.Sprintf("%s/procedures/%s", testServer.URL, id)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/sml+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						var proc map[string]interface{}
						err = json.NewDecoder(resp.Body).Decode(&proc)
						require.NoError(t, err)

						procType, ok := proc["type"].(string)
						assert.True(t, ok, "Procedure must have a 'type' field")
						assert.Contains(t, validTypes, procType, "Procedure type must be one of the valid SensorML process types")
					})
				}
			},
		},
		"/conf/sensorml/procedure-mappings": {
			testID:      "A.37",
			description: "Verify that procedure resources contain required mappings",
			testFunc: func(t *testing.T) {
				// Test A.37: /conf/sensorml/procedure-mappings
				// Requirement: /req/sensorml/procedure-mappings
				// Procedure resources SHALL contain required fields like uniqueId, label, etc.
				// See: https://docs.ogc.org/is/23-001/23-001.html#_conf_sensorml_procedure-mappings

				for _, id := range createdProcedureIDs {
					t.Run(fmt.Sprintf("check-mappings-%s", id), func(t *testing.T) {
						url := fmt.Sprintf("%s/procedures/%s", testServer.URL, id)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/sml+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						var proc map[string]interface{}
						err = json.NewDecoder(resp.Body).Decode(&proc)
						require.NoError(t, err)

						// Check required fields
						assert.NotEmpty(t, proc["uniqueId"], "Procedure must have a uniqueId")
						assert.NotEmpty(t, proc["label"], "Procedure must have a label")
						// definition is often required but depends on specific profile, checking it's present if set
						if _, ok := proc["definition"]; ok {
							assert.NotEmpty(t, proc["definition"], "Procedure definition should not be empty if present")
						}
					})
				}

				// Also validate GeoJSON mappings per OGC GeoJSON requirements
				// See: https://docs.ogc.org/is/23-001/23-001.html#_conf_geojson_procedure-mappings
				for _, id := range createdProcedureIDs {
					t.Run(fmt.Sprintf("check-geo-mappings-%s", id), func(t *testing.T) {
						url := fmt.Sprintf("%s/procedures/%s", testServer.URL, id)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/geo+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						var feat map[string]interface{}
						err = json.NewDecoder(resp.Body).Decode(&feat)
						require.NoError(t, err)

						// Ensure GeoJSON Feature properties contain expected mappings
						props, ok := feat["properties"].(map[string]interface{})
						require.True(t, ok, "GeoJSON Feature must have properties object")
						assert.NotEmpty(t, props["uid"], "GeoJSON properties must include uid")
						assert.NotEmpty(t, props["name"], "GeoJSON properties must include name/label")
					})
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Logf("Running test %s: %s", tc.testID, tc.description)
			tc.testFunc(t)
		})
	}
}

// Conformance Class: /conf/create-replace-delete/procedure
// Requirement: /req/create-replace-delete/procedure
// A.44 - Create
// See: https://docs.ogc.org/is/23-001/23-001.html#_req_create-replace-delete_procedure
func TestProcedureCRUD_Create(t *testing.T) {
	cleanupDB(t)

	formats := []struct {
		name        string
		contentType string
		makePayload func(suffix string) []byte
	}{
		{
			name:        "sml+json",
			contentType: "application/sml+json",
			makePayload: func(suffix string) []byte {
				payload := map[string]interface{}{
					"type":        "SimpleProcess",
					"uniqueId":    "urn:test:procedure:crud-" + suffix,
					"label":       "CRUD Test Procedure",
					"description": "Procedure for CRUD testing",
					"definition":  "http://www.w3.org/ns/sosa/Procedure",
				}
				b, _ := json.Marshal(payload)
				return b
			},
		},
		{
			name:        "geo+json",
			contentType: "application/geo+json",
			makePayload: func(suffix string) []byte {
				feat := map[string]interface{}{
					"type": "Feature",
					"properties": map[string]interface{}{
						"featureType": "http://www.w3.org/ns/sosa/Procedure",
						"uid":         "urn:test:procedure:crud-" + suffix,
						"name":        "CRUD Test Procedure (geo)",
						"description": "Procedure for CRUD testing (geo)",
					},
				}
				b, _ := json.Marshal(feat)
				return b
			},
		},
	}

	for i, f := range formats {
		t.Run(f.name, func(t *testing.T) {
			body := f.makePayload(fmt.Sprintf("%03d", i+1))
			req, err := http.NewRequest("POST", testServer.URL+"/procedures", bytes.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", f.contentType)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)
			location := resp.Header.Get("Location")
			assert.NotEmpty(t, location)
			id := parseID(location, "procedures")
			assert.NotEmpty(t, id)

			// Cleanup created resource
			reqDel, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/procedures/"+id, nil)
			http.DefaultClient.Do(reqDel)
		})
	}
}

// Conformance Class: /conf/create-replace-delete/procedure
// Requirement: /req/create-replace-delete/replace-procedure
// A.44 - Replace
// See: https://docs.ogc.org/is/23-001/23-001.html#_req_create-replace-delete_procedure
func TestProcedureCRUD_Replace(t *testing.T) {
	cleanupDB(t)

	formats := []struct {
		name              string
		contentType       string
		makeCreatePayload func(suffix string) []byte
		makeUpdatePayload func(suffix string) []byte
	}{
		{
			name:        "sml+json",
			contentType: "application/sml+json",
			makeCreatePayload: func(suffix string) []byte {
				payload := map[string]interface{}{
					"type":        "SimpleProcess",
					"uniqueId":    "urn:test:procedure:update-" + suffix,
					"label":       "Procedure to Update",
					"description": "Original description",
				}
				b, _ := json.Marshal(payload)
				return b
			},
			makeUpdatePayload: func(suffix string) []byte {
				payload := map[string]interface{}{
					"type":        "SimpleProcess",
					"uniqueId":    "urn:test:procedure:update-" + suffix,
					"label":       "Updated Procedure",
					"description": "Updated description",
				}
				b, _ := json.Marshal(payload)
				return b
			},
		},
		{
			name:        "geo+json",
			contentType: "application/geo+json",
			makeCreatePayload: func(suffix string) []byte {
				feat := map[string]interface{}{
					"type": "Feature",
					"properties": map[string]interface{}{
						"featureType": "http://www.w3.org/ns/sosa/Procedure",
						"uid":         "urn:test:procedure:update-" + suffix,
						"name":        "Procedure to Update (geo)",
						"description": "Original description",
					},
				}
				b, _ := json.Marshal(feat)
				return b
			},
			makeUpdatePayload: func(suffix string) []byte {
				feat := map[string]interface{}{
					"type": "Feature",
					"properties": map[string]interface{}{
						"featureType": "http://www.w3.org/ns/sosa/Procedure",
						"uid":         "urn:test:procedure:update-" + suffix,
						"name":        "Updated Procedure (geo)",
						"description": "Updated description",
					},
				}
				b, _ := json.Marshal(feat)
				return b
			},
		},
	}

	for i, f := range formats {
		t.Run(f.name, func(t *testing.T) {
			suffix := fmt.Sprintf("%03d", i+1)
			// create
			createBody := f.makeCreatePayload(suffix)
			req, _ := http.NewRequest("POST", testServer.URL+"/procedures", bytes.NewReader(createBody))
			req.Header.Set("Content-Type", f.contentType)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			location := resp.Header.Get("Location")
			resp.Body.Close()
			id := parseID(location, "procedures")

			// update
			updateBody := f.makeUpdatePayload(suffix)
			req, _ = http.NewRequest(http.MethodPut, testServer.URL+"/procedures/"+id, bytes.NewReader(updateBody))
			req.Header.Set("Content-Type", f.contentType)

			resp, err = http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)

			// Verify update (GET using Accept = content type being tested for consistency)
			req, _ = http.NewRequest("GET", location, nil)
			req.Header.Set("Accept", f.contentType)
			resp, _ = http.DefaultClient.Do(req)
			defer resp.Body.Close()

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)

			// label field is named differently for GeoJSON properties
			if f.contentType == "application/geo+json" {
				props, ok := result["properties"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "Updated Procedure (geo)", props["name"])
			} else {
				assert.Equal(t, "Updated Procedure", result["label"])
			}

			// cleanup
			reqDel, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/procedures/"+id, nil)
			http.DefaultClient.Do(reqDel)
		})
	}
}

// Conformance Class: /conf/create-replace-delete/procedure
// Requirement: /req/create-replace-delete/delete-procedure
// A.44 - Delete
// See: https://docs.ogc.org/is/23-001/23-001.html#_req_create-replace-delete_procedure
func TestProcedureCRUD_Delete(t *testing.T) {
	cleanupDB(t)

	// Delete behavior is content-type agnostic. Create a single procedure
	// (using SML payload) and verify delete/gone semantics.
	payload := map[string]interface{}{
		"type":     "SimpleProcess",
		"uniqueId": "urn:test:procedure:delete-001",
		"label":    "Procedure to Delete",
	}
	b, _ := json.Marshal(payload)

	// create
	req, _ := http.NewRequest("POST", testServer.URL+"/procedures", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/sml+json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	location := resp.Header.Get("Location")
	resp.Body.Close()
	id := parseID(location, "procedures")
	require.NotEmpty(t, id)

	// Delete
	req, _ = http.NewRequest(http.MethodDelete, testServer.URL+"/procedures/"+id, nil)
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify gone
	req, _ = http.NewRequest("GET", location, nil)
	resp, _ = http.DefaultClient.Do(req)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
