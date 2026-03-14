package queryparams

import (
	"net/http"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

type DeploymentsQueryParams struct {
	QueryParams

	DateTime           *common_shared.TimeRange
	ObservedProperty   []string
	ControlledProperty []string
	Parent             []string
	System             []string
	Foi                []string
	Recursive          bool
}

// BuildFromRequest parses common query parameters
func (DeploymentsQueryParams) BuildFromRequest(r *http.Request) *DeploymentsQueryParams {
	params := &DeploymentsQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r),
	}

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

	return params
}
