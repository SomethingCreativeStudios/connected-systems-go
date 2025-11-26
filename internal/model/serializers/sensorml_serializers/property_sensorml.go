package sensorml_serializers

import (
	"context"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

type PropertySensorMLSerializer struct {
	serializers.Serializer[domains.PropertySensorMLFeature, *domains.Property]
	repos *repository.Repositories
}

// NewCPropertySensorMLSerializer constructs a serializer with required repository readers.
func NewPropertySensorMLSerializer(repos *repository.Repositories) *PropertySensorMLSerializer {
	return &PropertySensorMLSerializer{repos: repos}
}

func (s *PropertySensorMLSerializer) Serialize(ctx context.Context, property *domains.Property) (domains.PropertySensorMLFeature, error) {
	features, err := s.SerializeAll(ctx, []*domains.Property{property})
	if err != nil {
		return domains.PropertySensorMLFeature{}, err
	}
	return features[0], nil
}

func (s *PropertySensorMLSerializer) SerializeAll(ctx context.Context, properties []*domains.Property) ([]domains.PropertySensorMLFeature, error) {
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
