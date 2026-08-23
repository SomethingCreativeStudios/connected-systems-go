package queryparams

import (
	"fmt"
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

func (SamplingFeatureQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) (*SamplingFeatureQueryParams, error) {
	base, err := QueryParams{}.BuildFromRequest(r, defaultLimit, CursorKindIDAsc)
	if err != nil {
		return nil, err
	}
	params := &SamplingFeatureQueryParams{
		QueryParams: *base,
	}
	bbox, geom, err := buildSpatialQueryParams(r)
	if err != nil {
		return nil, err
	}
	params.Bbox = bbox
	params.Geom = geom

	if controlledProperty := r.URL.Query().Get("controlledProperty"); controlledProperty != "" {
		params.ControlledProperty = strings.Split(controlledProperty, ",")
	}

	if observedProperty := r.URL.Query().Get("observedProperty"); observedProperty != "" {
		params.ObservedProperty = strings.Split(observedProperty, ",")
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
