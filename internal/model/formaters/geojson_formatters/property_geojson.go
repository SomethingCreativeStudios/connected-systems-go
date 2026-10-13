package geojson_formatters

import (
	"context"
	"fmt"
	"io"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

// PropertyGeoJSONFormatter handles serialization and deserialization of Property objects in GeoJSON format
type PropertyGeoJSONFormatter struct {
	formaters.Formatter[domains.PropertyGeoJSONFeature, *domains.Property]
	links *formaters.LinkResolver
}

// NewPropertyGeoJSONFormatter constructs a formatter with required repository readers
func NewPropertyGeoJSONFormatter(links *formaters.LinkResolver) *PropertyGeoJSONFormatter {
	return &PropertyGeoJSONFormatter{links: links}
}

func (f *PropertyGeoJSONFormatter) ContentType() string {
	return mediatype.GeoJSON
}

// --- Serialization ---

func (f *PropertyGeoJSONFormatter) Serialize(ctx context.Context, property *domains.Property) (domains.PropertyGeoJSONFeature, error) {
	return formaters.SerializeOne(ctx, property, f.SerializeAll)
}

func (f *PropertyGeoJSONFormatter) SerializeAll(ctx context.Context, properties []*domains.Property) ([]domains.PropertyGeoJSONFeature, error) {
	if len(properties) == 0 {
		return []domains.PropertyGeoJSONFeature{}, nil
	}

	var features []domains.PropertyGeoJSONFeature
	for _, property := range properties {
		feature := domains.PropertyGeoJSONFeature{
			Type:     "Feature",
			ID:       property.ID,
			Geometry: nil, // Properties don't have spatial geometry
			Properties: domains.PropertyGeoJSONProperties{
				UID:               property.UniqueIdentifier,
				Name:              property.Name,
				Description:       property.Description,
				Definition:        property.Definition,
				PropertyType:      property.PropertyType,
				BaseProperty:      property.BaseProperty,
				ObjectType:        property.ObjectType,
				Statistic:         property.Statistic,
				Qualifiers:        property.Qualifiers,
				UnitOfMeasurement: property.UnitOfMeasurement,
			},
			Links: property.Links,
		}
		features = append(features, feature)
	}

	return features, nil
}

// --- Deserialization ---

func (f *PropertyGeoJSONFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Property, error) {
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	geoJSON, err := common_shared.DecodeWithFieldErrors[struct {
		Type       string                            `json:"type"`
		ID         string                            `json:"id,omitempty"`
		Properties domains.PropertyGeoJSONProperties `json:"properties"`
		Geometry   any                               `json:"geometry,omitempty"`
		Links      common_shared.Links               `json:"links,omitempty"`
	}](body)
	if err != nil {
		return nil, err
	}

	// Required root fields per OGC CS Part 1 property (DerivedProperty) schema
	if geoJSON.Properties.UID == "" {
		return nil, fmt.Errorf("properties.uid is required")
	}
	if geoJSON.Properties.Name == "" {
		return nil, fmt.Errorf("properties.name is required")
	}
	if geoJSON.Properties.BaseProperty == nil || *geoJSON.Properties.BaseProperty == "" {
		return nil, fmt.Errorf("properties.baseProperty is required")
	}

	property := &domains.Property{
		Links: common_shared.StripAssociationLinks(geoJSON.Links),
	}

	// Extract properties
	property.UniqueIdentifier = domains.UniqueID(geoJSON.Properties.UID)
	property.Name = geoJSON.Properties.Name
	property.Description = geoJSON.Properties.Description
	property.Definition = geoJSON.Properties.Definition
	property.PropertyType = geoJSON.Properties.PropertyType
	property.BaseProperty = geoJSON.Properties.BaseProperty
	property.ObjectType = geoJSON.Properties.ObjectType
	property.Statistic = geoJSON.Properties.Statistic
	property.Qualifiers = geoJSON.Properties.Qualifiers
	property.UnitOfMeasurement = geoJSON.Properties.UnitOfMeasurement

	return property, nil
}
