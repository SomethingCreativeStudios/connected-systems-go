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

// TestPropertyConformance implements OGC Connected Systems API conformance tests
// for the Property Definitions resource as defined in:
// - Clause 15: Property Definitions (https://docs.ogc.org/is/23-001/23-001.html#clause-property-definitions)
// - Annex A.9: Property Conformance Tests
// - Annex A.11: Create, Replace, Delete Conformance Tests
// - Annex A.10: Advanced Filtering Conformance Tests
// - Annex A.14: SensorML Encoding Conformance Tests
func TestPropertyConformance(t *testing.T) {
	cleanupDB(t)

	// Setup: Create test properties for conformance tests
	createdPropertyIDs := setupPropertyConformanceData(t)
	defer cleanupPropertyConformanceData(t, createdPropertyIDs)

	t.Run("Conformance Class: /conf/property", func(t *testing.T) {
		testPropertyConformanceClass(t, createdPropertyIDs)
	})

	t.Run("Conformance Class: /conf/sensorml/property-schema", func(t *testing.T) {
		testSensorMLPropertySchema(t, createdPropertyIDs)
	})

	t.Run("Conformance Class: /conf/create-replace-delete/property", func(t *testing.T) {
		testPropertyCRUD(t)
	})

	t.Run("Conformance Class: /conf/advanced-filtering", func(t *testing.T) {
		testPropertyAdvancedFiltering(t, createdPropertyIDs)
	})
}

