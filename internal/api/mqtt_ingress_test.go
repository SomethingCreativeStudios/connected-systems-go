package api

import (
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/stretchr/testify/require"
)

func TestMQTTCreateQoSPolicy(t *testing.T) {
	for _, v := range []int{4, 5} {
		for qos := 0; qos <= 2; qos++ {
			for _, key := range []string{"", "operation"} {
				err := validatePublication(v, qos, false, "application/json", key)
				ok := (v != 4 || key == "") && (qos < 2 || v == 5 && key != "")
				require.Equal(t, ok, err == nil, "protocol %d qos %d key %q", v, qos, key)
			}
		}
	}
	require.Error(t, validatePublication(5, 1, true, "application/json", "key"))
	require.Error(t, validatePublication(5, 1, false, "", "key"))
	require.Error(t, validatePublication(5, 1, false, "application/json", string(make([]byte, 129))))
}
func TestMQTTFingerprintKeepsNumericPrecision(t *testing.T) {
	fingerprint := func(body string) string {
		t.Helper()
		value, err := objectBody([]byte(body))
		require.NoError(t, err)
		sum, err := requestFingerprint("/systems", "application/json", value)
		require.NoError(t, err)
		return sum
	}
	a := fingerprint(`{"a":9007199254740993,"b":{"c":2}}`)
	require.Equal(t, a, fingerprint("{\n\"b\":{\"c\":2}, \"a\":9007199254740993}"))
	require.NotEqual(t, a, fingerprint(`{"a":9007199254740992,"b":{"c":2}}`))
	require.NotEqual(t, a, fingerprint(`{"a":9007199254740993.0,"b":{"c":2}}`))
	_, err := objectBody([]byte(`{} {}`))
	require.Error(t, err)
}

func TestEveryClientCreateHasOneCreator(t *testing.T) {
	_, err := indexCreators(resourceCreators)
	require.NoError(t, err)
	_, err = indexCreators(resourceCreators[1:])
	require.ErrorContains(t, err, "has no creator")
	_, err = indexCreators(append([]resourceCreator{systemCreator}, resourceCreators...))
	require.ErrorContains(t, err, "duplicate creator")
	_, err = indexCreators(append([]resourceCreator{{Kind: "deployment"}}, resourceCreators...))
	require.ErrorContains(t, err, "no client-create capability")
	// Systems advertise GeoJSON and SensorML, so a JSON-only codec cannot serve them.
	for _, c := range []struct {
		codec   resourceCodec
		message string
	}{{nil, "has no codec"}, {jsonCodec[*domains.System]{}, "cannot encode application/geo+json"}} {
		broken := systemCreator
		broken.Codec = c.codec
		require.Equal(t, systemCreator.Kind, resourceCreators[0].Kind)
		_, err = indexCreators(append([]resourceCreator{broken}, resourceCreators[1:]...))
		require.ErrorContains(t, err, c.message)
	}
}
