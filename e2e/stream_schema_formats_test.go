package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// A stream keeps one schema per format: PUT on the schema endpoint with a new
// obsFormat/commandFormat registers an additional schema (the stream then
// supports that format), while PUT with an existing format replaces it. The
// read-only "formats" field lists all registered formats, and GET .../schema
// accepts a format query parameter to select a specific schema.
// =============================================================================

func postObservationExpectStatus(t *testing.T, datastreamID string, payload map[string]interface{}, want int) {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/datastreams/"+datastreamID+"/observations", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, want, resp.StatusCode)
}

func TestDatastream_MultiFormatSchemas(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Multi Format System"))
	dsID := createDatastreamViaAPI(t, "/systems/"+systemID+"/datastreams", baseDatastreamPayload())

	// Register a second schema in a new format.
	putJSONResource(t, "/datastreams/"+dsID+"/schema", map[string]interface{}{
		"obsFormat":     "application/x-protobuf",
		"messageSchema": "syntax = \"proto3\"; message ObservationResult { double value = 1; }",
	})

	// formats now advertises both registered formats.
	ds := getJSONResource(t, "/datastreams/"+dsID, "application/json")
	assert.ElementsMatch(t, []interface{}{"application/json", "application/x-protobuf"}, ds["formats"],
		"formats must list all registered schema formats")

	// schema is writeOnly: never present in responses. live is required and
	// defaults to false; resultType is required, derived from the result
	// schema (DataRecord -> "record"); observedProperties is required and
	// derived from the semantic definitions the base schema declares.
	_, hasSchema := ds["schema"]
	assert.False(t, hasSchema, "datastream responses must not include schema")
	assert.Equal(t, false, ds["live"], "live must default to false")
	assert.Equal(t, "record", ds["resultType"], "resultType must be derived from the result schema")
	op, hasObsProps := ds["observedProperties"]
	assert.True(t, hasObsProps, "observedProperties must be present")
	assert.NotNil(t, op, "observedProperties must be derived from the schema's semantic definitions")

	// No format param -> the current (most recently written) schema.
	schema := getJSONResource(t, "/datastreams/"+dsID+"/schema", "application/json")
	assert.Equal(t, "application/x-protobuf", schema["obsFormat"])

	// Format param selects the registered schema for that format.
	schema = getJSONResource(t, "/datastreams/"+dsID+"/schema?obsFormat=application/json", "application/json")
	assert.Equal(t, "application/json", schema["obsFormat"])
	assert.NotNil(t, schema["resultSchema"], "original JSON schema must be preserved")

	schema = getJSONResource(t, "/datastreams/"+dsID+"/schema?obsFormat=application/x-protobuf", "application/json")
	assert.Equal(t, "application/x-protobuf", schema["obsFormat"])

	// Unregistered format -> 404.
	resp := doGet(t, "/datastreams/"+dsID+"/schema?obsFormat=text/bogus")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// JSON observations still validate against the JSON schema even though the
	// current schema is protobuf: valid result is accepted, invalid rejected.
	postObservationExpectStatus(t, dsID, map[string]interface{}{
		"resultTime": "2026-07-01T00:00:00Z",
		"result":     map[string]interface{}{"temperature": 21.0, "humidity": 40.0},
	}, http.StatusCreated)
	postObservationExpectStatus(t, dsID, map[string]interface{}{
		"resultTime": "2026-07-01T00:00:00Z",
		"result":     map[string]interface{}{"humidity": 40.0},
	}, http.StatusBadRequest)

	// PUT with an existing format replaces that schema instead of adding one.
	putJSONResource(t, "/datastreams/"+dsID+"/schema", map[string]interface{}{
		"obsFormat": "application/json",
		"resultSchema": map[string]interface{}{
			"type": "DataRecord",
			"fields": []map[string]interface{}{
				{"name": "pressure", "type": "Quantity", "definition": "http://sensorml.com/ont/swe/property/Pressure", "label": "Pressure", "uom": map[string]interface{}{"code": "hPa"}},
			},
		},
	})

	ds = getJSONResource(t, "/datastreams/"+dsID, "application/json")
	assert.ElementsMatch(t, []interface{}{"application/json", "application/x-protobuf"}, ds["formats"],
		"replacing a schema must not add a format")

	schema = getJSONResource(t, "/datastreams/"+dsID+"/schema?obsFormat=application/json", "application/json")
	resultSchema, ok := schema["resultSchema"].(map[string]interface{})
	require.True(t, ok)
	fields, ok := resultSchema["fields"].([]interface{})
	require.True(t, ok)
	require.Len(t, fields, 1, "JSON schema must have been replaced")
}

