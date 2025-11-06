package queryparams

import (
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

type SystemQueryParams struct {
	QueryParams

	Bbox               *common_shared.BoundingBox
	Datetime           *common_shared.TimeRange
	Geom               string // WKT geometry
	Parent             []string
	Procedure          []string
	FOI                []string
	ObservedProperty   []string
	ControlledProperty []string
	Recursive          bool
}

func (SystemQueryParams) BuildFromRequest(r *http.Request) *SystemQueryParams {
	params := &SystemQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r),
	}

	params.Recursive = r.URL.Query().Get("recursive") == "true"

	if parent := r.URL.Query().Get("parent"); parent != "" {
		params.Parent = strings.Split(parent, ",")
	}

	// dateTime may be supplied as a single string or as repeated parameters
	if dateVals := r.URL.Query()["dateTime"]; len(dateVals) > 0 {
		var tr common_shared.TimeRange
		if len(dateVals) == 1 {
			tr = common_shared.ToTimeRange(dateVals[0])
		} else {
			tr = common_shared.ToTimeRangeFromSlice(dateVals)
		}
		params.Datetime = &tr
	}

	return params
}
