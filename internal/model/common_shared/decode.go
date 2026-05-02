package common_shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

var timeRangeType = reflect.TypeOf(TimeRange{})

// UnknownFieldError is returned when an incoming JSON payload contains a field
// that is not declared on the target Go type. Path is the dotted/indexed JSON
// path to the offending field (e.g. "observedProperties[0]"); Field is the
// unknown key itself.
type UnknownFieldError struct {
	Field string
	Path  string
}

func (e *UnknownFieldError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("unknown field %q", e.Field)
	}
	return fmt.Sprintf("unknown field %q in %s", e.Field, e.Path)
}

// DecodeWithFieldErrors decodes JSON bytes into T strictly: unknown JSON fields
// produce an UnknownFieldError with the field's JSON path. It also produces
// named field errors for any TimeRange fields, since custom UnmarshalJSON
// errors from TimeRange otherwise lose their field context.
func DecodeWithFieldErrors[T any](data []byte) (T, error) {
	var zero T

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return zero, err
	}

	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		if err := checkTimeRangeFields(t, raw); err != nil {
			return zero, err
		}
	}

	var out T
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		if key, ok := unknownFieldKey(err); ok {
			path, _ := findUnknownFieldPath(t, data, key)
			return zero, &UnknownFieldError{Field: key, Path: path}
		}
		return zero, err
	}
	return out, nil
}

// unknownFieldKey extracts the field name from an "json: unknown field" error.
func unknownFieldKey(err error) (string, bool) {
	msg := err.Error()
	const prefix = "json: unknown field "
	idx := strings.Index(msg, prefix)
	if idx < 0 {
		return "", false
	}
	rest := msg[idx+len(prefix):]
	rest = strings.TrimSpace(rest)
	if len(rest) >= 2 && rest[0] == '"' {
		end := strings.IndexByte(rest[1:], '"')
		if end >= 0 {
			return rest[1 : 1+end], true
		}
	}
	return "", false
}

func checkTimeRangeFields(t reflect.Type, raw map[string]json.RawMessage) error {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		ft := f.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		if f.Anonymous && ft.Kind() == reflect.Struct {
			if err := checkTimeRangeFields(ft, raw); err != nil {
				return err
			}
			continue
		}

		if ft != timeRangeType {
			continue
		}

		tag := f.Tag.Get("json")
		name, _, _ := strings.Cut(tag, ",")
		if name == "" || name == "-" {
			continue
		}

		data, ok := raw[name]
		if !ok {
			continue
		}

		var tr TimeRange
		if err := json.Unmarshal(data, &tr); err != nil {
			return fmt.Errorf("field '%s': %w", name, err)
		}
	}
	return nil
}

// findUnknownFieldPath walks the target type t alongside the raw JSON to find
// the JSON path of the first key that does not map to a declared struct field.
// Returns the dotted/indexed path of the *parent* container of the unknown
// key; the key itself is the caller's "field". Returns ("", false) if no
// path can be confidently resolved (e.g., the unknown field is inside an
// opaque type like map[string]any or json.RawMessage).
func findUnknownFieldPath(t reflect.Type, data []byte, key string) (string, bool) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return "", false
	}
	return walkForUnknown(t, v, key, "")
}

var (
	rawMessageType   = reflect.TypeOf(json.RawMessage(nil))
	jsonUnmarshaler  = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	emptyInterface   = reflect.TypeOf((*any)(nil)).Elem()
)

func walkForUnknown(t reflect.Type, v any, key, path string) (string, bool) {
	if t == nil {
		return "", false
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		obj, ok := v.(map[string]any)
		if !ok {
			return "", false
		}
		known := structJSONFields(t)
		for k := range obj {
			if _, isKnown := known[k]; !isKnown {
				if k == key {
					return path, true
				}
			}
		}
		// recurse into known fields
		for k, child := range obj {
			info, isKnown := known[k]
			if !isKnown {
				continue
			}
			childPath := joinPath(path, k)
			if p, found := walkForUnknown(info.fieldType, child, key, childPath); found {
				return p, true
			}
		}
	case reflect.Slice, reflect.Array:
		arr, ok := v.([]any)
		if !ok {
			return "", false
		}
		elemType := t.Elem()
		for i, child := range arr {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			if p, found := walkForUnknown(elemType, child, key, childPath); found {
				return p, true
			}
		}
	case reflect.Map:
		// map[string]X — keys are user data, recurse into values with the
		// element type but we cannot say a field is "unknown" at the map level.
		obj, ok := v.(map[string]any)
		if !ok {
			return "", false
		}
		elemType := t.Elem()
		for k, child := range obj {
			childPath := joinPath(path, k)
			if p, found := walkForUnknown(elemType, child, key, childPath); found {
				return p, true
			}
		}
	}
	return "", false
}

type fieldInfo struct {
	fieldType reflect.Type
}

// structJSONFields returns a map of JSON field name -> fieldInfo for a struct
// type, walking embedded structs (anonymous fields) so their promoted fields
// are visible. Fields whose type is opaque (json.RawMessage, interface{}, or
// implements json.Unmarshaler with no exposed structure) are still listed by
// name but their fieldType may be nil to skip recursion.
func structJSONFields(t reflect.Type) map[string]fieldInfo {
	out := map[string]fieldInfo{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		tag := f.Tag.Get("json")
		name, _, _ := strings.Cut(tag, ",")

		if f.Anonymous {
			ft := f.Type
			for ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if name == "" && ft.Kind() == reflect.Struct {
				for k, v := range structJSONFields(ft) {
					if _, exists := out[k]; !exists {
						out[k] = v
					}
				}
				continue
			}
		}

		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}

		ft := f.Type
		// Treat opaque types as "known but un-walkable".
		if ft == rawMessageType || ft == emptyInterface || ft.Kind() == reflect.Interface {
			out[name] = fieldInfo{fieldType: nil}
			continue
		}
		// Types with a custom UnmarshalJSON consume their payload via their
		// own logic; we cannot reliably enforce strictness inside them.
		if reflect.PointerTo(ft).Implements(jsonUnmarshaler) || ft.Implements(jsonUnmarshaler) {
			out[name] = fieldInfo{fieldType: nil}
			continue
		}
		out[name] = fieldInfo{fieldType: ft}
	}
	return out
}

func joinPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

var _ error = (*UnknownFieldError)(nil)
