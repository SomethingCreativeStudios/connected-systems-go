// Package queryables keeps the public CQL2 vocabulary and the private SQL
// mapping in one place.  CQL property names are never treated as SQL
// identifiers: every name below is deliberately mapped to a known column or
// application-authored expression.
package queryables

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	gocql2 "github.com/cwygoda/gocql2"
	"github.com/cwygoda/gocql2/api"
	cql2sql "github.com/cwygoda/gocql2/sql"
)

type Resource string

const (
	Systems          Resource = "systems"
	Deployments      Resource = "deployments"
	Procedures       Resource = "procedures"
	SamplingFeatures Resource = "samplingFeatures"
	Properties       Resource = "properties"
	Features         Resource = "features"
	Datastreams      Resource = "datastreams"
	Observations     Resource = "observations"
	ControlStreams   Resource = "controlstreams"
	Commands         Resource = "commands"
	CommandStatuses  Resource = "commandStatuses"
	CommandResults   Resource = "commandResults"
	SystemEvents     Resource = "systemEvents"
	SystemHistory    Resource = "systemHistory"
)

const (
	CQL2Text = "cql2-text"
	CQL2JSON = "cql2-json"
)

var ErrUnsupportedLanguage = errors.New("filter-lang must be cql2-text or cql2-json")

// ConformanceClasses are both configured in the parser and advertised by the
// API.  Custom functions are intentionally absent: CASEI and ACCENTI are the
// standard functions enabled by their respective CQL2 conformance classes.
var ConformanceClasses = []string{
	api.ConformanceBasicCQL2,
	api.ConformanceAdvancedComparisonOperators,
	api.ConformanceCaseInsensitiveComparison,
	api.ConformanceAccentInsensitiveComparison,
	api.ConformanceSpatialFunctions,
	api.ConformanceTemporalFunctions,
	api.ConformanceArrayFunctions,
	api.ConformancePropertyProperty,
	api.ConformanceArithmetic,
	api.ConformanceCQL2Text,
	api.ConformanceCQL2JSON,
}

type Compiled struct {
	Text string
	Args []any
}

type contextKey struct{}

// WithCompiled attaches the middleware-validated filter to the request.  It
// avoids parsing twice and lets every repository apply exactly the same ANDed
// predicate as the legacy endpoint filters.
func WithCompiled(r *http.Request, filter Compiled) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), contextKey{}, filter))
}

func FromRequest(r *http.Request) Compiled {
	if r == nil {
		return Compiled{}
	}
	filter, _ := r.Context().Value(contextKey{}).(Compiled)
	return filter
}

// Compile parses either registered CQL2 encoding and produces only a
// parameterized PostGIS fragment. Unknown properties, wrong operand types and
// unsupported language values are rejected before a repository sees them.
func Compile(resource Resource, filter, language string) (Compiled, error) {
	properties, ok := PropertiesFor(resource)
	if !ok {
		return Compiled{}, fmt.Errorf("resource %q does not support CQL2 filtering", resource)
	}
	return compile(properties, intervalColumnsFor(resource), filter, language)
}

// CompileWithCatalog extends the generic Feature catalog for exactly one
// collection. Callers must validate and persist the catalog before using it.
func CompileWithCatalog(resource Resource, catalog map[string]any, filter, language string) (Compiled, error) {
	if resource != Features {
		return Compile(resource, filter, language)
	}
	if err := ValidateCustomCatalog(catalog); err != nil {
		return Compiled{}, err
	}
	properties, _ := PropertiesFor(resource)
	properties = append(properties, CatalogProperties(catalog)...)
	return compile(properties, intervalColumnsFor(resource), filter, language)
}

func compile(properties []cql2sql.Property, intervals map[string]intervalColumns, filter, language string) (Compiled, error) {
	if strings.TrimSpace(filter) == "" {
		return Compiled{}, nil
	}
	if language == "" {
		language = CQL2Text
	}
	if language != CQL2Text && language != CQL2JSON {
		return Compiled{}, ErrUnsupportedLanguage
	}

	parser := gocql2.NewParser().
		WithConformance(ConformanceClasses...).
		WithAllowedProperties(cql2sql.PropertyDefinitions(properties...)...).
		WithMaxDepth(32)

	var (
		expr api.Expression
		err  error
	)
	if language == CQL2JSON {
		expr, err = parser.ParseJSON([]byte(filter))
	} else {
		expr, err = parser.ParseText(filter)
	}
	if err != nil {
		return Compiled{}, err
	}
	where, err := cql2sql.ToSQL(expr, intervalDialect{Dialect: cql2sql.PostGISDialect(cql2sql.WithSRID(4326)), intervals: intervals}, cql2sql.WithSQLProperties(properties...))
	if err != nil {
		return Compiled{}, err
	}
	return Compiled{Text: where.Text, Args: where.Args}, nil
}

