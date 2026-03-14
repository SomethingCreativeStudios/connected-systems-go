package generators

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// =============================================================================
// SensorML Shared Generators - Reusable across System, Procedure, Deployment
// =============================================================================

// -----------------------------------------------------------------------------
// Term / Identifier / Classifier Generators
// -----------------------------------------------------------------------------

// FakeTerm creates a single Term with realistic values
func FakeTerm() common_shared.Term {
	return common_shared.Term{
		Definition: fmt.Sprintf("http://vocab.example.org/terms/%s", f.Lorem().Word()),
		Label:      f.Lorem().Word(),
		CodeSpace:  fmt.Sprintf("http://vocab.example.org/codespace/%s", f.Lorem().Word()),
		Value:      f.Lorem().Word(),
	}
}

// FakeIdentifierTerm creates a Term suitable for use as an identifier
func FakeIdentifierTerm() common_shared.Term {
	identifierTypes := []string{"shortName", "longName", "modelNumber", "serialNumber", "missionId", "manufacturer"}
	idType := identifierTypes[rand.Intn(len(identifierTypes))]

	return common_shared.Term{
		Definition: fmt.Sprintf("http://sensorml.com/ont/swe/identifier/%s", idType),
		Label:      idType,
		CodeSpace:  "http://sensorml.com/ont/swe/codespace/identifiers",
		Value:      fmt.Sprintf("%s-%s", f.Lorem().Word(), f.RandomStringWithLength(8)),
	}
}

// FakeClassifierTerm creates a Term suitable for use as a classifier
func FakeClassifierTerm() common_shared.Term {
	classifierTypes := []string{"sensorType", "applicationDomain", "intendedApplication", "processType"}
	clsType := classifierTypes[rand.Intn(len(classifierTypes))]

	return common_shared.Term{
		Definition: fmt.Sprintf("http://sensorml.com/ont/swe/classifier/%s", clsType),
		Label:      clsType,
		CodeSpace:  "http://sensorml.com/ont/swe/codespace/classifiers",
		Value:      f.Lorem().Word(),
	}
}

// FakeIdentifiers creates multiple identifier Terms
func FakeIdentifiers(count int) common_shared.Terms {
	terms := make(common_shared.Terms, count)
	for i := 0; i < count; i++ {
		terms[i] = FakeIdentifierTerm()
	}
	return terms
}

// FakeClassifiers creates multiple classifier Terms
func FakeClassifiers(count int) common_shared.Terms {
	terms := make(common_shared.Terms, count)
	for i := 0; i < count; i++ {
		terms[i] = FakeClassifierTerm()
	}
	return terms
}

// -----------------------------------------------------------------------------
// Contact Generators (deeply nested)
// -----------------------------------------------------------------------------

// FakePhone creates a Phone with realistic data
func FakePhone() *common_shared.Phone {
	return &common_shared.Phone{
		Voice:     f.Phone().Number(),
		Facsimile: f.Phone().Number(),
	}
}

// FakeAddress creates a fully populated Address
func FakeAddress() *common_shared.Address {
	return &common_shared.Address{
		DeliveryPoint:         f.Address().StreetAddress(),
		City:                  f.Address().City(),
		AdministrativeArea:    f.Address().State(),
		PostalCode:            f.Address().PostCode(),
		Country:               f.Address().Country(),
		ElectronicMailAddress: f.Internet().Email(),
	}
}

// FakeContactInfoFull creates a fully populated ContactInfo
func FakeContactInfoFull() *common_shared.ContactInfo {
	return &common_shared.ContactInfo{
		Phone:               FakePhone(),
		Address:             FakeAddress(),
		Website:             f.Internet().URL(),
		HoursOfService:      "Mon-Fri 9:00-17:00",
		ContactInstructions: "Please email for fastest response",
	}
}

