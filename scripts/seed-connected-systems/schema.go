package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"regexp"
	"strings"
	"time"

	"github.com/emicklei/proto"
)

// StreamSchema mirrors the JSON fields that the Part 2 schema endpoint
// accepts. It is deliberately small but supports all component trees that the
// server validates for JSON-shaped observations and commands.
type StreamSchema struct {
	ObsFormat        string          `json:"obsFormat,omitempty"`
	CommandFormat    string          `json:"commandFormat,omitempty"`
	ResultSchema     *DataComponent  `json:"resultSchema,omitempty"`
	ParametersSchema *DataComponent  `json:"parametersSchema,omitempty"`
	RecordSchema     *DataComponent  `json:"recordSchema,omitempty"`
	Encoding         map[string]any  `json:"encoding,omitempty"`
	MessageSchema    json.RawMessage `json:"messageSchema,omitempty"`
}

type DataComponent struct {
	ID          string           `json:"id,omitempty"`
	Name        string           `json:"name,omitempty"`
	Type        string           `json:"type,omitempty"`
	Label       string           `json:"label,omitempty"`
	Description string           `json:"description,omitempty"`
	Definition  string           `json:"definition,omitempty"`
	Optional    *bool            `json:"optional,omitempty"`
	Fields      []NamedComponent `json:"fields,omitempty"`
	Coordinates []NamedComponent `json:"coordinates,omitempty"`
	Items       []NamedComponent `json:"items,omitempty"`
	ElementType *NamedComponent  `json:"elementType,omitempty"`
	Constraint  *DataConstraint  `json:"constraint,omitempty"`
	NilValues   []NilValue       `json:"nilValues,omitempty"`
	UOM         *DataUOM         `json:"uom,omitempty"`
	SRS         string           `json:"srs,omitempty"`
}

type DataUOM struct {
	Code string `json:"code,omitempty"`
	Href string `json:"href,omitempty"`
}

type NamedComponent struct {
	Name string `json:"name"`
	DataComponent
}

type DataConstraint struct {
	Intervals          json.RawMessage `json:"intervals,omitempty"`
	Values             json.RawMessage `json:"values,omitempty"`
	Pattern            string          `json:"pattern,omitempty"`
	SignificantFigures *int            `json:"significantFigures,omitempty"`
}

type NilValue struct {
	Reason string          `json:"reason,omitempty"`
	Value  json.RawMessage `json:"value,omitempty"`
}

func (s StreamSchema) ResultComponent() *DataComponent {
	if s.ResultSchema != nil {
		return s.ResultSchema
	}
	return s.RecordSchema
}

func (s StreamSchema) ParameterComponent() *DataComponent {
	if s.ParametersSchema != nil {
		return s.ParametersSchema
	}
	return s.RecordSchema
}