// PropertiesFor is intentionally a curated operational catalog rather than a
// reflection of every storage column. It is used for both queryables schemas
// and the fail-closed compiler map.
func PropertiesFor(resource Resource) ([]cql2sql.Property, bool) {
	table, common := tableFor(resource)
	if table == "" {
		return nil, false
	}
	props := make([]cql2sql.Property, 0, 12)
	add := func(name string, typ api.PropertyType, column string) {
		props = append(props, cql2sql.Property{Name: name, Type: typ, Expr: cql2sql.Column(table, column)})
	}
	addRaw := func(name string, typ api.PropertyType, expression string) {
		props = append(props, cql2sql.Property{Name: name, Type: typ, Expr: cql2sql.RawSQL(expression)})
	}
	add("id", api.PropertyTypeString, "id")
	if common {
		add("uid", api.PropertyTypeString, "unique_identifier")
		add("name", api.PropertyTypeString, "name")
		add("description", api.PropertyTypeString, "description")
	}

	switch resource {
	case Systems:
		add("parent", api.PropertyTypeString, "parent_system_id")
		addRaw("procedures", api.PropertyTypeArray, `ARRAY(SELECT sp.procedure_id FROM system_procedures AS sp WHERE sp.system_id = "systems"."id")`)
		add("geometry", api.PropertyTypeGeometry, "geometry")
		add("validTimeStart", api.PropertyTypeTimestamp, "valid_time_start")
		add("validTimeEnd", api.PropertyTypeTimestamp, "valid_time_end")
		add("validTime", api.PropertyTypeInterval, "valid_time_start")
	case Deployments:
		add("parent", api.PropertyTypeString, "parent_deployment_id")
		add("system", api.PropertyTypeString, "platform_id")
		add("geometry", api.PropertyTypeGeometry, "geometry")
		add("validTimeStart", api.PropertyTypeTimestamp, "valid_time_start")
		add("validTimeEnd", api.PropertyTypeTimestamp, "valid_time_end")
		add("validTime", api.PropertyTypeInterval, "valid_time_start")
	case Procedures:
		add("validTimeStart", api.PropertyTypeTimestamp, "valid_time_start")
		add("validTimeEnd", api.PropertyTypeTimestamp, "valid_time_end")
		add("validTime", api.PropertyTypeInterval, "valid_time_start")
	case SamplingFeatures:
		add("system", api.PropertyTypeString, "parent_system_id")
		add("sampledFeature", api.PropertyTypeString, "sampled_feature_id")
		add("geometry", api.PropertyTypeGeometry, "geometry")
		add("validTimeStart", api.PropertyTypeTimestamp, "valid_time_start")
		add("validTimeEnd", api.PropertyTypeTimestamp, "valid_time_end")
		add("validTime", api.PropertyTypeInterval, "valid_time_start")
	case Properties:
		add("propertyType", api.PropertyTypeString, "property_type")
		add("baseProperty", api.PropertyTypeString, "base_property")
	case Features:
		add("collectionId", api.PropertyTypeString, "collection_id")
		add("dateTime", api.PropertyTypeTimestamp, "date_time")
		add("validTimeStart", api.PropertyTypeTimestamp, "valid_time_start")
		add("validTimeEnd", api.PropertyTypeTimestamp, "valid_time_end")
		add("validTime", api.PropertyTypeInterval, "valid_time_start")
		add("geometry", api.PropertyTypeGeometry, "geometry")
	case Datastreams:
		add("system", api.PropertyTypeString, "system_id")
		add("procedure", api.PropertyTypeString, "procedure_id")
		add("deployment", api.PropertyTypeString, "deployment_id")
		add("featureOfInterest", api.PropertyTypeString, "feature_of_interest_id")
		add("samplingFeature", api.PropertyTypeString, "sampling_feature_id")
		add("validTime", api.PropertyTypeInterval, "valid_time_start")
		add("phenomenonTimeStart", api.PropertyTypeTimestamp, "phenomenon_time_start")
		add("phenomenonTimeEnd", api.PropertyTypeTimestamp, "phenomenon_time_end")
		add("phenomenonTime", api.PropertyTypeInterval, "phenomenon_time_start")
		add("resultTimeStart", api.PropertyTypeTimestamp, "result_time_start")
		add("resultTimeEnd", api.PropertyTypeTimestamp, "result_time_end")
		add("resultTime", api.PropertyTypeInterval, "result_time_start")
	case Observations:
		add("datastream", api.PropertyTypeString, "datastream_id")
		add("samplingFeature", api.PropertyTypeString, "sampling_feature_id")
		add("phenomenonTime", api.PropertyTypeTimestamp, "phenomenon_time")
		add("resultTime", api.PropertyTypeTimestamp, "result_time")
	case ControlStreams:
		add("system", api.PropertyTypeString, "system_id")
		add("procedure", api.PropertyTypeString, "procedure_id")
		add("deployment", api.PropertyTypeString, "deployment_id")
		add("featureOfInterest", api.PropertyTypeString, "feature_of_interest_id")
		add("samplingFeature", api.PropertyTypeString, "sampling_feature_id")
		add("validTime", api.PropertyTypeInterval, "valid_time_start")
		add("issueTimeStart", api.PropertyTypeTimestamp, "issue_time_start")
		add("issueTimeEnd", api.PropertyTypeTimestamp, "issue_time_end")
		add("issueTime", api.PropertyTypeInterval, "issue_time_start")
	case Commands:
		add("controlStream", api.PropertyTypeString, "control_stream_id")
		add("samplingFeature", api.PropertyTypeString, "sampling_feature_id")
		add("issueTime", api.PropertyTypeTimestamp, "issue_time")
		add("executionTimeStart", api.PropertyTypeTimestamp, "execution_time_start")
		add("executionTimeEnd", api.PropertyTypeTimestamp, "execution_time_end")
		add("executionTime", api.PropertyTypeInterval, "execution_time_start")
		add("currentStatus", api.PropertyTypeString, "current_status")
	case CommandStatuses:
		add("command", api.PropertyTypeString, "command_id")
		add("reportTime", api.PropertyTypeTimestamp, "report_time")
		add("statusCode", api.PropertyTypeString, "status_code")
		add("percentCompletion", api.PropertyTypeNumber, "percent_completion")
		add("executionTime", api.PropertyTypeInterval, "execution_time_start")
	case CommandResults:
		add("command", api.PropertyTypeString, "command_id")
	case SystemEvents:
		add("system", api.PropertyTypeString, "system_id")
		add("eventType", api.PropertyTypeString, "definition")
		add("timeStart", api.PropertyTypeTimestamp, "time_start")
		add("timeEnd", api.PropertyTypeTimestamp, "time_end")
		add("time", api.PropertyTypeInterval, "time_start")
	case SystemHistory:
		add("system", api.PropertyTypeString, "system_id")
		add("validTimeStart", api.PropertyTypeTimestamp, "valid_time_start")
		add("validTimeEnd", api.PropertyTypeTimestamp, "valid_time_end")
		add("validTime", api.PropertyTypeInterval, "valid_time_start")
	}

	return props, true
}

