package queryables

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompileTextAndJSONHaveEquivalentBoundArguments(t *testing.T) {
	text, err := Compile(Systems, "name = 'harbor sensor' AND id <> 'legacy'", CQL2Text)
	require.NoError(t, err)
	json, err := Compile(Systems, `{"op":"and","args":[{"op":"=","args":[{"property":"name"},"harbor sensor"]},{"op":"<>","args":[{"property":"id"},"legacy"]}]}`, CQL2JSON)
	require.NoError(t, err)
	require.Equal(t, text.Args, json.Args)
	require.Contains(t, text.Text, "$1")
	require.NotContains(t, text.Text, "harbor sensor")
}

func TestCompileRejectsUnknownOrUnsafeQueryable(t *testing.T) {
	_, err := Compile(Systems, "properties.owner = 'untrusted'", CQL2Text)
	require.Error(t, err)
	_, err = Compile(Systems, "name = 'x'", "cql2-sql")
	require.ErrorIs(t, err, ErrUnsupportedLanguage)
}

func TestCatalogRequiresAnIndexablePropertySchema(t *testing.T) {
	catalog := map[string]any{
		"properties": map[string]any{
			"siteCode": map[string]any{"type": "string"},
			"tags":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"additionalProperties": false,
	}
	require.NoError(t, ValidateCustomCatalog(catalog))
	compiled, err := CompileWithCatalog(Features, catalog, "siteCode = 'A-12'", CQL2Text)
	require.NoError(t, err)
	require.True(t, strings.Contains(compiled.Text, `"features"."properties"`))

	catalog["additionalProperties"] = true
	require.Error(t, ValidateCustomCatalog(catalog))
}

func TestSchemaIsClosedAndServerOwned(t *testing.T) {
	schema, err := Schema(Systems, "https://example.test/collections/systems/queryables")
	require.NoError(t, err)
	require.Equal(t, false, schema["additionalProperties"])
	require.Equal(t, "https://json-schema.org/draft/2020-12/schema", schema["$schema"])
	require.Contains(t, schema["properties"], "procedures")
}

func TestTemporalIntervalsUseBothConnectedSystemsColumns(t *testing.T) {
	compiled, err := Compile(Systems, `T_INTERSECTS(validTime,INTERVAL('2026-01-01T00:00:00Z','2026-01-02T00:00:00Z'))`, CQL2Text)
	require.NoError(t, err)
	require.Contains(t, compiled.Text, `"systems"."valid_time_start"`)
	require.Contains(t, compiled.Text, `"systems"."valid_time_end"`)
	require.Len(t, compiled.Args, 2)
}

func TestAdvertisedCQL2OperatorsCompileWithCuratedQueryables(t *testing.T) {
	cases := []struct {
		resource Resource
		filter   string
	}{
		{Systems, `CASEI(name) LIKE CASEI('harbor%')`},
		{Systems, `ACCENTI(name) = ACCENTI('cafe')`},
		{Systems, `S_INTERSECTS(geometry,POINT(-70 40))`},
		{Systems, `A_CONTAINS(procedures,('procedure-1'))`},
		{Systems, `id = uid`},
		{CommandStatuses, `percentCompletion + 1 >= 50`},
	}
	for _, testCase := range cases {
		t.Run(testCase.filter, func(t *testing.T) {
			compiled, err := Compile(testCase.resource, testCase.filter, CQL2Text)
			require.NoError(t, err)
			require.NotEmpty(t, compiled.Text)
		})
	}
}

func TestDeclaredFeatureQueryableValuesAreTypeChecked(t *testing.T) {
	catalog := map[string]any{
		"properties": map[string]any{
			"temperature": map[string]any{"type": "number"},
			"recordedAt":  map[string]any{"type": "string", "format": "date-time"},
		},
		"additionalProperties": false,
	}
	require.NoError(t, ValidateFeatureProperties(catalog, map[string]any{"temperature": 12.5, "recordedAt": "2026-01-02T03:04:05Z"}))
	require.Error(t, ValidateFeatureProperties(catalog, map[string]any{"temperature": "hot"}))
	require.Error(t, ValidateFeatureProperties(catalog, map[string]any{"recordedAt": "tomorrow"}))
}
