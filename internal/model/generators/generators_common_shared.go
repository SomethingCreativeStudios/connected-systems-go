package generators

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"time"

	faker "github.com/jaswdr/faker/v2"
	geom "github.com/twpayne/go-geom"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

var f = faker.New()

// FakeTerms creates a Terms slice with one random term
func FakeTerms() common_shared.Terms {
	return common_shared.Terms{
		{
			Definition: f.Internet().URL(),
			Label:      f.Person().FirstName(),
			CodeSpace:  f.Internet().URL(),
			Value:      f.Lorem().Word(),
		},
	}
}

// FakeLink creates a basic Link
func FakeLink() common_shared.Link {
	return common_shared.Link{
		Href:  f.Internet().URL(),
		Title: f.Lorem().Word(),
		Rel:   "related",
		Type:  "application/json",
	}
}

// FakeLinks returns a Links slice
func FakeLinks() common_shared.Links {
	return common_shared.Links{FakeLink()}
}

// FakeContactInfo creates a ContactInfo
func FakeContactInfo() *common_shared.ContactInfo {
	return &common_shared.ContactInfo{
		Phone: &common_shared.Phone{Voice: f.Phone().Number()},
		Address: &common_shared.Address{
			City:                  f.Address().City(),
			Country:               f.Address().Country(),
			ElectronicMailAddress: f.Internet().Email(),
		},
		Website:             f.Internet().URL(),
		HoursOfService:      "",
		ContactInstructions: "",
	}
}

// FakeContactPersonOrg creates a ContactPersonOrg
func FakeContactPersonOrg() *common_shared.ContactPersonOrg {
	return &common_shared.ContactPersonOrg{
		IndividualName:   f.Person().Name(),
		OrganisationName: f.Company().Name(),
		PositionName:     f.Person().Title(),
		ContactInfo:      FakeContactInfo(),
		Role:             f.Internet().URL(),
	}
}

// FakeContactLink creates a ContactLink
func FakeContactLink() *common_shared.ContactLink {
	return &common_shared.ContactLink{
		Role: f.Internet().URL(),
		Name: f.Person().Name(),
		Link: FakeLink(),
	}
}

// FakeContactWrapper returns a ContactWrapper with person or link variant
func FakeContactWrapper() common_shared.ContactWrapper {
	if rand.Intn(2) == 0 {
		p := FakeContactPersonOrg()
		raw, _ := json.Marshal(p)
		return common_shared.ContactWrapper{Person: p, Raw: raw}
	}
	l := FakeContactLink()
	raw, _ := json.Marshal(l)
	return common_shared.ContactWrapper{LinkRef: l, Raw: raw}
}

// FakeComponentWrapper creates a random ComponentWrapper and one of the concrete component variants
func FakeComponentWrapper() common_shared.ComponentWrapper {
	// choose a type
	types := []string{"Boolean", "Count", "Quantity", "Text"}
	t := types[rand.Intn(len(types))]

	up := rand.Intn(2) == 0
	opt := rand.Intn(2) == 0
	cw := common_shared.ComponentWrapper{
		Type:           t,
		Definition:     f.Internet().URL(),
		Label:          f.Lorem().Word(),
		ReferenceFrame: f.Lorem().Word(),
		AxisID:         f.Lorem().Word(),
		LocalFrame:     f.Lorem().Word(),
		Updatable:      &up,
		Optional:       &opt,
	}

	switch t {
	case "Boolean":
		b, _ := json.Marshal(true)
		cw.Value = b
	case "Count":
		v, _ := json.Marshal(42)
		cw.Value = v
	case "Quantity":
		q, _ := json.Marshal(3.14)
		cw.Value = q
	case "Text":
		s, _ := json.Marshal(f.Lorem().Sentence(3))
		cw.Value = s
	}

	// set UOM/Constraint/NilValues for richer payloads
	uom, _ := json.Marshal(map[string]string{"uom": "m"})
	constr, _ := json.Marshal(map[string]interface{}{"type": "AllowedValues", "values": []int{1, 2, 3}})
	nilv, _ := json.Marshal([]string{})
	cw.UOM = uom
	cw.Constraint = constr
	cw.NilValues = nilv

	// Raw: marshal a representative payload
	rep := map[string]interface{}{
		"type":       cw.Type,
		"definition": cw.Definition,
		"label":      cw.Label,
		"value":      json.RawMessage(cw.Value),
	}
	if rawRep, err := json.Marshal(rep); err == nil {
		cw.Raw = rawRep
	}

	return cw
}

// FakeCharacteristicGroup returns a simple CharacteristicGroup
func FakeCharacteristicGroup() common_shared.CharacteristicGroup {
	return common_shared.CharacteristicGroup{
		ID:              f.Lorem().Word(),
		Label:           f.Lorem().Word(),
		Description:     f.Lorem().Sentence(3),
		Definition:      f.Internet().URL(),
		Conditions:      []common_shared.ComponentWrapper{FakeComponentWrapper()},
		Characteristics: []common_shared.ComponentWrapper{FakeComponentWrapper()},
	}
}

// FakeCapabilityGroup returns a simple CapabilityGroup
func FakeCapabilityGroup() common_shared.CapabilityGroup {
	return common_shared.CapabilityGroup{
		ID:           f.Lorem().Word(),
		Label:        f.Lorem().Word(),
		Description:  f.Lorem().Sentence(3),
		Definition:   f.Internet().URL(),
		Capabilities: []common_shared.ComponentWrapper{FakeComponentWrapper()},
	}
}

// FakeTimeRange returns a TimeRange with start and end
func FakeTimeRange() *common_shared.TimeRange {
	now := time.Now()
	start := now.Add(-time.Hour * 24)
	end := now.Add(time.Hour * 24)
	return &common_shared.TimeRange{Start: &start, End: &end}
}

// FakeGoGeomPoint returns a simple point geometry wrapper
func FakeGoGeomPoint() *common_shared.GoGeom {
	lat := rand.Float64()*180 - 90
	lon := rand.Float64()*360 - 180
	p := geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{lon, lat}).SetSRID(4326)
	return &common_shared.GoGeom{T: p}
}

// FakeDeployedSystemItem returns a domains.DeployedSystemItem
func FakeDeployedSystemItem() domains.DeployedSystemItem {
	name := f.Lorem().Word()
	return domains.DeployedSystemItem{
		Name:        name,
		Description: f.Lorem().Sentence(4),
		System: common_shared.Link{
			Href:  "http://example.com/systems/" + strconv.Itoa(f.RandomNumber(12)),
			Title: name,
			Rel:   "system",
		},
		Configuration: FakeConfigurationSettings(),
	}
}

// FakeConfigurationSettings returns a simple ConfigurationSettings
func FakeConfigurationSettings() common_shared.ConfigurationSettings {
	v := 3.14
	sv := common_shared.SetValue{Ref: "components/comp1/inputs/samp", Value: v}
	sm := common_shared.SetMode{Ref: "modes/1", Value: "default"}
	return common_shared.ConfigurationSettings{
		SetValues: []common_shared.SetValue{sv},
		SetModes:  []common_shared.SetMode{sm},
		SetStatus: []common_shared.SetStatus{{Ref: "components/comp1", Value: "enabled"}},
	}
}
