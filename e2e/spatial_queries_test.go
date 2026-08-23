package e2e

import (
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func getSpatialCollectionIDs(t *testing.T, path string, query url.Values) []string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, testServer.URL+path+"?"+query.Encode(), nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/geo+json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	return getFeatureCollectionIDs(t, body)
}

func pointGeometry(lon, lat float64) map[string]interface{} {
	return map[string]interface{}{
		"type":        "Point",
		"coordinates": []float64{lon, lat},
	}
}

func polygonGeometry(minLon, minLat, maxLon, maxLat float64) map[string]interface{} {
	return map[string]interface{}{
		"type": "Polygon",
		"coordinates": [][][]float64{{
			{minLon, minLat},
			{maxLon, minLat},
			{maxLon, maxLat},
			{minLon, maxLat},
			{minLon, minLat},
		}},
	}
}

func requireSpatialFiltersSelectOnly(t *testing.T, path, includedID, excludedID string) {
	t.Helper()
	queries := map[string]url.Values{
		"bbox": {
			"bbox": {"-118,32,-117,33"},
		},
		"geom": {
			"geom": {"POLYGON((-118 32,-117 32,-117 33,-118 33,-118 32))"},
		},
		"bbox and geom": {
			"bbox": {"-118,32,-117,33"},
			"geom": {"POLYGON((-118 32,-117 32,-117 33,-118 33,-118 32))"},
		},
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			ids := getSpatialCollectionIDs(t, path, query)
			require.Contains(t, ids, includedID)
			require.NotContains(t, ids, excludedID)
		})
	}
}

func TestSystemSpatialQueries(t *testing.T) {
	cleanupDB(t)

	included := baseSystemPayload("System Inside Spatial Query")
	included["geometry"] = pointGeometry(-117.1625, 32.715)
	includedID := createSystemViaAPI(t, "/systems", included)

	excluded := baseSystemPayload("System Outside Spatial Query")
	excluded["geometry"] = pointGeometry(-73.9857, 40.7484)
	excludedID := createSystemViaAPI(t, "/systems", excluded)

	requireSpatialFiltersSelectOnly(t, "/systems", includedID, excludedID)
}

func TestSubsystemSpatialQueries(t *testing.T) {
	cleanupDB(t)

	parentID := createSystemViaAPI(t, "/systems", baseSystemPayload("Spatial Subsystem Parent"))
	included := baseSystemPayload("Subsystem Inside Spatial Query")
	included["geometry"] = pointGeometry(-117.1625, 32.715)
	includedID := createSystemViaAPI(t, "/systems/"+parentID+"/subsystems", included)

	excluded := baseSystemPayload("Subsystem Outside Spatial Query")
	excluded["geometry"] = pointGeometry(-73.9857, 40.7484)
	excludedID := createSystemViaAPI(t, "/systems/"+parentID+"/subsystems", excluded)

	requireSpatialFiltersSelectOnly(t, "/systems/"+parentID+"/subsystems", includedID, excludedID)
}

func TestSamplingFeatureSpatialQueries(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Spatial Sampling Feature Parent"))
	included := baseSamplingFeaturePayload("Sampling Feature Inside Spatial Query")
	included["geometry"] = pointGeometry(-117.1625, 32.715)
	includedID := createSamplingFeatureViaAPI(t, systemID, included)

	excluded := baseSamplingFeaturePayload("Sampling Feature Outside Spatial Query")
	excluded["geometry"] = pointGeometry(-73.9857, 40.7484)
	excludedID := createSamplingFeatureViaAPI(t, systemID, excluded)

	requireSpatialFiltersSelectOnly(t, "/samplingFeatures", includedID, excludedID)
}

func TestDeploymentSpatialQueries(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Spatial Deployment System"))
	included := baseDeploymentPayload("Deployment Inside Spatial Query", systemID)
	included["geometry"] = polygonGeometry(-117.2, 32.7, -117.1, 32.8)
	includedID := createDeploymentViaAPI(t, "/deployments", included)

	excluded := baseDeploymentPayload("Deployment Outside Spatial Query", systemID)
	excluded["geometry"] = polygonGeometry(-74.1, 40.7, -73.9, 40.8)
	excludedID := createDeploymentViaAPI(t, "/deployments", excluded)

	requireSpatialFiltersSelectOnly(t, "/deployments", includedID, excludedID)
}
