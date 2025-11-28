package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// SchemaValidator manages JSON schema validation for OGC conformance tests
type SchemaValidator struct {
	compiler *jsonschema.Compiler
	schemas  map[string]*jsonschema.Schema
	mu       sync.RWMutex
}

// Global validator instance
var (
	validator     *SchemaValidator
	validatorOnce sync.Once
)

// GetSchemaValidator returns the singleton schema validator
func GetSchemaValidator() *SchemaValidator {
	validatorOnce.Do(func() {
		validator = NewSchemaValidator()
	})
	return validator
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	compiler := jsonschema.NewCompiler()
	return &SchemaValidator{
		compiler: compiler,
		schemas:  make(map[string]*jsonschema.Schema),
	}
}

// getSchemaDir returns the absolute path to the schemas directory
func getSchemaDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Join(filepath.Dir(filename), "schemas")
}

// LoadSchema loads and compiles a JSON schema from the schemas directory
func (v *SchemaValidator) LoadSchema(schemaName string) (*jsonschema.Schema, error) {
	v.mu.RLock()
	if schema, exists := v.schemas[schemaName]; exists {
		v.mu.RUnlock()
		return schema, nil
	}
	v.mu.RUnlock()

	v.mu.Lock()
	defer v.mu.Unlock()

	// Double-check after acquiring write lock
	if schema, exists := v.schemas[schemaName]; exists {
		return schema, nil
	}

	schemaPath := filepath.Join(getSchemaDir(), schemaName)
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file %s: %w", schemaName, err)
	}

	// Parse the schema JSON
	var schemaDoc any
	schemaDoc, err = jsonschema.UnmarshalJSON(bytes.NewReader(schemaData))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema JSON: %w", err)
	}

	// Add the schema to the compiler
	schemaURL := "file:///" + schemaName
	if err := v.compiler.AddResource(schemaURL, schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := v.compiler.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema %s: %w", schemaName, err)
	}

	v.schemas[schemaName] = schema
	return schema, nil
}

// ValidateJSON validates JSON data against a named schema
func (v *SchemaValidator) ValidateJSON(schemaName string, data []byte) error {
	schema, err := v.LoadSchema(schemaName)
	if err != nil {
		return err
	}

	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("failed to unmarshal JSON data: %w", err)
	}

	if err := schema.Validate(jsonData); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	return nil
}

// ValidateInterface validates an interface{} against a named schema
func (v *SchemaValidator) ValidateInterface(schemaName string, data interface{}) error {
	schema, err := v.LoadSchema(schemaName)
	if err != nil {
		return err
	}

	if err := schema.Validate(data); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	return nil
}

// Schema file constants for OGC Connected Systems API
const (
	PropertySchema = "property-bundled.json"
)
