package queryparams

import (
	"fmt"
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

func (SystemEventsQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) (*SystemEventsQueryParams, error) {
	params := &SystemEventsQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r, defaultLimit),
	}

	// Accept both "datetime" (OGC standard) and "eventTime" (legacy alias)
	timeKey := "datetime"
	if vals := r.URL.Query()["datetime"]; len(vals) == 0 {
		timeKey = "eventTime"
	}
	if vals := r.URL.Query()[timeKey]; len(vals) > 0 {
		tr, err := common_shared.ParseTimeRangeStrict(vals)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", timeKey, err)
		}
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

	return params, nil
}
