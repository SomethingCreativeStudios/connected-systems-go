package common_shared

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/ewkbhex"
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
			stored, err := ewkbhex.Decode(value.(string))
			require.NoError(t, err)
			require.Equal(t, DefaultGeometrySRID, stored.SRID())
		})
	}
}

// Altitude and full coordinate precision survive storage and output, which
// the earlier WKT-with-%f encoding truncated to XY and six decimals.
func TestGoGeomRoundTripKeepsAltitudeAndPrecision(t *testing.T) {
	tests := map[string]string{
		"point with altitude": `{"type":"Point","coordinates":[-117.16250123456789,32.71500987654321,12.5]}`,
		"line string":         `{"type":"LineString","coordinates":[[-118.123456789,32.1,5],[-117,33,6]]}`,
		"collection":          `{"type":"GeometryCollection","geometries":[{"type":"Point","coordinates":[1,2,3]}]}`,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			var geometry GoGeom
			require.NoError(t, geometry.UnmarshalJSON([]byte(raw)))
			value, err := geometry.Value()
			require.NoError(t, err)

			var scanned GoGeom
			require.NoError(t, scanned.Scan(value))
			out, err := json.Marshal(scanned)
			require.NoError(t, err)
			require.JSONEq(t, raw, string(out))
		})
	}
}

func TestGoGeomUnmarshalJSONClosesPolygonRings(t *testing.T) {
	var geometry GoGeom
	require.NoError(t, geometry.UnmarshalJSON([]byte(`{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,1]]]}`)))
	out, err := json.Marshal(geometry)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,1],[0,0]]]}`, string(out))
}

func TestGoGeomUnmarshalJSONRejectsMalformedGeometries(t *testing.T) {
	tests := map[string]string{
		"missing type":            `{"coordinates":[1,2]}`,
		"unknown type":            `{"type":"Circle","coordinates":[1,2]}`,
		"missing coordinates":     `{"type":"Point"}`,
		"empty point":             `{"type":"Point","coordinates":[]}`,
		"single position line":    `{"type":"LineString","coordinates":[[1,2]]}`,
		"mixed dimensions":        `{"type":"LineString","coordinates":[[1,2],[3,4,5]]}`,
		"too many dimensions":     `{"type":"Point","coordinates":[1,2,3,4,5]}`,
		"degenerate ring":         `{"type":"Polygon","coordinates":[[[0,0],[1,1],[0,0]]]}`,
		"empty collection":        `{"type":"GeometryCollection","geometries":[]}`,
		"invalid collection item": `{"type":"GeometryCollection","geometries":[{"type":"Point","coordinates":[1]}]}`,
		"non-numeric position":    `{"type":"Point","coordinates":["1","2"]}`,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			var geometry GoGeom
			require.ErrorContains(t, geometry.UnmarshalJSON([]byte(raw)), "invalid geometry")
		})
	}
}

func TestGoGeomNull(t *testing.T) {
	var geometry GoGeom
	require.NoError(t, geometry.UnmarshalJSON([]byte(`null`)))
	require.Nil(t, geometry.T)
	out, err := json.Marshal(geometry)
	require.NoError(t, err)
	require.Equal(t, "null", string(out))
	value, err := geometry.Value()
	require.NoError(t, err)
	require.Nil(t, value)
}

// Malformed positions are a client error, not a panic.
func TestGoGeomUnmarshalJSONRejectsShortPositions(t *testing.T) {
	tests := map[string]string{
		"line string":   `{"type":"LineString","coordinates":[[1],[2]]}`,
		"polygon":       `{"type":"Polygon","coordinates":[[[1,2],[3]]]}`,
		"multi polygon": `{"type":"MultiPolygon","coordinates":[[[[1,2],[3]]]]}`,
		"point":         `{"type":"Point","coordinates":[1]}`,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			var geometry GoGeom
			require.NotPanics(t, func() {
				require.Error(t, geometry.UnmarshalJSON([]byte(raw)))
			})
		})
	}
}

// PostGIS hands geometry back as hex EWKB text or as raw EWKB bytes,
// depending on the wire format; both must decode, and garbage must error.
func TestGoGeomScan(t *testing.T) {
	point := geom.NewPointFlat(geom.XY, []float64{-117.1625, 32.715}).SetSRID(DefaultGeometrySRID)
	raw, err := ewkb.Marshal(point, ewkb.NDR)
	require.NoError(t, err)

	for name, value := range map[string]any{"hex text": hex.EncodeToString(raw), "binary": raw} {
		t.Run(name, func(t *testing.T) {
			var geometry GoGeom
			require.NoError(t, geometry.Scan(value))
			require.NotNil(t, geometry.T)
			require.Equal(t, []float64{-117.1625, 32.715}, geometry.T.FlatCoords())
		})
	}

	var geometry GoGeom
	require.NoError(t, geometry.Scan(nil))
	require.Nil(t, geometry.T)
	require.Error(t, geometry.Scan("not a geometry"))
	require.Error(t, geometry.Scan(42))
}
