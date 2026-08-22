package queryparams

import (
	"fmt"
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
func (ControlStreamsQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) (*ControlStreamsQueryParams, error) {
	base, err := QueryParams{}.BuildFromRequest(r, defaultLimit, CursorKindIDAsc)
	if err != nil {
		return nil, err
	}
	params := &ControlStreamsQueryParams{
		QueryParams: *base,
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
