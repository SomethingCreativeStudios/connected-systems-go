package queryparams

import (
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

type ProceduresQueryParams struct {
	QueryParams

	DateTime *common_shared.TimeRange

	ObservedProperty   []string
	ControlledProperty []string
}

// parseQueryParams parses common query parameters
func (ProceduresQueryParams) BuildFromRequest(r *http.Request) *ProceduresQueryParams {
	params := &ProceduresQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r),
	}

	if controlledProperties := r.URL.Query().Get("controlledProperty"); controlledProperties != "" {
		params.ControlledProperty = strings.Split(controlledProperties, ",")
	}

	if observedProperties := r.URL.Query().Get("observedProperty"); observedProperties != "" {
		params.ObservedProperty = strings.Split(observedProperties, ",")
	}

	// dateTime may be provided as a single value (string) or as repeated query params
	// where index 0 = start, index 1 = end.
	if dateVals := r.URL.Query()["dateTime"]; len(dateVals) > 0 {
		var tr common_shared.TimeRange
		if len(dateVals) == 1 {
			tr = common_shared.ToTimeRange(dateVals[0])
		} else {
			tr = common_shared.ToTimeRangeFromSlice(dateVals)
		}
		params.DateTime = &tr
	}

	return params
}
