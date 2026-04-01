package geojson_formatters

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// FeatureGeoJSONFormatter handles serialization and deserialization of Feature objects in GeoJSON format
type FeatureGeoJSONFormatter struct {
	formaters.Formatter[domains.FeatureGeoJSONFeature, *domains.Feature]
	repos *repository.Repositories
}

// NewFeatureGeoJSONFormatter constructs a formatter with required repository readers
func NewFeatureGeoJSONFormatter(repos *repository.Repositories) *FeatureGeoJSONFormatter {
	return &FeatureGeoJSONFormatter{repos: repos}
}

func (f *FeatureGeoJSONFormatter) ContentType() string {
	return GeoJSONContentType
}

// --- Serialization ---

func (f *FeatureGeoJSONFormatter) Serialize(ctx context.Context, feature *domains.Feature) (domains.FeatureGeoJSONFeature, error) {
	features, err := f.SerializeAll(ctx, []*domains.Feature{feature})
	if err != nil {
		return domains.FeatureGeoJSONFeature{}, err
	}
	return features[0], nil
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
	// Decode GeoJSON Feature format
	var geoJSON struct {
		Type       string                 `json:"type"`
		ID         string                 `json:"id,omitempty"`
		Properties map[string]interface{} `json:"properties"`
		Geometry   *common_shared.GoGeom  `json:"geometry,omitempty"`
		Links      common_shared.Links    `json:"links,omitempty"`
	}

	if err := json.NewDecoder(reader).Decode(&geoJSON); err != nil {
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

	// Parse dateTime if present
	if dtStr, ok := geoJSON.Properties["dateTime"].(string); ok {
		if dt, err := time.Parse(time.RFC3339, dtStr); err == nil {
			feature.DateTime = &dt
		}
	}

	// Parse validTime — accepts both array ["start","end"] and object {"start":...,"end":...} forms.
	if vt, ok := geoJSON.Properties["validTime"]; ok && vt != nil {
		tr := common_shared.ParseTimeRange(vt)
		if tr.Start != nil || tr.End != nil {
			feature.ValidTime = &tr
		}
	}

	return &feature, nil
}
