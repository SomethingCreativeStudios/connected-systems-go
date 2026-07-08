package queryparams

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// CommandStatusQueryParams defines filtering options for command status reports.
type CommandStatusQueryParams struct {
	QueryParams

	ReportTime *common_shared.TimeRange
	StatusCode []string
}

// BuildFromRequest parses command status query parameters from request.
func (CommandStatusQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) (*CommandStatusQueryParams, error) {
	params := &CommandStatusQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r, defaultLimit),
	}

	if status := r.URL.Query().Get("statusCode"); status != "" {
		params.StatusCode = strings.Split(status, ",")
	}

	if vals := r.URL.Query()["reportTime"]; len(vals) > 0 {
		tr, err := common_shared.ParseTimeRangeStrict(vals)
		if err != nil {
			return nil, fmt.Errorf("invalid reportTime: %w", err)
		}
		params.ReportTime = &tr
	}

	return params, nil
}

// CommandResultQueryParams defines filtering options for command results.
type CommandResultQueryParams struct {
	QueryParams
}

// BuildFromRequest parses command result query parameters from request.
func (CommandResultQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) *CommandResultQueryParams {
	return &CommandResultQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r, defaultLimit),
	}
}
