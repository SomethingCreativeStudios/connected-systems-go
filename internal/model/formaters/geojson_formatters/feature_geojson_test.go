package geojson_formatters

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Invalid feature times are a client error rather than being dropped.
func TestFeatureDeserialize_RejectsInvalidTimes(t *testing.T) {
	formatter := NewFeatureGeoJSONFormatter(nil)
	decode := func(properties string) error {
		_, err := formatter.Deserialize(context.Background(), strings.NewReader(
			`{"type": "Feature", "geometry": null, "properties": {"name": "Gauge", `+properties+`}}`))
		return err
	}

	require.NoError(t, decode(`"validTime": ["2026-01-01T00:00:00Z", ".."], "dateTime": "2026-02-01T00:00:00Z"`))
	require.Error(t, decode(`"validTime": ["sometime", ".."]`))
	require.Error(t, decode(`"validTime": "yesterday/today"`))
	require.Error(t, decode(`"dateTime": "last tuesday"`))
}