func intervalColumnsFor(resource Resource) map[string]intervalColumns {
	table, _ := tableFor(resource)
	qualified := func(start, end string) intervalColumns {
		return intervalColumns{start: `"` + table + `"."` + start + `"`, end: `"` + table + `"."` + end + `"`}
	}
	switch resource {
	case Systems, Deployments, Procedures, SamplingFeatures, Features, SystemHistory:
		return map[string]intervalColumns{"validTime": qualified("valid_time_start", "valid_time_end")}
	case Datastreams:
		return map[string]intervalColumns{"validTime": qualified("valid_time_start", "valid_time_end"), "phenomenonTime": qualified("phenomenon_time_start", "phenomenon_time_end"), "resultTime": qualified("result_time_start", "result_time_end")}
	case ControlStreams:
		return map[string]intervalColumns{"validTime": qualified("valid_time_start", "valid_time_end"), "issueTime": qualified("issue_time_start", "issue_time_end"), "executionTime": qualified("execution_time_start", "execution_time_end")}
	case Commands, CommandStatuses:
		return map[string]intervalColumns{"executionTime": qualified("execution_time_start", "execution_time_end")}
	case SystemEvents:
		return map[string]intervalColumns{"time": qualified("time_start", "time_end")}
	default:
		return nil
	}
}

