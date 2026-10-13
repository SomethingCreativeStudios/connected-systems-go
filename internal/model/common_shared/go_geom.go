package common_shared

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/ewkbhex"
	"github.com/twpayne/go-geom/encoding/geojson"
)

// GoGeom wraps a go-geom geometry so it can be stored in a PostGIS geometry
// column and read and written as a GeoJSON geometry. Positions keep their
// altitude (and measure) rather than being flattened to XY.
type GoGeom struct {
	T geom.T
}

// DefaultGeometrySRID is the CRS84-compatible PostGIS SRID used by Connected
// Systems GeoJSON resources and spatial query parameters.
const DefaultGeometrySRID = 4326

// Value stores the geometry as hex-encoded EWKB, which PostGIS parses without
// losing precision. GeoJSON coordinates are always CRS84, so a geometry
// without an SRID is stored as 4326.
func (gg GoGeom) Value() (driver.Value, error) {
	if gg.T == nil {
		return nil, nil
	}
	t := gg.T
	if t.SRID() == 0 {
		var err error
		if t, err = geom.SetSRID(t, DefaultGeometrySRID); err != nil {
			return nil, fmt.Errorf("set geometry SRID: %w", err)
		}
	}
	return ewkbhex.Encode(t, ewkbhex.NDR)
}

// Scan decodes the EWKB PostGIS returns for a geometry column: hex-encoded
// text in the text wire format, raw bytes in the binary one.
func (gg *GoGeom) Scan(value any) error {
	gg.T = nil

	var raw []byte
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		raw = []byte(v)
	case []byte:
		raw = v
	default:
		return fmt.Errorf("scan geometry: unsupported value type %T", value)
	}

	// Binary EWKB starts with a byte-order byte (0x00 or 0x01), which is
	// never a hex digit, so only the text form decodes.
	if decoded, err := hex.DecodeString(string(raw)); err == nil {
		raw = decoded
	}

	t, err := ewkb.Unmarshal(raw)
	if err != nil {
		return fmt.Errorf("scan geometry: %w", err)
	}
	gg.T = t
	return nil
}

// MarshalJSON encodes the geometry as a GeoJSON geometry object.
func (gg GoGeom) MarshalJSON() ([]byte, error) {
	return geojson.Marshal(gg.T)
}

// UnmarshalJSON decodes a GeoJSON geometry object. Malformed geometries are
// rejected here rather than by PostGIS, so clients get a 400 that says what
// is wrong. Unclosed polygon rings are closed.
func (gg *GoGeom) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		gg.T = nil
		return nil
	}

	var raw geojson.Geometry
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid geometry: %w", err)
	}
	t, err := decodeGeometry(&raw)
	if err != nil {
		return fmt.Errorf("invalid geometry: %w", err)
	}
	if t, err = geom.SetSRID(t, DefaultGeometrySRID); err != nil {
		return fmt.Errorf("set geometry SRID: %w", err)
	}
	gg.T = t
	return nil
}

var errPositionSize = errors.New("positions must have 2, 3 or 4 coordinates")

func decodeGeometry(raw *geojson.Geometry) (geom.T, error) {
	switch raw.Type {
	case "GeometryCollection":
		return decodeGeometryCollection(raw)
	case "Point", "LineString", "Polygon", "MultiPoint", "MultiLineString", "MultiPolygon":
	default:
		return nil, fmt.Errorf("unsupported geometry type %q", raw.Type)
	}
	if raw.Coordinates == nil {
		return nil, fmt.Errorf("%s has no coordinates", raw.Type)
	}

	t, err := raw.Decode()
	var tooLow geojson.ErrDimensionalityTooLow
	var mixed geom.ErrStrideMismatch
	switch {
	case errors.As(err, &tooLow):
		return nil, errPositionSize
	case errors.As(err, &mixed):
		return nil, errors.New("positions must all have the same number of coordinates")
	case err != nil:
		return nil, err
	}
	if t.Empty() {
		return nil, fmt.Errorf("%s has no coordinates", raw.Type)
	}
	if layout := t.Layout(); layout != geom.XY && layout != geom.XYZ && layout != geom.XYZM {
		return nil, errPositionSize
	}

	switch t := t.(type) {
	case *geom.LineString:
		if t.NumCoords() < 2 {
			return nil, errors.New("LineString needs at least two positions")
		}
	case *geom.MultiLineString:
		for i := range t.NumLineStrings() {
			if t.LineString(i).NumCoords() < 2 {
				return nil, errors.New("each MultiLineString line needs at least two positions")
			}
		}
	case *geom.Polygon:
		rings, err := closeRings(t.Coords())
		if err != nil {
			return nil, err
		}
		return geom.NewPolygon(t.Layout()).SetCoords(rings)
	case *geom.MultiPolygon:
		polygons := t.Coords()
		for i, rings := range polygons {
			if polygons[i], err = closeRings(rings); err != nil {
				return nil, err
			}
		}
		return geom.NewMultiPolygon(t.Layout()).SetCoords(polygons)
	}
	return t, nil
}

func decodeGeometryCollection(raw *geojson.Geometry) (geom.T, error) {
	var members []geojson.Geometry
	if raw.Geometries != nil {
		if err := json.Unmarshal(*raw.Geometries, &members); err != nil {
			return nil, err
		}
	}
	if len(members) == 0 {
		return nil, errors.New("GeometryCollection has no geometries")
	}
	collection := geom.NewGeometryCollection()
	for i := range members {
		member, err := decodeGeometry(&members[i])
		if err != nil {
			return nil, fmt.Errorf("geometries[%d]: %w", i, err)
		}
		if err := collection.Push(member); err != nil {
			return nil, fmt.Errorf("geometries[%d]: %w", i, err)
		}
	}
	return collection, nil
}

// closeRings appends the first position to every open ring. A closed ring
// needs at least four positions.
func closeRings(rings [][]geom.Coord) ([][]geom.Coord, error) {
	for i, ring := range rings {
		if len(ring) > 0 && !slices.Equal(ring[0], ring[len(ring)-1]) {
			rings[i] = append(ring, slices.Clone(ring[0]))
		}
		if len(rings[i]) < 4 {
			return nil, errors.New("each polygon ring needs at least four positions")
		}
	}
	return rings, nil
}
