package e2e

import (
	"encoding/json"
	"sync"

	"github.com/yourusername/connected-systems-go/internal/contractvalidation"
)

// SchemaValidator keeps the E2E conformance tests on the exact schema bundle
// embedded by the production request validator. This prevents test-only schema
// drift and avoids depending on a working-directory-relative schema path.
type SchemaValidator struct {
	validator *contractvalidation.Validator
}

var (
	validator     *SchemaValidator
	validatorOnce sync.Once
)

func GetSchemaValidator() *SchemaValidator {
	validatorOnce.Do(func() {
		validator = NewSchemaValidator()
	})
	return validator
}

func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{validator: contractvalidation.New()}
}

func (v *SchemaValidator) ValidateJSON(schemaName string, data []byte) error {
	return v.validator.ValidateSchema(schemaName, data)
}

func (v *SchemaValidator) ValidateInterface(schemaName string, data interface{}) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return v.ValidateJSON(schemaName, encoded)
}

const (
	PropertySchema = "sensorml/property-bundled.json"
)
