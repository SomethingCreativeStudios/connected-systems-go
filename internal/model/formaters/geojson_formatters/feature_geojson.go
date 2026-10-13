package geojson_formatters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

// FeatureGeoJSONFormatter handles serialization and deserialization of Feature objects in GeoJSON format
type FeatureGeoJSONFormatter struct {
	formaters.Formatter[domains.FeatureGeoJSONFeature, *domains.Feature]
	links *formaters.LinkResolver
}

// NewFeatureGeoJSONFormatter constructs a formatter with required repository readers
func NewFeatureGeoJSONFormatter(links *formaters.LinkResolver) *FeatureGeoJSONFormatter {
	return &FeatureGeoJSONFormatter{links: links}
}

func (f *FeatureGeoJSONFormatter) ContentType() string {
	return mediatype.GeoJSON
}

// --- Serialization ---

func (f *FeatureGeoJSONFormatter) Serialize(ctx context.Context, feature *domains.Feature) (domains.FeatureGeoJSONFeature, error) {
	return formaters.SerializeOne(ctx, feature, f.SerializeAll)
}

func (f *FeatureGeoJSONFormatter) SerializeAll(ctx context.Context, features []*domains.Feature) ([]domains.FeatureGeoJSONFeature, error) {
	result := make([]domains.FeatureGeoJSONFeature, 0, len(features))

	for _, feature := range features {
		if feature == nil {
			continue
		}
		result = append(result, feature.ToGeoJSON())
	}

	return result, nil
}

// --- Deserialization ---

func (f *FeatureGeoJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Feature, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	geoJSON, err := common_shared.DecodeWithFieldErrors[struct {
		Type       string                `json:"type"`
		ID         string                `json:"id,omitempty"`
		Properties map[string]any        `json:"properties"`
		Geometry   *common_shared.GoGeom `json:"geometry,omitempty"`
		Links      common_shared.Links   `json:"links,omitempty"`
	}](body)
	if err != nil {
		return nil, err
	}

	// Strip known fields from properties; only extra/arbitrary props are stored in Properties column
	knownKeys := map[string]bool{
		"uid": true, "name": true, "description": true, "collectionId": true,
		"dateTime": true, "validTime": true,
	}
	extraProps := make(common_shared.Properties)
	for k, v := range geoJSON.Properties {
		if !knownKeys[k] {
			extraProps[k] = v
		}
	}

	// Convert GeoJSON properties to Feature model
	feature := domains.Feature{
		Links:      common_shared.StripAssociationLinks(geoJSON.Links),
		Properties: extraProps,
	}
	// assign geometry (decoded directly into GoGeom)
	if geoJSON.Geometry != nil {
		feature.Geometry = geoJSON.Geometry
	}

	// Extract standard properties
	if uid, ok := geoJSON.Properties["uid"].(string); ok {
		feature.UniqueIdentifier = domains.UniqueID(uid)
	}
	if name, ok := geoJSON.Properties["name"].(string); ok {
		feature.Name = name
	}
	if desc, ok := geoJSON.Properties["description"].(string); ok {
		feature.Description = desc
	}
	if collectionID, ok := geoJSON.Properties["collectionId"].(string); ok {
		feature.CollectionID = collectionID
	}

	if raw, ok := geoJSON.Properties["dateTime"]; ok && raw != nil {
		dtStr, isString := raw.(string)
		dt, err := time.Parse(time.RFC3339, dtStr)
		if !isString || err != nil {
			return nil, fmt.Errorf("properties.dateTime must be an RFC 3339 date-time")
		}
		feature.DateTime = &dt
	}

	// validTime goes through the same strict decoder as every other body time
	// range (array, object or "start/end" string forms).
	if vt, ok := geoJSON.Properties["validTime"]; ok && vt != nil {
		raw, err := json.Marshal(vt)
		if err != nil {
			return nil, err
		}
		var tr common_shared.TimeRange
		if err := tr.UnmarshalJSON(raw); err != nil {
			return nil, fmt.Errorf("properties.validTime: %w", err)
		}
		feature.ValidTime = common_shared.NonEmptyTimeRange(&tr)
	}

	return &feature, nil
}
