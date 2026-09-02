// Package contractvalidation validates external Connected Systems request
// representations against the same bundled OGC JSON Schemas used by the
// conformance tests. Keeping the schemas here makes them available to the
// server binary rather than only to E2E tests on disk.
package contractvalidation

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//go:embed schemas
var schemaFiles embed.FS

// Resource identifies an externally writable Connected Systems resource.
type Resource string

const (
	System          Resource = "system"
	Deployment      Resource = "deployment"
	Procedure       Resource = "procedure"
	SamplingFeature Resource = "samplingFeature"
	Property        Resource = "property"
	Datastream      Resource = "datastream"
	ControlStream   Resource = "controlStream"
	Observation     Resource = "observation"
	Command         Resource = "command"
	SystemEvent     Resource = "systemEvent"
	CommandStatus   Resource = "commandStatus"
	CommandResult   Resource = "commandResult"
)

const (
	geoJSONContentType  = "application/geo+json"
	sensorMLContentType = "application/sml+json"
	jsonContentType     = "application/json"
)

var resourceSchemas = map[Resource]map[string]string{
	Deployment: {
		geoJSONContentType:  "geojson/deployment-bundled.json",
		sensorMLContentType: "sensorml/deployment-bundled.json",
	},
	Procedure: {
		geoJSONContentType:  "geojson/procedure-bundled.json",
		sensorMLContentType: "sensorml/procedure-bundled.json",
	},
	SamplingFeature: {geoJSONContentType: "geojson/samplingFeature-bundled.json"},
	// The remaining writable resources use strict typed decoding and their
	// endpoint-specific semantic validators. Their bundled response schemas
	// contain invalid local JSON pointers or server-assigned read-only fields,
	// so applying those response documents to a create request would reject
	// valid requests. They remain embedded for E2E response conformance.
}

// Violation identifies one actionable problem in a request payload. Path uses
// the dotted/indexed style already used by the server's decoder errors.
type Violation struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// Error is returned when a request violates an external resource contract.
// It deliberately contains no schema locations, which are server internals.
type Error struct {
	Resource    Resource
	ContentType string
	Details     []Violation
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("request body violates the %s schema", e.ContentType)
}

// Validator compiles the embedded schemas lazily and is safe for concurrent
// requests. A single instance should be shared by all formatter collections.
type Validator struct {
	once     sync.Once
	compiler *jsonschema.Compiler
	initErr  error
	mu       sync.RWMutex
	schemas  map[string]*jsonschema.Schema
}

func New() *Validator {
	return &Validator{schemas: make(map[string]*jsonschema.Schema)}
}

// Validate checks the raw JSON before it is deserialized into a domain type.
// Resources with no standalone bundled schema still receive representation
// specific semantic validation below; their typed decoders remain responsible
// for strict field/type validation.
func (v *Validator) Validate(resource Resource, contentType string, body []byte) error {
	mediaType := normalizeContentType(contentType)
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return err
	}
	if err := validateRepresentation(resource, mediaType, value); err != nil {
		return err
	}

	schemaName := resourceSchemas[resource][mediaType]
	if schemaName == "" {
		return nil
	}

	schema, err := v.load(schemaName)
	if err != nil {
		return fmt.Errorf("load %s request schema: %w", resource, err)
	}
	if err := schema.Validate(value); err != nil {
		return schemaError(resource, mediaType, err)
	}
	return nil
}

// For returns a formatter-compatible validation hook for one resource.
func (v *Validator) For(resource Resource) func(string, []byte) error {
	return func(contentType string, body []byte) error {
		return v.Validate(resource, contentType, body)
	}
}

// ValidateSchema is used by E2E response conformance checks. It intentionally
// takes a schema-relative name to retain their existing test call sites.
func (v *Validator) ValidateSchema(schemaName string, body []byte) error {
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return err
	}
	schema, err := v.load(schemaName)
	if err != nil {
		return err
	}
	return schema.Validate(value)
}

func (v *Validator) load(schemaName string) (*jsonschema.Schema, error) {
	v.once.Do(v.initialize)
	if v.initErr != nil {
		return nil, v.initErr
	}

	v.mu.RLock()
	if schema := v.schemas[schemaName]; schema != nil {
		v.mu.RUnlock()
		return schema, nil
	}
	v.mu.RUnlock()

	v.mu.Lock()
	defer v.mu.Unlock()
	if schema := v.schemas[schemaName]; schema != nil {
		return schema, nil
	}

	schema, err := v.compiler.Compile("file:///schemas/" + schemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema %s: %w", schemaName, err)
	}
	v.schemas[schemaName] = schema
	return schema, nil
}

