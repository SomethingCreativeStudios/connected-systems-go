package queryparams

import (
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// ObservationsQueryParams defines filtering options for observations endpoints.
type ObservationsQueryParams struct {
	QueryParams

	PhenomenonTime *common_shared.TimeRange
	ResultTime     *common_shared.TimeRange

	DataStream       []string
	System           []string
	FOI              []string
	ObservedProperty []string
}

// BuildFromRequest parses observation query parameters from request.
func (ObservationsQueryParams) BuildFromRequest(r *http.Request) *ObservationsQueryParams {
	params := &ObservationsQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r),
	}

	if foi := r.URL.Query().Get("foi"); foi != "" {
		params.FOI = strings.Split(foi, ",")
	}

	if dataStreams := r.URL.Query().Get("dataStream"); dataStreams != "" {
		params.DataStream = strings.Split(dataStreams, ",")
	}

	if systems := r.URL.Query().Get("system"); systems != "" {
		params.System = strings.Split(systems, ",")
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
