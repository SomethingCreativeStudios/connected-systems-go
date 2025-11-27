package model

import (
	"context"
	"net/url"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	serializers "github.com/yourusername/connected-systems-go/internal/model/formaters"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
)

type FeatureCollection[T any, S any] struct {
	Type           string              `json:"type"`
	Features       []T                 `json:"features"`
	NumberMatched  *int                `json:"numberMatched,omitempty"`
	NumberReturned int                 `json:"numberReturned"`
	Links          common_shared.Links `json:"links"`
}

func (FeatureCollection[T, S]) BuildCollection(items []S, serializer serializers.Serializer[T, S], basePath string, total int, requestParams url.Values, queryParams queryparams.QueryParams) FeatureCollection[T, S] {
	features, err := serializer.SerializeAll(context.Background(), items)

	if err != nil {
		features = []T{}
	}

	totalInt := int(total)
	return FeatureCollection[T, S]{
		Type:           "FeatureCollection",
		Features:       features,
		NumberMatched:  &totalInt,
		NumberReturned: len(items),
		Links:          queryParams.BuildPagintationLinks(basePath, requestParams, &totalInt, len(items)),
	}
}
