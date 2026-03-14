package generators

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	geom "github.com/twpayne/go-geom"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

// =============================================================================
// Configuration Settings Generators (for deployedSystems)
// =============================================================================

// FakeSetValue creates a SetValue for configuration
func FakeSetValue() common_shared.SetValue {
	// Randomly choose between number and string value
	if rand.Intn(2) == 0 {
		return common_shared.SetValue{
			Ref:   fmt.Sprintf("parameters/%s", f.Lorem().Word()),
			Value: rand.Float64() * 100,
		}
	}
	return common_shared.SetValue{
		Ref:   fmt.Sprintf("parameters/%s", f.Lorem().Word()),
		Value: f.Lorem().Word(),
	}
}

// FakeSetArrayValue creates a SetArrayValue for configuration
func FakeSetArrayValue() common_shared.SetArrayValue {
	values := make([]interface{}, rand.Intn(5)+2)
	for i := range values {
		values[i] = rand.Float64() * 100
	}
	return common_shared.SetArrayValue{
		Ref:   fmt.Sprintf("parameters/%sArray", f.Lorem().Word()),
		Value: values,
	}
}

// FakeSetMode creates a SetMode for configuration
func FakeSetMode() common_shared.SetMode {
	modes := []string{"normal", "standby", "highPrecision", "lowPower", "calibration"}
	return common_shared.SetMode{
		Ref:   "modes/operatingMode",
		Value: modes[rand.Intn(len(modes))],
	}
}

// FakeAllowedTokens creates an AllowedTokens constraint
func FakeAllowedTokens() common_shared.AllowedTokens {
	return common_shared.AllowedTokens{
		Type:   "AllowedTokens",
		Values: []string{"value1", "value2", "value3"},
	}
}

// FakeAllowedValues creates an AllowedValues constraint
func FakeAllowedValues() common_shared.AllowedValues {
	return common_shared.AllowedValues{
		Type: "AllowedValues",
		Values: []common_shared.ValueItem{
			{Number: ptrFloat64(0)},
			{Number: ptrFloat64(100)},
		},
		Intervals: [][]common_shared.ValueItem{
			{{Number: ptrFloat64(0)}, {Number: ptrFloat64(50)}},
		},
	}
}

// FakeConstraint creates a configuration Constraint
func FakeConstraint() common_shared.Constraint {
	if rand.Intn(2) == 0 {
		tokens := FakeAllowedTokens()
		return common_shared.Constraint{
			Type:   "AllowedTokens",
			Ref:    fmt.Sprintf("parameters/%s", f.Lorem().Word()),
			Tokens: &tokens,
		}
	}
	values := FakeAllowedValues()
	return common_shared.Constraint{
		Type:   "AllowedValues",
		Ref:    fmt.Sprintf("parameters/%s", f.Lorem().Word()),
		Values: &values,
	}
}

// FakeSetStatus creates a SetStatus for configuration
// Per the schema, setStatus values must be one of: 'enabled', 'disabled'
func FakeSetStatus() common_shared.SetStatus {
	statuses := []string{"enabled", "disabled"}
	return common_shared.SetStatus{
		Ref:   fmt.Sprintf("outputs/%s", f.Lorem().Word()),
		Value: statuses[rand.Intn(len(statuses))],
	}
}

// FakeConfigurationSettingsMinimal creates minimal configuration settings
func FakeConfigurationSettingsMinimal() common_shared.ConfigurationSettings {
	return common_shared.ConfigurationSettings{
		SetValues: []common_shared.SetValue{FakeSetValue()},
	}
}

// FakeConfigurationSettingsFull creates fully populated configuration settings
// Note: SetConstraints omitted due to complex schema validation requirements
func FakeConfigurationSettingsFull() common_shared.ConfigurationSettings {
	return common_shared.ConfigurationSettings{
		SetValues:      []common_shared.SetValue{FakeSetValue(), FakeSetValue()},
		SetArrayValues: []common_shared.SetArrayValue{FakeSetArrayValue()},
		SetModes:       []common_shared.SetMode{FakeSetMode()},
		SetStatus:      []common_shared.SetStatus{FakeSetStatus()},
		// SetConstraints omitted - complex oneOf validation in schema
	}
}

// =============================================================================
// DeployedSystemItem Generators
// =============================================================================

