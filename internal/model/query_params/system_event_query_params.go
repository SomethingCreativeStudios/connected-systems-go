package queryparams

import (
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// SystemEventsQueryParams defines filtering options for system event endpoints.
type SystemEventsQueryParams struct {
	QueryParams

	EventTime *common_shared.TimeRange
	EventType []string
	Keyword   []string
	System    []string
}

func (SystemEventsQueryParams) BuildFromRequest(r *http.Request) *SystemEventsQueryParams {
	params := &SystemEventsQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r),
	}

	if vals := r.URL.Query()["datetime"]; len(vals) > 0 {
		tr := common_shared.ParseTimeRange(vals)
		params.EventTime = &tr
	} else if vals := r.URL.Query()["eventTime"]; len(vals) > 0 {
		tr := common_shared.ParseTimeRange(vals)
		params.EventTime = &tr
	}

	if eventType := r.URL.Query().Get("eventType"); eventType != "" {
		params.EventType = strings.Split(eventType, ",")
	}

	if keyword := r.URL.Query().Get("keyword"); keyword != "" {
		params.Keyword = strings.Split(keyword, ",")
	}

	if system := r.URL.Query().Get("system"); system != "" {
		params.System = strings.Split(system, ",")
	}

	return params
}
