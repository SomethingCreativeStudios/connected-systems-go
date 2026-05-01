package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deleteViaAPI sends a DELETE request and returns the response status code.
func deleteViaAPI(t *testing.T, path string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, testServer.URL+path, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	return resp.StatusCode
}

// =============================================================================
// System DELETE status codes
// =============================================================================

func TestSystemDelete_Returns404ForNonExistentID(t *testing.T) {
	cleanupDB(t)
	assert.Equal(t, http.StatusNotFound, deleteViaAPI(t, "/systems/does-not-exist"))
}

func TestSystemDelete_Returns409WhenSubsystemExistsNoCascade(t *testing.T) {
	cleanupDB(t)

	parentID := createSystemViaAPI(t, "/systems", baseSystemPayload("Delete 409 Parent"))
	createSystemViaAPI(t, "/systems/"+parentID+"/subsystems", baseSystemPayload("Delete 409 Child"))

	assert.Equal(t, http.StatusConflict, deleteViaAPI(t, "/systems/"+parentID))
}

func TestSystemDelete_Returns204WithCascade(t *testing.T) {
	cleanupDB(t)

	parentID := createSystemViaAPI(t, "/systems", baseSystemPayload("Cascade Delete Parent"))
	childID := createSystemViaAPI(t, "/systems/"+parentID+"/subsystems", baseSystemPayload("Cascade Delete Child"))

	assert.Equal(t, http.StatusNoContent, deleteViaAPI(t, "/systems/"+parentID+"?cascade=true"))

	assert.Equal(t, http.StatusNotFound, deleteViaAPI(t, "/systems/"+parentID))
	assert.Equal(t, http.StatusNotFound, deleteViaAPI(t, "/systems/"+childID))
}

// =============================================================================
// Datastream DELETE status codes
// =============================================================================

func TestDatastreamDelete_Returns404ForNonExistentID(t *testing.T) {
	cleanupDB(t)
	assert.Equal(t, http.StatusNotFound, deleteViaAPI(t, "/datastreams/does-not-exist"))
}

func TestDatastreamDelete_Returns409WhenObservationsExistNoCascade(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("DS Delete 409 System"))
	datastreamID := createDatastreamViaAPI(t, "/systems/"+systemID+"/datastreams", baseDatastreamPayload())

	obsPayload := map[string]interface{}{
		"resultTime": time.Now().UTC().Format(time.RFC3339),
		"result":     map[string]interface{}{"temperature": 22.5, "humidity": 55.0},
	}
	createObservationViaAPI(t, datastreamID, obsPayload)

	assert.Equal(t, http.StatusConflict, deleteViaAPI(t, "/datastreams/"+datastreamID))
}

func TestDatastreamDelete_Returns204WithCascade(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("DS Cascade Delete System"))
	datastreamID := createDatastreamViaAPI(t, "/systems/"+systemID+"/datastreams", baseDatastreamPayload())

	obsPayload := map[string]interface{}{
		"resultTime": time.Now().UTC().Format(time.RFC3339),
		"result":     map[string]interface{}{"temperature": 22.5, "humidity": 55.0},
	}
	createObservationViaAPI(t, datastreamID, obsPayload)

	assert.Equal(t, http.StatusNoContent, deleteViaAPI(t, "/datastreams/"+datastreamID+"?cascade=true"))
	assert.Equal(t, http.StatusNotFound, deleteViaAPI(t, "/datastreams/"+datastreamID))
}

// =============================================================================
// Deployment DELETE status codes
// =============================================================================

func TestDeploymentDelete_Returns404ForNonExistentID(t *testing.T) {
	cleanupDB(t)
	assert.Equal(t, http.StatusNotFound, deleteViaAPI(t, "/deployments/does-not-exist"))
}

func TestDeploymentDelete_Returns409WhenSubdeploymentExistsNoCascade(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Dep Delete 409 System"))
	parentDepID := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Delete 409 Parent Dep", systemID))
	createDeploymentViaAPI(t, "/deployments/"+parentDepID+"/subdeployments", baseDeploymentPayload("Delete 409 Child Dep", systemID))

	assert.Equal(t, http.StatusConflict, deleteViaAPI(t, "/deployments/"+parentDepID))
}

func TestDeploymentDelete_Returns204WithCascade(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Dep Cascade Delete System"))
	parentDepID := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Cascade Delete Parent Dep", systemID))
	childDepID := createDeploymentViaAPI(t, "/deployments/"+parentDepID+"/subdeployments", baseDeploymentPayload("Cascade Delete Child Dep", systemID))

	assert.Equal(t, http.StatusNoContent, deleteViaAPI(t, "/deployments/"+parentDepID+"?cascade=true"))

	assert.Equal(t, http.StatusNotFound, deleteViaAPI(t, "/deployments/"+parentDepID))
	assert.Equal(t, http.StatusNotFound, deleteViaAPI(t, "/deployments/"+childDepID))
}

func TestDeploymentDelete_CascadeDoesNotDeleteAssociatedSystems(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("System Surviving Cascade"))
	depID := createDeploymentViaAPI(t, "/deployments", baseDeploymentPayload("Dep To Cascade Delete", systemID))

	assert.Equal(t, http.StatusNoContent, deleteViaAPI(t, "/deployments/"+depID+"?cascade=true"))

	// System must still exist after deployment cascade delete
	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/systems/"+systemID, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
