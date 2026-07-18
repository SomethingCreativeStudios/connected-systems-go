package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRequiredRootFields verifies that creating a resource without one of its
// required root fields is rejected with 400 Bad Request, per the required
// (and writable) fields of the OGC API - Connected Systems Part 1 and Part 2
// OpenAPI schemas.
func TestRequiredRootFields(t *testing.T) {
	cleanupDB(t)

	// Parents for nested resources.
	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Required Fields Parent System"))
	controlStreamID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	smlSystemPayload := func() map[string]interface{} {
		return map[string]interface{}{
			"type":       "PhysicalSystem",
			"label":      "SML Required Fields System",
			"uniqueId":   "urn:uuid:" + uuid.NewString(),
			"definition": "http://www.w3.org/ns/sosa/Sensor",
		}
	}
	smlProcedurePayload := func() map[string]interface{} {
		return map[string]interface{}{
			"type":       "SimpleProcess",
			"label":      "SML Required Fields Procedure",
			"uniqueId":   "urn:uuid:" + uuid.NewString(),
			"definition": "http://www.w3.org/ns/sosa/Procedure",
		}
	}
	smlDeploymentPayload := func() map[string]interface{} {
		return map[string]interface{}{
			"type":       "Deployment",
			"label":      "SML Required Fields Deployment",
			"uniqueId":   "urn:uuid:" + uuid.NewString(),
			"definition": "http://www.w3.org/ns/sosa/Deployment",
		}
	}
	smlPropertyPayload := func() map[string]interface{} {
		return map[string]interface{}{
			"label":        "SML Required Fields Property",
			"uniqueId":     "urn:uuid:" + uuid.NewString(),
			"baseProperty": "https://qudt.org/vocab/quantitykind/Temperature",
		}
	}
	datastreamPayload := baseDatastreamPayload
	systemEventPayload := func() map[string]interface{} {
		return map[string]interface{}{
			"definition": "https://example.org/event/calibration",
			"label":      "Required Fields Event",
			"time":       "2026-07-01T00:00:00Z",
		}
	}
	observationPayload := func() map[string]interface{} {
		return map[string]interface{}{
			"resultTime": "2026-07-01T00:00:00Z",
			"result":     22.5,
		}
	}
	commandPayload := func() map[string]interface{} {
		return map[string]interface{}{
			"parameters": map[string]interface{}{"setPoint": 21.0},
		}
	}

	cases := []struct {
		resource    string
		endpoint    string
		contentType string
		payload     func() map[string]interface{}
		// required keys; "properties.x" removes a nested key
		required []string
	}{
		{"system geo+json", "/systems", "application/geo+json",
			func() map[string]interface{} { return baseSystemPayload("GeoJSON Required Fields System") },
			[]string{"properties.uid", "properties.name", "properties.featureType"}},
		{"system sml+json", "/systems", "application/sml+json",
			smlSystemPayload,
			[]string{"type", "uniqueId", "label", "definition"}},
		{"procedure geo+json", "/procedures", "application/geo+json",
			func() map[string]interface{} {
				return map[string]interface{}{
					"type": "Feature",
					"properties": map[string]interface{}{
						"uid":         "urn:uuid:" + uuid.NewString(),
						"name":        "GeoJSON Required Fields Procedure",
						"featureType": "http://www.w3.org/ns/sosa/Procedure",
					},
				}
			},
			[]string{"properties.uid", "properties.name", "properties.featureType"}},
		{"procedure sml+json", "/procedures", "application/sml+json",
			smlProcedurePayload,
			[]string{"type", "uniqueId", "label", "definition"}},
		{"deployment geo+json", "/deployments", "application/geo+json",
			func() map[string]interface{} {
				return baseDeploymentPayload("GeoJSON Required Fields Deployment", systemID)
			},
			[]string{"properties.uid", "properties.name", "properties.featureType", "properties.validTime"}},
		{"deployment sml+json", "/deployments", "application/sml+json",
			smlDeploymentPayload,
			[]string{"type", "uniqueId", "label", "definition"}},
		{"samplingFeature geo+json", "/systems/" + systemID + "/samplingFeatures", "application/geo+json",
			func() map[string]interface{} {
				return baseSamplingFeaturePayload("GeoJSON Required Fields SF")
			},
			[]string{"properties.uid", "properties.name", "properties.featureType", "properties.sampledFeature@link"}},
		{"property sml+json", "/properties", "application/sml+json",
			smlPropertyPayload,
			[]string{"uniqueId", "label", "baseProperty"}},
		{"datastream json", "/systems/" + systemID + "/datastreams", "application/json",
			datastreamPayload,
			[]string{"name", "schema"}},
		{"controlstream json", "/systems/" + systemID + "/controlstreams", "application/json",
			func() map[string]interface{} { return baseControlStreamPayload() },
			[]string{"name", "schema"}},
		{"observation json", "/datastreams/{ds}/observations", "application/json",
			observationPayload,
			[]string{"resultTime", "result"}},
		{"command json", "/controlstreams/" + controlStreamID + "/commands", "application/json",
			commandPayload,
			[]string{"parameters"}},
		{"systemEvent json", "/systems/" + systemID + "/events", "application/json",
			systemEventPayload,
			[]string{"definition", "label", "time"}},
	}

	// Observations need a datastream parent; create one lazily.
	datastreamID := createDatastreamViaAPI(t, "/systems/"+systemID+"/datastreams", datastreamPayload())

	for _, tc := range cases {
		endpoint := strings.ReplaceAll(tc.endpoint, "{ds}", datastreamID)

		for _, field := range tc.required {
			t.Run(tc.resource+" missing "+field, func(t *testing.T) {
				payload := tc.payload()
				if nested, ok := strings.CutPrefix(field, "properties."); ok {
					props, isMap := payload["properties"].(map[string]interface{})
					require.True(t, isMap, "payload must have a properties object")
					delete(props, nested)
				} else {
					delete(payload, field)
				}

				body, err := json.Marshal(payload)
				require.NoError(t, err)

				req, err := http.NewRequest(http.MethodPost, testServer.URL+endpoint, bytes.NewReader(body))
				require.NoError(t, err)
				req.Header.Set("Content-Type", tc.contentType)

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
					"POST %s without %s must return 400 Bad Request", endpoint, field)
			})
		}
	}
}