// FakeContactPersonOrgFull creates a fully populated ContactPersonOrg
// Note: Schema oneOf requires EITHER individualName OR organisationName, not both
func FakeContactPersonOrgFull() *common_shared.ContactPersonOrg {
	roles := []string{
		"http://sensorml.com/ont/swe/role/Manufacturer",
		"http://sensorml.com/ont/swe/role/Operator",
		"http://sensorml.com/ont/swe/role/Owner",
		"http://sensorml.com/ont/swe/role/DataProvider",
		"http://sensorml.com/ont/swe/role/Maintainer",
	}

	contact := &common_shared.ContactPersonOrg{
		ContactInfo:  FakeContactInfoFull(),
		PositionName: f.Company().JobTitle(),
		Role:         roles[rand.Intn(len(roles))],
	}

	// Schema requires oneOf: either individualName OR organisationName
	if rand.Intn(2) == 0 {
		contact.IndividualName = f.Person().Name()
	} else {
		contact.OrganisationName = f.Company().Name()
	}

	return contact
}

// FakeContactLinkFull creates a ContactLink
func FakeContactLinkFull() *common_shared.ContactLink {
	return &common_shared.ContactLink{
		Role: "http://sensorml.com/ont/swe/role/Contact",
		Name: f.Person().Name(),
		Link: common_shared.Link{
			Href:  f.Internet().URL(),
			Title: "Contact Information",
			Rel:   "related",
			Type:  "text/html",
		},
	}
}

// FakeContactWrapperPerson creates a ContactWrapper with person/org variant
func FakeContactWrapperPerson() common_shared.ContactWrapper {
	p := FakeContactPersonOrgFull()
	raw, _ := json.Marshal(p)
	return common_shared.ContactWrapper{Person: p, Raw: raw}
}

// FakeContactWrapperLink creates a ContactWrapper with link variant
func FakeContactWrapperLink() common_shared.ContactWrapper {
	l := FakeContactLinkFull()
	raw, _ := json.Marshal(l)
	return common_shared.ContactWrapper{LinkRef: l, Raw: raw}
}

// FakeContactWrappers creates a mixed list of contact wrappers
func FakeContactWrappers(count int) common_shared.ContactWrappers {
	wrappers := make(common_shared.ContactWrappers, count)
	for i := 0; i < count; i++ {
		if rand.Intn(2) == 0 {
			wrappers[i] = FakeContactWrapperPerson()
		} else {
			wrappers[i] = FakeContactWrapperLink()
		}
	}
	return wrappers
}

// -----------------------------------------------------------------------------
// Document Generators
// -----------------------------------------------------------------------------

// FakeDocumentFull creates a fully populated Document
func FakeDocumentFull() common_shared.Document {
	roles := []string{
		"http://sensorml.com/ont/swe/role/Datasheet",
		"http://sensorml.com/ont/swe/role/Manual",
		"http://sensorml.com/ont/swe/role/Specification",
		"http://sensorml.com/ont/swe/role/Photo",
		"http://sensorml.com/ont/swe/role/Video",
	}

	return common_shared.Document{
		Role:        roles[rand.Intn(len(roles))],
		Name:        fmt.Sprintf("%s %s", f.Lorem().Word(), "Document"),
		Description: f.Lorem().Sentence(5),
		Link: common_shared.Link{
			Href:  f.Internet().URL(),
			Title: f.Lorem().Word(),
			Rel:   "describedby",
			Type:  "application/pdf",
		},
	}
}

// FakeDocumentsFull creates multiple Documents
func FakeDocumentsFull(count int) common_shared.Documents {
	docs := make(common_shared.Documents, count)
	for i := 0; i < count; i++ {
		docs[i] = FakeDocumentFull()
	}
	return docs
}

// -----------------------------------------------------------------------------
// Security and Legal Constraint Generators
// -----------------------------------------------------------------------------

// FakeSecurityConstraint creates a SecurityConstraint with extra properties
func FakeSecurityConstraint() common_shared.SecurityConstraint {
	return common_shared.SecurityConstraint{
		Type: "http://sensorml.com/ont/swe/security/Classification",
		Extra: map[string]interface{}{
			"level":        "unclassified",
			"releasableTo": []string{"public"},
		},
	}
}

// FakeSecurityConstraintsFull creates multiple SecurityConstraints
func FakeSecurityConstraintsFull(count int) common_shared.SecurityConstraints {
	constraints := make(common_shared.SecurityConstraints, count)
	for i := 0; i < count; i++ {
		constraints[i] = FakeSecurityConstraint()
	}
	return constraints
}

// FakeCodeList creates a CodeList entry
func FakeCodeList() common_shared.CodeList {
	return common_shared.CodeList{
		CodeSpace: "http://vocab.example.org/constraints",
		Value:     f.Lorem().Word(),
	}
}

