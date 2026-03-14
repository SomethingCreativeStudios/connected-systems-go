package generators

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

func FakeDatastreamJSONScalarSchema() *domains.DatastreamSchema {
	return &domains.DatastreamSchema{
		ObsFormat: "application/json",
		ResultSchema: &domains.DatastreamDataComponent{
			Type:       "Quantity",
			Name:       "temperature",
			Label:      "Temperature",
			Definition: "https://example.org/def/property/temperature",
			UOM: &domains.DatastreamUOM{
				Code: "Cel",
			},
		},
	}
}

func FakeDatastreamJSONRecordSchema() *domains.DatastreamSchema {
	optional := true
	return &domains.DatastreamSchema{
		ObsFormat: "application/json",
		ResultSchema: &domains.DatastreamDataComponent{
			Type:       "DataRecord",
			Name:       "weatherRecord",
			Label:      "Weather Record",
			Definition: "https://example.org/def/record/weather",
			Fields: []domains.DatastreamNamedComponent{
				{
					Name: "temperature",
					DatastreamDataComponent: domains.DatastreamDataComponent{
						Type:       "Quantity",
						Definition: "https://example.org/def/property/temperature",
						UOM:        &domains.DatastreamUOM{Code: "Cel"},
					},
				},
				{
					Name: "humidity",
					DatastreamDataComponent: domains.DatastreamDataComponent{
						Type:       "Quantity",
						Definition: "https://example.org/def/property/humidity",
						UOM:        &domains.DatastreamUOM{Code: "%"},
					},
				},
				{
					Name: "status",
					DatastreamDataComponent: domains.DatastreamDataComponent{
						Type:       "Category",
						Definition: "https://example.org/def/property/status",
						Optional:   &optional,
					},
				},
			},
		},
	}
}

func FakeDatastreamSWEJSONSchema() *domains.DatastreamSchema {
	falseVal := false
	return &domains.DatastreamSchema{
		ObsFormat: "application/swe+json",
		RecordSchema: &domains.DatastreamDataComponent{
			Type:       "DataRecord",
			Name:       "positionRecord",
			Definition: "https://example.org/def/record/position",
			Fields: []domains.DatastreamNamedComponent{
				{
					Name: "x",
					DatastreamDataComponent: domains.DatastreamDataComponent{
						Type:       "Quantity",
						Definition: "https://example.org/def/property/x",
						UOM:        &domains.DatastreamUOM{Code: "m"},
					},
				},
				{
					Name: "y",
					DatastreamDataComponent: domains.DatastreamDataComponent{
						Type:       "Quantity",
						Definition: "https://example.org/def/property/y",
						UOM:        &domains.DatastreamUOM{Code: "m"},
					},
				},
			},
		},
		Encoding: &domains.DatastreamEncoding{
			Type:            "JSONEncoding",
			RecordsAsArrays: &falseVal,
			VectorsAsArrays: &falseVal,
		},
	}
}

func FakeDatastreamSWECsvSchema() *domains.DatastreamSchema {
	return &domains.DatastreamSchema{
		ObsFormat: "application/swe+csv",
		RecordSchema: &domains.DatastreamDataComponent{
			Type:       "DataRecord",
			Name:       "scalarRecord",
			Definition: "https://example.org/def/record/scalar",
			Fields: []domains.DatastreamNamedComponent{
				{
					Name: "v",
					DatastreamDataComponent: domains.DatastreamDataComponent{
						Type:       "Quantity",
						Definition: "https://example.org/def/property/value",
						UOM:        &domains.DatastreamUOM{Code: "1"},
					},
				},
			},
		},
		Encoding: &domains.DatastreamEncoding{
			Type:             "TextEncoding",
			TokenSeparator:   ",",
			BlockSeparator:   "\n",
			DecimalSeparator: ".",
		},
	}
}

func FakeDatastreamProtobufSchema() *domains.DatastreamSchema {
	protoSchema := `syntax = "proto3";
message ObservationResult {
  double temperature = 1;
  double humidity = 2;
  string status = 3;
}`
	return &domains.DatastreamSchema{
		ObsFormat: "application/x-protobuf",
		MessageSchema: &domains.DatastreamMessageSchema{
			Inline: &protoSchema,
		},
	}
}

