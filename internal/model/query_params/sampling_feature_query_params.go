package queryparams

import (
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

type SamplingFeatureQueryParams struct {
	QueryParams

	Geom string // WKT geometry

	DateTime *common_shared.TimeRange
	Bbox     *common_shared.BoundingBox

	ObservedProperty   []string
	ControlledProperty []string
	FOI                []string
}

func (SamplingFeatureQueryParams) BuildFromRequest(r *http.Request) (*SamplingFeatureQueryParams, error) {
	params := &SamplingFeatureQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r),
	}

	if controlledProperty := r.URL.Query().Get("controlledProperty"); controlledProperty != "" {
		params.ControlledProperty = strings.Split(controlledProperty, ",")
	}

	if observedProperty := r.URL.Query().Get("observedProperty"); observedProperty != "" {
		params.ObservedProperty = strings.Split(observedProperty, ",")
	}

	return params, nil
}
