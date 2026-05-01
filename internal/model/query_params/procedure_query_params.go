package queryparams

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

type ProceduresQueryParams struct {
	QueryParams

	DateTime *common_shared.TimeRange

	ObservedProperty   []string
	ControlledProperty []string
}

func (ProceduresQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) (*ProceduresQueryParams, error) {
	params := &ProceduresQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r, defaultLimit),
	}

	if controlledProperties := r.URL.Query().Get("controlledProperty"); controlledProperties != "" {
		params.ControlledProperty = strings.Split(controlledProperties, ",")
	}

	if observedProperties := r.URL.Query().Get("observedProperty"); observedProperties != "" {
		params.ObservedProperty = strings.Split(observedProperties, ",")
	}

	if dateVals := r.URL.Query()["dateTime"]; len(dateVals) > 0 {
		tr, err := common_shared.ParseTimeRangeStrict(dateVals)
		if err != nil {
			return nil, fmt.Errorf("invalid dateTime: %w", err)
		}
		params.DateTime = &tr
	}

	return params, nil
}