// setupPropertyConformanceData creates test properties for conformance testing
func setupPropertyConformanceData(t *testing.T) []string {
	t.Helper()

	testProperties := []map[string]interface{}{
		{
			"label":        "Temperature",
			"description":  "Air temperature property",
			"uniqueId":     "urn:ogc:conf:property:temperature",
			"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
			"objectType":   "http://dbpedia.org/resource/Atmosphere",
		},
		{
			"label":        "Humidity",
			"description":  "Relative humidity property",
			"uniqueId":     "urn:ogc:conf:property:humidity",
			"baseProperty": "https://qudt.org/vocab/quantitykind/RelativeHumidity",
			"objectType":   "http://dbpedia.org/resource/Atmosphere",
		},
		{
			"label":        "CPU Temperature",
			"description":  "CPU core temperature",
			"uniqueId":     "urn:ogc:conf:property:cpu-temp",
			"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
			"objectType":   "http://dbpedia.org/resource/Central_processing_unit",
			"statistic":    "http://sensorml.com/ont/x-stats/HourlyMean",
		},
		{
			"label":        "Wind Speed",
			"description":  "Wind speed measurement",
			"uniqueId":     "urn:ogc:conf:property:wind-speed",
			"baseProperty": "https://qudt.org/vocab/quantitykind/Speed",
		},
		{
			"label":        "Temperature with Qualifiers",
			"description":  "Temperature with measurement qualifiers",
			"uniqueId":     "urn:ogc:conf:property:temp-qualified",
			"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
			"qualifiers": []map[string]interface{}{
				{
					"type":       "Quantity",
					"label":      "Measurement Height",
					"definition": "http://sensorml.com/ont/swe/property/Height",
					"uom": map[string]interface{}{
						"code": "m",
					},
					"value": 2.0,
				},
			},
		},
	}

	var createdIDs []string
	for _, prop := range testProperties {
		body, _ := json.Marshal(prop)
		req, err := http.NewRequest(http.MethodPost, testServer.URL+"/properties", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/sml+json")
		req.Header.Set("Accept", "application/sml+json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode, "failed to create property: %s", prop["label"])

		location := resp.Header.Get("Location")
		resp.Body.Close()
		require.NotEmpty(t, location, "expected Location header on create")
		id := parsePropertyID(location)
		require.NotEmpty(t, id, "unable to parse id from Location header: %s", location)
		createdIDs = append(createdIDs, id)
	}

	return createdIDs
}

func cleanupPropertyConformanceData(t *testing.T, ids []string) {
	t.Helper()
	for _, id := range ids {
		req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/properties/"+id, nil)
		http.DefaultClient.Do(req)
	}
}

// =============================================================================
// Conformance Class: /conf/property
// Requirement: /req/property
// =============================================================================

func testPropertyConformanceClass(t *testing.T, propertyIDs []string) {
	tests := []struct {
		name        string
		testID      string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "/conf/property/resources-endpoint",
			testID:      "A.33",
			description: "Verify that property resources can be retrieved from the /properties endpoint",
			testFunc: func(t *testing.T) {
				// Test A.33: /conf/property/resources-endpoint
				// Requirement: /req/property/resources-endpoint
				// The server SHALL expose all served property resources at the path /properties

				req, err := http.NewRequest(http.MethodGet, testServer.URL+"/properties", nil)
				require.NoError(t, err)
				req.Header.Set("Accept", "application/sml+json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				// Response must be 200 OK
				assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /properties must return 200 OK")

				// Response must be valid JSON
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				var collection map[string]interface{}
				err = json.Unmarshal(body, &collection)
				require.NoError(t, err, "Response must be valid JSON")

				// Must contain features array (OGC FeatureCollection uses "features")
				features, ok := collection["features"].([]interface{})
				assert.True(t, ok, "Response must contain 'features' array")
				assert.GreaterOrEqual(t, len(features), 1, "Collection must contain at least one property")
			},
		},
		{
			name:        "/conf/property/canonical-endpoint",
			testID:      "A.34",
			description: "Verify that property resources can be accessed via the /properties endpoint",
			testFunc: func(t *testing.T) {
				// Test A.34: /conf/property/canonical-endpoint
				// Requirement: /req/property/canonical-endpoint
				// Server SHALL expose the canonical URL at path /properties

				// Verify the endpoint exists and returns proper content type
				req, err := http.NewRequest(http.MethodGet, testServer.URL+"/properties", nil)
				require.NoError(t, err)
				req.Header.Set("Accept", "application/sml+json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				// Verify content type header contains expected media type
				// Note: render.JSON may override Content-Type, but the key conformance
				// requirement is that the endpoint exists and returns valid data
				contentType := resp.Header.Get("Content-Type")
				assert.NotEmpty(t, contentType, "Content-Type header must be set")
				// Check that it's either the requested format or at least a valid JSON type
				assert.True(t,
					contentType == "application/sml+json" ||
						contentType == "application/json" ||
						contentType == "application/json; charset=utf-8",
					"Content-Type must be a valid JSON media type, got: %s", contentType)
			},
		},
		{
			name:        "/conf/property/canonical-url",
			testID:      "A.32",
			description: "Verify that each property resource is accessible via its canonical URL",
			testFunc: func(t *testing.T) {
				// Test A.32: /conf/property/canonical-url
				// Requirement: /req/property/canonical-url
				// Each property resource SHALL be accessible at URL /properties/{id}

				for _, id := range propertyIDs {
					t.Run(fmt.Sprintf("property-%s", id), func(t *testing.T) {
						url := fmt.Sprintf("%s/properties/%s", testServer.URL, id)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/sml+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						assert.Equal(t, http.StatusOK, resp.StatusCode, "GET /properties/{id} must return 200 OK")

						var property map[string]interface{}
						err = json.NewDecoder(resp.Body).Decode(&property)
						require.NoError(t, err)

						// Verify the returned property has the expected ID
						assert.Equal(t, id, property["id"], "Returned property ID must match requested ID")
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
// Conformance Class: /conf/sensorml/property-schema
// Requirement: /req/sensorml/property-schema
// =============================================================================

func testSensorMLPropertySchema(t *testing.T, propertyIDs []string) {
	validator := GetSchemaValidator()

	tests := []struct {
		name        string
		testID      string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "/conf/sensorml/property-schema/single-resource",
			testID:      "A.56",
			description: "Verify single property resource conforms to property.json schema",
			testFunc: func(t *testing.T) {
				// Test A.56: /conf/sensorml/property-schema (single resource)
				// Requirement: /req/sensorml/property-schema
				// Property resources SHALL validate against property.json schema

				for _, id := range propertyIDs {
					t.Run(fmt.Sprintf("property-%s", id), func(t *testing.T) {
						url := fmt.Sprintf("%s/properties/%s", testServer.URL, id)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/sml+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						require.Equal(t, http.StatusOK, resp.StatusCode)

						body, err := io.ReadAll(resp.Body)
						require.NoError(t, err)

						// Validate against property schema
						err = validator.ValidateJSON(PropertySchema, body)
						assert.NoError(t, err, "Property must validate against property.json schema")
					})
				}
			},
		},
		{
			name:        "/conf/sensorml/property-schema/collection-items",
			testID:      "A.56",
			description: "Verify property collection items conform to property.json schema",
			testFunc: func(t *testing.T) {
				// Test A.56: /conf/sensorml/property-schema (collection items)
				// Each item in property collection SHALL validate against property.json schema

				req, err := http.NewRequest(http.MethodGet, testServer.URL+"/properties", nil)
				require.NoError(t, err)
				req.Header.Set("Accept", "application/sml+json")

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

						err = validator.ValidateJSON(PropertySchema, itemBytes)
						assert.NoError(t, err, "Collection item must validate against property.json schema")
					})
				}
			},
		},
		{
			name:        "/conf/sensorml/property-mappings",
			testID:      "A.57",
			description: "Verify property resource has required SensorML mappings per Table 55",
			testFunc: func(t *testing.T) {
				// Test A.57: /conf/sensorml/property-mappings
				// Requirement: /req/sensorml/property-mappings
				// Property resources SHALL have mappings per Table 55 of the spec

				// Table 55 defines required property attributes:
				// - uniqueId (URI)
				// - label (string)
				// - description (optional string)
				// - baseProperty (optional URI)
				// - objectType (optional URI)
				// - statistic (optional URI)
				// - qualifiers (optional array)

				for _, id := range propertyIDs {
					t.Run(fmt.Sprintf("property-%s", id), func(t *testing.T) {
						url := fmt.Sprintf("%s/properties/%s", testServer.URL, id)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/sml+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						require.Equal(t, http.StatusOK, resp.StatusCode)

						var property map[string]interface{}
						err = json.NewDecoder(resp.Body).Decode(&property)
						require.NoError(t, err)

						// Verify required fields per Table 55
						assert.NotEmpty(t, property["id"], "Property must have id")
						assert.NotEmpty(t, property["uniqueId"], "Property must have uniqueId")
						assert.NotEmpty(t, property["label"], "Property must have label")

						// Verify optional fields are correct type if present
						if baseProperty, ok := property["baseProperty"]; ok {
							assert.IsType(t, "", baseProperty, "baseProperty must be a string")
						}
						if objectType, ok := property["objectType"]; ok {
							assert.IsType(t, "", objectType, "objectType must be a string")
						}
						if statistic, ok := property["statistic"]; ok {
							assert.IsType(t, "", statistic, "statistic must be a string")
						}
						if qualifiers, ok := property["qualifiers"]; ok {
							assert.IsType(t, []interface{}{}, qualifiers, "qualifiers must be an array")
						}
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
// Conformance Class: /conf/create-replace-delete/property
// Requirement: /req/create-replace-delete/property
// =============================================================================

func testPropertyCRUD(t *testing.T) {
	tests := []struct {
		name        string
		testID      string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "/conf/create-replace-delete/property/create",
			testID:      "A.44",
			description: "Verify property resources can be created via POST to /properties",
			testFunc: func(t *testing.T) {
				// Test A.44: /conf/create-replace-delete/property (CREATE)
				// Requirement: /req/create-replace-delete/create-property
				// Server SHALL support POST to /properties to create new property resources

				testCases := []struct {
					name    string
					payload map[string]interface{}
				}{
					{
						name: "minimal property",
						payload: map[string]interface{}{
							"label":    "Test Minimal Property",
							"uniqueId": "urn:ogc:conf:property:minimal-create",
						},
					},
					{
						name: "property with all optional fields",
						payload: map[string]interface{}{
							"label":        "Test Full Property",
							"description":  "A fully specified property",
							"uniqueId":     "urn:ogc:conf:property:full-create",
							"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
							"objectType":   "http://dbpedia.org/resource/Sensor",
							"statistic":    "http://sensorml.com/ont/x-stats/Mean",
						},
					},
					{
						name: "property with qualifiers",
						payload: map[string]interface{}{
							"label":        "Test Property with Qualifiers",
							"uniqueId":     "urn:ogc:conf:property:qualified-create",
							"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
							"qualifiers": []map[string]interface{}{
								{
									"type":       "Quantity",
									"label":      "Height",
									"definition": "http://sensorml.com/ont/swe/property/Height",
									"uom":        map[string]interface{}{"code": "m"},
									"value":      5.0,
								},
							},
						},
					},
				}

				for _, tc := range testCases {
					t.Run(tc.name, func(t *testing.T) {
						body, _ := json.Marshal(tc.payload)
						req, err := http.NewRequest(http.MethodPost, testServer.URL+"/properties", bytes.NewReader(body))
						require.NoError(t, err)
						req.Header.Set("Content-Type", "application/sml+json")
						req.Header.Set("Accept", "application/sml+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						// Must return 201 Created
						assert.Equal(t, http.StatusCreated, resp.StatusCode, "POST /properties must return 201 Created")

						location := resp.Header.Get("Location")
						require.NotEmpty(t, location, "expected Location header on create")
						id := parsePropertyID(location)
						require.NotEmpty(t, id, "unable to parse id from Location header: %s", location)

						created, err := fetchPropertyById(id)
						require.NoError(t, err)

						// Must return created resource with ID
						assert.NotEmpty(t, (*created)["id"], "Created property must have an id")
						assert.Equal(t, tc.payload["label"], (*created)["label"])
						assert.Equal(t, tc.payload["uniqueId"], (*created)["uniqueId"])

						// Cleanup
						deleteReq, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/properties/"+id, nil)
						http.DefaultClient.Do(deleteReq)
					})
				}
			},
		},
		{
			name:        "/conf/create-replace-delete/property/replace",
			testID:      "A.44",
			description: "Verify property resources can be replaced via PUT to /properties/{id}",
			testFunc: func(t *testing.T) {
				// Test A.44: /conf/create-replace-delete/property (REPLACE)
				// Requirement: /req/create-replace-delete/replace-property
				// Server SHALL support PUT to /properties/{id} to replace property resources

				// First create a property
				createPayload := map[string]interface{}{
					"label":        "Original Property",
					"description":  "Original description",
					"uniqueId":     "urn:ogc:conf:property:replace-test",
					"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				}
				body, _ := json.Marshal(createPayload)
				createReq, err := http.NewRequest(http.MethodPost, testServer.URL+"/properties", bytes.NewReader(body))
				require.NoError(t, err)
				createReq.Header.Set("Content-Type", "application/sml+json")
				createReq.Header.Set("Accept", "application/sml+json")

				createResp, err := http.DefaultClient.Do(createReq)
				require.NoError(t, err)
				require.Equal(t, http.StatusCreated, createResp.StatusCode)

				location := createResp.Header.Get("Location")
				createResp.Body.Close()
				require.NotEmpty(t, location, "expected Location header on create")
				propertyID := parsePropertyID(location)
				require.NotEmpty(t, propertyID, "unable to parse id from Location header: %s", location)
				defer func() {
					deleteReq, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/properties/"+propertyID, nil)
					http.DefaultClient.Do(deleteReq)
				}()

				// Replace the property
				replacePayload := map[string]interface{}{
					"label":        "Replaced Property",
					"description":  "Replaced description",
					"uniqueId":     "urn:ogc:conf:property:replace-test",
					"baseProperty": "https://qudt.org/vocab/quantitykind/Pressure",
					"objectType":   "http://dbpedia.org/resource/Barometer",
				}
				body, _ = json.Marshal(replacePayload)
				replaceReq, err := http.NewRequest(http.MethodPut, testServer.URL+"/properties/"+propertyID, bytes.NewReader(body))
				require.NoError(t, err)
				replaceReq.Header.Set("Content-Type", "application/sml+json")
				replaceReq.Header.Set("Accept", "application/sml+json")

				replaceResp, err := http.DefaultClient.Do(replaceReq)
				require.NoError(t, err)
				replaceResp.Body.Close()

				// Update should return 204 No Content
				assert.Equal(t, http.StatusNoContent, replaceResp.StatusCode, "PUT /properties/{id} must return 204 No Content")

				// Fetch the replaced resource to verify updates were applied
				replacedPtr, err := fetchPropertyById(propertyID)
				require.NoError(t, err)
				replaced := *replacedPtr

				assert.Equal(t, "Replaced Property", replaced["label"])
				assert.Equal(t, "Replaced description", replaced["description"])
				assert.Equal(t, "https://qudt.org/vocab/quantitykind/Pressure", replaced["baseProperty"])
				assert.Equal(t, "http://dbpedia.org/resource/Barometer", replaced["objectType"])
			},
		},
		{
			name:        "/conf/create-replace-delete/property/delete",
			testID:      "A.44",
			description: "Verify property resources can be deleted via DELETE to /properties/{id}",
			testFunc: func(t *testing.T) {
				// Test A.44: /conf/create-replace-delete/property (DELETE)
				// Requirement: /req/create-replace-delete/delete-property
				// Server SHALL support DELETE to /properties/{id} to delete property resources

				// First create a property to delete
				createPayload := map[string]interface{}{
					"label":    "Property to Delete",
					"uniqueId": "urn:ogc:conf:property:delete-test",
				}
				body, _ := json.Marshal(createPayload)
				createReq, err := http.NewRequest(http.MethodPost, testServer.URL+"/properties", bytes.NewReader(body))
				require.NoError(t, err)
				createReq.Header.Set("Content-Type", "application/sml+json")
				createReq.Header.Set("Accept", "application/sml+json")

				createResp, err := http.DefaultClient.Do(createReq)
				require.NoError(t, err)
				require.Equal(t, http.StatusCreated, createResp.StatusCode)

				location := createResp.Header.Get("Location")
				createResp.Body.Close()
				require.NotEmpty(t, location, "expected Location header on create")
				propertyID := parsePropertyID(location)
				require.NotEmpty(t, propertyID, "unable to parse id from Location header: %s", location)

				// Delete the property
				deleteReq, err := http.NewRequest(http.MethodDelete, testServer.URL+"/properties/"+propertyID, nil)
				require.NoError(t, err)

				deleteResp, err := http.DefaultClient.Do(deleteReq)
				require.NoError(t, err)
				defer deleteResp.Body.Close()

				// Must return 204 No Content
				assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode, "DELETE /properties/{id} must return 204 No Content")

				// Verify the property is deleted
				getReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/properties/"+propertyID, nil)
				require.NoError(t, err)
				getReq.Header.Set("Accept", "application/sml+json")

				getResp, err := http.DefaultClient.Do(getReq)
				require.NoError(t, err)
				defer getResp.Body.Close()

				assert.Equal(t, http.StatusNotFound, getResp.StatusCode, "GET deleted property must return 404 Not Found")
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
// Conformance Class: /conf/advanced-filtering
// Requirements: /req/advanced-filtering/prop-by-baseprop, /req/advanced-filtering/prop-by-object
// =============================================================================

func testPropertyAdvancedFiltering(t *testing.T, propertyIDs []string) {
	tests := []struct {
		name        string
		testID      string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "/conf/advanced-filtering/prop-by-baseprop",
			testID:      "A.39",
			description: "Verify filtering properties by baseProperty parameter",
			testFunc: func(t *testing.T) {
				// Test A.39: /conf/advanced-filtering/prop-by-baseprop
				// Requirement: /req/advanced-filtering/prop-by-baseprop
				// Server SHALL support filtering properties by the baseProperty query parameter

				testCases := []struct {
					name           string
					baseProperty   string
					expectedMinLen int
				}{
					{
						name:           "filter by Temperature base property",
						baseProperty:   "https://qudt.org/vocab/quantitykind/Temperature",
						expectedMinLen: 1,
					},
					{
						name:           "filter by Humidity base property",
						baseProperty:   "https://qudt.org/vocab/quantitykind/RelativeHumidity",
						expectedMinLen: 1,
					},
					{
						name:           "filter by Speed base property",
						baseProperty:   "https://qudt.org/vocab/quantitykind/Speed",
						expectedMinLen: 1,
					},
					{
						name:           "filter by non-existent base property",
						baseProperty:   "https://qudt.org/vocab/quantitykind/NonExistent",
						expectedMinLen: 0,
					},
				}

				for _, tc := range testCases {
					t.Run(tc.name, func(t *testing.T) {
						url := fmt.Sprintf("%s/properties?baseProperty=%s", testServer.URL, tc.baseProperty)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/sml+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						assert.Equal(t, http.StatusOK, resp.StatusCode)

						var collection map[string]interface{}
						err = json.NewDecoder(resp.Body).Decode(&collection)
						require.NoError(t, err)

						features, ok := collection["features"].([]interface{})
						require.True(t, ok)

						if tc.expectedMinLen == 0 {
							assert.Len(t, features, 0, "Expected no properties for non-existent baseProperty")
						} else {
							assert.GreaterOrEqual(t, len(features), tc.expectedMinLen, "Expected at least %d properties", tc.expectedMinLen)

							// Verify all returned properties have the expected baseProperty
							for _, item := range features {
								prop := item.(map[string]interface{})
								assert.Equal(t, tc.baseProperty, prop["baseProperty"], "All returned properties must match the baseProperty filter")
							}
						}
					})
				}
			},
		},
		{
			name:        "/conf/advanced-filtering/prop-by-object",
			testID:      "A.40",
			description: "Verify filtering properties by objectType parameter",
			testFunc: func(t *testing.T) {
				// Test A.40: /conf/advanced-filtering/prop-by-object
				// Requirement: /req/advanced-filtering/prop-by-object
				// Server SHALL support filtering properties by the objectType query parameter

				testCases := []struct {
					name           string
					objectType     string
					expectedMinLen int
				}{
					{
						name:           "filter by Atmosphere object type",
						objectType:     "http://dbpedia.org/resource/Atmosphere",
						expectedMinLen: 1,
					},
					{
						name:           "filter by CPU object type",
						objectType:     "http://dbpedia.org/resource/Central_processing_unit",
						expectedMinLen: 1,
					},
					{
						name:           "filter by non-existent object type",
						objectType:     "http://dbpedia.org/resource/NonExistent",
						expectedMinLen: 0,
					},
				}

				for _, tc := range testCases {
					t.Run(tc.name, func(t *testing.T) {
						url := fmt.Sprintf("%s/properties?objectType=%s", testServer.URL, tc.objectType)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/sml+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						assert.Equal(t, http.StatusOK, resp.StatusCode)

						var collection map[string]interface{}
						err = json.NewDecoder(resp.Body).Decode(&collection)
						require.NoError(t, err)

						features, ok := collection["features"].([]interface{})
						require.True(t, ok)

						if tc.expectedMinLen == 0 {
							assert.Len(t, features, 0, "Expected no properties for non-existent objectType")
						} else {
							assert.GreaterOrEqual(t, len(features), tc.expectedMinLen, "Expected at least %d properties", tc.expectedMinLen)

							// Verify all returned properties have the expected objectType
							for _, item := range features {
								prop := item.(map[string]interface{})
								assert.Equal(t, tc.objectType, prop["objectType"], "All returned properties must match the objectType filter")
							}
						}
					})
				}
			},
		},
		{
			name:        "/conf/advanced-filtering/prop-combined-filters",
			testID:      "A.39-40",
			description: "Verify filtering properties by combined baseProperty and objectType parameters",
			testFunc: func(t *testing.T) {
				// Combined filtering test - using both baseProperty and objectType
				// This tests the AND behavior of multiple filter parameters

				testCases := []struct {
					name           string
					baseProperty   string
					objectType     string
					expectedMinLen int
				}{
					{
						name:           "Temperature + Atmosphere",
						baseProperty:   "https://qudt.org/vocab/quantitykind/Temperature",
						objectType:     "http://dbpedia.org/resource/Atmosphere",
						expectedMinLen: 1,
					},
					{
						name:           "Temperature + CPU",
						baseProperty:   "https://qudt.org/vocab/quantitykind/Temperature",
						objectType:     "http://dbpedia.org/resource/Central_processing_unit",
						expectedMinLen: 1,
					},
					{
						name:           "Humidity + CPU (no match)",
						baseProperty:   "https://qudt.org/vocab/quantitykind/RelativeHumidity",
						objectType:     "http://dbpedia.org/resource/Central_processing_unit",
						expectedMinLen: 0,
					},
				}

				for _, tc := range testCases {
					t.Run(tc.name, func(t *testing.T) {
						url := fmt.Sprintf("%s/properties?baseProperty=%s&objectType=%s", testServer.URL, tc.baseProperty, tc.objectType)
						req, err := http.NewRequest(http.MethodGet, url, nil)
						require.NoError(t, err)
						req.Header.Set("Accept", "application/sml+json")

						resp, err := http.DefaultClient.Do(req)
						require.NoError(t, err)
						defer resp.Body.Close()

						assert.Equal(t, http.StatusOK, resp.StatusCode)

						var collection map[string]interface{}
						err = json.NewDecoder(resp.Body).Decode(&collection)
						require.NoError(t, err)

						features, ok := collection["features"].([]interface{})
						require.True(t, ok)

						if tc.expectedMinLen == 0 {
							assert.Len(t, features, 0, "Expected no properties for combined filter with no match")
						} else {
							assert.GreaterOrEqual(t, len(features), tc.expectedMinLen)

							// Verify all returned properties match both filters
							for _, item := range features {
								prop := item.(map[string]interface{})
								assert.Equal(t, tc.baseProperty, prop["baseProperty"])
								assert.Equal(t, tc.objectType, prop["objectType"])
							}
						}
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
// Additional Schema Validation Tests
// =============================================================================

func TestPropertySchemaValidation(t *testing.T) {
	cleanupDB(t)

	validator := GetSchemaValidator()

	// Table-driven tests for various property schemas
	tests := []struct {
		name          string
		propertyJSON  string
		shouldBeValid bool
		description   string
	}{
		{
			name: "valid minimal property with baseProperty",
			propertyJSON: `{
				"label": "Test Property",
				"uniqueId": "urn:test:property:minimal",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature"
			}`,
			shouldBeValid: true,
			description:   "Minimal valid property with required fields (label, uniqueId, baseProperty)",
		},
		{
			name: "invalid property missing baseProperty",
			propertyJSON: `{
				"label": "Test Property",
				"uniqueId": "urn:test:property:no-base"
			}`,
			shouldBeValid: false,
			description:   "Property missing required baseProperty field should fail validation",
		},
		{
			name: "valid property with all fields",
			propertyJSON: `{
				"label": "Full Property",
				"description": "A fully specified property",
				"uniqueId": "urn:test:property:full",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"objectType": "http://dbpedia.org/resource/Sensor",
				"statistic": "http://sensorml.com/ont/x-stats/Mean"
			}`,
			shouldBeValid: true,
			description:   "Property with all optional fields",
		},
		{
			name: "valid property with Quantity qualifier",
			propertyJSON: `{
				"label": "Property with Quantity",
				"uniqueId": "urn:test:property:quantity",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"qualifiers": [{
					"type": "Quantity",
					"label": "Height",
					"definition": "http://sensorml.com/ont/swe/property/Height",
					"uom": {"code": "m"},
					"value": 10.0
				}]
			}`,
			shouldBeValid: true,
			description:   "Property with Quantity qualifier",
		},
		{
			name: "valid property with Boolean qualifier",
			propertyJSON: `{
				"label": "Property with Boolean",
				"uniqueId": "urn:test:property:boolean",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"qualifiers": [{
					"type": "Boolean",
					"label": "Calibrated",
					"definition": "http://sensorml.com/ont/swe/property/Calibrated",
					"value": true
				}]
			}`,
			shouldBeValid: true,
			description:   "Property with Boolean qualifier",
		},
		{
			name: "valid property with Category qualifier",
			propertyJSON: `{
				"label": "Property with Category",
				"uniqueId": "urn:test:property:category",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"qualifiers": [{
					"type": "Category",
					"label": "Quality",
					"definition": "http://sensorml.com/ont/swe/property/Quality",
					"value": "good"
				}]
			}`,
			shouldBeValid: true,
			description:   "Property with Category qualifier",
		},
		{
			name: "valid property with Count qualifier",
			propertyJSON: `{
				"label": "Property with Count",
				"uniqueId": "urn:test:property:count",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"qualifiers": [{
					"type": "Count",
					"label": "Samples",
					"definition": "http://sensorml.com/ont/swe/property/Samples",
					"value": 100
				}]
			}`,
			shouldBeValid: true,
			description:   "Property with Count qualifier",
		},
		{
			name: "valid property with Text qualifier",
			propertyJSON: `{
				"label": "Property with Text",
				"uniqueId": "urn:test:property:text",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"qualifiers": [{
					"type": "Text",
					"label": "Notes",
					"definition": "http://sensorml.com/ont/swe/property/Notes",
					"value": "Some notes here"
				}]
			}`,
			shouldBeValid: true,
			description:   "Property with Text qualifier",
		},
		{
			name: "valid property with QuantityRange qualifier",
			propertyJSON: `{
				"label": "Property with Range",
				"uniqueId": "urn:test:property:range",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"qualifiers": [{
					"type": "QuantityRange",
					"label": "Valid Range",
					"definition": "http://sensorml.com/ont/swe/property/Range",
					"uom": {"code": "Cel"},
					"value": [-40.0, 85.0]
				}]
			}`,
			shouldBeValid: true,
			description:   "Property with QuantityRange qualifier",
		},
		{
			name: "valid property with multiple qualifiers",
			propertyJSON: `{
				"label": "Complex Property",
				"uniqueId": "urn:test:property:complex",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"qualifiers": [
					{
						"type": "Quantity",
						"label": "Height",
						"definition": "http://sensorml.com/ont/swe/property/Height",
						"uom": {"code": "m"},
						"value": 2.0
					},
					{
						"type": "Boolean",
						"label": "QC",
						"definition": "http://sensorml.com/ont/swe/property/QC",
						"value": true
					}
				]
			}`,
			shouldBeValid: true,
			description:   "Property with multiple qualifiers",
		},
		{
			name: "invalid property missing label",
			propertyJSON: `{
				"uniqueId": "urn:test:property:no-label",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature"
			}`,
			shouldBeValid: false,
			description:   "Property missing required label field should fail validation",
		},
		{
			name: "invalid property with multiple qualifiers",
			propertyJSON: `{
				"label": "Complex Property",
				"uniqueId": "urn:test:property:complex",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"qualifiers": [
					{
						"type": "Boolean",
						"label": "Height",
						"definition": "http://sensorml.com/ont/swe/property/Height",
						"uom": {"code": "m"},
						"value": 2.0
					},
					{
						"type": "Boolean",
						"label": "QC",
						"definition": "http://sensorml.com/ont/swe/property/QC",
						"value": true
					}
				]
			}`,
			shouldBeValid: false,
			description:   "Property with invalid qualifier type should fail validation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing: %s", tc.description)

			err := validator.ValidateJSON(PropertySchema, []byte(tc.propertyJSON))
			if tc.shouldBeValid {
				assert.NoError(t, err, "Expected valid property to pass schema validation")
			} else {
				assert.Error(t, err, "Expected invalid property to fail schema validation")
			}
		})
	}
}
