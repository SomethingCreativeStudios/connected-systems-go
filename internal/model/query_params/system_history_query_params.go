package queryparams

import (
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// SystemHistoryQueryParams defines filtering for /systems/{id}/history endpoints.
type SystemHistoryQueryParams struct {
	QueryParams

	ValidTime *common_shared.TimeRange
	Keyword   []string
}

func (SystemHistoryQueryParams) BuildFromRequest(r *http.Request) *SystemHistoryQueryParams {
	params := &SystemHistoryQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r),
	}

	if vals := r.URL.Query()["validTime"]; len(vals) > 0 {
		tr := common_shared.ParseTimeRange(vals)
		params.ValidTime = &tr
	}

	if keyword := r.URL.Query().Get("keyword"); keyword != "" {
		params.Keyword = strings.Split(keyword, ",")
	}

	return params
}
