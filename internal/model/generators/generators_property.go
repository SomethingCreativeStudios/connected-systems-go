package generators

import (
	"fmt"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/google/uuid"
)

// FakeProperty returns a populated domains.Property
func FakeProperty() domains.Property {
	name := f.Lorem().Word()
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	uom := "unit"
	return domains.Property{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      f.Lorem().Sentence(2),
		},
		Definition:        f.Internet().URL(),
		PropertyType:      domains.PropertyTypeObservable,
		ObjectType:        new("Sensor"),
		BaseProperty:      new("baseProp"),
		Statistic:         new("statistic"),
		Qualifiers:        common_shared.ComponentWrappers{FakeComponentWrapper()},
		UnitOfMeasurement: &uom,
		Links:             FakeLinks(),
		Properties:        common_shared.Properties{},
	}
}
