package queryparams

import (
	"net/http"
)

type DeploymentsQueryParams struct {
	QueryParams

	BaseProperty []string
	ObjectType   []string
}

// BuildFromRequest parses common query parameters
func (DeploymentsQueryParams) BuildFromRequest(r *http.Request) *DeploymentsQueryParams {
	params := &DeploymentsQueryParams{
		QueryParams: *QueryParams{}.BuildFromRequest(r),
	}

	return params
}
