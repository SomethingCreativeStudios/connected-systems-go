package common_shared

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/wkb"
	"github.com/twpayne/go-geom/encoding/wkt"
)

// GoGeom is a thin wrapper around go-geom's geom.T that implements
// sql.Scanner / driver.Valuer and JSON marshal/unmarshal by converting
// to/from the existing GeoJSON-friendly Geometry struct (in this package).
// This file deliberately does not modify the existing `geometry.go`.
type GoGeom struct {
	T geom.T
}

// Value returns WKB bytes for storage in PostGIS; falls back to GeoJSON bytes on error
func (gg GoGeom) Value() (driver.Value, error) {
	if gg.T == nil {
		return nil, nil
	}
	// Prefer returning a PostGIS-friendly WKT string (with SRID if present).
	if wkt := wktFromGeom(gg.T); wkt != "" {
		// Try to include SRID if available
		if s, ok := gg.T.(interface{ SRID() int }); ok {
			srid := s.SRID()
			if srid != 0 {
				return fmt.Sprintf("SRID=%d;%s", srid, wkt), nil
			}
		}
		return wkt, nil
	}
	// Fallback to binary WKB if WKT generation failed
	if b, err := wkb.Marshal(gg.T, nil); err == nil {
		return b, nil
	}
	// Last-resort: marshal to JSON-friendly object
	out := fromGeomToGeoJSON(gg.T)
	return json.Marshal(out)
}

// Scan accepts WKB bytes, WKT strings (optionally SRID-prefixed), or GeoJSON bytes
// and sets the inner geom.T.
func (gg *GoGeom) Scan(value interface{}) error {
	if value == nil {
		gg.T = nil
		return nil
	}

	// Helper to try JSON/GeoJSON -> geom
	tryGeoJSON := func(b []byte) (geom.T, error) {
		var raw interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			return nil, err
		}
		return toGeomFromGeoJSON(raw)
	}

	// Debug log value type and content
	log.Printf("GoGeom.Scan: value type=%T value=%v", value, value)

	switch v := value.(type) {
	case []byte:
		s := strings.TrimSpace(string(v))
		// Try hex decode if looks like hex
		if isHexString(s) {
			if bb, err := hex.DecodeString(s); err == nil {
				// If EWKB (PostGIS), first 4 bytes are SRID if type code has high bit set
				t, err := wkb.Unmarshal(bb)
				if err != nil && err.Error() == "unknown type" && len(bb) > 8 {
					// Try stripping first 4 bytes (SRID)
					log.Printf("GoGeom.Scan: detected EWKB, stripping SRID")
					t, err = wkb.Unmarshal(bb[4:])
				}
				if err != nil {
					log.Printf("GoGeom.Scan: wkb.Unmarshal failed for hex WKB: hex=%q err=%v", s, err)
				}
				if t != nil && err == nil {
					gg.T = t
					log.Printf("GoGeom.Scan: decoded hex WKB from []byte")
					return nil
				}
			} else {
				log.Printf("GoGeom.Scan: hex.DecodeString failed: hex=%q err=%v", s, err)
			}
		}
		// Try WKB directly
		if t, err := wkb.Unmarshal(v); err == nil && t != nil {
			gg.T = t
			log.Printf("GoGeom.Scan: decoded WKB from []byte")
			return nil
		}
		// Try WKT (byte -> string)
		if t, err := wkt.Unmarshal(s); err == nil && t != nil {
			gg.T = t
			log.Printf("GoGeom.Scan: decoded WKT from []byte")
			return nil
		}
		// Try JSON/GeoJSON
		if tg, err := tryGeoJSON(v); err == nil {
			gg.T = tg
			log.Printf("GoGeom.Scan: decoded GeoJSON from []byte")
			return nil
		}
		log.Printf("GoGeom.Scan: unable to scan from []byte: %q", s)
		return fmt.Errorf("unable to scan GoGeom from []byte: %q", s)
	case string:
		s := strings.TrimSpace(v)
		// Remove SRID=####; prefix if present
		if strings.HasPrefix(strings.ToUpper(s), "SRID=") {
			if idx := strings.Index(s, ";"); idx != -1 {
				s = s[idx+1:]
			}
		}
		// Try hex decode if looks like hex
		if isHexString(s) {
			if bb, err := hex.DecodeString(s); err == nil {
				// If EWKB (PostGIS), first 4 bytes are SRID if type code has high bit set
				t, err := wkb.Unmarshal(bb)
				if err != nil && err.Error() == "unknown type" && len(bb) > 8 {
					// Try stripping first 4 bytes (SRID)
					log.Printf("GoGeom.Scan: detected EWKB, stripping SRID")
					t, err = wkb.Unmarshal(bb[4:])
				}
				if err != nil {
					log.Printf("GoGeom.Scan: wkb.Unmarshal failed for hex WKB: hex=%q err=%v", s, err)
				}
				if t != nil && err == nil {
					gg.T = t
					log.Printf("GoGeom.Scan: decoded hex WKB from string")
					return nil
				}
			} else {
				log.Printf("GoGeom.Scan: hex.DecodeString failed: hex=%q err=%v", s, err)
			}
		}
		// Try WKT
		if t, err := wkt.Unmarshal(s); err == nil && t != nil {
			gg.T = t
			log.Printf("GoGeom.Scan: decoded WKT from string")
			return nil
		}
		// Try JSON
		if tg, err := tryGeoJSON([]byte(s)); err == nil {
			gg.T = tg
			log.Printf("GoGeom.Scan: decoded GeoJSON from string")
			return nil
		}
		log.Printf("GoGeom.Scan: unable to scan from string: %q", s)
		return fmt.Errorf("unable to scan GoGeom from string: %q", s)
	default:
		log.Printf("GoGeom.Scan: unsupported scan type: %T", value)
		return fmt.Errorf("unsupported scan type for GoGeom: %T", value)
	}
}

