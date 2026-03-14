package queryparams

import (
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// ControlStreamsQueryParams defines filtering options for control stream list endpoints.
type ControlStreamsQueryParams struct {
	QueryParams

	IssueTime     *common_shared.TimeRange
	ExecutionTime *common_shared.TimeRange

	System             []string
	FOI                []string
	ControlledProperty []string
}

// BuildFromRequest parses control stream query parameters from request.
func (ControlStreamsQueryParams) BuildFromRequest(r *http.Request) *ControlStreamsQueryParams {
	params := &ControlStreamsQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r),
	}

	if system := r.URL.Query().Get("system"); system != "" {
		params.System = strings.Split(system, ",")
	}

	if foi := r.URL.Query().Get("foi"); foi != "" {
		params.FOI = strings.Split(foi, ",")
	}

	if cp := r.URL.Query().Get("controlledProperty"); cp != "" {
		params.ControlledProperty = strings.Split(cp, ",")
	}

	if vals := r.URL.Query()["issueTime"]; len(vals) > 0 {
		tr := common_shared.ParseTimeRange(vals)
		params.IssueTime = &tr
	}

	if vals := r.URL.Query()["executionTime"]; len(vals) > 0 {
		tr := common_shared.ParseTimeRange(vals)
		params.ExecutionTime = &tr
	}

	return params
}