func generateSchemaValue(rng *rand.Rand, component *DataComponent) (any, error) {
	if component == nil {
		return nil, fmt.Errorf("schema has no result component")
	}
	typ := strings.ToLower(component.Type)
	if typ == "" && len(component.Fields) > 0 {
		typ = "datarecord"
	}
	switch typ {
	case "datarecord":
		value := make(map[string]any, len(component.Fields))
		for _, field := range component.Fields {
			if field.Name == "" {
				return nil, fmt.Errorf("DataRecord field has no name")
			}
			if field.Optional != nil && *field.Optional && rng.IntN(5) == 0 {
				continue
			}
			fieldValue, err := generateSchemaValue(rng, &field.DataComponent)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w", field.Name, err)
			}
			value[field.Name] = fieldValue
		}
		return value, nil
	case "vector":
		value := make(map[string]any, len(component.Coordinates))
		for _, coordinate := range component.Coordinates {
			if coordinate.Name == "" {
				return nil, fmt.Errorf("Vector coordinate has no name")
			}
			coordinateValue, err := generateSchemaValue(rng, &coordinate.DataComponent)
			if err != nil {
				return nil, fmt.Errorf("coordinate %s: %w", coordinate.Name, err)
			}
			value[coordinate.Name] = coordinateValue
		}
		return value, nil
	case "dataarray", "matrix":
		if component.ElementType == nil {
			return nil, fmt.Errorf("%s has no elementType", component.Type)
		}
		length := 1 + rng.IntN(3)
		value := make([]any, 0, length)
		for range length {
			element, err := generateSchemaValue(rng, &component.ElementType.DataComponent)
			if err != nil {
				return nil, err
			}
			value = append(value, element)
		}
		return value, nil
	case "datachoice":
		if len(component.Items) == 0 {
			return nil, fmt.Errorf("DataChoice has no items")
		}
		for _, item := range rng.Perm(len(component.Items)) {
			value, err := generateSchemaValue(rng, &component.Items[item].DataComponent)
			if err == nil {
				return value, nil
			}
		}
		return nil, fmt.Errorf("no DataChoice item can be generated")
	case "geometry":
		return map[string]any{"type": "Point", "coordinates": []float64{-117.1611 + rng.Float64()/100, 32.7157 + rng.Float64()/100}}, nil
	case "boolean":
		if value, ok := constrainedValue(rng, component.Constraint); ok {
			if boolean, ok := value.(bool); ok {
				return boolean, nil
			}
		}
		return rng.IntN(2) == 0, nil
	case "count":
		return generatedNumber(rng, component.Constraint, true)
	case "quantity":
		return generatedNumber(rng, component.Constraint, false)
	case "countrange", "quantityrange":
		first, err := generatedNumber(rng, component.Constraint, typ == "countrange")
		if err != nil {
			return nil, err
		}
		second, err := generatedNumber(rng, component.Constraint, typ == "countrange")
		if err != nil {
			return nil, err
		}
		if first.(float64) > second.(float64) {
			first, second = second, first
		}
		return []any{first, second}, nil
	case "time":
		if value, ok := constrainedValue(rng, component.Constraint); ok {
			if text, ok := value.(string); ok {
				return text, nil
			}
		}
		return time.Now().UTC().Format(time.RFC3339), nil
	case "timerange":
		now := time.Now().UTC().Truncate(time.Second)
		return []string{now.Add(-time.Minute).Format(time.RFC3339), now.Format(time.RFC3339)}, nil
	case "category", "text":
		return generatedText(rng, component.Constraint)
	case "categoryrange":
		value, err := generatedText(rng, component.Constraint)
		if err != nil {
			return nil, err
		}
		return []string{value.(string), value.(string)}, nil
	default:
		return nil, fmt.Errorf("unsupported data component type %q", component.Type)
	}
}

func constrainedValue(rng *rand.Rand, constraint *DataConstraint) (any, bool) {
	if constraint == nil || len(constraint.Values) == 0 {
		return nil, false
	}
	var values []any
	if json.Unmarshal(constraint.Values, &values) != nil || len(values) == 0 {
		return nil, false
	}
	return values[rng.IntN(len(values))], true
}

func generatedNumber(rng *rand.Rand, constraint *DataConstraint, integer bool) (any, error) {
	if value, ok := constrainedValue(rng, constraint); ok {
		number, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf("numeric constraint values contain %T", value)
		}
		if integer {
			number = math.Round(number)
		}
		return number, nil
	}
	minimum, maximum := 0.0, 100.0
	if constraint != nil && len(constraint.Intervals) > 0 {
		var intervals [][]float64
		if err := json.Unmarshal(constraint.Intervals, &intervals); err != nil || len(intervals) == 0 || len(intervals[0]) != 2 {
			return nil, fmt.Errorf("numeric intervals are not a non-empty two-value array")
		}
		minimum, maximum = intervals[0][0], intervals[0][1]
		if maximum < minimum {
			return nil, fmt.Errorf("numeric interval maximum is before minimum")
		}
	}
	value := minimum + rng.Float64()*(maximum-minimum)
	if integer {
		value = math.Round(value)
	}
	if constraint != nil && constraint.SignificantFigures != nil && *constraint.SignificantFigures > 0 && value != 0 {
		value = roundSignificant(value, *constraint.SignificantFigures)
	}
	return value, nil
}

