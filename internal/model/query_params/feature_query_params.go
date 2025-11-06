package queryparams

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// FeatureQueryParams holds OGC API Features query parameters
type FeatureQueryParams struct {
	QueryParams

	// OGC API Features standard parameters
	BBox     []float64   `json:"bbox,omitempty"`     // Bounding box filter [minLon,minLat,maxLon,maxLat]
	DateTime *TimeFilter `json:"datetime,omitempty"` // Temporal filter

	// Collection ID (for features within a collection)
	CollectionID string `json:"collectionId,omitempty"`
}

// TimeFilter represents a temporal filter (instant or interval)
type TimeFilter struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// BuildFromRequest parses query parameters from HTTP request
func (FeatureQueryParams) BuildFromRequest(r *http.Request) *FeatureQueryParams {
	params := &FeatureQueryParams{}
	baseParams := QueryParams{}.BuildFromRequest(r)
	if baseParams != nil {
		params.QueryParams = *baseParams
	}

	// Parse bbox parameter
	if bboxStr := r.URL.Query().Get("bbox"); bboxStr != "" {
		coords := strings.Split(bboxStr, ",")
		bbox := make([]float64, 0, len(coords))
		for _, coord := range coords {
			if val, err := strconv.ParseFloat(strings.TrimSpace(coord), 64); err == nil {
				bbox = append(bbox, val)
			}
		}
		if len(bbox) == 4 || len(bbox) == 6 {
			params.BBox = bbox
		}
	}

	// Parse datetime parameter
	if dtStr := r.URL.Query().Get("datetime"); dtStr != "" {
		params.DateTime = parseDateTime(dtStr)
	}

	return params
}

// parseDateTime parses OGC API datetime parameter
// Supports:
// - Single instant: "2018-02-12T23:20:50Z"
// - Interval: "2018-02-12T00:00:00Z/2018-03-18T12:31:12Z"
// - Open start: "../2018-03-18T12:31:12Z"
// - Open end: "2018-02-12T00:00:00Z/.."
func parseDateTime(dtStr string) *TimeFilter {
	filter := &TimeFilter{}

	// Check for interval (contains "/")
	if strings.Contains(dtStr, "/") {
		parts := strings.Split(dtStr, "/")
		if len(parts) == 2 {
			// Parse start
			if parts[0] != "" && parts[0] != ".." {
				if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
					filter.Start = &t
				}
			}
			// Parse end
			if parts[1] != "" && parts[1] != ".." {
				if t, err := time.Parse(time.RFC3339, parts[1]); err == nil {
					filter.End = &t
				}
			}
		}
	} else {
		// Single instant - use as both start and end
		if t, err := time.Parse(time.RFC3339, dtStr); err == nil {
			filter.Start = &t
			filter.End = &t
		}
	}

	return filter
}
