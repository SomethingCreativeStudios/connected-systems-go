package generators

import (
	"fmt"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/google/uuid"
)

// FakeSamplingFeature returns a populated SamplingFeature
func FakeSamplingFeature() domains.SamplingFeature {
	name := f.Lorem().Word()
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.SamplingFeature{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      f.Lorem().Sentence(2),
		},
		FeatureType: domains.SamplingFeatureTypeSample,
		ValidTime:   FakeTimeRange(),
		Geometry:    FakeGoGeomPoint(),
		Links:       FakeLinks(),
		Properties:  common_shared.Properties{},
	}
}
