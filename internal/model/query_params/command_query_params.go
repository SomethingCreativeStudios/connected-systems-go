package queryparams

import (
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// CommandsQueryParams defines filtering options for command list endpoints.
type CommandsQueryParams struct {
	QueryParams

	IssueTime     *common_shared.TimeRange
	ExecutionTime *common_shared.TimeRange

	ControlStream []string
	System        []string
	FOI           []string
	CurrentStatus []string
}

// BuildFromRequest parses command query parameters from request.
func (CommandsQueryParams) BuildFromRequest(r *http.Request) *CommandsQueryParams {
	params := &CommandsQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r),
	}

	if cs := r.URL.Query().Get("controlStream"); cs != "" {
		params.ControlStream = strings.Split(cs, ",")
	}

	if system := r.URL.Query().Get("system"); system != "" {
		params.System = strings.Split(system, ",")
	}

	if foi := r.URL.Query().Get("foi"); foi != "" {
		params.FOI = strings.Split(foi, ",")
	}

	if status := r.URL.Query().Get("currentStatus"); status != "" {
		params.CurrentStatus = strings.Split(status, ",")
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