func (v *Validator) initialize() {
	compiler := jsonschema.NewCompiler()
	err := fs.WalkDir(schemaFiles, "schemas", func(fileName string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			return nil
		}
		contents, err := schemaFiles.ReadFile(fileName)
		if err != nil {
			return err
		}
		var schemaDocument any
		if err := json.Unmarshal(contents, &schemaDocument); err != nil {
			return fmt.Errorf("decode %s: %w", fileName, err)
		}
		return compiler.AddResource("file:///"+path.Clean(fileName), schemaDocument)
	})
	if err != nil {
		v.initErr = fmt.Errorf("load embedded request schemas: %w", err)
		return
	}
	v.compiler = compiler
}

func normalizeContentType(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && mediaType != "" {
		return strings.ToLower(mediaType)
	}
	return strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
}

func schemaError(resource Resource, contentType string, err error) error {
	var validationErr *jsonschema.ValidationError
	if !errors.As(err, &validationErr) {
		return err
	}

	printer := message.NewPrinter(language.English)
	violations := make([]Violation, 0)
	collectViolations(validationErr, printer, &violations)
	violations = uniqueViolations(violations)
	if len(violations) == 0 {
		violations = []Violation{{Path: "$", Message: "does not match the resource schema"}}
	}
	return &Error{Resource: resource, ContentType: contentType, Details: violations}
}

func collectViolations(err *jsonschema.ValidationError, printer *message.Printer, out *[]Violation) {
	if err == nil {
		return
	}
	if len(err.Causes) == 0 {
		path := dottedPath(err.InstanceLocation)
		message := err.ErrorKind.LocalizedString(printer)
		path = requiredFieldPath(path, message)
		*out = append(*out, Violation{Path: path, Message: message})
		return
	}
	// oneOf/anyOf include every failed alternative. Return only the closest
	// alternative so callers see the concrete bad field rather than the
	// implementation details of every incompatible representation.
	switch err.ErrorKind.(type) {
	case *kind.OneOf, *kind.AnyOf:
		collectViolations(bestUnionCause(err.Causes), printer, out)
		return
	}
	for _, cause := range err.Causes {
		collectViolations(cause, printer, out)
	}
}

func bestUnionCause(causes []*jsonschema.ValidationError) *jsonschema.ValidationError {
	if len(causes) == 0 {
		return nil
	}
	best := causes[0]
	bestLeaves, bestDepth := errorShape(best)
	for _, candidate := range causes[1:] {
		leaves, depth := errorShape(candidate)
		if leaves < bestLeaves || (leaves == bestLeaves && depth > bestDepth) {
			best = candidate
			bestLeaves, bestDepth = leaves, depth
		}
	}
	return best
}

func errorShape(err *jsonschema.ValidationError) (leaves, deepestPath int) {
	if err == nil {
		return 0, 0
	}
	if len(err.Causes) == 0 {
		return 1, len(err.InstanceLocation)
	}
	for _, cause := range err.Causes {
		childLeaves, childDepth := errorShape(cause)
		leaves += childLeaves
		if childDepth > deepestPath {
			deepestPath = childDepth
		}
	}
	return leaves, deepestPath
}

func dottedPath(parts []string) string {
	if len(parts) == 0 {
		return "$"
	}
	var out strings.Builder
	for _, part := range parts {
		if _, err := fmt.Sscanf(part, "%d", new(int)); err == nil {
			out.WriteString("[")
			out.WriteString(part)
			out.WriteString("]")
			continue
		}
		if out.Len() > 0 {
			out.WriteByte('.')
		}
		out.WriteString(part)
	}
	return out.String()
}

func requiredFieldPath(path, message string) string {
	for _, marker := range []string{"missing property '", "missing properties "} {
		idx := strings.Index(message, marker)
		if idx < 0 {
			continue
		}
		rest := message[idx+len(marker):]
		if marker == "missing properties " {
			rest = strings.TrimSpace(rest)
			if len(rest) == 0 || rest[0] != '\'' {
				continue
			}
			rest = rest[1:]
		}
		if end := strings.IndexByte(rest, '\''); end >= 0 {
			field := rest[:end]
			if path == "$" {
				return field
			}
			return path + "." + field
		}
	}
	return path
}