// FakeCodeListsFull creates multiple CodeList entries
func FakeCodeListsFull(count int) common_shared.CodeLists {
	lists := make(common_shared.CodeLists, count)
	for i := 0; i < count; i++ {
		lists[i] = FakeCodeList()
	}
	return lists
}

// FakeLegalConstraint creates a fully populated LegalConstraint
// Note: otherConstraints omitted - schema expects []string but model uses Terms
func FakeLegalConstraint() common_shared.LegalConstraint {
	limitation := "For authorized use only"
	return common_shared.LegalConstraint{
		AccessConstraints: FakeCodeListsFull(1),
		UseConstraints:    FakeCodeListsFull(1),
		// OtherConstraints omitted - schema expects []string, model has Terms
		UserLimitations: &limitation,
	}
}

// FakeLegalConstraintsFull creates multiple LegalConstraints
func FakeLegalConstraintsFull(count int) common_shared.LegalConstraints {
	constraints := make(common_shared.LegalConstraints, count)
	for i := 0; i < count; i++ {
		constraints[i] = FakeLegalConstraint()
	}
	return constraints
}

// -----------------------------------------------------------------------------
// Component / Characteristic / Capability Generators (SWE Common)
// -----------------------------------------------------------------------------

// FakeComponentWrapperBoolean creates a Boolean component
func FakeComponentWrapperBoolean() common_shared.ComponentWrapper {
	boolVal := rand.Intn(2) == 0
	valJSON, _ := json.Marshal(boolVal)

	cw := common_shared.ComponentWrapper{
		Type:       "Boolean",
		Definition: "http://sensorml.com/ont/swe/property/Boolean",
		Label:      f.Lorem().Word(),
		Value:      valJSON,
	}

	raw, _ := json.Marshal(map[string]interface{}{
		"type":       cw.Type,
		"definition": cw.Definition,
		"label":      cw.Label,
		"value":      boolVal,
	})
	cw.Raw = raw

	return cw
}

// FakeComponentWrapperCount creates a Count component
func FakeComponentWrapperCount() common_shared.ComponentWrapper {
	countVal := rand.Intn(1000)
	valJSON, _ := json.Marshal(countVal)

	cw := common_shared.ComponentWrapper{
		Type:       "Count",
		Definition: "http://sensorml.com/ont/swe/property/Count",
		Label:      f.Lorem().Word(),
		Value:      valJSON,
	}

	raw, _ := json.Marshal(map[string]interface{}{
		"type":       cw.Type,
		"definition": cw.Definition,
		"label":      cw.Label,
		"value":      countVal,
	})
	cw.Raw = raw

	return cw
}

// FakeComponentWrapperQuantity creates a Quantity component with UOM
func FakeComponentWrapperQuantity() common_shared.ComponentWrapper {
	quantityVal := rand.Float64() * 100
	valJSON, _ := json.Marshal(quantityVal)

	uoms := []string{"m", "cm", "mm", "km", "kg", "g", "s", "Hz", "Pa", "K", "°C", "m/s", "m^2"}
	uom := uoms[rand.Intn(len(uoms))]
	uomJSON, _ := json.Marshal(map[string]string{"code": uom})

	cw := common_shared.ComponentWrapper{
		Type:       "Quantity",
		Definition: "http://sensorml.com/ont/swe/property/Measurement",
		Label:      f.Lorem().Word(),
		Value:      valJSON,
		UOM:        uomJSON,
	}

	raw, _ := json.Marshal(map[string]interface{}{
		"type":       cw.Type,
		"definition": cw.Definition,
		"label":      cw.Label,
		"value":      quantityVal,
		"uom":        map[string]string{"code": uom},
	})
	cw.Raw = raw

	return cw
}

// FakeComponentWrapperText creates a Text component
func FakeComponentWrapperText() common_shared.ComponentWrapper {
	textVal := f.Lorem().Sentence(3)
	valJSON, _ := json.Marshal(textVal)

	cw := common_shared.ComponentWrapper{
		Type:       "Text",
		Definition: "http://sensorml.com/ont/swe/property/Description",
		Label:      f.Lorem().Word(),
		Value:      valJSON,
	}

	raw, _ := json.Marshal(map[string]interface{}{
		"type":       cw.Type,
		"definition": cw.Definition,
		"label":      cw.Label,
		"value":      textVal,
	})
	cw.Raw = raw

	return cw
}

