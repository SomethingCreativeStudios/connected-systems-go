package queryparams

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
)

type ProceduresQueryParams struct {
	QueryParams

	DateTime *common_shared.TimeRange

	ObservedProperty   []string
	ControlledProperty []string
}

func (ProceduresQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) (*ProceduresQueryParams, error) {
	base, err := QueryParams{}.BuildFromRequest(r, defaultLimit, CursorKindIDAsc)
	if err != nil {
		return nil, err
	}
	params := &ProceduresQueryParams{
		QueryParams: *base,
	}

	if controlledProperties := r.URL.Query().Get("controlledProperty"); controlledProperties != "" {
		params.ControlledProperty = strings.Split(controlledProperties, ",")
	}

	if observedProperties := r.URL.Query().Get("observedProperty"); observedProperties != "" {
		params.ObservedProperty = strings.Split(observedProperties, ",")
	}

	dateVals, err := datetimeValues(r)
	if err != nil {
		return nil, err
	}
	if len(dateVals) > 0 {
		tr, err := common_shared.ParseTimeRangeStrict(dateVals)
		if err != nil {
			return nil, fmt.Errorf("invalid datetime: %w", err)
		}
		params.DateTime = &tr
	}

	return params, nil
}