func uniqueViolations(in []Violation) []Violation {
	seen := make(map[string]struct{}, len(in))
	out := make([]Violation, 0, len(in))
	for _, violation := range in {
		key := violation.Path + "\x00" + violation.Message
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, violation)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].Message < out[j].Message
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func validateRepresentation(resource Resource, contentType string, value any) error {
	if resource == Procedure && contentType == geoJSONContentType {
		return &Error{
			Resource: resource, ContentType: contentType,
			Details: []Violation{{
				Path:    "Content-Type",
				Message: "application/geo+json cannot represent the required SensorML field 'type'; use application/sml+json",
			}},
		}
	}

	if resource != Procedure && resource != System {
		return nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return semanticError(resource, contentType, "$", "must be a JSON object")
	}

	if contentType == sensorMLContentType {
		if processType, exists := object["type"]; exists && !validProcessType(processType) {
			return semanticError(resource, contentType, "type", "must be one of SimpleProcess, AggregateProcess, PhysicalComponent, or PhysicalSystem")
		}
		if definition, exists := object["definition"]; exists && !validDefinition(resource, definition) {
			return semanticError(resource, contentType, "definition", allowedDefinitionsMessage(resource))
		}
		if resource == System {
			for _, rawClassifier := range rawObjects(object["classifiers"]) {
				if rawClassifier.value["definition"] != "cs:AssetType" {
					continue
				}
				if !validAssetType(rawClassifier.value["value"]) {
					return semanticError(resource, contentType, fmt.Sprintf("classifiers[%d].value", rawClassifier.index), "must be one of Equipment, Human, LivingThing, Simulation, Process, Group, or Other")
				}
			}
		}
	}

	if resource == System && contentType == geoJSONContentType {
		if properties, ok := object["properties"].(map[string]any); ok {
			if definition, exists := properties["featureType"]; exists && !validDefinition(resource, definition) {
				return semanticError(resource, contentType, "properties.featureType", allowedDefinitionsMessage(resource))
			}
			if assetType, exists := properties["assetType"]; exists && !validAssetType(assetType) {
				return semanticError(resource, contentType, "properties.assetType", "must be one of Equipment, Human, LivingThing, Simulation, Process, Group, or Other")
			}
		}
	}
	return nil
}

func validDefinition(resource Resource, value any) bool {
	definition, ok := value.(string)
	if !ok {
		return false
	}
	allowed := map[Resource]map[string]struct{}{
		System: {
			"http://www.w3.org/ns/sosa/Sensor":   {},
			"http://www.w3.org/ns/sosa/Actuator": {},
			"http://www.w3.org/ns/sosa/Sampler":  {},
			"http://www.w3.org/ns/sosa/Platform": {},
			"http://www.w3.org/ns/sosa/System":   {},
			"sosa:Sensor":                        {},
			"sosa:Actuator":                      {},
			"sosa:Sampler":                       {},
			"sosa:Platform":                      {},
			"sosa:System":                        {},
		},
		Procedure: {
			"http://www.w3.org/ns/sosa/Procedure":          {},
			"http://www.w3.org/ns/sosa/ObservingProcedure": {},
			"http://www.w3.org/ns/sosa/SamplingProcedure":  {},
			"http://www.w3.org/ns/sosa/ActuatingProcedure": {},
			"http://www.w3.org/ns/sosa/System":             {},
			"http://www.w3.org/ns/sosa/Sensor":             {},
			"http://www.w3.org/ns/sosa/Actuator":           {},
			"http://www.w3.org/ns/sosa/Sampler":            {},
			"http://www.w3.org/ns/sosa/Platform":           {},
			"sosa:Procedure":                               {},
			"sosa:ObservingProcedure":                      {},
			"sosa:SamplingProcedure":                       {},
			"sosa:ActuatingProcedure":                      {},
			"sosa:System":                                  {},
			"sosa:Sensor":                                  {},
			"sosa:Actuator":                                {},
			"sosa:Sampler":                                 {},
			"sosa:Platform":                                {},
		},
	}
	_, ok = allowed[resource][definition]
	return ok
}

func allowedDefinitionsMessage(resource Resource) string {
	if resource == System {
		return "must be a supported SOSA system type (Sensor, Actuator, Sampler, Platform, or System)"
	}
	return "must be a supported SOSA procedure type"
}

func semanticError(resource Resource, contentType, path, message string) error {
	return &Error{Resource: resource, ContentType: contentType, Details: []Violation{{Path: path, Message: message}}}
}

func validProcessType(value any) bool {
	valueString, ok := value.(string)
	if !ok {
		return false
	}
	switch valueString {
	case "SimpleProcess", "AggregateProcess", "PhysicalComponent", "PhysicalSystem":
		return true
	default:
		return false
	}
}

func validAssetType(value any) bool {
	valueString, ok := value.(string)
	if !ok {
		return false
	}
	switch valueString {
	case "Equipment", "Human", "LivingThing", "Simulation", "Process", "Group", "Other":
		return true
	default:
		return false
	}
}

type indexedObject struct {
	index int
	value map[string]any
}

func rawObjects(value any) []indexedObject {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]indexedObject, 0, len(values))
	for index, value := range values {
		if object, ok := value.(map[string]any); ok {
			result = append(result, indexedObject{index: index, value: object})
		}
	}
	return result
}
