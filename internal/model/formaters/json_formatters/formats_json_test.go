package json_formatters

import (
	"context"
	"reflect"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

// formats is read-only and derived from the stream's registered schema;
// client-supplied values are overridden and empty schemas fall back to JSON.
func TestDatastreamJSONSerialize_FormatsDerivedFromSchema(t *testing.T) {
	formatter := NewDatastreamJSONFormatter(nil)

	cases := []struct {
		name     string
		schema   *domains.DatastreamSchema
		schemas  domains.DatastreamSchemas
		expected common_shared.StringArray
	}{
		{"from schema obsFormat", &domains.DatastreamSchema{ObsFormat: "application/swe+csv"}, nil, common_shared.StringArray{"application/swe+csv"}},
		{"no schema falls back to json", nil, nil, common_shared.StringArray{JSONContentType}},
		{"all registered schemas", &domains.DatastreamSchema{ObsFormat: "application/x-protobuf"},
			domains.DatastreamSchemas{
				{ObsFormat: "application/json"},
				{ObsFormat: "application/x-protobuf"},
			},
			common_shared.StringArray{"application/json", "application/x-protobuf"}},
	}

	systemID := "sys-1"
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			datastream := &domains.Datastream{
				Base:     domains.Base{ID: "ds-1"},
				SystemID: &systemID,
				Formats:  common_shared.StringArray{"text/bogus"},
				Schema:   tc.schema,
				Schemas:  tc.schemas,
			}
			out, err := formatter.Serialize(context.Background(), datastream)
			if err != nil {
				t.Fatalf("serialize failed: %v", err)
			}
			if !reflect.DeepEqual(out.Formats, tc.expected) {
				t.Fatalf("expected formats %v, got %v", tc.expected, out.Formats)
			}
		})
	}
}

func TestControlStreamJSONSerialize_FormatsDerivedFromSchema(t *testing.T) {
	formatter := NewControlStreamJSONFormatter(nil)

	cases := []struct {
		name     string
		schema   *domains.ControlStreamSchema
		expected common_shared.StringArray
	}{
		{"from schema commandFormat", &domains.ControlStreamSchema{CommandFormat: "application/swe+binary"}, common_shared.StringArray{"application/swe+binary"}},
		{"no schema falls back to json", nil, common_shared.StringArray{JSONContentType}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cs := &domains.ControlStream{
				Base:    domains.Base{ID: "cs-1"},
				Formats: common_shared.StringArray{"text/bogus"},
				Schema:  tc.schema,
			}
			out, err := formatter.Serialize(context.Background(), cs)
			if err != nil {
				t.Fatalf("serialize failed: %v", err)
			}
			if !reflect.DeepEqual(out.Formats, tc.expected) {
				t.Fatalf("expected formats %v, got %v", tc.expected, out.Formats)
			}
		})
	}
}
