package queryparams

import (
	"fmt"
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
func (CommandsQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) (*CommandsQueryParams, error) {
	base, err := QueryParams{}.BuildFromRequest(r, defaultLimit, CursorKindTimeDesc)
	if err != nil {
		return nil, err
	}
	params := &CommandsQueryParams{
		QueryParams: *base,
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
		tr, err := common_shared.ParseTimeRangeStrict(vals)
		if err != nil {
			return nil, fmt.Errorf("invalid issueTime: %w", err)
		}
		params.IssueTime = &tr
	}

	if vals := r.URL.Query()["executionTime"]; len(vals) > 0 {
		tr, err := common_shared.ParseTimeRangeStrict(vals)
		if err != nil {
			return nil, fmt.Errorf("invalid executionTime: %w", err)
		}
		params.ExecutionTime = &tr
	}

	return params, nil
}
