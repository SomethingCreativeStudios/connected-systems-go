package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getJSONResource(t *testing.T, path, accept string) map[string]interface{} {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, testServer.URL+path, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", accept)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body
}

func requireObjectMember(t *testing.T, obj map[string]interface{}, name string) map[string]interface{} {
	t.Helper()
	value, ok := obj[name]
	require.True(t, ok, "expected %s member", name)
	require.NotNil(t, value, "expected %s member to be non-null", name)
	member, ok := value.(map[string]interface{})
	require.True(t, ok, "%s must be an object", name)
	return member
}

func requireGeoJSONPoint(t *testing.T, geom map[string]interface{}, lon, lat float64) {
	t.Helper()
	require.Equal(t, "Point", geom["type"])
	coords, ok := geom["coordinates"].([]interface{})
	require.True(t, ok, "Point coordinates must be an array")
	require.Len(t, coords, 2)
	assert.InDelta(t, lon, coords[0], 0.000001)
	assert.InDelta(t, lat, coords[1], 0.000001)
}

func createSamplingFeatureSMLViaAPI(t *testing.T, systemID string, payload map[string]interface{}) string {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/samplingFeatures", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/sml+json")
	req.Header.Set("Accept", "application/sml+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	location := resp.Header.Get("Location")
	require.NotEmpty(t, location)
	id := parseID(location, "/samplingFeatures/")
	require.NotEmpty(t, id)
	return id
}

func TestSystemSensorML_PositionDerivedFromGeoJSONGeometry(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("System GeoJSON Geometry To SML Position"))

	sml := getJSONResource(t, "/systems/"+systemID, "application/sml+json")
	requireGeoJSONPoint(t, requireObjectMember(t, sml, "position"), -117.1625, 32.715)
}

func TestSystemGeoJSON_GeometryDerivedFromSensorMLPosition(t *testing.T) {
	cleanupDB(t)

	payload := map[string]interface{}{
		"type":       "PhysicalSystem",
		"label":      "SML Position To GeoJSON Geometry",
		"uniqueId":   "urn:uuid:" + uuid.NewString(),
		"definition": "http://www.w3.org/ns/sosa/Sensor",
		"position": map[string]interface{}{
			"type":        "Point",
			"coordinates": []float64{-118.25, 34.05},
		},
	}
	systemID := createSystemSMLViaAPI(t, "/systems", payload)

	feature := getJSONResource(t, "/systems/"+systemID, "application/geo+json")
	requireGeoJSONPoint(t, requireObjectMember(t, feature, "geometry"), -118.25, 34.05)
}

func TestDeploymentSensorML_LocationDerivedFromGeoJSONGeometry(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Deployment Location System"))
	deploymentID := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Deployment GeoJSON Geometry To SML Location", systemID))

	sml := getJSONResource(t, "/deployments/"+deploymentID, "application/sml+json")
	location := requireObjectMember(t, sml, "location")
	require.Equal(t, "Polygon", location["type"])
}

func TestSamplingFeatureSensorML_PositionDerivedFromGeoJSONGeometry(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Sampling Feature Position System"))
	samplingFeatureID := createSamplingFeatureViaAPI(t, systemID, baseSamplingFeaturePayload("SF GeoJSON Geometry To SML Position"))

	sml := getJSONResource(t, "/samplingFeatures/"+samplingFeatureID, "application/sml+json")
	requireGeoJSONPoint(t, requireObjectMember(t, sml, "position"), -117.1625, 32.715)
}

func TestSamplingFeatureGeoJSON_GeometryDerivedFromSensorMLPosition(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Sampling Feature Geometry System"))
	payload := map[string]interface{}{
		"type":       "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
		"label":      "SF SML Position To GeoJSON Geometry",
		"uniqueId":   "urn:uuid:" + uuid.NewString(),
		"definition": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
		"sampledFeature": map[string]interface{}{
			"href": "http://example.org/features/test-foi",
		},
		"position": map[string]interface{}{
			"type":        "Point",
			"coordinates": []float64{-119, 35},
		},
	}
	samplingFeatureID := createSamplingFeatureSMLViaAPI(t, systemID, payload)

	feature := getJSONResource(t, "/samplingFeatures/"+samplingFeatureID, "application/geo+json")
	requireGeoJSONPoint(t, requireObjectMember(t, feature, "geometry"), -119, 35)
}
