package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// postControlStreamStatus creates a control stream under a system and returns
// the raw HTTP status, so tests can assert rejection of malformed schemas.
func postControlStreamStatus(t *testing.T, systemID string, payload map[string]interface{}) int {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/systems/"+systemID+"/controlstreams", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}

// controlStreamPayloadWithParamField returns a control stream create payload
// whose parametersSchema is a DataRecord with a single field. The field is
// merged from the given map so tests can omit required attributes.
func controlStreamPayloadWithParamField(field map[string]interface{}) map[string]interface{} {
	p := baseControlStreamPayload()
	p["schema"] = map[string]interface{}{
		"commandFormat": "application/json",
		"parametersSchema": map[string]interface{}{
			"type":   "DataRecord",
			"fields": []map[string]interface{}{field},
		},
	}
	return p
}

// TestControlStreamSchema_QuantityRequiresUOM reproduces the original defect: a
// Quantity component without a uom must be rejected on create, not silently
// stored.
func TestControlStreamSchema_QuantityRequiresUOM(t *testing.T) {
	cleanupDB(t)
	systemID := createSystemForControlStreamTest(t)

	// Missing uom -> rejected.
	status := postControlStreamStatus(t, systemID, controlStreamPayloadWithParamField(map[string]interface{}{
		"name":       "speed",
		"type":       "Quantity",
		"definition": "urn:onvif:ptz:speed",
		"label":      "Move Speed",
	}))
	assert.Equal(t, http.StatusBadRequest, status, "Quantity without uom must be rejected")

	// With a uom -> accepted.
	status = postControlStreamStatus(t, systemID, controlStreamPayloadWithParamField(map[string]interface{}{
		"name":       "speed",
		"type":       "Quantity",
		"definition": "urn:onvif:ptz:speed",
		"label":      "Move Speed",
		"uom":        map[string]interface{}{"code": "1"},
	}))
	assert.Equal(t, http.StatusCreated, status, "Quantity with uom must be accepted")
}

// TestControlStreamSchema_ScalarRequiresDefinitionAndLabel verifies that simple
// scalars still require definition and label per the SWE Common schema.
func TestControlStreamSchema_ScalarRequiresDefinitionAndLabel(t *testing.T) {
	cleanupDB(t)
	systemID := createSystemForControlStreamTest(t)

	// Category missing definition and label.
	status := postControlStreamStatus(t, systemID, controlStreamPayloadWithParamField(map[string]interface{}{
		"name": "action",
		"type": "Category",
	}))
	assert.Equal(t, http.StatusBadRequest, status, "Category without definition/label must be rejected")

	// Category missing only label.
	status = postControlStreamStatus(t, systemID, controlStreamPayloadWithParamField(map[string]interface{}{
		"name":       "action",
		"type":       "Category",
		"definition": "urn:onvif:ptz:preset-action",
	}))
	assert.Equal(t, http.StatusBadRequest, status, "Category without label must be rejected")

	// Complete Category (definition + label, no uom needed) is accepted.
	status = postControlStreamStatus(t, systemID, controlStreamPayloadWithParamField(map[string]interface{}{
		"name":       "action",
		"type":       "Category",
		"definition": "urn:onvif:ptz:preset-action",
		"label":      "Action",
	}))
	assert.Equal(t, http.StatusCreated, status, "complete Category must be accepted")
}

// TestControlStreamSchema_ValidationOnSchemaPut verifies the schema PUT endpoint
// enforces the same structural rules as create.
func TestControlStreamSchema_ValidationOnSchemaPut(t *testing.T) {
	cleanupDB(t)
	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	// A schema PUT with a uom-less Quantity must be rejected.
	body, err := json.Marshal(map[string]interface{}{
		"commandFormat": "application/json",
		"parametersSchema": map[string]interface{}{
			"type": "DataRecord",
			"fields": []map[string]interface{}{
				{"name": "speed", "type": "Quantity", "definition": "urn:onvif:ptz:speed", "label": "Move Speed"},
			},
		},
	})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, testServer.URL+"/controlstreams/"+csID+"/schema", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "schema PUT must reject a uom-less Quantity")
}
