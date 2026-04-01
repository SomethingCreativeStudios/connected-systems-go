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

	if procedure := r.URL.Query().Get("procedure"); procedure != "" {
		params.Procedure = strings.Split(procedure, ",")
	}

	if foi := r.URL.Query().Get("foi"); foi != "" {
		params.FOI = strings.Split(foi, ",")
	}

	if observedProperty := r.URL.Query().Get("observedProperty"); observedProperty != "" {
		params.ObservedProperty = strings.Split(observedProperty, ",")
	}

	if controlledProperty := r.URL.Query().Get("controlledProperty"); controlledProperty != "" {
		params.ControlledProperty = strings.Split(controlledProperty, ",")
	}

	if geom := r.URL.Query().Get("geom"); geom != "" {
		params.Geom = geom
	}

	return params
}