func tableFor(resource Resource) (table string, common bool) {
	switch resource {
	case Systems:
		return "systems", true
	case Deployments:
		return "deployments", true
	case Procedures:
		return "procedures", true
	case SamplingFeatures:
		return "sampling_features", true
	case Properties:
		return "properties", true
	case Features:
		return "features", true
	case Datastreams:
		return "datastreams", true
	case Observations:
		return "observations", false
	case ControlStreams:
		return "control_streams", true
	case Commands:
		return "commands", false
	case CommandStatuses:
		return "command_status_reports", false
	case CommandResults:
		return "command_results", false
	case SystemEvents:
		return "system_events", false
	case SystemHistory:
		return "system_history_revisions", false
	default:
		return "", false
	}
}

// Schema is a JSON Schema 2020-12 catalog suitable for the OGC API Features
// Part 3 queryables resource. It rejects undeclared names by design.
func Schema(resource Resource, id string) (map[string]any, error) {
	props, ok := PropertiesFor(resource)
	if !ok {
		return nil, fmt.Errorf("unknown queryables resource %q", resource)
	}
	properties := make(map[string]any, len(props))
	for _, prop := range props {
		properties[prop.Name] = schemaForType(prop.Type)
	}
	return map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$id":                  id,
		"type":                 "object",
		"title":                string(resource) + " queryables",
		"properties":           properties,
		"additionalProperties": false,
	}, nil
}

// SchemaWithCatalog merges a validated generic Feature catalog into the
// built-in safe fields. It is intentionally only used for the collection that
// owns the catalog, so a property declaration never leaks between collections.
func SchemaWithCatalog(resource Resource, id string, catalog map[string]any) (map[string]any, error) {
	if err := ValidateCustomCatalog(catalog); err != nil {
		return nil, err
	}
	schema, err := Schema(resource, id)
	if err != nil {
		return nil, err
	}
	properties := schema["properties"].(map[string]any)
	for name, value := range catalogProperties(catalog) {
		properties[name] = value
	}
	return schema, nil
}

