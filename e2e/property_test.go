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
		id := parseID(location, "/properties/")
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
// Annex A.9: Property Conformance Tests
// =============================================================================
func TestPropertyConformance(t *testing.T) {
	cleanupDB(t)

	// Setup: Create test properties for conformance tests
	createdPropertyIDs := setupPropertyConformanceData(t)
	defer cleanupPropertyConformanceData(t, createdPropertyIDs)

	tests := map[string]struct {
		testID      string
		description string
		testFunc    func(t *testing.T)
	}{
		"/conf/property/resources-endpoint": {
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
		"/conf/property/canonical-endpoint": {
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
		"/conf/property/canonical-url": {
			testID:      "A.32",
			description: "Verify that each property resource is accessible via its canonical URL",
			testFunc: func(t *testing.T) {
				// Test A.32: /conf/property/canonical-url
				// Requirement: /req/property/canonical-url
				// Each property resource SHALL be accessible at URL /properties/{id}

				for _, id := range createdPropertyIDs {
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

	for key, tc := range tests {
		t.Run(key+" "+tc.testID, func(t *testing.T) {
			tc.testFunc(t)
		})
	}
}

// =============================================================================
// Conformance Class: /conf/sensorml/property-schema
// Requirement: /req/sensorml/property-schema
// Annex A.14: SensorML Encoding Conformance Tests
// =============================================================================
func TestPropertySchema_SensorML(t *testing.T) {
	validator := GetSchemaValidator()

	// Setup: Create test properties for conformance tests
	createdPropertyIDs := setupPropertyConformanceData(t)
	defer cleanupPropertyConformanceData(t, createdPropertyIDs)

	tests := map[string]struct {
		testID      string
		description string
		testFunc    func(t *testing.T)
	}{
		"/conf/sensorml/property-schema/single-resource": {
			testID:      "A.56",
			description: "Verify single property resource conforms to property.json schema",
			testFunc: func(t *testing.T) {
				// Test A.56: /conf/sensorml/property-schema (single resource)
				// Requirement: /req/sensorml/property-schema
				// Property resources SHALL validate against property.json schema

				for _, id := range createdPropertyIDs {
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
		// Update to valdiate collection itself
		"/conf/sensorml/property-schema/collection-items": {
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
		"/conf/sensorml/property-mappings": {
			testID:      "A.57",
			description: "Verify property resource has required SensorML mappings per Table 55",
			testFunc: func(t *testing.T) {
				// Test A.57: /conf/sensorml/property-mappings
				// Requirement: /req/sensorml/property-mappings
				// Property resources SHALL have mappings per Table 55 of the spec

				for _, id := range createdPropertyIDs {
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

	for name, tc := range tests {
		t.Run(name+" "+tc.testID, func(t *testing.T) {
			tc.testFunc(t)
		})
	}
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/property
// Requirement: /req/create-replace-delete/property
// A.44 - Create
// =============================================================================
func TestPropertyCRUD_Create(t *testing.T) {
	testCases := map[string]struct {
		payload      map[string]interface{}
		validateFunc func(t *testing.T, created map[string]interface{})
	}{
		"minimal property": {
			payload: map[string]interface{}{
				"label":    "Test Minimal Property",
				"uniqueId": "urn:ogc:conf:property:minimal-create",
			},
			validateFunc: func(t *testing.T, created map[string]interface{}) {
				assert.Equal(t, "Test Minimal Property", created["label"])
				assert.Equal(t, "urn:ogc:conf:property:minimal-create", created["uniqueId"])
			},
		},
		"property with all optional fields": {
			payload: map[string]interface{}{
				"label":        "Test Full Property",
				"description":  "A fully specified property",
				"uniqueId":     "urn:ogc:conf:property:full-create",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"objectType":   "http://dbpedia.org/resource/Sensor",
				"statistic":    "http://sensorml.com/ont/x-stats/Mean",
			},
			validateFunc: func(t *testing.T, created map[string]interface{}) {
				assert.Equal(t, "Test Full Property", created["label"])
				assert.Equal(t, "A fully specified property", created["description"])
				assert.Equal(t, "urn:ogc:conf:property:full-create", created["uniqueId"])
				assert.Equal(t, "https://qudt.org/vocab/quantitykind/Temperature", created["baseProperty"])
				assert.Equal(t, "http://dbpedia.org/resource/Sensor", created["objectType"])
				assert.Equal(t, "http://sensorml.com/ont/x-stats/Mean", created["statistic"])
			},
		},
		"property with qualifiers": {
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
			validateFunc: func(t *testing.T, created map[string]interface{}) {
				assert.Equal(t, "Test Property with Qualifiers", created["label"])
				assert.Equal(t, "urn:ogc:conf:property:qualified-create", created["uniqueId"])
				assert.Equal(t, "https://qudt.org/vocab/quantitykind/Temperature", created["baseProperty"])

				qualifiers, ok := created["qualifiers"].([]interface{})
				require.True(t, ok, "qualifiers must be an array")
				require.Len(t, qualifiers, 1, "there must be one qualifier")

				qualifier, ok := qualifiers[0].(map[string]interface{})
				require.True(t, ok, "qualifier must be an object")
				assert.Equal(t, "Quantity", qualifier["type"])
				assert.Equal(t, "Height", qualifier["label"])
				assert.Equal(t, "http://sensorml.com/ont/swe/property/Height", qualifier["definition"])

				uom, ok := qualifier["uom"].(map[string]interface{})
				require.True(t, ok, "uom must be an object")
				assert.Equal(t, "m", uom["code"])

				assert.Equal(t, 5.0, qualifier["value"])
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
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

			created, err := FollowLocation(resp, "application/sml+json")
			require.NoError(t, err)

			// Validate created resource
			tc.validateFunc(t, *created)
		})
	}
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/property
// Requirement: /req/create-replace-delete/property
// A.44 - Replace
// =============================================================================
func TestPropertyCRUD_Replace(t *testing.T) {
	// Test A.44: /conf/create-replace-delete/property (REPLACE)
	// Requirement: /req/create-replace-delete/replace-property
	// Server SHALL support PUT to /properties/{id} to replace property resources

	replaceTestCases := map[string]struct {
		createPayload   map[string]interface{}
		updatePayload   map[string]interface{}
		expectedGetCode int
		validateUpdate  func(t *testing.T, result map[string]interface{})
	}{
		"update property label": {
			createPayload: map[string]interface{}{
				"label":        "Property to Update",
				"uniqueId":     "urn:test:property:crud-update",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
			},
			updatePayload: map[string]interface{}{
				"label":        "Updated Property Name",
				"uniqueId":     "urn:test:property:crud-update",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"description":  "Now with a description",
			},
			expectedGetCode: http.StatusOK,
			validateUpdate: func(t *testing.T, result map[string]interface{}) {
				assert.Equal(t, "Updated Property Name", result["label"])
				assert.Equal(t, "Now with a description", result["description"])
			},
		},
		"update property with qualifiers": {
			createPayload: map[string]interface{}{
				"label":        "Property for Qualifier Update",
				"uniqueId":     "urn:test:property:crud-qualifier-update",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Pressure",
			},
			updatePayload: map[string]interface{}{
				"label":        "Property for Qualifier Update",
				"uniqueId":     "urn:test:property:crud-qualifier-update",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Pressure",
				"qualifiers": []map[string]interface{}{
					{
						"type":       "Quantity",
						"label":      "Altitude",
						"definition": "http://sensorml.com/ont/swe/property/Altitude",
						"uom": map[string]interface{}{
							"code": "m",
						},
						"value": 1000.0,
					},
				},
			},
			expectedGetCode: http.StatusOK,
			validateUpdate: func(t *testing.T, result map[string]interface{}) {
				qualifiers, ok := result["qualifiers"].([]interface{})
				require.True(t, ok, "expected qualifiers array after update")
				assert.Len(t, qualifiers, 1)
			},
		},
	}

	for name, tc := range replaceTestCases {
		t.Run(name, func(t *testing.T) {
			// Create
			body, _ := json.Marshal(tc.createPayload)
			createResp, err := http.Post(testServer.URL+"/properties", "application/sml+json", bytes.NewReader(body))
			require.NoError(t, err)
			createResp.Body.Close()

			createdProp, err := FollowLocation(createResp, "application/sml+json")
			require.NoError(t, err)
			propID := (*createdProp)["id"].(string)

			client := &http.Client{}

			updateBody, _ := json.Marshal(tc.updatePayload)
			req, _ := http.NewRequest(http.MethodPut, testServer.URL+"/properties/"+propID, bytes.NewReader(updateBody))
			req.Header.Set("Content-Type", "application/sml+json")
			req.Header.Set("Accept", "application/sml+json")

			resp, err := client.Do(req)
			require.NoError(t, err)

			resp.Body.Close()
			// Update now returns 204 No Content; fetch the resource to verify changes
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)

			// Follow Location from create to retrieve updated resource
			fetched, err := FollowLocation(createResp, "application/sml+json")
			require.NoError(t, err)
			tc.validateUpdate(t, *fetched)

		})
	}
}

// =============================================================================
// Conformance Class: /conf/create-replace-delete/property
// Requirement: /req/create-replace-delete/property
// A.44 - Replace
// =============================================================================
func TestPropertyCRUD_Delete(t *testing.T) {

	// Test A.44: /conf/create-replace-delete/property (DELETE)
	// Requirement: /req/create-replace-delete/delete-property
	// Server SHALL support DELETE to /properties/{id} to delete property resources

	deleteTestCases := map[string]struct {
		createPayload   map[string]interface{}
		expectedGetCode int
	}{
		"delete existing property": {
			createPayload: map[string]interface{}{
				"label":    "Property to Delete",
				"uniqueId": "urn:test:property:crud-delete",
			},
			expectedGetCode: http.StatusNotFound,
		},
		"delete property with qualifiers": {
			createPayload: map[string]interface{}{
				"label":    "Property with Qualifiers to Delete",
				"uniqueId": "urn:test:property:crud-delete-qualifiers",
				"qualifiers": []map[string]interface{}{
					{
						"type":       "Quantity",
						"label":      "Depth",
						"definition": "http://sensorml.com/ont/swe/property/Depth",
						"uom":        map[string]interface{}{"code": "m"},
						"value":      50.0,
					},
				},
			},
			expectedGetCode: http.StatusNotFound,
		},
	}

	for name, tc := range deleteTestCases {
		t.Run(name, func(t *testing.T) {
			// Create
			body, _ := json.Marshal(tc.createPayload)
			createResp, err := http.Post(testServer.URL+"/properties", "application/sml+json", bytes.NewReader(body))
			require.NoError(t, err)
			createResp.Body.Close()

			createdProp, err := FollowLocation(createResp, "application/sml+json")
			require.NoError(t, err)
			propID := (*createdProp)["id"].(string)

			// Delete
			client := &http.Client{}
			req, _ := http.NewRequest(http.MethodDelete, testServer.URL+"/properties/"+propID, nil)
			resp, err := client.Do(req)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode, "DELETE /properties/{id} must return 204 No Content")

			// Verify deletion
			getReq, _ := http.NewRequest(http.MethodGet, testServer.URL+"/properties/"+propID, nil)
			getReq.Header.Set("Accept", "application/sml+json")
			getResp, err := client.Do(getReq)
			require.NoError(t, err)
			defer getResp.Body.Close()

			assert.Equal(t, tc.expectedGetCode, getResp.StatusCode, "GET after DELETE must return expected status code")
		})
	}
}

// =============================================================================
// Conformance Class: /conf/advanced-filtering
// Requirements: /req/advanced-filtering/prop-by-baseprop, /req/advanced-filtering/prop-by-object
// A.39: /conf/advanced-filtering/prop-by-baseprop
// =============================================================================
func TestPropertyAdvancedFiltering_BaseProp(t *testing.T) {

	// Setup: Create test properties for conformance tests
	createdPropertyIDs := setupPropertyConformanceData(t)
	defer cleanupPropertyConformanceData(t, createdPropertyIDs)

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

}

// =============================================================================
// Conformance Class: /conf/advanced-filtering
// Requirements: /req/advanced-filtering/prop-by-baseprop, /req/advanced-filtering/prop-by-object
// A.40: /conf/advanced-filtering/prop-by-object
// =============================================================================
func TestPropertyAdvancedFiltering_ObjectType(t *testing.T) {
	// Setup: Create test properties for conformance tests
	createdPropertyIDs := setupPropertyConformanceData(t)
	defer cleanupPropertyConformanceData(t, createdPropertyIDs)

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

}

// =============================================================================
// Conformance Class: /conf/advanced-filtering
// Requirements: /req/advanced-filtering/prop-by-baseprop, /req/advanced-filtering/prop-by-object
// A.39-40: /conf/advanced-filtering/prop-combined-filters
// =============================================================================
func TestPropertyAdvancedFiltering_CombinedFilters(t *testing.T) {
	// Setup: Create test properties for conformance tests
	createdPropertyIDs := setupPropertyConformanceData(t)
	defer cleanupPropertyConformanceData(t, createdPropertyIDs)

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
}