// FakeDeployedSystemItemMinimal creates a minimal DeployedSystemItem (name + system link only)
func FakeDeployedSystemItemMinimal() domains.DeployedSystemItem {
	name := f.Lorem().Word()
	return domains.DeployedSystemItem{
		Name: name,
		System: common_shared.Link{
			Href:  fmt.Sprintf("http://example.org/api/systems/%s", uuid.New().String()),
			Title: name + " System",
		},
	}
}

// FakeDeployedSystemItemWithDescription creates a DeployedSystemItem with description
func FakeDeployedSystemItemWithDescription() domains.DeployedSystemItem {
	item := FakeDeployedSystemItemMinimal()
	item.Description = f.Lorem().Sentence(5)
	return item
}

// FakeDeployedSystemItemWithConfig creates a DeployedSystemItem with configuration
func FakeDeployedSystemItemWithConfig() domains.DeployedSystemItem {
	item := FakeDeployedSystemItemWithDescription()
	item.Configuration = FakeConfigurationSettingsMinimal()
	return item
}

// FakeDeployedSystemItemFull creates a fully populated DeployedSystemItem
func FakeDeployedSystemItemFull() domains.DeployedSystemItem {
	name := fmt.Sprintf("%s-%s", f.Lorem().Word(), f.RandomStringWithLength(4))
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())
	return domains.DeployedSystemItem{
		Name:        name,
		Description: f.Lorem().Sentence(8),
		System: common_shared.Link{
			Href:  fmt.Sprintf("http://example.org/api/systems/%s", uuid.New().String()),
			Rel:   "system",
			Type:  "application/sml+json",
			Title: f.Company().Name() + " Sensor",
			UID:   &uid,
		},
		Configuration: FakeConfigurationSettingsFull(),
	}
}

// FakeDeployedSystemItems creates multiple DeployedSystemItems
func FakeDeployedSystemItems(count int) domains.DeployedSystemItems {
	items := make(domains.DeployedSystemItems, count)
	for i := 0; i < count; i++ {
		// Mix of different complexity levels
		switch rand.Intn(3) {
		case 0:
			items[i] = FakeDeployedSystemItemMinimal()
		case 1:
			items[i] = FakeDeployedSystemItemWithConfig()
		default:
			items[i] = FakeDeployedSystemItemFull()
		}
	}
	return items
}

// =============================================================================
// Platform Generators (platform is also a DeployedSystemItem)
// =============================================================================

// FakePlatformMinimal creates a minimal platform reference
func FakePlatformMinimal() *domains.DeployedSystemItem {
	return &domains.DeployedSystemItem{
		Name: "platform",
		System: common_shared.Link{
			Href:  fmt.Sprintf("http://example.org/api/systems/%s", uuid.New().String()),
			Title: "Platform",
		},
	}
}

// FakePlatformFull creates a fully detailed platform
func FakePlatformFull() *domains.DeployedSystemItem {
	platformTypes := []string{"Vehicle", "Drone", "Buoy", "Station", "Satellite", "Ship", "Aircraft"}
	platformType := platformTypes[rand.Intn(len(platformTypes))]
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return &domains.DeployedSystemItem{
		Name:        "platform",
		Description: fmt.Sprintf("%s platform hosting the deployed systems", platformType),
		System: common_shared.Link{
			Href:  fmt.Sprintf("http://example.org/api/systems/%s", uuid.New().String()),
			Rel:   "platform",
			Type:  "application/sml+json",
			Title: fmt.Sprintf("%s %s", f.Company().Name(), platformType),
			UID:   &uid,
		},
		Configuration: FakeConfigurationSettingsFull(),
	}
}

// =============================================================================
// Geometry Generators for Deployments
// =============================================================================

// FakeDeploymentPointGeometry creates a point geometry for fixed deployments
func FakeDeploymentPointGeometry() *common_shared.GoGeom {
	lat := rand.Float64()*180 - 90
	lon := rand.Float64()*360 - 180
	p := geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{lon, lat}).SetSRID(4326)
	return &common_shared.GoGeom{T: p}
}

// FakeDeploymentPolygonGeometry creates a polygon geometry for area deployments
func FakeDeploymentPolygonGeometry() *common_shared.GoGeom {
	// Create a simple square polygon around a random point
	centerLat := rand.Float64()*160 - 80 // Avoid poles
	centerLon := rand.Float64()*360 - 180
	size := 0.01 // ~1km at equator

	coords := []geom.Coord{
		{centerLon - size, centerLat - size},
		{centerLon + size, centerLat - size},
		{centerLon + size, centerLat + size},
		{centerLon - size, centerLat + size},
		{centerLon - size, centerLat - size}, // Close the ring
	}

	p := geom.NewPolygon(geom.XY).MustSetCoords([][]geom.Coord{coords}).SetSRID(4326)
	return &common_shared.GoGeom{T: p}
}