var queryableName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,62}$`)

// CatalogOrDefault gives pre-queryables Feature collections the closed built-in
// catalog rather than turning an upgrade into an accidental arbitrary-JSON
// query surface. New collections are still required to declare a catalog.
func CatalogOrDefault(catalog map[string]any) map[string]any {
	if catalog != nil {
		return catalog
	}
	return map[string]any{"properties": map[string]any{}, "additionalProperties": false}
}

// ValidateCustomCatalog accepts a narrow, indexable JSON Schema property map.
// Nested object paths, union types and arbitrary JSON are deliberately
// rejected because they cannot be given a stable type-safe SQL expression.
func ValidateCustomCatalog(catalog map[string]any) error {
	if catalog == nil {
		return errors.New("generic Feature collections require a queryables catalog")
	}
	if additional, ok := catalog["additionalProperties"].(bool); !ok || additional {
		return errors.New("queryables.additionalProperties must be false")
	}
	properties, ok := catalog["properties"].(map[string]any)
	if !ok {
		return errors.New("queryables.properties must be an object")
	}
	for name, raw := range properties {
		if !queryableName.MatchString(name) {
			return fmt.Errorf("invalid queryable name %q", name)
		}
		schema, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("queryable %q must be a schema object", name)
		}
		typ, _ := schema["type"].(string)
		switch typ {
		case "string", "number", "integer", "boolean":
		case "array":
			items, ok := schema["items"].(map[string]any)
			if !ok {
				return fmt.Errorf("array queryable %q requires scalar items", name)
			}
			itemType, _ := items["type"].(string)
			if itemType != "string" && itemType != "number" && itemType != "integer" && itemType != "boolean" {
				return fmt.Errorf("array queryable %q has unsupported item type", name)
			}
		default:
			return fmt.Errorf("queryable %q has unsupported type %q", name, typ)
		}
		for key := range schema {
			if key != "type" && key != "format" && key != "items" && key != "title" && key != "description" {
				return fmt.Errorf("queryable %q uses unsupported schema keyword %q", name, key)
			}
		}
	}
	return nil
}

func catalogProperties(catalog map[string]any) map[string]any {
	properties, _ := catalog["properties"].(map[string]any)
	return properties
}

// CatalogProperties creates SQL mappings only after catalog validation. Every
// expression is application-authored from a constrained property name; no CQL
// input is ever inserted into the SQL text.
func CatalogProperties(catalog map[string]any) []cql2sql.Property {
	properties := catalogProperties(catalog)
	out := make([]cql2sql.Property, 0, len(properties))
	for name, raw := range properties {
		schema, _ := raw.(map[string]any)
		typ, _ := schema["type"].(string)
		format, _ := schema["format"].(string)
		key := "'" + strings.ReplaceAll(name, "'", "''") + "'"
		expr := `"features"."properties" ->> ` + key
		propertyType := api.PropertyTypeString
		switch typ {
		case "string":
			if format == "date-time" {
				propertyType, expr = api.PropertyTypeTimestamp, "NULLIF("+expr+", '')::timestamptz"
			}
		case "number":
			propertyType, expr = api.PropertyTypeNumber, "NULLIF("+expr+", '')::numeric"
		case "integer":
			propertyType, expr = api.PropertyTypeInteger, "NULLIF("+expr+", '')::bigint"
		case "boolean":
			propertyType, expr = api.PropertyTypeBoolean, "NULLIF("+expr+", '')::boolean"
		case "array":
			propertyType = api.PropertyTypeArray
			expr = `cql2_jsonb_text_array("features"."properties" -> ` + key + ")"
		}
		out = append(out, cql2sql.Property{Name: name, Type: propertyType, Expr: cql2sql.RawSQL(expr)})
	}
	return out
}

// ValidateFeatureProperties prevents a declared custom queryable from being
// stored with a value that cannot be compiled and indexed consistently later.
// Undeclared feature properties remain allowed; they are simply not CQL2
// queryables.
func ValidateFeatureProperties(catalog, values map[string]any) error {
	if err := ValidateCustomCatalog(catalog); err != nil {
		return err
	}
	for name, raw := range catalogProperties(catalog) {
		value, present := values[name]
		if !present || value == nil {
			continue
		}
		schema := raw.(map[string]any)
		typ, _ := schema["type"].(string)
		format, _ := schema["format"].(string)
		if err := validateQueryableValue(name, typ, format, schema, value); err != nil {
			return err
		}
	}
	return nil
}

func validateQueryableValue(name, typ, format string, schema map[string]any, value any) error {
	switch typ {
	case "string":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("queryable %q must be a string", name)
		}
		if format == "date-time" {
			if _, err := time.Parse(time.RFC3339, text); err != nil {
				return fmt.Errorf("queryable %q must be an RFC3339 date-time", name)
			}
		}
	case "number":
		if !isNumber(value) {
			return fmt.Errorf("queryable %q must be a number", name)
		}
	case "integer":
		if !isInteger(value) {
			return fmt.Errorf("queryable %q must be an integer", name)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("queryable %q must be a boolean", name)
		}
	case "array":
		items, ok := asSlice(value)
		if !ok {
			return fmt.Errorf("queryable %q must be an array", name)
		}
		itemSchema := schema["items"].(map[string]any)
		itemType, _ := itemSchema["type"].(string)
		for _, item := range items {
			if err := validateQueryableValue(name, itemType, "", nil, item); err != nil {
				return err
			}
		}
	}
	return nil
}

func isNumber(value any) bool {
	switch value.(type) {
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}
func isInteger(value any) bool {
	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float64:
		return v == float64(int64(v))
	case float32:
		return v == float32(int64(v))
	default:
		return false
	}
}
func asSlice(value any) ([]any, bool) {
	switch v := value.(type) {
	case []any:
		return v, true
	case []string:
		out := make([]any, len(v))
		for i := range v {
			out[i] = v[i]
		}
		return out, true
	default:
		return nil, false
	}
}

func schemaForType(typ api.PropertyType) map[string]any {
	switch typ {
	case api.PropertyTypeNumber:
		return map[string]any{"type": "number"}
	case api.PropertyTypeInteger:
		return map[string]any{"type": "integer"}
	case api.PropertyTypeBoolean:
		return map[string]any{"type": "boolean"}
	case api.PropertyTypeTimestamp:
		return map[string]any{"type": "string", "format": "date-time"}
	case api.PropertyTypeDate:
		return map[string]any{"type": "string", "format": "date"}
	case api.PropertyTypeGeometry, api.PropertyTypePoint, api.PropertyTypePolygon, api.PropertyTypeMultiPolygon:
		return map[string]any{"$ref": "https://geojson.org/schema/Geometry.json"}
	case api.PropertyTypeArray:
		return map[string]any{"type": "array"}
	default:
		return map[string]any{"type": "string"}
	}
}
