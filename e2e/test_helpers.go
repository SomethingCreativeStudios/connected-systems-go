package e2e

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// getJSONItems decodes a JSON endpoint collection response and returns the "items" array.
// Use this for Part 2 / plain JSON endpoints (datastreams, control streams, observations,
// commands, system events, etc.) where the collection envelope uses "items" rather than
// "features".
func getJSONItems(t *testing.T, body []byte) []any {
	t.Helper()
	var collection map[string]any
	require.NoError(t, json.Unmarshal(body, &collection))
	items, ok := collection["items"].([]any)
	require.True(t, ok, "response must contain 'items' array; got: %s", string(body))
	links, ok := collection["links"].([]any)
	require.True(t, ok, "response must contain 'links' array; got: %s", string(body))
	require.GreaterOrEqual(t, len(links), 1, "response links must include at least self")
	return items
}

// validateAgainstSchema is a helper function to validate JSON data against a schema
func validateAgainstSchema(t *testing.T, jsonData []byte, schemaName string) error {
	t.Helper()
	// The caller decides whether a validation error fails the test.
	return GetSchemaValidator().ValidateJSON(schemaName, jsonData)
}
