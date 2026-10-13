package formaters

import "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"

// AnyFeatureCollection represents a feature collection where the features can be any type
type AnyFeatureCollection struct {
	Type           string              `json:"type"`
	Features       []any               `json:"features"`
	NumberMatched  *int                `json:"numberMatched,omitempty"`
	NumberReturned int                 `json:"numberReturned"`
	Links          common_shared.Links `json:"links"`
}

// AnyItemCollection represents a SensorML (application/sml+json) collection,
// which uses a plain "items" array rather than a GeoJSON FeatureCollection envelope.
type AnyItemCollection struct {
	Items []any               `json:"items"`
	Links common_shared.Links `json:"links"`
}
