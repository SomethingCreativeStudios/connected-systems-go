package generators

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
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
		ObjectType:        ptrString("Sensor"),
		BaseProperty:      ptrString("baseProp"),
		Statistic:         ptrString("statistic"),
		Qualifiers:        common_shared.ComponentWrappers{FakeComponentWrapper()},
		UnitOfMeasurement: &uom,
		Links:             FakeLinks(),
		Properties:        common_shared.Properties{},
	}
}