func TestControlStream_MultiFormatSchemas(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemForControlStreamTest(t)
	csID := createControlStreamViaAPI(t, systemID, baseControlStreamPayload())

	// Register a second schema in a new format.
	putJSONResource(t, "/controlstreams/"+csID+"/schema", map[string]interface{}{
		"commandFormat": "application/swe+json",
		"recordSchema": map[string]interface{}{
			"type": "DataRecord",
			"fields": []map[string]interface{}{
				{"name": "setPoint", "type": "Quantity", "definition": "http://sensorml.com/ont/swe/property/Temperature", "label": "Set Point", "uom": map[string]interface{}{"code": "Cel"}},
			},
		},
	})

	cs := getJSONResource(t, "/controlstreams/"+csID, "application/json")
	assert.ElementsMatch(t, []interface{}{"application/json", "application/swe+json"}, cs["formats"],
		"formats must list all registered schema formats")

	// schema is writeOnly: never present in responses. live and async are
	// required booleans defaulting to false.
	_, hasSchema := cs["schema"]
	assert.False(t, hasSchema, "control stream responses must not include schema")
	assert.Equal(t, false, cs["live"], "live must default to false")
	assert.Equal(t, false, cs["async"], "async must default to false")

	// No format param -> the current (most recently written) schema.
	schema := getJSONResource(t, "/controlstreams/"+csID+"/schema", "application/json")
	assert.Equal(t, "application/swe+json", schema["commandFormat"])

	// Format param selects the registered schema for that format.
	schema = getJSONResource(t, "/controlstreams/"+csID+"/schema?commandFormat=application/json", "application/json")
	assert.Equal(t, "application/json", schema["commandFormat"])
	assert.NotNil(t, schema["parametersSchema"], "original JSON schema must be preserved")

	schema = getJSONResource(t, "/controlstreams/"+csID+"/schema?commandFormat="+url.QueryEscape("application/swe+json"), "application/json")
	assert.Equal(t, "application/swe+json", schema["commandFormat"])

	// Unregistered format -> 404.
	resp := doGet(t, "/controlstreams/"+csID+"/schema?commandFormat=text/bogus")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Commands still post fine with the JSON format.
	createCommandViaAPI(t, csID, map[string]interface{}{
		"parameters": map[string]interface{}{"setPoint": 21.0},
	})
}

// =============================================================================
// observedProperties (datastream) and controlledProperties (control stream)
// are read-only, derived from the semantic definitions declared in the
// registered schemas. Client-supplied values are ignored.
// =============================================================================

func TestStream_SemanticProperties_DerivedFromSchemas(t *testing.T) {
	cleanupDB(t)

	systemID := createSystemViaAPI(t, "/systems", baseSystemPayload("Semantic Props System"))

	dsPayload := baseDatastreamPayload()
	dsPayload["schema"] = map[string]interface{}{
		"obsFormat": "application/json",
		"resultSchema": map[string]interface{}{
			"type": "DataRecord",
			"fields": []map[string]interface{}{
				{"name": "temperature", "type": "Quantity",
					"definition": "https://qudt.org/vocab/quantitykind/Temperature", "label": "Air Temperature",
					"uom": map[string]interface{}{"code": "Cel"}},
				{"name": "humidity", "type": "Quantity",
					"definition": "https://qudt.org/vocab/quantitykind/RelativeHumidity", "label": "Relative Humidity",
					"uom": map[string]interface{}{"code": "%"}},
			},
		},
	}
	// Client-supplied observedProperties must be ignored (readOnly).
	dsPayload["observedProperties"] = []map[string]interface{}{
		{"definition": "https://example.org/bogus", "label": "Bogus"},
	}
	dsID := createDatastreamViaAPI(t, "/systems/"+systemID+"/datastreams", dsPayload)

	ds := getJSONResource(t, "/datastreams/"+dsID, "application/json")
	obsProps, ok := ds["observedProperties"].([]interface{})
	require.True(t, ok, "expected derived observedProperties array, got %v", ds["observedProperties"])
	require.Len(t, obsProps, 2)
	first, ok := obsProps[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "https://qudt.org/vocab/quantitykind/Temperature", first["definition"])
	assert.Equal(t, "Air Temperature", first["label"])
	second, ok := obsProps[1].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "https://qudt.org/vocab/quantitykind/RelativeHumidity", second["definition"])
	assert.Equal(t, "Relative Humidity", second["label"])

	csPayload := baseControlStreamPayload()
	csPayload["schema"] = map[string]interface{}{
		"commandFormat": "application/json",
		"parametersSchema": map[string]interface{}{
			"type": "DataRecord",
			"fields": []map[string]interface{}{
				{"name": "setPoint", "type": "Quantity",
					"definition": "https://qudt.org/vocab/quantitykind/Temperature", "label": "Set Point",
					"uom": map[string]interface{}{"code": "Cel"}},
			},
		},
	}
	csID := createControlStreamViaAPI(t, systemID, csPayload)

	cs := getJSONResource(t, "/controlstreams/"+csID, "application/json")
	ctlProps, ok := cs["controlledProperties"].([]interface{})
	require.True(t, ok, "expected derived controlledProperties array, got %v", cs["controlledProperties"])
	require.Len(t, ctlProps, 1)
	ctl, ok := ctlProps[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "https://qudt.org/vocab/quantitykind/Temperature", ctl["definition"])
	assert.Equal(t, "Set Point", ctl["label"])
}