func roundSignificant(value float64, figures int) float64 {
	scale := math.Pow(10, float64(figures)-math.Floor(math.Log10(math.Abs(value)))-1)
	return math.Round(value*scale) / scale
}

func generatedText(rng *rand.Rand, constraint *DataConstraint) (any, error) {
	if value, ok := constrainedValue(rng, constraint); ok {
		if text, ok := value.(string); ok {
			return text, nil
		}
		return nil, fmt.Errorf("text constraint values contain %T", value)
	}
	if constraint == nil || constraint.Pattern == "" {
		return []string{"normal", "active", "stable", "seeded"}[rng.IntN(4)], nil
	}
	pattern, err := regexp.Compile(constraint.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid text pattern: %w", err)
	}
	for _, candidate := range []string{"seeded", "normal", "active", "ABC-1234", "station-001", "ok"} {
		if pattern.MatchString(candidate) {
			return candidate, nil
		}
	}
	return nil, fmt.Errorf("cannot generate a value for pattern %q", constraint.Pattern)
}

func generateProtobufValue(rng *rand.Rand, raw json.RawMessage) (any, error) {
	var inline string
	if err := json.Unmarshal(raw, &inline); err != nil {
		var wrapped struct {
			Inline string `json:"inline"`
		}
		if wrappedErr := json.Unmarshal(raw, &wrapped); wrappedErr != nil || wrapped.Inline == "" {
			return nil, fmt.Errorf("protobuf messageSchema must be an inline schema string")
		}
		inline = wrapped.Inline
	}
	definition, err := proto.NewParser(strings.NewReader(inline)).Parse()
	if err != nil {
		return nil, fmt.Errorf("parse protobuf schema: %w", err)
	}
	for _, element := range definition.Elements {
		if message, ok := element.(*proto.Message); ok {
			return generateProtoMessage(rng, message)
		}
	}
	return nil, fmt.Errorf("protobuf schema defines no message")
}

func generateProtoMessage(rng *rand.Rand, message *proto.Message) (map[string]any, error) {
	value := make(map[string]any)
	nested := map[string]*proto.Message{}
	for _, element := range message.Elements {
		if child, ok := element.(*proto.Message); ok {
			nested[child.Name] = child
		}
	}
	for _, element := range message.Elements {
		field, ok := element.(*proto.NormalField)
		if !ok {
			continue
		}
		fieldValue, err := generateProtoFieldValue(rng, field.Type, nested)
		if err != nil {
			return nil, err
		}
		if field.Repeated {
			fieldValue = []any{fieldValue}
		}
		value[field.Name] = fieldValue
	}
	return value, nil
}

func generateProtoFieldValue(rng *rand.Rand, typ string, nested map[string]*proto.Message) (any, error) {
	if message, ok := nested[typ]; ok {
		return generateProtoMessage(rng, message)
	}
	switch strings.ToLower(typ) {
	case "double", "float":
		return 10 + rng.Float64()*90, nil
	case "int32", "sint32", "sfixed32", "fixed32", "uint32", "int64", "sint64", "sfixed64", "fixed64", "uint64":
		return float64(rng.IntN(100)), nil
	case "bool":
		return rng.IntN(2) == 0, nil
	case "string", "bytes":
		return "seeded", nil
	default:
		return nil, fmt.Errorf("unsupported protobuf field type %q", typ)
	}
}

func resultForSchema(rng *rand.Rand, schema StreamSchema) (any, error) {
	if component := schema.ResultComponent(); component != nil {
		return generateSchemaValue(rng, component)
	}
	if len(schema.MessageSchema) > 0 {
		return generateProtobufValue(rng, schema.MessageSchema)
	}
	return nil, fmt.Errorf("schema has no supported result or protobuf message")
}

func parametersForSchema(rng *rand.Rand, schema StreamSchema) (any, error) {
	component := schema.ParameterComponent()
	if component == nil {
		return nil, fmt.Errorf("schema has no supported parameters component")
	}
	return generateSchemaValue(rng, component)
}
