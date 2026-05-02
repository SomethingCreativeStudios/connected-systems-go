package common_shared

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

var timeRangeType = reflect.TypeOf(TimeRange{})

// DecodeWithFieldErrors decodes JSON bytes into T, providing named field errors
// for any TimeRange fields. Without this, custom UnmarshalJSON errors from
// TimeRange lose their field context when decoded as part of a larger struct.
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
	if err := json.Unmarshal(data, &out); err != nil {
		return zero, err
	}
	return out, nil
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