// FakeDeploymentLineStringGeometry creates a linestring for mobile/track deployments
func FakeDeploymentLineStringGeometry() *common_shared.GoGeom {
	startLat := rand.Float64()*160 - 80
	startLon := rand.Float64()*360 - 180
	numPoints := rand.Intn(5) + 3

	coords := make([]geom.Coord, numPoints)
	for i := 0; i < numPoints; i++ {
		coords[i] = geom.Coord{
			startLon + float64(i)*0.01,
			startLat + float64(i)*0.005,
		}
	}

	ls := geom.NewLineString(geom.XY).MustSetCoords(coords).SetSRID(4326)
	return &common_shared.GoGeom{T: ls}
}

// =============================================================================
// Deployment Generators - Various Types and Complexity Levels
// =============================================================================

// FakeDeploymentMinimal creates a minimal valid Deployment (required fields only per schema)
// Schema requires: type, label, uniqueId, definition
func FakeDeploymentMinimal() domains.Deployment {
	name := f.Lorem().Word()
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.Deployment{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
		},
		DeploymentType: domains.DeploymentTypeDeployment,
		Links:          FakeLinksFull(1), // links required with minItems: 1
	}
}

// FakeDeploymentWithDescription adds description to minimal deployment
func FakeDeploymentWithDescription() domains.Deployment {
	dep := FakeDeploymentMinimal()
	dep.Description = f.Lorem().Sentence(5)
	return dep
}

// FakeDeploymentWithValidTime adds validTime to deployment
func FakeDeploymentWithValidTime() domains.Deployment {
	dep := FakeDeploymentWithDescription()
	dep.ValidTime = FakeValidTimeCurrent()
	return dep
}

// FakeDeploymentWithLocation creates a deployment with location (fixed deployment)
func FakeDeploymentWithLocation() domains.Deployment {
	dep := FakeDeploymentWithValidTime()
	dep.Geometry = FakeDeploymentPointGeometry()
	return dep
}

// FakeDeploymentFixed creates a fixed-location deployment with full metadata
func FakeDeploymentFixed() domains.Deployment {
	name := fmt.Sprintf("%s Fixed Deployment", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.Deployment{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      f.Lorem().Paragraph(1),
		},
		DeploymentType:      domains.DeploymentTypeDeployment,
		ValidTime:           FakeValidTimeCurrent(),
		Geometry:            FakeDeploymentPointGeometry(),
		Lang:                ptrString("en"),
		Keywords:            common_shared.StringArray{"fixed", "monitoring", f.Lorem().Word()},
		Identifiers:         FakeIdentifiers(2),
		Classifiers:         FakeClassifiers(2),
		Contacts:            FakeContactWrappers(1),
		SecurityConstraints: FakeSecurityConstraintsFull(1),
		LegalConstraints:    FakeLegalConstraintsFull(1),
		Characteristics:     FakeCharacteristicGroupsFull(1),
		Capabilities:        FakeCapabilityGroupsFull(1),
		Documentation:       FakeDocumentsFull(1),
		History:             FakeHistoryFull(1),
		DeployedSystems:     FakeDeployedSystemItems(2),
		Platform:            FakePlatformMinimal(),
		Links:               FakeLinksFull(2),
	}
}

// FakeDeploymentMobile creates a mobile deployment (track geometry)
func FakeDeploymentMobile() domains.Deployment {
	name := fmt.Sprintf("%s Mobile Deployment", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.Deployment{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "Mobile sensor deployment with trajectory",
		},
		DeploymentType:  domains.DeploymentTypeDeployment,
		ValidTime:       FakeValidTimeCurrent(),
		Geometry:        FakeDeploymentLineStringGeometry(),
		Lang:            ptrString("en"),
		Keywords:        common_shared.StringArray{"mobile", "trajectory", "survey"},
		Identifiers:     FakeIdentifiers(2),
		Classifiers:     FakeClassifiers(1),
		Contacts:        FakeContactWrappers(1),
		Characteristics: FakeCharacteristicGroupsFull(1),
		Capabilities:    FakeCapabilityGroupsFull(1),
		DeployedSystems: FakeDeployedSystemItems(1),
		Platform:        FakePlatformFull(),
		Links:           FakeLinksFull(1),
	}
}

