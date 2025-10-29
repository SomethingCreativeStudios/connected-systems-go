package queryparams

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

type QueryParams struct {
	IDs []string
	Q   string // Full-text search

	Limit  int
	Offset int // Not part of standard, but useful for pagination (till i do curorsors)
}

func (QueryParams) BuildFromRequest(r *http.Request) *QueryParams {
	params := &QueryParams{
		Limit:  10,
		Offset: 0,
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			params.Limit = val
		}
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if val, err := strconv.Atoi(offset); err == nil {
			params.Offset = val
		}
	}

	if ids := r.URL.Query().Get("id"); ids != "" {
		params.IDs = strings.Split(ids, ",")
	}

	params.Q = r.URL.Query().Get("q")

	return params
}

func (qp *QueryParams) BuildPagintationLinks(baseURL string, params url.Values, total *int, returned int) common_shared.Links {
	currentOffsetStr := params.Get("offset")
	currentOffset := 0

	if currentOffsetStr != "" {
		if val, err := strconv.Atoi(currentOffsetStr); err == nil {
			currentOffset = val
		}
	}

	links := common_shared.Links{
		common_shared.Link{Href: baseURL + "?" + params.Encode(), Rel: "self"},
	}

	if (currentOffset + returned) < *total {
		nextLink := params
		nextLink.Set("offset", strconv.Itoa(currentOffset+returned))

		links = append(links, common_shared.Link{
			Rel:  "next",
			Href: baseURL + "?" + nextLink.Encode(),
		})
	}

	if currentOffset > 0 {
		prevLink := params
		if currentOffset-qp.Limit <= 0 {
			prevLink.Del("offset")
		} else {
			prevLink.Set("offset", strconv.Itoa(currentOffset-qp.Limit))
		}

		links = append(links, common_shared.Link{
			Rel:  "prev",
			Href: baseURL + "?" + prevLink.Encode(),
		})
	}

	return links
}