func FakeDatastreamOtherFormatSchema() *domains.DatastreamSchema {
	return &domains.DatastreamSchema{
		ObsFormat: "application/vnd.example.custom+json",
		Any: common_shared.Properties{
			"description": "custom payload format",
			"version":     "1.0",
		},
	}
}

func FakeDatastreamWithSchema(schema *domains.DatastreamSchema) domains.Datastream {
	id := uuid.NewString()
	systemID := uuid.NewString()
	foiID := uuid.NewString()
	live := true
	resultType := domains.DatastreamResultTypeMeasure
	if schema != nil && schema.ResultSchema != nil && schema.ResultSchema.Type == "DataRecord" {
		resultType = domains.DatastreamResultTypeRecord
	}

	obsProps := domains.DatastreamObservedProperties{
		{
			Definition: "https://example.org/def/property/temperature",
			Label:      "Temperature",
		},
	}

	formats := common_shared.StringArray{"application/json"}
	if schema != nil && schema.ObsFormat != "" {
		formats = common_shared.StringArray{schema.ObsFormat}
	}

	return domains.Datastream{
		Base: domains.Base{ID: id},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(fmt.Sprintf("urn:uuid:%s", id)),
			Name:             "Datastream " + f.Lorem().Word(),
			Description:      f.Lorem().Sentence(6),
		},
		ValidTime:          FakeValidTimeCurrent(),
		Formats:            formats,
		SystemID:           &systemID,
		SystemLink:         &common_shared.Link{Href: "/systems/" + systemID},
		FeatureOfInterest:  &common_shared.Link{Href: "/features/" + foiID},
		OutputName:         "output-" + f.Lorem().Word(),
		ObservedProperties: &obsProps,
		PhenomenonTime:     FakeValidTimeCurrent(),
		ResultTime:         FakeValidTimeCurrent(),
		Type:               domains.DatastreamTypeObservation,
		ResultType:         &resultType,
		Live:               &live,
		Schema:             schema,
		Links:              FakeLinksFull(1),
	}
}

func FakeDatastreamJSONScalar() domains.Datastream {
	return FakeDatastreamWithSchema(FakeDatastreamJSONScalarSchema())
}

func FakeDatastreamJSONRecord() domains.Datastream {
	return FakeDatastreamWithSchema(FakeDatastreamJSONRecordSchema())
}

func FakeDatastreamSWEJSON() domains.Datastream {
	return FakeDatastreamWithSchema(FakeDatastreamSWEJSONSchema())
}

func FakeDatastreamSWECsv() domains.Datastream {
	return FakeDatastreamWithSchema(FakeDatastreamSWECsvSchema())
}

func FakeDatastreamProtobuf() domains.Datastream {
	return FakeDatastreamWithSchema(FakeDatastreamProtobufSchema())
}

func FakeDatastreamOtherFormat() domains.Datastream {
	return FakeDatastreamWithSchema(FakeDatastreamOtherFormatSchema())
}

// FakeDatastreamSchemaMix returns a representative mix of datastreams with
// different schema branches for e2e and validator coverage.
func FakeDatastreamSchemaMix() []domains.Datastream {
	return []domains.Datastream{
		FakeDatastreamJSONScalar(),
		FakeDatastreamJSONRecord(),
		FakeDatastreamSWEJSON(),
		FakeDatastreamSWECsv(),
		FakeDatastreamProtobuf(),
		FakeDatastreamOtherFormat(),
	}
}

// FakeDatastreamRandom picks one schema branch at random.
func FakeDatastreamRandom() domains.Datastream {
	builders := []func() domains.Datastream{
		FakeDatastreamJSONScalar,
		FakeDatastreamJSONRecord,
		FakeDatastreamSWEJSON,
		FakeDatastreamSWECsv,
		FakeDatastreamProtobuf,
		FakeDatastreamOtherFormat,
	}
	return builders[rand.Intn(len(builders))]()
}