// FakeDeploymentArea creates an area-based deployment (polygon geometry)
func FakeDeploymentArea() domains.Deployment {
	name := fmt.Sprintf("%s Area Deployment", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.Deployment{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "Area-based sensor network deployment",
		},
		DeploymentType:  domains.DeploymentTypeDeployment,
		ValidTime:       FakeValidTimeCurrent(),
		Geometry:        FakeDeploymentPolygonGeometry(),
		Lang:            ptrString("en"),
		Keywords:        common_shared.StringArray{"area", "network", "distributed"},
		Identifiers:     FakeIdentifiers(1),
		Classifiers:     FakeClassifiers(1),
		Contacts:        FakeContactWrappers(2),
		Characteristics: FakeCharacteristicGroupsFull(1),
		Capabilities:    FakeCapabilityGroupsFull(1),
		DeployedSystems: FakeDeployedSystemItems(5),
		Links:           FakeLinksFull(1),
	}
}

// FakeDeploymentWithPlatform creates a deployment with detailed platform info
func FakeDeploymentWithPlatform() domains.Deployment {
	dep := FakeDeploymentFixed()
	dep.Platform = FakePlatformFull()
	return dep
}

// FakeDeploymentWithMultipleSystems creates a deployment with many systems
func FakeDeploymentWithMultipleSystems() domains.Deployment {
	dep := FakeDeploymentFixed()
	dep.DeployedSystems = FakeDeployedSystemItems(rand.Intn(5) + 3)
	return dep
}

// FakeDeploymentWithHistory creates a deployment with historical events
func FakeDeploymentWithHistory() domains.Deployment {
	dep := FakeDeploymentFixed()
	dep.History = FakeHistoryFull(rand.Intn(3) + 2)
	return dep
}

// FakeDeploymentFull creates a fully populated Deployment with all optional fields
func FakeDeploymentFull() domains.Deployment {
	name := fmt.Sprintf("%s Complete Deployment", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.Deployment{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      f.Lorem().Paragraph(2),
		},
		DeploymentType:      domains.DeploymentTypeDeployment,
		ValidTime:           FakeValidTimeCurrent(),
		Geometry:            FakeDeploymentPointGeometry(),
		Lang:                ptrString("en"),
		Keywords:            common_shared.StringArray{f.Lorem().Word(), f.Lorem().Word(), f.Lorem().Word()},
		Identifiers:         FakeIdentifiers(3),
		Classifiers:         FakeClassifiers(2),
		SecurityConstraints: FakeSecurityConstraintsFull(1),
		LegalConstraints:    FakeLegalConstraintsFull(1),
		Characteristics:     FakeCharacteristicGroupsFull(2),
		Capabilities:        FakeCapabilityGroupsFull(2),
		Contacts:            FakeContactWrappers(3),
		Documentation:       FakeDocumentsFull(2),
		History:             FakeHistoryFull(3),
		DeployedSystems:     FakeDeployedSystemItems(4),
		Platform:            FakePlatformFull(),
		Links:               FakeLinksFull(3),
	}
}

// FakeDeployment returns a standard populated Deployment (backward compatible)
func FakeDeployment() domains.Deployment {
	return FakeDeploymentFixed()
}

func FakeDeploymentWithSystem(systemId string) domains.Deployment {
	dep := FakeDeploymentFixed()

	for i := range dep.DeployedSystems {
		dep.DeployedSystems[i].System = common_shared.Link{
			Href:  fmt.Sprintf("http://example.org/api/systems/%s", systemId),
			Title: dep.DeployedSystems[i].Name + " System",
		}
		// Only set the first one
		break
	}

	for i := range *dep.SystemIds {
		(*dep.SystemIds)[i] = systemId
		// Only set the first one
		break
	}

	return dep
}

// FakeDeploymentRandom returns a randomly chosen deployment type
func FakeDeploymentRandom() domains.Deployment {
	generators := []func() domains.Deployment{
		FakeDeploymentMinimal,
		FakeDeploymentFixed,
		FakeDeploymentMobile,
		FakeDeploymentArea,
		FakeDeploymentFull,
	}
	return generators[rand.Intn(len(generators))]()
}

// =============================================================================
// Helper Functions
// =============================================================================

func ptrFloat64(f float64) *float64 {
	return &f
}