// FakeComponentWrapperCategory creates a Category component
func FakeComponentWrapperCategory() common_shared.ComponentWrapper {
	categories := []string{"high", "medium", "low", "excellent", "good", "fair", "poor"}
	catVal := categories[rand.Intn(len(categories))]
	valJSON, _ := json.Marshal(catVal)

	cw := common_shared.ComponentWrapper{
		Type:       "Category",
		Definition: "http://sensorml.com/ont/swe/property/Category",
		Label:      f.Lorem().Word(),
		Value:      valJSON,
	}

	raw, _ := json.Marshal(map[string]interface{}{
		"type":       cw.Type,
		"definition": cw.Definition,
		"label":      cw.Label,
		"value":      catVal,
	})
	cw.Raw = raw

	return cw
}

// FakeComponentWrapperQuantityRange creates a QuantityRange component
func FakeComponentWrapperQuantityRange() common_shared.ComponentWrapper {
	min := rand.Float64() * 50
	max := min + rand.Float64()*50
	valJSON, _ := json.Marshal([]float64{min, max})

	uomJSON, _ := json.Marshal(map[string]string{"code": "m"})

	cw := common_shared.ComponentWrapper{
		Type:       "QuantityRange",
		Definition: "http://sensorml.com/ont/swe/property/Range",
		Label:      f.Lorem().Word(),
		Value:      valJSON,
		UOM:        uomJSON,
	}

	raw, _ := json.Marshal(map[string]interface{}{
		"type":       cw.Type,
		"definition": cw.Definition,
		"label":      cw.Label,
		"value":      []float64{min, max},
		"uom":        map[string]string{"code": "m"},
	})
	cw.Raw = raw

	return cw
}

// FakeComponentWrapperTime creates a Time component
func FakeComponentWrapperTime() common_shared.ComponentWrapper {
	timeVal := time.Now().Format(time.RFC3339)
	valJSON, _ := json.Marshal(timeVal)

	uomJSON, _ := json.Marshal(map[string]string{"href": "http://www.opengis.net/def/uom/ISO-8601/0/Gregorian"})

	cw := common_shared.ComponentWrapper{
		Type:       "Time",
		Definition: "http://sensorml.com/ont/swe/property/SamplingTime",
		Label:      "samplingTime",
		Value:      valJSON,
		UOM:        uomJSON,
	}

	raw, _ := json.Marshal(map[string]interface{}{
		"type":       cw.Type,
		"definition": cw.Definition,
		"label":      cw.Label,
		"value":      timeVal,
		"uom":        map[string]string{"href": "http://www.opengis.net/def/uom/ISO-8601/0/Gregorian"},
	})
	cw.Raw = raw

	return cw
}

// FakeComponentWrapperRandom returns a random component type
func FakeComponentWrapperRandom() common_shared.ComponentWrapper {
	generators := []func() common_shared.ComponentWrapper{
		FakeComponentWrapperBoolean,
		FakeComponentWrapperCount,
		FakeComponentWrapperQuantity,
		FakeComponentWrapperText,
		FakeComponentWrapperCategory,
		FakeComponentWrapperQuantityRange,
		FakeComponentWrapperTime,
	}
	return generators[rand.Intn(len(generators))]()
}

// FakeComponentWrappers creates a list of random component wrappers
func FakeComponentWrappers(count int) []common_shared.ComponentWrapper {
	wrappers := make([]common_shared.ComponentWrapper, count)
	for i := 0; i < count; i++ {
		wrappers[i] = FakeComponentWrapperRandom()
	}
	return wrappers
}

// FakeCharacteristicGroupFull creates a fully populated CharacteristicGroup
func FakeCharacteristicGroupFull() common_shared.CharacteristicGroup {
	return common_shared.CharacteristicGroup{
		ID:              fmt.Sprintf("char-%s", uuid.New().String()[:8]),
		Label:           f.Lorem().Word() + " Characteristics",
		Description:     f.Lorem().Sentence(5),
		Definition:      "http://sensorml.com/ont/swe/property/PhysicalCharacteristics",
		Conditions:      FakeComponentWrappers(1),
		Characteristics: FakeComponentWrappers(rand.Intn(3) + 2),
	}
}

