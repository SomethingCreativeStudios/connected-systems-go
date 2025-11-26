package sensorml_formatters

import (
	"context"
	"encoding/json"
	"io"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// PropertySensorMLFormatter handles serialization and deserialization of Property objects in SensorML format
type PropertySensorMLFormatter struct {
	serializers.Formatter[domains.PropertySensorMLFeature, *domains.Property]
	repos *repository.Repositories
}

// NewPropertySensorMLFormatter constructs a formatter with required repository readers
func NewPropertySensorMLFormatter(repos *repository.Repositories) *PropertySensorMLFormatter {
	return &PropertySensorMLFormatter{repos: repos}
}

func (f *PropertySensorMLFormatter) ContentType() string {
	return SensorMLContentType
}

// --- Serialization ---

func (f *PropertySensorMLFormatter) Serialize(ctx context.Context, property *domains.Property) (domains.PropertySensorMLFeature, error) {
	features, err := f.SerializeAll(ctx, []*domains.Property{property})
	if err != nil {
		return domains.PropertySensorMLFeature{}, err
	}
	return features[0], nil
}

func (f *PropertySensorMLFormatter) SerializeAll(ctx context.Context, properties []*domains.Property) ([]domains.PropertySensorMLFeature, error) {
	if len(properties) == 0 {
		return []domains.PropertySensorMLFeature{}, nil
	}

	var features []domains.PropertySensorMLFeature
	for _, property := range properties {
		feature := domains.PropertySensorMLFeature{
			ID:           property.ID,
			Label:        property.Name,
			Description:  property.Description,
			UniqueID:     string(property.UniqueIdentifier),
			BaseProperty: property.BaseProperty,
			ObjectType:   property.ObjectType,
			Statistic:    property.Statistic,
			Qualifiers:   property.Qualifiers,
			Links:        property.Links,
		}
		features = append(features, feature)
	}
	return features, nil
}

// --- Deserialization ---

func (f *PropertySensorMLFormatter) Deserialize(ctx context.Context, reader io.Reader) (*domains.Property, error) {
	var sensorML domains.PropertySensorMLFeature

	if err := json.NewDecoder(reader).Decode(&sensorML); err != nil {
		return nil, err
	}

	property := &domains.Property{
		Links: sensorML.Links,
	}

	property.UniqueIdentifier = domains.UniqueID(sensorML.UniqueID)
	property.Name = sensorML.Label
	property.Description = sensorML.Description
	property.ObjectType = sensorML.ObjectType
	property.BaseProperty = sensorML.BaseProperty
	property.Statistic = sensorML.Statistic
	property.Qualifiers = sensorML.Qualifiers

	return property, nil
}
