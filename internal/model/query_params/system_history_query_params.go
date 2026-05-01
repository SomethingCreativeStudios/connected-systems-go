package queryparams

import (
	"fmt"
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

func (SystemHistoryQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) (*SystemHistoryQueryParams, error) {
	params := &SystemHistoryQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r, defaultLimit),
	}

	if vals := r.URL.Query()["validTime"]; len(vals) > 0 {
		tr, err := common_shared.ParseTimeRangeStrict(vals)
		if err != nil {
			return nil, fmt.Errorf("invalid validTime: %w", err)
		}
		params.ValidTime = &tr
	}

	if keyword := r.URL.Query().Get("keyword"); keyword != "" {
		params.Keyword = strings.Split(keyword, ",")
	}

	return params, nil
}