// isHexString returns true if s contains only hexadecimal characters and has even length.
func isHexString(s string) bool {
	if s == "" || len(s)%2 != 0 {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// MarshalJSON encodes as GeoJSON using a JSON-friendly representation
func (gg GoGeom) MarshalJSON() ([]byte, error) {
	if gg.T == nil {
		return json.Marshal(nil)
	}
	out := fromGeomToGeoJSON(gg.T)
	return json.Marshal(out)
}

// UnmarshalJSON decodes GeoJSON into geom.T
func (gg *GoGeom) UnmarshalJSON(data []byte) error {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if tg, err := toGeomFromGeoJSON(raw); err == nil {
		gg.T = tg
		return nil
	}
	return fmt.Errorf("invalid geometry JSON")
}

// ----------------- conversion helpers -----------------

// toGeomFromGeoJSON accepts either the existing Geometry struct (unmarshaled
// into a map/object by the caller) or a raw map[string]interface{} and
// constructs a geom.T.
func toGeomFromGeoJSON(v interface{}) (geom.T, error) {
	if v == nil {
		return nil, fmt.Errorf("nil geometry")
	}

	// If it's already a map[string]interface{} (raw JSON), use that
	if raw, ok := v.(map[string]interface{}); ok {
		if tval, _ := raw["type"].(string); tval != "" {
			switch tval {
			case "Point":
				if coords, ok := ifaceToFloat64Slice(raw["coordinates"]); ok && len(coords) >= 2 {
					return geom.NewPointFlat(geom.XY, coords[:2]), nil
				}
			case "LineString":
				if coords, ok := ifaceTo2DFloat64Slice(raw["coordinates"]); ok {
					return geom.NewLineStringFlat(geom.XY, flatten2D(coords)), nil
				}
			case "Polygon":
				if rings, ok := ifaceTo3DFloat64Slice(raw["coordinates"]); ok {
					return geom.NewPolygonFlat(geom.XY, flattenRings(rings), ringEnds(rings)), nil
				}
			case "MultiPoint":
				if coords, ok := ifaceTo2DFloat64Slice(raw["coordinates"]); ok {
					return geom.NewMultiPointFlat(geom.XY, flatten2D(coords)), nil
				}
			case "MultiLineString":
				if lines, ok := ifaceTo3DFloat64Slice(raw["coordinates"]); ok {
					return geom.NewMultiLineStringFlat(geom.XY, flattenRings(lines), ringEnds(lines)), nil
				}
			case "MultiPolygon":
				if polys, ok := ifaceTo4DFloat64Slice(raw["coordinates"]); ok {
					mp := geom.NewMultiPolygon(geom.XY)
					for _, poly := range polys {
						p := geom.NewPolygonFlat(geom.XY, flattenRings(poly), ringEnds(poly))
						mp.Push(p)
					}
					return mp, nil
				}
			case "GeometryCollection":
				geomsRaw, ok := raw["geometries"].([]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid geometries for GeometryCollection")
				}
				if len(geomsRaw) == 0 {
					return nil, fmt.Errorf("empty geometrycollection")
				}
				gc := geom.NewGeometryCollection()
				for _, gr := range geomsRaw {
					if tg, err := toGeomFromGeoJSON(gr); err == nil && tg != nil {
						gc.Push(tg)
					}
				}
				return gc, nil
			}
		}
		return nil, fmt.Errorf("unsupported or invalid geometry type in raw JSON")
	}

	// If it's been unmarshaled into the lightweight Geometry struct, we'll
	// handle the common types. Attempt to cast.
	if g, ok := v.(*Geometry); ok {
		switch g.Type {
		case "Point":
			if coords, ok := ifaceToFloat64Slice(g.Coordinates); ok && len(coords) >= 2 {
				return geom.NewPointFlat(geom.XY, coords[:2]), nil
			}
		case "LineString":
			if coords, ok := ifaceTo2DFloat64Slice(g.Coordinates); ok {
				return geom.NewLineStringFlat(geom.XY, flatten2D(coords)), nil
			}
		case "Polygon":
			if rings, ok := ifaceTo3DFloat64Slice(g.Coordinates); ok {
				return geom.NewPolygonFlat(geom.XY, flattenRings(rings), ringEnds(rings)), nil
			}
		case "MultiPoint":
			if coords, ok := ifaceTo2DFloat64Slice(g.Coordinates); ok {
				return geom.NewMultiPointFlat(geom.XY, flatten2D(coords)), nil
			}
		case "MultiLineString":
			if lines, ok := ifaceTo3DFloat64Slice(g.Coordinates); ok {
				return geom.NewMultiLineStringFlat(geom.XY, flattenRings(lines), ringEnds(lines)), nil
			}
		case "MultiPolygon":
			if polys, ok := ifaceTo4DFloat64Slice(g.Coordinates); ok {
				mp := geom.NewMultiPolygon(geom.XY)
				for _, poly := range polys {
					p := geom.NewPolygonFlat(geom.XY, flattenRings(poly), ringEnds(poly))
					mp.Push(p)
				}
				return mp, nil
			}
		}
		return nil, fmt.Errorf("unsupported or invalid geometry type: %s", g.Type)
	}

	return nil, fmt.Errorf("unsupported geojson value type: %T", v)
}

// fromGeomToGeoJSON returns a JSON-friendly representation (either *Geometry
// for simple types or a map[string]interface{} for collections/complex types).
func fromGeomToGeoJSON(t geom.T) interface{} {
	if t == nil {
		return &Geometry{}
	}
	switch tt := t.(type) {
	case *geom.Point:
		coords := tt.FlatCoords()
		if len(coords) >= 2 {
			return &Geometry{Type: "Point", Coordinates: []float64{coords[0], coords[1]}}
		}
	case *geom.LineString:
		coords := tt.FlatCoords()
		return &Geometry{Type: "LineString", Coordinates: unflattenCoords(coords)}
	case *geom.Polygon:
		coords := tt.FlatCoords()
		ends := tt.Ends()
		rings := unflattenRings(coords, ends)
		// ensure rings are closed for GeoJSON output
		for i, r := range rings {
			rings[i] = closeRing(r)
		}
		return &Geometry{Type: "Polygon", Coordinates: rings}
	case *geom.MultiPoint:
		coords := tt.FlatCoords()
		return &Geometry{Type: "MultiPoint", Coordinates: unflattenCoords(coords)}
	case *geom.MultiLineString:
		coords := tt.FlatCoords()
		ends := tt.Ends()
		return &Geometry{Type: "MultiLineString", Coordinates: unflattenLines(coords, ends)}
	case *geom.MultiPolygon:
		var polys [][][][]float64
		for i := 0; i < tt.NumPolygons(); i++ {
			p := tt.Polygon(i)
			flat := p.FlatCoords()
			ends := p.Ends()
			rings := unflattenRings(flat, ends)
			for j, r := range rings {
				rings[j] = closeRing(r)
			}
			polys = append(polys, rings)
		}
		return map[string]interface{}{"type": "MultiPolygon", "coordinates": polys}
	case *geom.GeometryCollection:
		var geoms []interface{}
		for i := 0; i < tt.NumGeoms(); i++ {
			sub := tt.Geom(i)
			geoms = append(geoms, fromGeomToGeoJSON(sub))
		}
		return map[string]interface{}{"type": "GeometryCollection", "geometries": geoms}
	default:
		b, _ := json.Marshal(t)
		var out interface{}
		_ = json.Unmarshal(b, &out)
		return out
	}
	return &Geometry{}
}

// ----------------- small coordinate helpers -----------------

func ifaceToFloat64Slice(v interface{}) ([]float64, bool) {
	if v == nil {
		return nil, false
	}
	// common case: []interface{} from generic JSON decoding
	if arr, ok := v.([]interface{}); ok {
		out := make([]float64, 0, len(arr))
		for _, x := range arr {
			f, ok := x.(float64)
			if !ok {
				return nil, false
			}
			out = append(out, f)
		}
		return out, true
	}
	// already []float64
	if ff, ok := v.([]float64); ok {
		return ff, true
	}
	return nil, false
}

func ifaceTo2DFloat64Slice(v interface{}) ([][]float64, bool) {
	if v == nil {
		return nil, false
	}
	if arr, ok := v.([]interface{}); ok {
		out := make([][]float64, 0, len(arr))
		for _, xi := range arr {
			if x, ok := ifaceToFloat64Slice(xi); ok {
				out = append(out, x)
			} else {
				return nil, false
			}
		}
		return out, true
	}
	if a2, ok := v.([][]float64); ok {
		return a2, true
	}
	return nil, false
}

func ifaceTo3DFloat64Slice(v interface{}) ([][][]float64, bool) {
	if v == nil {
		return nil, false
	}
	if arr, ok := v.([]interface{}); ok {
		out := make([][][]float64, 0, len(arr))
		for _, xi := range arr {
			if x, ok := ifaceTo2DFloat64Slice(xi); ok {
				out = append(out, x)
			} else {
				return nil, false
			}
		}
		return out, true
	}
	if a2, ok := v.([][][]float64); ok {
		return a2, true
	}
	return nil, false
}

func ifaceTo4DFloat64Slice(v interface{}) ([][][][]float64, bool) {
	if v == nil {
		return nil, false
	}
	if arr, ok := v.([]interface{}); ok {
		out := make([][][][]float64, 0, len(arr))
		for _, xi := range arr {
			if x, ok := ifaceTo3DFloat64Slice(xi); ok {
				out = append(out, x)
			} else {
				return nil, false
			}
		}
		return out, true
	}
	if a2, ok := v.([][][][]float64); ok {
		return a2, true
	}
	return nil, false
}

func flatten2D(coords [][]float64) []float64 {
	var out []float64
	for _, c := range coords {
		out = append(out, c[0], c[1])
	}
	return out
}

func flattenRings(rings [][][]float64) []float64 {
	var out []float64
	for _, ring := range rings {
		for _, pt := range ring {
			out = append(out, pt[0], pt[1])
		}
	}
	return out
}

func ringEnds(rings [][][]float64) []int {
	var ends []int
	idx := 0
	for _, ring := range rings {
		// ends are indexes into the flat coordinate array (stride=2 for XY)
		idx += len(ring) * 2
		ends = append(ends, idx)
	}
	return ends
}

func unflattenCoords(flat []float64) [][]float64 {
	var out [][]float64
	for i := 0; i < len(flat); i += 2 {
		out = append(out, []float64{flat[i], flat[i+1]})
	}
	return out
}

func unflattenRings(flat []float64, ends []int) [][][]float64 {
	var out [][][]float64
	start := 0
	for _, end := range ends {
		var ring [][]float64
		for i := start; i < end; i += 2 {
			ring = append(ring, []float64{flat[i], flat[i+1]})
		}
		out = append(out, ring)
		start = end
	}
	return out
}

func unflattenLines(flat []float64, ends []int) [][][]float64 {
	return unflattenRings(flat, ends)
}

// wktFromGeom returns a WKT representation for common geom.T types.
func wktFromGeom(t geom.T) string {
	if t == nil {
		return ""
	}
	switch tt := t.(type) {
	case *geom.Point:
		c := tt.FlatCoords()
		if len(c) >= 2 {
			return fmt.Sprintf("POINT(%f %f)", c[0], c[1])
		}
	case *geom.LineString:
		coords := unflattenCoords(tt.FlatCoords())
		var pts []string
		for _, p := range coords {
			if len(p) >= 2 {
				pts = append(pts, fmt.Sprintf("%f %f", p[0], p[1]))
			}
		}
		return fmt.Sprintf("LINESTRING(%s)", joinWKT(pts))
	case *geom.Polygon:
		flat := tt.FlatCoords()
		ends := tt.Ends()
		rings := unflattenRings(flat, ends)
		var ringStrs []string
		for _, ring := range rings {
			// ensure ring is closed (first == last) for valid WKT
			closed := closeRing(ring)
			var pts []string
			for _, p := range closed {
				if len(p) >= 2 {
					pts = append(pts, fmt.Sprintf("%f %f", p[0], p[1]))
				}
			}
			ringStrs = append(ringStrs, fmt.Sprintf("(%s)", joinWKT(pts)))
		}
		return fmt.Sprintf("POLYGON(%s)", joinWKT(ringStrs))
	case *geom.MultiPoint:
		coords := unflattenCoords(tt.FlatCoords())
		var pts []string
		for _, p := range coords {
			pts = append(pts, fmt.Sprintf("(%f %f)", p[0], p[1]))
		}
		return fmt.Sprintf("MULTIPOINT(%s)", joinWKT(pts))
	case *geom.MultiLineString:
		var lineStrs []string
		for i := 0; i < tt.NumLineStrings(); i++ {
			ls := tt.LineString(i)
			coords := unflattenCoords(ls.FlatCoords())
			var pts []string
			for _, p := range coords {
				pts = append(pts, fmt.Sprintf("%f %f", p[0], p[1]))
			}
			lineStrs = append(lineStrs, fmt.Sprintf("(%s)", joinWKT(pts)))
		}
		return fmt.Sprintf("MULTILINESTRING(%s)", joinWKT(lineStrs))
	case *geom.MultiPolygon:
		var polyStrs []string
		for i := 0; i < tt.NumPolygons(); i++ {
			p := tt.Polygon(i)
			rings := unflattenRings(p.FlatCoords(), p.Ends())
			var ringStrs []string
			for _, ring := range rings {
				closed := closeRing(ring)
				var pts []string
				for _, pt := range closed {
					pts = append(pts, fmt.Sprintf("%f %f", pt[0], pt[1]))
				}
				ringStrs = append(ringStrs, fmt.Sprintf("(%s)", joinWKT(pts)))
			}
			polyStrs = append(polyStrs, fmt.Sprintf("(%s)", joinWKT(ringStrs)))
		}
		return fmt.Sprintf("MULTIPOLYGON(%s)", joinWKT(polyStrs))
	case *geom.GeometryCollection:
		var parts []string
		for i := 0; i < tt.NumGeoms(); i++ {
			sub := tt.Geom(i)
			if s := wktFromGeom(sub); s != "" {
				parts = append(parts, s)
			}
		}
		return fmt.Sprintf("GEOMETRYCOLLECTION(%s)", joinWKT(parts))
	}
	return ""
}

// joinWKT reuses the helper defined earlier
func joinWKT(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += ", " + p
	}
	return out
}

// closeRing ensures the first and last coordinate are identical (closing the ring)
func closeRing(ring [][]float64) [][]float64 {
	if len(ring) == 0 {
		return ring
	}
	first := ring[0]
	last := ring[len(ring)-1]
	if len(first) >= 2 && len(last) >= 2 {
		if first[0] == last[0] && first[1] == last[1] {
			return ring
		}
	}
	// append a copy of the first point
	closed := make([][]float64, len(ring)+1)
	copy(closed, ring)
	closed[len(closed)-1] = []float64{first[0], first[1]}
	return closed
}
