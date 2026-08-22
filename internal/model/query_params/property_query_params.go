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
func (PropertiesQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) (*PropertiesQueryParams, error) {
	base, err := QueryParams{}.BuildFromRequest(r, defaultLimit, CursorKindIDAsc)
	if err != nil {
		return nil, err
	}
	params := &PropertiesQueryParams{
		QueryParams: *base,
	}

	if baseProps := r.URL.Query().Get("baseProperty"); baseProps != "" {
		params.BaseProperty = strings.Split(baseProps, ",")
	}

	if objTypes := r.URL.Query().Get("objectType"); objTypes != "" {
		params.ObjectType = strings.Split(objTypes, ",")
	}

	return params, nil
}
