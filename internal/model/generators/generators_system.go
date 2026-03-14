package generators

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

// =============================================================================
// System Generators - Various Types and Complexity Levels
// =============================================================================

// FakeSystemMinimal creates a minimal valid System (required fields only)
// Schema requires: type, label, uniqueId, definition
func FakeSystemMinimal() domains.System {
	name := f.Company().Name()
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.System{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
		},
		SystemType: domains.SystemTypeSensor,
		Links:      FakeLinksFull(1),
	}
}

// FakeSystemSensor creates a Sensor system
func FakeSystemSensor() domains.System {
	name := fmt.Sprintf("%s Temperature Sensor", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.System{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "High-precision temperature measurement sensor",
		},
		SystemType:    domains.SystemTypeSensor,
		AssetType:     ptrString(domains.AssetTypeEquipment),
		ValidTime:     FakeValidTimeCurrent(),
		Geometry:      FakeDeploymentPointGeometry(),
		Lang:          ptrString("en"),
		Keywords:      []string{"sensor", "temperature", "measurement"},
		Identifiers:   FakeIdentifiers(2),
		Classifiers:   FakeClassifiers(2),
		Contacts:      FakeContactWrappers(1),
		Documentation: FakeDocumentsFull(1),
		History:       FakeHistoryFull(1),
		TypeOf:        &common_shared.Link{Href: f.Internet().URL(), Title: "Sensor Datasheet", Rel: "typeOf"},
		Inputs:        FakeIOListFull(1),
		Outputs:       FakeIOListFull(2),
		Parameters:    FakeIOListFull(1),
		Links:         FakeLinksFull(2),
	}
}

// FakeSystemActuator creates an Actuator system
func FakeSystemActuator() domains.System {
	name := fmt.Sprintf("%s Valve Actuator", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.System{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "Precision valve control actuator",
		},
		SystemType:    domains.SystemTypeActuator,
		AssetType:     ptrString(domains.AssetTypeEquipment),
		ValidTime:     FakeValidTimeCurrent(),
		Geometry:      FakeDeploymentPointGeometry(),
		Lang:          ptrString("en"),
		Keywords:      []string{"actuator", "valve", "control"},
		Identifiers:   FakeIdentifiers(2),
		Classifiers:   FakeClassifiers(1),
		Contacts:      FakeContactWrappers(1),
		Documentation: FakeDocumentsFull(1),
		TypeOf:        &common_shared.Link{Href: f.Internet().URL(), Title: "Actuator Datasheet", Rel: "typeOf"},
		Inputs:        FakeIOListFull(2),
		Outputs:       FakeIOListFull(1),
		Links:         FakeLinksFull(1),
	}
}

// FakeSystemSampler creates a Sampler system
func FakeSystemSampler() domains.System {
	name := fmt.Sprintf("%s Water Sampler", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.System{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "Automated water sampling system",
		},
		SystemType:    domains.SystemTypeSampler,
		AssetType:     ptrString(domains.AssetTypeEquipment),
		ValidTime:     FakeValidTimeCurrent(),
		Geometry:      FakeDeploymentPointGeometry(),
		Lang:          ptrString("en"),
		Keywords:      []string{"sampler", "water", "collection"},
		Identifiers:   FakeIdentifiers(1),
		Classifiers:   FakeClassifiers(1),
		Contacts:      FakeContactWrappers(1),
		Documentation: FakeDocumentsFull(1),
		Outputs:       FakeIOListFull(1),
		Links:         FakeLinksFull(1),
	}
}

// FakeSystemPlatform creates a Platform system
func FakeSystemPlatform() domains.System {
	platformTypes := []string{"Buoy", "Drone", "Vehicle", "Station", "Satellite"}
	platformType := platformTypes[rand.Intn(len(platformTypes))]
	name := fmt.Sprintf("%s %s Platform", f.Company().Name(), platformType)
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.System{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      fmt.Sprintf("Multi-sensor %s platform", platformType),
		},
		SystemType:           domains.SystemTypePlatform,
		AssetType:            ptrString(domains.AssetTypePlatform),
		ValidTime:            FakeValidTimeCurrent(),
		Geometry:             FakeDeploymentPointGeometry(),
		Lang:                 ptrString("en"),
		Keywords:             []string{"platform", platformType, "hosting"},
		Identifiers:          FakeIdentifiers(2),
		Classifiers:          FakeClassifiers(2),
		Contacts:             FakeContactWrappers(2),
		Documentation:        FakeDocumentsFull(2),
		History:              FakeHistoryFull(2),
		LocalReferenceFrames: []common_shared.SpatialFrame{FakeSpatialFrameFull()},
		LocalTimeFrames:      []common_shared.TemporalFrame{FakeTemporalFrameFull()},
		Links:                FakeLinksFull(2),
	}
}

