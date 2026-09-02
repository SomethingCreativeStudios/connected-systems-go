package contractvalidation

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcedureSensorMLValidationReportsProcessType(t *testing.T) {
	validator := New()
	body := []byte(`{
        "type":"UnsupportedProcess",
        "label":"Invalid procedure",
        "uniqueId":"urn:uuid:procedure-invalid",
        "definition":"http://www.w3.org/ns/sosa/Procedure"
    }`)

	err := validator.Validate(Procedure, sensorMLContentType, body)
	var validationErr *Error
	require.ErrorAs(t, err, &validationErr)
	require.Equal(t, []Violation{{
		Path:    "type",
		Message: "must be one of SimpleProcess, AggregateProcess, PhysicalComponent, or PhysicalSystem",
	}}, validationErr.Details)
}

func TestSystemAssetTypeValidationUsesRepresentationPath(t *testing.T) {
	validator := New()
	geoJSONBody := []byte(`{
        "type":"Feature",
        "properties":{
            "uid":"urn:uuid:invalid-asset",
            "name":"Invalid asset",
            "featureType":"http://www.w3.org/ns/sosa/Platform",
            "assetType":"Platform"
        }
    }`)

	err := validator.Validate(System, geoJSONContentType, geoJSONBody)
	var validationErr *Error
	require.ErrorAs(t, err, &validationErr)
	require.Equal(t, "properties.assetType", validationErr.Details[0].Path)
	require.Contains(t, validationErr.Details[0].Message, "Equipment")

	sensorMLBody := []byte(`{
        "type":"PhysicalSystem",
        "label":"Invalid asset",
        "uniqueId":"urn:uuid:invalid-sml-asset",
        "definition":"http://www.w3.org/ns/sosa/Platform",
        "classifiers":[{"definition":"cs:AssetType","value":"Platform"}]
    }`)
	err = validator.Validate(System, sensorMLContentType, sensorMLBody)
	require.ErrorAs(t, err, &validationErr)
	require.Equal(t, "classifiers[0].value", validationErr.Details[0].Path)
}

func TestProcedureGeoJSONWritesAreRejectedWithGuidance(t *testing.T) {
	validator := New()
	err := validator.Validate(Procedure, geoJSONContentType, []byte(`{"type":"Feature","properties":{}}`))
	var validationErr *Error
	require.ErrorAs(t, err, &validationErr)
	require.Equal(t, "Content-Type", validationErr.Details[0].Path)
	require.Contains(t, validationErr.Details[0].Message, "application/sml+json")
}

func TestValidatorCompilesEmbeddedSchemaAndReportsInvalidDefinition(t *testing.T) {
	validator := New()
	payload := map[string]any{
		"type":       "PhysicalComponent",
		"label":      "Invalid definition",
		"uniqueId":   "urn:uuid:definition-invalid",
		"definition": "https://example.test/not-a-sosa-type",
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	err = validator.Validate(Procedure, sensorMLContentType, body)
	var validationErr *Error
	require.True(t, errors.As(err, &validationErr), "expected contract validation error, got %v", err)
	require.NotEmpty(t, validationErr.Details)
}

func TestSchemaValidationReportsStableNestedPath(t *testing.T) {
	validator := New()
	body := []byte(`{
        "type":"Feature",
        "properties":{
            "uid":"urn:uuid:sampling-feature-invalid",
            "name":"Invalid sampling feature",
            "featureType":"http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint"
        },
        "geometry":{"type":"Point","coordinates":"not-an-array"}
    }`)

	err := validator.Validate(SamplingFeature, geoJSONContentType, body)
	var validationErr *Error
	require.ErrorAs(t, err, &validationErr)
	require.Equal(t, []Violation{
		{Path: "geometry.coordinates", Message: "got string, want array"},
		{Path: "properties.sampledFeature@link", Message: "missing property 'sampledFeature@link'"},
	}, validationErr.Details)
	for _, violation := range validationErr.Details {
		require.NotContains(t, violation.Message, "oneOf", "union internals must not leak into API details")
	}
}

func TestEmbeddedSchemasCompile(t *testing.T) {
	validator := New()
	for _, schemaName := range []string{
		"geojson/system-bundled.json",
		"geojson/deployment-bundled.json",
		"geojson/procedure-bundled.json",
		"geojson/samplingFeature-bundled.json",
		"sensorml/deployment-bundled.json",
		"sensorml/procedure-bundled.json",
		"json/observation-bundled.json",
		"json/command-bundled.json",
		"json/systemEvent-bundled.json",
	} {
		t.Run(schemaName, func(t *testing.T) {
			_, err := validator.load(schemaName)
			require.NoError(t, err)
		})
	}
}