// FakeCharacteristicGroupsFull creates multiple CharacteristicGroups
func FakeCharacteristicGroupsFull(count int) common_shared.CharacteristicGroups {
	groups := make(common_shared.CharacteristicGroups, count)
	for i := 0; i < count; i++ {
		groups[i] = FakeCharacteristicGroupFull()
	}
	return groups
}

// FakeCapabilityGroupFull creates a fully populated CapabilityGroup
func FakeCapabilityGroupFull() common_shared.CapabilityGroup {
	return common_shared.CapabilityGroup{
		ID:           fmt.Sprintf("cap-%s", uuid.New().String()[:8]),
		Label:        f.Lorem().Word() + " Capabilities",
		Description:  f.Lorem().Sentence(5),
		Definition:   "http://sensorml.com/ont/swe/property/SystemCapabilities",
		Conditions:   FakeComponentWrappers(1),
		Capabilities: FakeComponentWrappers(rand.Intn(3) + 2),
	}
}

// FakeCapabilityGroupsFull creates multiple CapabilityGroups
func FakeCapabilityGroupsFull(count int) common_shared.CapabilityGroups {
	groups := make(common_shared.CapabilityGroups, count)
	for i := 0; i < count; i++ {
		groups[i] = FakeCapabilityGroupFull()
	}
	return groups
}

// -----------------------------------------------------------------------------
// History Event Generators
// -----------------------------------------------------------------------------

// FakeHistoryTimePeriod creates a HistoryTime with a time range
func FakeHistoryTimePeriod() common_shared.HistoryTime {
	tr := FakeTimeRange()
	return common_shared.HistoryTime{Range: tr}
}

// FakeHistoryTimeInstant creates a HistoryTime with an instant
func FakeHistoryTimeInstant() common_shared.HistoryTime {
	now := time.Now()
	return common_shared.HistoryTime{Instant: &now}
}

// FakeHistoryEventFull creates a fully populated HistoryEvent
func FakeHistoryEventFull() common_shared.HistoryEvent {
	eventTypes := []string{
		"http://sensorml.com/ont/swe/event/Installation",
		"http://sensorml.com/ont/swe/event/Calibration",
		"http://sensorml.com/ont/swe/event/Maintenance",
		"http://sensorml.com/ont/swe/event/Upgrade",
		"http://sensorml.com/ont/swe/event/Deployment",
	}

	props := FakeComponentWrappers(2)
	configJSON, _ := json.Marshal(map[string]interface{}{
		"setting1": f.Lorem().Word(),
		"setting2": rand.Float64() * 100,
	})

	return common_shared.HistoryEvent{
		ID:            fmt.Sprintf("event-%s", uuid.New().String()[:8]),
		Label:         f.Lorem().Word() + " Event",
		Description:   f.Lorem().Sentence(5),
		Definition:    eventTypes[rand.Intn(len(eventTypes))],
		Identifiers:   FakeIdentifiers(1),
		Classifiers:   FakeClassifiers(1),
		Contacts:      nil, // Omit to avoid oneOf validation complexity in nested history
		Documentation: FakeDocumentsFull(1),
		Time:          FakeHistoryTimeInstant(),
		Properties:    props,
		Configuration: configJSON,
	}
}

// FakeHistoryFull creates multiple HistoryEvents
func FakeHistoryFull(count int) common_shared.History {
	events := make(common_shared.History, count)
	for i := 0; i < count; i++ {
		events[i] = FakeHistoryEventFull()
	}
	return events
}

// -----------------------------------------------------------------------------
// Link Generators
// -----------------------------------------------------------------------------

// FakeLinkFull creates a fully populated Link
func FakeLinkFull() common_shared.Link {
	rels := []string{"self", "alternate", "related", "describedby", "collection", "item"}
	types := []string{"application/json", "application/geo+json", "application/sml+json", "text/html"}
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return common_shared.Link{
		Href:  f.Internet().URL(),
		Rel:   rels[rand.Intn(len(rels))],
		Type:  types[rand.Intn(len(types))],
		Title: f.Lorem().Word(),
		UID:   &uid,
	}
}

// FakeLinksFull creates multiple Links
func FakeLinksFull(count int) common_shared.Links {
	links := make(common_shared.Links, count)
	for i := 0; i < count; i++ {
		links[i] = FakeLinkFull()
	}
	return links
}