// FakeSystemPhysicalSystem creates a PhysicalSystem (composite system)
func FakeSystemPhysicalSystem() domains.System {
	name := fmt.Sprintf("%s Monitoring Station", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	configJSON, _ := json.Marshal(FakeConfigurationSettingsFull())

	return domains.System{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "Integrated environmental monitoring station",
		},
		SystemType:           domains.SystemTypeSystem,
		AssetType:            ptrString(domains.AssetTypeEquipment),
		ValidTime:            FakeValidTimeCurrent(),
		Geometry:             FakeDeploymentPointGeometry(),
		Lang:                 ptrString("en"),
		Keywords:             []string{"station", "monitoring", "integrated"},
		Identifiers:          FakeIdentifiers(3),
		Classifiers:          FakeClassifiers(2),
		SecurityConstraints:  FakeSecurityConstraintsFull(1),
		LegalConstraints:     FakeLegalConstraintsFull(1),
		Contacts:             FakeContactWrappers(2),
		Documentation:        FakeDocumentsFull(2),
		History:              FakeHistoryFull(2),
		TypeOf:               &common_shared.Link{Href: f.Internet().URL(), Title: "System Type", Rel: "typeOf"},
		Configuration:        configJSON,
		FeaturesOfInterest:   FakeLinksFull(1),
		Inputs:               FakeIOListFull(2),
		Outputs:              FakeIOListFull(3),
		Parameters:           FakeIOListFull(2),
		AttachedTo:           &common_shared.Link{Href: f.Internet().URL(), Title: "Parent Platform", Rel: "attachedTo"},
		LocalReferenceFrames: []common_shared.SpatialFrame{FakeSpatialFrameFull()},
		LocalTimeFrames:      []common_shared.TemporalFrame{FakeTemporalFrameFull()},
		Links:                FakeLinksFull(3),
	}
}

// FakeSystemWithPosition creates a system with position information
func FakeSystemWithPosition() domains.System {
	sys := FakeSystemSensor()

	positionJSON, _ := json.Marshal(map[string]interface{}{
		"type": "Point",
		"coordinates": []float64{
			rand.Float64()*360 - 180,
			rand.Float64()*180 - 90,
			rand.Float64() * 100,
		},
	})
	sys.Position = positionJSON
	sys.LocalReferenceFrames = []common_shared.SpatialFrame{FakeSpatialFrameFull()}

	return sys
}

// FakeSystemFull creates a fully populated System with all optional fields
func FakeSystemFull() domains.System {
	name := fmt.Sprintf("%s Complete System", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	configJSON, _ := json.Marshal(FakeConfigurationSettingsFull())
	modesJSON, _ := json.Marshal(map[string]interface{}{
		"modes": []map[string]interface{}{
			{
				"id":    "normal",
				"label": "Normal Operation",
				"configuration": map[string]interface{}{
					"setValues": []map[string]interface{}{
						{"ref": "parameters/samplingRate", "value": 1.0},
					},
				},
			},
			{
				"id":    "highPrecision",
				"label": "High Precision Mode",
				"configuration": map[string]interface{}{
					"setValues": []map[string]interface{}{
						{"ref": "parameters/samplingRate", "value": 10.0},
					},
				},
			},
		},
	})
	positionJSON, _ := json.Marshal(map[string]interface{}{
		"type": "Point",
		"coordinates": []float64{
			rand.Float64()*360 - 180,
			rand.Float64()*180 - 90,
		},
	})

	return domains.System{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      f.Lorem().Paragraph(2),
		},
		SystemType:           domains.SystemTypeSensor,
		AssetType:            ptrString(domains.AssetTypeEquipment),
		ValidTime:            FakeValidTimeCurrent(),
		Geometry:             FakeDeploymentPointGeometry(),
		Lang:                 ptrString("en"),
		Keywords:             []string{f.Lorem().Word(), f.Lorem().Word(), f.Lorem().Word()},
		Identifiers:          FakeIdentifiers(3),
		Classifiers:          FakeClassifiers(2),
		SecurityConstraints:  FakeSecurityConstraintsFull(1),
		LegalConstraints:     FakeLegalConstraintsFull(1),
		Contacts:             FakeContactWrappers(3),
		Documentation:        FakeDocumentsFull(2),
		History:              FakeHistoryFull(3),
		TypeOf:               &common_shared.Link{Href: f.Internet().URL(), Title: "System Datasheet", Rel: "typeOf"},
		Configuration:        configJSON,
		FeaturesOfInterest:   FakeLinksFull(2),
		Inputs:               FakeIOListFull(2),
		Outputs:              FakeIOListFull(3),
		Parameters:           FakeIOListFull(2),
		Modes:                modesJSON,
		AttachedTo:           &common_shared.Link{Href: f.Internet().URL(), Title: "Platform", Rel: "attachedTo"},
		LocalReferenceFrames: []common_shared.SpatialFrame{FakeSpatialFrameFull()},
		LocalTimeFrames:      []common_shared.TemporalFrame{FakeTemporalFrameFull()},
		Position:             positionJSON,
		Links:                FakeLinksFull(3),
	}
}

// FakeSystem returns a standard populated System (backward compatible)
func FakeSystem() domains.System {
	return FakeSystemSensor()
}

// FakeSystemRandom returns a randomly chosen system type
func FakeSystemRandom() domains.System {
	generators := []func() domains.System{
		FakeSystemMinimal,
		FakeSystemSensor,
		FakeSystemActuator,
		FakeSystemSampler,
		FakeSystemPlatform,
		FakeSystemPhysicalSystem,
		FakeSystemFull,
	}
	return generators[rand.Intn(len(generators))]()
}

// FakeSystemByType returns a system of the specified type
func FakeSystemByType(systemType string) domains.System {
	switch systemType {
	case domains.SystemTypeSensor:
		return FakeSystemSensor()
	case domains.SystemTypeActuator:
		return FakeSystemActuator()
	case domains.SystemTypeSampler:
		return FakeSystemSampler()
	case domains.SystemTypePlatform:
		return FakeSystemPlatform()
	case domains.SystemTypeSystem:
		return FakeSystemPhysicalSystem()
	default:
		return FakeSystemSensor()
	}
}
