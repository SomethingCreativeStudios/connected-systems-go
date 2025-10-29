package queryparams

import (
	"net/http"
	"strings"
)

type PropertiesQueryParams struct {
	QueryParams

	BaseProperty []string
	ObjectType   []string
}

// parseQueryParams parses common query parameters
func (PropertiesQueryParams) BuildFromRequest(r *http.Request) *PropertiesQueryParams {
	params := &PropertiesQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r),
	}

	if baseProps := r.URL.Query().Get("baseProperty"); baseProps != "" {
		params.BaseProperty = strings.Split(baseProps, ",")
	}

	if objTypes := r.URL.Query().Get("objectType"); objTypes != "" {
		params.ObjectType = strings.Split(objTypes, ",")
	}

	return params
}
