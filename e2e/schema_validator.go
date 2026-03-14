package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

	// Preload all schema files under the schemas directory so that
	// relative $ref references (e.g., "commonDefs.json") resolve.
	schemaDir := getSchemaDir()
	if schemaDir != "" {
		_ = filepath.WalkDir(schemaDir, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".json") {
				return nil
			}
			rel, err := filepath.Rel(schemaDir, p)
			if err != nil {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			// Add resource as raw bytes reader so the compiler can resolve
			// internal json-pointers consistently.
			url := "file:///" + filepath.ToSlash(rel)
			_ = compiler.AddResource(url, bytes.NewReader(data))
			return nil
		})
	}

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
	if _, err := os.Stat(schemaPath); err != nil {
		return nil, fmt.Errorf("failed to read schema file %s: %w", schemaName, err)
	}

	// We preload schema resources during initialization; read the file and
	// explicitly add this schema resource under the same URL to ensure
	// any internal json-pointer references are available to the compiler.
	schemaURL := "file://" + filepath.ToSlash(schemaPath)
	// Compile using an absolute file URL so relative $ref references resolve
	// against files on disk under the schemas directory.
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
	PropertySchema = "sensorml/property-bundled.json"
)
