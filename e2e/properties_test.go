package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropertiesAPI_E2E(t *testing.T) {
	cleanupDB(t)

	// SensorML format payloads for properties with various qualifier types
	createTestCases := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		validateResult func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "simple property with label and uniqueId",
			payload: map[string]interface{}{
				"label":        "Temperature",
				"description":  "Air temperature property",
				"uniqueId":     "urn:test:property:simple-temp",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
			},
			expectedStatus: http.StatusCreated,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.NotEmpty(t, result["id"])
				assert.Equal(t, "Temperature", result["label"])
				assert.Equal(t, "urn:test:property:simple-temp", result["uniqueId"])
				assert.Equal(t, "https://qudt.org/vocab/quantitykind/Temperature", result["baseProperty"])
			},
		},
		{
			name: "property with objectType and statistic",
			payload: map[string]interface{}{
				"label":        "Average CPU Temperature",
				"description":  "Hourly average of CPU temperature",
				"uniqueId":     "urn:test:property:avg-cpu-temp",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"objectType":   "http://dbpedia.org/resource/Central_processing_unit",
				"statistic":    "http://sensorml.com/ont/x-stats/HourlyMean",
			},
			expectedStatus: http.StatusCreated,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.NotEmpty(t, result["id"])
				assert.Equal(t, "Average CPU Temperature", result["label"])
				assert.Equal(t, "http://dbpedia.org/resource/Central_processing_unit", result["objectType"])
				assert.Equal(t, "http://sensorml.com/ont/x-stats/HourlyMean", result["statistic"])
			},
		},
		{
			name: "property with Quantity qualifier",
			payload: map[string]interface{}{
				"label":        "Temperature at Height",
				"uniqueId":     "urn:test:property:temp-with-height",
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
			expectedStatus: http.StatusCreated,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.NotEmpty(t, result["id"])
				qualifiers, ok := result["qualifiers"].([]interface{})
				require.True(t, ok, "expected qualifiers array")
				require.Len(t, qualifiers, 1)
				q := qualifiers[0].(map[string]interface{})
				assert.Equal(t, "Quantity", q["type"])
				assert.Equal(t, "Measurement Height", q["label"])
			},
		},
		{
			name: "property with Category qualifier",
			payload: map[string]interface{}{
				"label":        "Wind Speed by Terrain",
				"uniqueId":     "urn:test:property:wind-terrain",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Speed",
				"qualifiers": []map[string]interface{}{
					{
						"type":       "Category",
						"label":      "Terrain Type",
						"definition": "http://sensorml.com/ont/swe/property/TerrainType",
						"codeSpace":  "http://example.org/terrains",
						"value":      "urban",
					},
				},
			},
			expectedStatus: http.StatusCreated,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.NotEmpty(t, result["id"])
				qualifiers, ok := result["qualifiers"].([]interface{})
				require.True(t, ok, "expected qualifiers array")
				require.Len(t, qualifiers, 1)
				q := qualifiers[0].(map[string]interface{})
				assert.Equal(t, "Category", q["type"])
				assert.Equal(t, "urban", q["value"])
			},
		},
		{
			name: "property with Boolean qualifier",
			payload: map[string]interface{}{
				"label":        "Calibrated Temperature",
				"uniqueId":     "urn:test:property:calibrated-temp",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"qualifiers": []map[string]interface{}{
					{
						"type":       "Boolean",
						"label":      "Is Calibrated",
						"definition": "http://sensorml.com/ont/swe/property/IsCalibrated",
						"value":      true,
					},
				},
			},
			expectedStatus: http.StatusCreated,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.NotEmpty(t, result["id"])
				qualifiers, ok := result["qualifiers"].([]interface{})
				require.True(t, ok, "expected qualifiers array")
				require.Len(t, qualifiers, 1)
				q := qualifiers[0].(map[string]interface{})
				assert.Equal(t, "Boolean", q["type"])
				assert.Equal(t, true, q["value"])
			},
		},
		{
			name: "property with Count qualifier",
			payload: map[string]interface{}{
				"label":        "Average Over Samples",
				"uniqueId":     "urn:test:property:avg-samples",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"statistic":    "http://sensorml.com/ont/x-stats/Average",
				"qualifiers": []map[string]interface{}{
					{
						"type":       "Count",
						"label":      "Sample Count",
						"definition": "http://sensorml.com/ont/swe/property/SampleCount",
						"value":      100,
					},
				},
			},
			expectedStatus: http.StatusCreated,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.NotEmpty(t, result["id"])
				qualifiers, ok := result["qualifiers"].([]interface{})
				require.True(t, ok, "expected qualifiers array")
				require.Len(t, qualifiers, 1)
				q := qualifiers[0].(map[string]interface{})
				assert.Equal(t, "Count", q["type"])
				// JSON decodes numbers as float64
				assert.Equal(t, float64(100), q["value"])
			},
		},
		{
			name: "property with Text qualifier",
			payload: map[string]interface{}{
				"label":        "Temperature with Notes",
				"uniqueId":     "urn:test:property:temp-notes",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"qualifiers": []map[string]interface{}{
					{
						"type":       "Text",
						"label":      "Measurement Notes",
						"definition": "http://sensorml.com/ont/swe/property/Notes",
						"value":      "Measured in shade, away from heat sources",
					},
				},
			},
			expectedStatus: http.StatusCreated,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.NotEmpty(t, result["id"])
				qualifiers, ok := result["qualifiers"].([]interface{})
				require.True(t, ok, "expected qualifiers array")
				require.Len(t, qualifiers, 1)
				q := qualifiers[0].(map[string]interface{})
				assert.Equal(t, "Text", q["type"])
				assert.Equal(t, "Measured in shade, away from heat sources", q["value"])
			},
		},
		{
			name: "property with Time qualifier",
			payload: map[string]interface{}{
				"label":        "Temperature at Time",
				"uniqueId":     "urn:test:property:temp-time",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"qualifiers": []map[string]interface{}{
					{
						"type":       "Time",
						"label":      "Measurement Time",
						"definition": "http://sensorml.com/ont/swe/property/SamplingTime",
						"uom": map[string]interface{}{
							"href": "http://www.opengis.net/def/uom/ISO-8601/0/Gregorian",
						},
						"value": "2025-11-27T12:00:00Z",
					},
				},
			},
			expectedStatus: http.StatusCreated,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.NotEmpty(t, result["id"])
				qualifiers, ok := result["qualifiers"].([]interface{})
				require.True(t, ok, "expected qualifiers array")
				require.Len(t, qualifiers, 1)
				q := qualifiers[0].(map[string]interface{})
				assert.Equal(t, "Time", q["type"])
				assert.Equal(t, "2025-11-27T12:00:00Z", q["value"])
			},
		},
		{
			name: "property with QuantityRange qualifier",
			payload: map[string]interface{}{
				"label":        "Temperature in Range",
				"uniqueId":     "urn:test:property:temp-range",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"qualifiers": []map[string]interface{}{
					{
						"type":       "QuantityRange",
						"label":      "Valid Range",
						"definition": "http://sensorml.com/ont/swe/property/ValidRange",
						"uom": map[string]interface{}{
							"code": "Cel",
						},
						"value": []float64{-40.0, 85.0},
					},
				},
			},
			expectedStatus: http.StatusCreated,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.NotEmpty(t, result["id"])
				qualifiers, ok := result["qualifiers"].([]interface{})
				require.True(t, ok, "expected qualifiers array")
				require.Len(t, qualifiers, 1)
				q := qualifiers[0].(map[string]interface{})
				assert.Equal(t, "QuantityRange", q["type"])
				rangeVal, ok := q["value"].([]interface{})
				require.True(t, ok, "expected value to be array")
				assert.Len(t, rangeVal, 2)
			},
		},
		{
			name: "property with multiple qualifiers",
			payload: map[string]interface{}{
				"label":        "Complex Temperature Measurement",
				"description":  "Temperature with multiple qualifiers",
				"uniqueId":     "urn:test:property:complex-temp",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
				"objectType":   "http://dbpedia.org/resource/Atmosphere",
				"statistic":    "http://sensorml.com/ont/x-stats/Average",
				"qualifiers": []map[string]interface{}{
					{
						"type":       "Quantity",
						"label":      "Measurement Height",
						"definition": "http://sensorml.com/ont/swe/property/Height",
						"uom": map[string]interface{}{
							"code": "m",
						},
						"value": 10.0,
					},
					{
						"type":       "Category",
						"label":      "Environment",
						"definition": "http://sensorml.com/ont/swe/property/Environment",
						"value":      "outdoor",
					},
					{
						"type":       "Boolean",
						"label":      "Quality Controlled",
						"definition": "http://sensorml.com/ont/swe/property/QualityControlled",
						"value":      true,
					},
					{
						"type":       "QuantityRange",
						"label":      "Frequency Range",
						"definition": "http://sensorml.com/ont/swe/property/FrequencyRange",
						"uom": map[string]interface{}{
							"code": "Hz",
						},
						"value": []float64{0.1, 10.0},
					},
				},
			},
			expectedStatus: http.StatusCreated,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.NotEmpty(t, result["id"])
				assert.Equal(t, "Complex Temperature Measurement", result["label"])
				assert.Equal(t, "http://dbpedia.org/resource/Atmosphere", result["objectType"])
				assert.Equal(t, "http://sensorml.com/ont/x-stats/Average", result["statistic"])

				qualifiers, ok := result["qualifiers"].([]interface{})
				require.True(t, ok, "expected qualifiers array")
				assert.Len(t, qualifiers, 4)

				// Verify each qualifier type is present
				types := make(map[string]bool)
				for _, q := range qualifiers {
					qMap := q.(map[string]interface{})
					types[qMap["type"].(string)] = true
				}
				assert.True(t, types["Quantity"], "expected Quantity qualifier")
				assert.True(t, types["Category"], "expected Category qualifier")
				assert.True(t, types["Boolean"], "expected Boolean qualifier")
				assert.True(t, types["QuantityRange"], "expected QuantityRange qualifier")
			},
		},
		{
			name: "property with Quantity constraint",
			payload: map[string]interface{}{
				"label":        "Constrained Height",
				"uniqueId":     "urn:test:property:constrained-height",
				"baseProperty": "https://qudt.org/vocab/quantitykind/Height",
				"qualifiers": []map[string]interface{}{
					{
						"type":       "Quantity",
						"label":      "Height Above Ground",
						"definition": "http://sensorml.com/ont/swe/property/HeightAboveGround",
						"uom": map[string]interface{}{
							"code":   "m",
							"label":  "meters",
							"symbol": "m",
						},
						"constraint": map[string]interface{}{
							"type":      "AllowedValues",
							"intervals": [][]float64{{0.0, 100.0}},
						},
						"value": 50.0,
					},
				},
			},
			expectedStatus: http.StatusCreated,
			validateResult: func(t *testing.T, result map[string]interface{}) {
				assert.NotEmpty(t, result["id"])
				qualifiers, ok := result["qualifiers"].([]interface{})
				require.True(t, ok, "expected qualifiers array")
				require.Len(t, qualifiers, 1)
				q := qualifiers[0].(map[string]interface{})
				assert.Equal(t, "Quantity", q["type"])
				constraint, ok := q["constraint"].(map[string]interface{})
				require.True(t, ok, "expected constraint object")
				assert.Equal(t, "AllowedValues", constraint["type"])
			},
		},
	}

	for _, tc := range createTestCases {
		t.Run("POST /properties - "+tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			resp, err := http.Post(testServer.URL+"/properties", "application/sml+json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			if tc.validateResult != nil {
				if resp.StatusCode == http.StatusCreated {
					// Expect Location header and fetch the created resource for validation
					location := resp.Header.Get("Location")
					require.NotEmpty(t, location, "expected Location header on create")
					id := parsePropertyID(location)
					require.NotEmpty(t, id, "unable to parse id from Location header: %s", location)
					created, err := fetchPropertyById(id)
					require.NoError(t, err)
					tc.validateResult(t, *created)
				} else {
					var result map[string]interface{}
					err = json.NewDecoder(resp.Body).Decode(&result)
					require.NoError(t, err)
					tc.validateResult(t, result)
				}
			}
		})
	}

	// CRUD operation test cases
	crudTestCases := []struct {
		name            string
		createPayload   map[string]interface{}
		updatePayload   map[string]interface{}
		expectedGetCode int
		validateUpdate  func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "get specific property",
			createPayload: map[string]interface{}{
				"label":        "Humidity",
				"uniqueId":     "urn:test:property:crud-humidity",
				"baseProperty": "https://qudt.org/vocab/quantitykind/RelativeHumidity",
			},
			expectedGetCode: http.StatusOK,
		},
		{
			name: "update property label",
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
		{
			name: "update property with qualifiers",
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

	for _, tc := range crudTestCases {
		t.Run("CRUD - "+tc.name, func(t *testing.T) {
			// Create
			body, _ := json.Marshal(tc.createPayload)
			createResp, err := http.Post(testServer.URL+"/properties", "application/sml+json", bytes.NewReader(body))
			require.NoError(t, err)
			createResp.Body.Close()

			// Extract ID from Location header (create returns 201 + Location, no body)
			require.Equal(t, http.StatusCreated, createResp.StatusCode)
			location := createResp.Header.Get("Location")
			require.NotEmpty(t, location, "expected Location header on create")
			propID := parsePropertyID(location)
			require.NotEmpty(t, propID, "unable to parse id from Location header: %s", location)

			// Get
			req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/properties/"+propID, nil)
			req.Header.Set("Accept", "application/sml+json")
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			resp.Body.Close()
			assert.Equal(t, tc.expectedGetCode, resp.StatusCode)

			// Update if payload provided
			if tc.updatePayload != nil {
				updateBody, _ := json.Marshal(tc.updatePayload)
				req, _ := http.NewRequest(http.MethodPut, testServer.URL+"/properties/"+propID, bytes.NewReader(updateBody))
				req.Header.Set("Content-Type", "application/sml+json")
				req.Header.Set("Accept", "application/sml+json")

				resp, err := client.Do(req)
				require.NoError(t, err)

				resp.Body.Close()
				// Update now returns 204 No Content; fetch the resource to verify changes
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)

				if tc.validateUpdate != nil {
					fetched, err := fetchPropertyById(propID)
					require.NoError(t, err)
					tc.validateUpdate(t, *fetched)
				}
			}
		})
	}
}

func parsePropertyID(locationHeader string) string {
	parts := strings.Split(locationHeader, "/properties/")

	if len(parts) == 2 {
		// strip any trailing slash or query
		id := parts[1]
		if idx := strings.IndexAny(id, "?#/"); idx != -1 {
			id = id[:idx]
		}
		return id
	}

	return ""
}

func fetchPropertyById(propertyID string) (*map[string]interface{}, error) {
	url := fmt.Sprintf("%s/properties/%s", testServer.URL, propertyID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/sml+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch property: status %d", resp.StatusCode)
	}

	var p map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&p)
	if err != nil {
		return nil, err
	}

	return &p, nil
}
