package api

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/emicklei/proto"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

func validateObservationAgainstDatastreamSchema(obs *domains.Observation, ds *domains.Datastream, contentType string) error {
	if obs == nil || ds == nil {
		return nil
	}
	if ds.Schema == nil {
		return nil
	}

	obsFormat := strings.TrimSpace(strings.ToLower(ds.Schema.ObsFormat))
	ct := strings.ToLower(contentType)

	if obsFormat == "" {
		if strings.Contains(ct, "protobuf") {
			obsFormat = "application/x-protobuf"
		} else {
			obsFormat = "application/json"
		}
	}

	if strings.Contains(obsFormat, "protobuf") {
		return validateObservationResultWithProtobufSchema(obs, ds.Schema)
	}

	if strings.Contains(obsFormat, "json") || strings.Contains(obsFormat, "swe") || strings.Contains(ct, "json") {
		return validateObservationResultWithJSONLikeSchema(obs, ds.Schema)
	}

	// Other formats are accepted for now.
	return nil
}

func validateObservationResultWithJSONLikeSchema(obs *domains.Observation, schema *domains.DatastreamSchema) error {
	if obs.ResultLink != nil || len(obs.Result) == 0 {
		return nil
	}

	component := schema.ResultSchema
	if component == nil {
		component = schema.RecordSchema
	}
	if component == nil {
		return nil
	}

	var value any
	if err := json.Unmarshal(obs.Result, &value); err != nil {
		return fmt.Errorf("result is not valid JSON: %w", err)
	}

	if err := validateDataComponentValue(component, value, "result"); err != nil {
		return err
	}

	return nil
}

func validateDataComponentValue(component *domains.DatastreamDataComponent, value any, path string) error {
	if component == nil {
		return nil
	}

	// Infer a record shape if explicit type is not provided but fields exist.
	componentType := strings.ToLower(component.Type)
	if componentType == "" && len(component.Fields) > 0 {
		componentType = "datarecord"
	}

	switch componentType {
	case "datarecord":
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be an object", path)
		}
		for _, field := range component.Fields {
			if field.Name == "" {
				continue
			}
			fieldVal, exists := obj[field.Name]
			if !exists {
				if field.Optional != nil && *field.Optional {
					continue
				}
				return fmt.Errorf("%s.%s is required by datastream schema", path, field.Name)
			}
			if err := validateDataComponentValue(&field.DatastreamDataComponent, fieldVal, path+"."+field.Name); err != nil {
				return err
			}
		}
		return nil

	case "vector":
		// Vectors are commonly encoded as objects in this API.
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be an object for Vector schema", path)
		}
		for _, coord := range component.Coordinates {
			if coord.Name == "" {
				continue
			}
			coordVal, exists := obj[coord.Name]
			if !exists {
				if coord.Optional != nil && *coord.Optional {
					continue
				}
				return fmt.Errorf("%s.%s is required by datastream vector schema", path, coord.Name)
			}
			if err := validateDataComponentValue(&coord.DatastreamDataComponent, coordVal, path+"."+coord.Name); err != nil {
				return err
			}
		}
		return nil

	case "dataarray", "matrix":
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s must be an array", path)
		}
		if component.ElementType != nil {
			for i, item := range arr {
				if err := validateDataComponentValue(&component.ElementType.DatastreamDataComponent, item, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
		return nil

	case "datachoice":
		for _, item := range component.Items {
			if err := validateDataComponentValue(&item.DatastreamDataComponent, value, path); err == nil {
				return nil
			}
		}
		return fmt.Errorf("%s does not match any allowed DataChoice item", path)

	case "geometry":
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be a geometry object", path)
		}
		if _, ok := obj["type"]; !ok {
			return fmt.Errorf("%s.type is required for geometry", path)
		}
		if _, ok := obj["coordinates"]; !ok {
			return fmt.Errorf("%s.coordinates is required for geometry", path)
		}
		return nil

	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s must be a boolean", path)
		}
		return nil

	case "count":
		if !isIntegerNumber(value) {
			return fmt.Errorf("%s must be an integer", path)
		}
		return nil

	case "quantity":
		if !isNumber(value) {
			return fmt.Errorf("%s must be a number", path)
		}
		return nil

	case "time", "category", "text":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s must be a string", path)
		}
		return nil

	case "countrange", "quantityrange", "timerange", "categoryrange":
		arr, ok := value.([]any)
		if !ok || len(arr) != 2 {
			return fmt.Errorf("%s must be a 2-item array", path)
		}
		return nil

	default:
		// Unknown/extension component: accept.
		return nil
	}
}

