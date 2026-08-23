package common_shared

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoGeomUnmarshalJSONAssignsDefaultSRID(t *testing.T) {
	tests := map[string]string{
		"point":       `{"type":"Point","coordinates":[-117.1625,32.715]}`,
		"line string": `{"type":"LineString","coordinates":[[-118,32],[-117,33]]}`,
		"polygon":     `{"type":"Polygon","coordinates":[[[-118,32],[-117,32],[-117,33],[-118,33],[-118,32]]]}`,
	}

	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			var geometry GoGeom
			require.NoError(t, geometry.UnmarshalJSON([]byte(raw)))
			require.NotNil(t, geometry.T)
			require.Equal(t, DefaultGeometrySRID, geometry.T.SRID())

			value, err := geometry.Value()
			require.NoError(t, err)
			require.Contains(t, value, "SRID=4326;")
		})
	}
}
