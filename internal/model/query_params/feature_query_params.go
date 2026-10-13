package queryparams

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
)

// FeatureQueryParams holds OGC API Features query parameters
type FeatureQueryParams struct {
	QueryParams

	// OGC API Features standard parameters
	BBox     *common_shared.BoundingBox `json:"bbox,omitempty"`     // Bounding box filter
	DateTime *common_shared.TimeRange   `json:"datetime,omitempty"` // Temporal filter

	// Collection ID (for features within a collection)
	CollectionID string `json:"collectionId,omitempty"`
}

// BuildFromRequest parses query parameters from HTTP request
func (FeatureQueryParams) BuildFromRequest(r *http.Request, defaultLimit int) (*FeatureQueryParams, error) {
	baseParams, err := QueryParams{}.BuildFromRequest(r, defaultLimit, CursorKindIDAsc)
	if err != nil {
		return nil, err
	}
	params := &FeatureQueryParams{}
	params.QueryParams = *baseParams

	if raw := strings.TrimSpace(r.URL.Query().Get("bbox")); raw != "" {
		if params.BBox, err = parseBoundingBox(raw); err != nil {
			return nil, err
		}
	}

	dateVals, err := datetimeValues(r)
	if err != nil {
		return nil, err
	}
	if len(dateVals) > 0 {
		tr, err := common_shared.ParseTimeRangeStrict(dateVals)
		if err != nil {
			return nil, fmt.Errorf("invalid datetime: %w", err)
		}
		params.DateTime = &tr
	}

	return params, nil
}
