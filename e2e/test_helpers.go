package e2e

import (
	"testing"
)

// validateAgainstSchema is a helper function to validate JSON data against a schema
func validateAgainstSchema(t *testing.T, jsonData []byte, schemaName string) error {
	t.Helper()
	validator := GetSchemaValidator()
	err := validator.ValidateJSON(schemaName, jsonData)
	if err != nil {
		// Log the error but don't fail immediately, let the caller decide
		// t.Logf("Schema validation error: %v", err)
	}
	return err
}
