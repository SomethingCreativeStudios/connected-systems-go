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

	if dateTime := r.URL.Query().Get("dateTime"); dateTime != "" {
		tr := common_shared.ToTimeRange(dateTime)
		params.DateTime = &tr
	}

	return params
}
