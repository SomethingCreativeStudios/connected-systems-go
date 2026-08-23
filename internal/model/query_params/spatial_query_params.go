package queryparams

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/twpayne/go-geom/encoding/wkt"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// buildSpatialQueryParams parses the Part 1 spatial collection filters. The
// API currently stores resource geometries in CRS84/EPSG:4326 and performs
// two-dimensional intersection tests. A six-value bbox is accepted per the
// contract; its vertical bounds are validated while the XY bounds are used for
// the intersection because the stored geometries are two-dimensional.
func buildSpatialQueryParams(r *http.Request) (*common_shared.BoundingBox, string, error) {
	var bbox *common_shared.BoundingBox
	if raw := strings.TrimSpace(r.URL.Query().Get("bbox")); raw != "" {
		parsed, err := parseBoundingBox(raw)
		if err != nil {
			return nil, "", err
		}
		bbox = parsed
	}

	geom := strings.TrimSpace(r.URL.Query().Get("geom"))
	if geom != "" {
		if _, err := wkt.Unmarshal(geom); err != nil {
			return nil, "", fmt.Errorf("invalid geom: %w", err)
		}
	}

	return bbox, geom, nil
}

func parseBoundingBox(raw string) (*common_shared.BoundingBox, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 4 && len(parts) != 6 {
		return nil, fmt.Errorf("invalid bbox: expected 4 or 6 comma-separated numbers")
	}

	values := make([]float64, len(parts))
	for i, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("invalid bbox: coordinate %d must be a finite number", i+1)
		}
		values[i] = value
	}

	maxXIndex, maxYIndex := 2, 3
	if len(values) == 6 {
		maxXIndex, maxYIndex = 3, 4
		if values[2] > values[5] {
			return nil, fmt.Errorf("invalid bbox: minimum vertical coordinate must not exceed maximum vertical coordinate")
		}
	}

	minX, minY := values[0], values[1]
	maxX, maxY := values[maxXIndex], values[maxYIndex]
	if minX < -180 || minX > 180 || maxX < -180 || maxX > 180 {
		return nil, fmt.Errorf("invalid bbox: longitude must be between -180 and 180")
	}
	if minY < -90 || minY > 90 || maxY < -90 || maxY > 90 {
		return nil, fmt.Errorf("invalid bbox: latitude must be between -90 and 90")
	}
	if minY > maxY {
		return nil, fmt.Errorf("invalid bbox: minimum latitude must not exceed maximum latitude")
	}

	// minX > maxX is valid and represents a bbox crossing the antimeridian.
	return &common_shared.BoundingBox{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY}, nil
}