// FakeSystemLink creates a Link to a system resource
func FakeSystemLink() common_shared.Link {
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())
	return common_shared.Link{
		Href:  fmt.Sprintf("http://example.org/api/systems/%s", uuid.New().String()),
		Rel:   "system",
		Type:  "application/sml+json",
		Title: f.Company().Name() + " System",
		UID:   &uid,
	}
}

// -----------------------------------------------------------------------------
// Time Range Generators
// -----------------------------------------------------------------------------

// FakeValidTimePast creates a TimeRange in the past
func FakeValidTimePast() *common_shared.TimeRange {
	now := time.Now()
	start := now.Add(-time.Hour * 24 * 365) // 1 year ago
	end := now.Add(-time.Hour * 24 * 30)    // 1 month ago
	return &common_shared.TimeRange{Start: &start, End: &end}
}

// FakeValidTimeCurrent creates a TimeRange from past to future
func FakeValidTimeCurrent() *common_shared.TimeRange {
	now := time.Now()
	start := now.Add(-time.Hour * 24 * 30) // 1 month ago
	end := now.Add(time.Hour * 24 * 365)   // 1 year from now
	return &common_shared.TimeRange{Start: &start, End: &end}
}

// FakeValidTimeFuture creates a TimeRange in the future
func FakeValidTimeFuture() *common_shared.TimeRange {
	now := time.Now()
	start := now.Add(time.Hour * 24 * 30) // 1 month from now
	end := now.Add(time.Hour * 24 * 365)  // 1 year from now
	return &common_shared.TimeRange{Start: &start, End: &end}
}

// FakeValidTimeOpen creates a TimeRange with open end (now)
func FakeValidTimeOpen() *common_shared.TimeRange {
	now := time.Now()
	start := now.Add(-time.Hour * 24 * 30) // 1 month ago
	return &common_shared.TimeRange{Start: &start, End: nil}
}

// -----------------------------------------------------------------------------
// IO List Generators (for inputs/outputs/parameters)
// -----------------------------------------------------------------------------

// FakeIOItem creates an IOItem (wrapped component for IO lists)
func FakeIOItem() common_shared.IOItem {
	cw := FakeComponentWrapperQuantity()
	raw, _ := json.Marshal(cw)
	return common_shared.IOItem{
		Component: &cw,
		Raw:       raw,
	}
}

// FakeIOListFull creates an IOList with multiple IOItems
func FakeIOListFull(count int) common_shared.IOList {
	list := make(common_shared.IOList, count)
	for i := 0; i < count; i++ {
		list[i] = FakeIOItem()
	}
	return list
}

// -----------------------------------------------------------------------------
// Spatial/Temporal Frame Generators (for Systems)
// -----------------------------------------------------------------------------

// FakeSpatialFrameFull creates a fully populated SpatialFrame
func FakeSpatialFrameFull() common_shared.SpatialFrame {
	return common_shared.SpatialFrame{
		ID:          fmt.Sprintf("frame-%s", uuid.New().String()[:8]),
		Label:       "Local Reference Frame",
		Description: "Local coordinate system relative to sensor housing",
		Origin:      "Center of sensor housing",
		Axes: []common_shared.Axis{
			{Name: "X", Description: "Positive along the direction of travel"},
			{Name: "Y", Description: "Positive to the right of travel direction"},
			{Name: "Z", Description: "Positive upward"},
		},
	}
}

// FakeTemporalFrameFull creates a fully populated TemporalFrame
func FakeTemporalFrameFull() common_shared.TemporalFrame {
	return common_shared.TemporalFrame{
		ID:          fmt.Sprintf("tframe-%s", uuid.New().String()[:8]),
		Label:       "Mission Time",
		Description: "Time since mission start",
		Origin:      time.Now().Add(-time.Hour * 24 * 30).Format(time.RFC3339),
	}
}

// -----------------------------------------------------------------------------
// Method Generator (for Procedures)
// -----------------------------------------------------------------------------

// FakeMethodFull creates a fully populated Method
func FakeMethodFull() common_shared.Method {
	return common_shared.Method{
		Algorithm:   fmt.Sprintf("urn:ogc:def:algorithm:%s", f.Lorem().Word()),
		Description: f.Lorem().Paragraph(2),
	}
}
