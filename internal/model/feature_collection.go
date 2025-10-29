package model

import (
	"net/url"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/model/seriallizers"
)

type FeatureCollection[T any, S seriallizers.GeoJsonSeriallizable[T]] struct {
	Type           string              `json:"type"`
	Features       []S                 `json:"features"`
	NumberMatched  *int                `json:"numberMatched,omitempty"`
	NumberReturned int                 `json:"numberReturned"`
	Links          common_shared.Links `json:"links"`
}

func (FeatureCollection[T, S]) BuildCollection(items []S, basePath string, total int, requestParams url.Values, queryParams queryparams.QueryParams) FeatureCollection[T, S] {
	features := make([]T, len(items))

	for i, item := range items {
		features[i] = item.ToGeoJSON()
	}

	totalInt := int(total)
	return FeatureCollection[T, S]{
		Type:           "FeatureCollection",
		Features:       items,
		NumberMatched:  &totalInt,
		NumberReturned: len(items),
		Links:          queryParams.BuildPagintationLinks(basePath, requestParams, &totalInt, len(items)),
	}
}
