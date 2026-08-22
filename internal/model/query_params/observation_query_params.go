package queryparams

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// ObservationsQueryParams defines filtering options for observations endpoints.
type ObservationsQueryParams struct {
	QueryParams

	PhenomenonTime *common_shared.TimeRange
	ResultTime     *common_shared.TimeRange

	DataStream       []string
	System           []string
	FOI              []string
	ObservedProperty []string
}

// BuildFromRequest parses observation query parameters from request.
// Returns an error if a temporal parameter value is present but unparseable.
func (ObservationsQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) (*ObservationsQueryParams, error) {
	base, err := QueryParams{}.BuildFromRequest(r, defaultLimit, CursorKindTimeDesc)
	if err != nil {
		return nil, err
	}
	params := &ObservationsQueryParams{
		QueryParams: *base,
	}

	if foi := r.URL.Query().Get("foi"); foi != "" {
		params.FOI = strings.Split(foi, ",")
	}

	if dataStreams := r.URL.Query().Get("dataStream"); dataStreams != "" {
		params.DataStream = strings.Split(dataStreams, ",")
	}

	if systems := r.URL.Query().Get("system"); systems != "" {
		params.System = strings.Split(systems, ",")
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
