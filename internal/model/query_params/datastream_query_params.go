package queryparams

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// DatastreamsQueryParams defines filtering options for datastream list endpoints.
type DatastreamsQueryParams struct {
	QueryParams

	PhenomenonTime *common_shared.TimeRange
	ResultTime     *common_shared.TimeRange

	System           []string
	FOI              []string
	ObservedProperty []string
}

// BuildFromRequest parses datastream query parameters from request.
func (DatastreamsQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) (*DatastreamsQueryParams, error) {
	params := &DatastreamsQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r, defaultLimit),
	}

	if system := r.URL.Query().Get("system"); system != "" {
		params.System = strings.Split(system, ",")
	}

	if foi := r.URL.Query().Get("foi"); foi != "" {
		params.FOI = strings.Split(foi, ",")
	}

	if observedProperty := r.URL.Query().Get("observedProperty"); observedProperty != "" {
		params.ObservedProperty = strings.Split(observedProperty, ",")
	}

	if vals := r.URL.Query()["phenomenonTime"]; len(vals) > 0 {
		tr, err := common_shared.ParseTimeRangeStrict(vals)
		if err != nil {
			return nil, fmt.Errorf("invalid phenomenonTime: %w", err)
		}
		params.PhenomenonTime = &tr
	}

	if vals := r.URL.Query()["resultTime"]; len(vals) > 0 {
		tr, err := common_shared.ParseTimeRangeStrict(vals)
		if err != nil {
			return nil, fmt.Errorf("invalid resultTime: %w", err)
		}
		params.ResultTime = &tr
	}

	return params, nil
}
