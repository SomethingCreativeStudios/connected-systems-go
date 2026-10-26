package common_shared

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// JSONEq decodes numbers as float64, which could hide precision loss here.
func requirePreservedJSON(t *testing.T, expected string, actual []byte) {
	t.Helper()
	decode := func(raw []byte) any {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var value any
		require.NoError(t, decoder.Decode(&value))
		return value
	}
	require.Equal(t, decode([]byte(expected)), decode(actual))
}

func TestComponentWrapperPreservesDecodedJSON(t *testing.T) {
	for name, payload := range map[string]string{
		"metadata and false flags": `{"id":"serial","type":"Text","definition":"urn:test:serial","label":"Serial","description":"Factory serial number","optional":false,"updatable":false,"value":"ABC"}`,
		"true flags":               `{"type":"Text","label":"Serial","optional":true,"updatable":true,"value":"ABC"}`,
		"vector coordinates":       `{"id":"orientation","type":"Vector","label":"Orientation","referenceFrame":"urn:test:frame","coordinates":[{"name":"x","type":"Quantity","axisID":"X","uom":{"code":"m"},"value":1.234567890123456789}]}`,
		"unmodeled component":      `{"type":"DataRecord","fields":[{"name":"count","type":"Count","value":9007199254740993}],"description":"A record"}`,
	} {
		t.Run(name, func(t *testing.T) {
			var component ComponentWrapper
			require.NoError(t, json.Unmarshal([]byte(payload), &component))
			encoded, err := json.Marshal(component)
			require.NoError(t, err)
			requirePreservedJSON(t, payload, encoded)

			// Captured JSON remains authoritative, like IOItem and ContactWrapper.
			component.Label = "Changed typed view"
			encoded, err = json.Marshal(component)
			require.NoError(t, err)
			requirePreservedJSON(t, payload, encoded)
		})
	}
}

func TestComponentWrapperModeledFields(t *testing.T) {
	payload := `{"id":"serial","type":"Text","definition":"urn:test:serial","label":"Serial","description":"Factory serial number","optional":false,"updatable":false,"value":"ABC"}`
	var component ComponentWrapper
	require.NoError(t, json.Unmarshal([]byte(payload), &component))
	require.Equal(t, "serial", component.ID)
	require.Equal(t, "Factory serial number", component.Description)
	require.NotNil(t, component.Optional)
	require.False(t, *component.Optional)
	require.NotNil(t, component.Updatable)
	require.False(t, *component.Updatable)

	component.Raw = nil
	encoded, err := json.Marshal(component)
	require.NoError(t, err)
	requirePreservedJSON(t, payload, encoded)

	// A second decode must not inherit flags or a concrete component from the first.
	require.NoError(t, json.Unmarshal([]byte(`{"type":"DataRecord","fields":[]}`), &component))
	require.Nil(t, component.Optional)
	require.Nil(t, component.Updatable)
	require.Nil(t, component.Component)
}

func TestComponentWrapperConstructedWithoutRaw(t *testing.T) {
	optional, updatable := false, true
	component := ComponentWrapper{
		ID: "serial", Type: "Text", Definition: "urn:test:serial", Label: "Serial",
		Description: "Factory serial number", Optional: &optional, Updatable: &updatable,
		Value: json.RawMessage(`"ABC"`),
	}
	encoded, err := json.Marshal(component)
	require.NoError(t, err)
	requirePreservedJSON(t, `{"id":"serial","type":"Text","definition":"urn:test:serial","label":"Serial","description":"Factory serial number","optional":false,"updatable":true,"value":"ABC"}`, encoded)
}
