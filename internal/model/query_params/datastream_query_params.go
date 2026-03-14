package queryparams

import (
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// DatastreamsQueryParams defines filtering options for datastream list endpoints.
type DatastreamsQueryParams struct {
	QueryParams

	PhenomenonTime *common_shared.TimeRange
	ResultTime     *common_shared.TimeRange

	System           []string
	FOI              []string
	ObservedProperty []string
}

// BuildFromRequest parses datastream query parameters from request.
func (DatastreamsQueryParams) BuildFromRequest(r *http.Request) *DatastreamsQueryParams {
	params := &DatastreamsQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r),
	}

	if system := r.URL.Query().Get("system"); system != "" {
		params.System = strings.Split(system, ",")
	}

	if foi := r.URL.Query().Get("foi"); foi != "" {
		params.FOI = strings.Split(foi, ",")
	}

	if observedProperty := r.URL.Query().Get("observedProperty"); observedProperty != "" {
		params.ObservedProperty = strings.Split(observedProperty, ",")
	}

	if vals := r.URL.Query()["phenomenonTime"]; len(vals) > 0 {
		tr := common_shared.ParseTimeRange(vals)
		params.PhenomenonTime = &tr
	}

	if vals := r.URL.Query()["resultTime"]; len(vals) > 0 {
		tr := common_shared.ParseTimeRange(vals)
		params.ResultTime = &tr
	}

	return params
}
