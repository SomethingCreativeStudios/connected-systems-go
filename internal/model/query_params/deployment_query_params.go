package queryparams

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

type DeploymentsQueryParams struct {
	QueryParams

	Bbox               *common_shared.BoundingBox
	Geom               string // WKT geometry
	DateTime           *common_shared.TimeRange
	ObservedProperty   []string
	ControlledProperty []string
	Parent             []string
	System             []string
	Foi                []string
	Recursive          bool
}

func (DeploymentsQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) (*DeploymentsQueryParams, error) {
	base, err := QueryParams{}.BuildFromRequest(r, defaultLimit, CursorKindIDAsc)
	if err != nil {
		return nil, err
	}
	params := &DeploymentsQueryParams{
		QueryParams: *base,
	}
	bbox, geom, err := buildSpatialQueryParams(r)
	if err != nil {
		return nil, err
	}
	params.Bbox = bbox
	params.Geom = geom

	if observedProperty := r.URL.Query().Get("observedProperty"); observedProperty != "" {
		params.ObservedProperty = strings.Split(observedProperty, ",")
	}

	if controlledProperty := r.URL.Query().Get("controlledProperty"); controlledProperty != "" {
		params.ControlledProperty = strings.Split(controlledProperty, ",")
	}

	if system := r.URL.Query().Get("system"); system != "" {
		params.System = strings.Split(system, ",")
	}

	if foi := r.URL.Query().Get("foi"); foi != "" {
		params.Foi = strings.Split(foi, ",")
	}

	if parent := r.URL.Query().Get("parent"); parent != "" {
		params.Parent = strings.Split(parent, ",")
	}

	if r.URL.Query().Get("recursive") == "true" {
		params.Recursive = true
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