func validateObservationResultWithProtobufSchema(obs *domains.Observation, schema *domains.DatastreamSchema) error {
	if obs.ResultLink != nil || len(obs.Result) == 0 {
		return nil
	}

	if schema.MessageSchema == nil || schema.MessageSchema.Inline == nil || strings.TrimSpace(*schema.MessageSchema.Inline) == "" {
		return fmt.Errorf("datastream protobuf schema is missing inline messageSchema")
	}

	// Parse the .proto schema using a protobuf parser library.
	parser := proto.NewParser(strings.NewReader(*schema.MessageSchema.Inline))
	definition, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("invalid protobuf messageSchema: %w", err)
	}

	message := firstMessage(definition)
	if message == nil {
		return fmt.Errorf("protobuf messageSchema must define at least one message")
	}

	var result any
	if err := json.Unmarshal(obs.Result, &result); err != nil {
		return fmt.Errorf("protobuf result must be JSON object in this API: %w", err)
	}

	obj, ok := result.(map[string]any)
	if !ok {
		return fmt.Errorf("protobuf result must be a JSON object")
	}

	if err := validateAgainstProtoMessage(message, obj, "result"); err != nil {
		return err
	}

	return nil
}

func firstMessage(definition *proto.Proto) *proto.Message {
	if definition == nil {
		return nil
	}
	for _, element := range definition.Elements {
		if msg, ok := element.(*proto.Message); ok {
			return msg
		}
	}
	return nil
}

func validateAgainstProtoMessage(msg *proto.Message, obj map[string]any, path string) error {
	if msg == nil {
		return nil
	}

	fields := map[string]*proto.NormalField{}
	mapFields := map[string]*proto.MapField{}
	nested := map[string]*proto.Message{}

	for _, element := range msg.Elements {
		switch e := element.(type) {
		case *proto.NormalField:
			fields[e.Name] = e
		case *proto.MapField:
			mapFields[e.Name] = e
		case *proto.Message:
			nested[e.Name] = e
		}
	}

	for name, field := range fields {
		val, exists := obj[name]
		if !exists {
			if field.Required {
				return fmt.Errorf("%s.%s is required by protobuf schema", path, name)
			}
			continue
		}

		if field.Repeated {
			arr, ok := val.([]any)
			if !ok {
				return fmt.Errorf("%s.%s must be an array (repeated field)", path, name)
			}
			for i, item := range arr {
				if err := validateProtoScalarOrMessage(field.Type, nested, item, fmt.Sprintf("%s.%s[%d]", path, name, i)); err != nil {
					return err
				}
			}
			continue
		}

		if err := validateProtoScalarOrMessage(field.Type, nested, val, path+"."+name); err != nil {
			return err
		}
	}

	for name := range mapFields {
		if val, exists := obj[name]; exists {
			if _, ok := val.(map[string]any); !ok {
				return fmt.Errorf("%s.%s must be an object (map field)", path, name)
			}
		}
	}

	return nil
}

func validateProtoScalarOrMessage(fieldType string, nested map[string]*proto.Message, value any, path string) error {
	lower := strings.ToLower(fieldType)

	if nestedMsg, ok := nested[fieldType]; ok {
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s must be an object for message type %s", path, fieldType)
		}
		return validateAgainstProtoMessage(nestedMsg, obj, path)
	}

	switch lower {
	case "double", "float":
		if !isNumber(value) {
			return fmt.Errorf("%s must be numeric", path)
		}
	case "int32", "sint32", "sfixed32", "fixed32", "uint32", "int64", "sint64", "sfixed64", "fixed64", "uint64":
		if !isIntegerNumber(value) {
			return fmt.Errorf("%s must be an integer", path)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s must be a boolean", path)
		}
	case "string", "bytes":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s must be a string", path)
		}
	default:
		// Unknown/custom type: allow object or scalar.
	}

	return nil
}

func isNumber(v any) bool {
	_, ok := v.(float64)
	return ok
}

func isIntegerNumber(v any) bool {
	f, ok := v.(float64)
	if !ok {
		return false
	}
	return math.Mod(f, 1) == 0
}
