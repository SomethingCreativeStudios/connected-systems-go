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
// Procedure Generators - Various Types and Complexity Levels
// =============================================================================

// FakeProcedureMinimal creates a minimal valid Procedure (required fields only)
// Schema requires: type, label, uniqueId, definition
func FakeProcedureMinimal() domains.Procedure {
	name := f.Lorem().Word()
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.Procedure{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
		},
		ProcedureType: domains.ProcedureTypeProcedure,
		ProcessType:   "SimpleProcess",
		Links:         FakeLinksFull(1),
	}
}

// FakeProcedureObserving creates an ObservingProcedure (observation method)
func FakeProcedureObserving() domains.Procedure {
	name := fmt.Sprintf("%s Observation Method", f.Lorem().Word())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.Procedure{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "Standard observation procedure for environmental monitoring",
		},
		ProcedureType:   domains.ProcedureTypeObserving,
		ProcessType:     "SimpleProcess",
		Lang:            ptrString("en"),
		Keywords:        common_shared.StringArray{"observation", "method", "monitoring"},
		Identifiers:     FakeIdentifiers(2),
		Classifiers:     FakeClassifiers(1),
		Characteristics: FakeCharacteristicGroupsFull(1),
		Capabilities:    FakeCapabilityGroupsFull(1),
		Contacts:        FakeContactWrappers(1),
		Documentation:   FakeDocumentsFull(1),
		Inputs:          FakeIOListFull(1),
		Outputs:         FakeIOListFull(2),
		Parameters:      FakeIOListFull(1),
		Method:          FakeMethodFull(),
		ValidTime:       FakeValidTimeCurrent(),
		Links:           FakeLinksFull(1),
	}
}

// FakeProcedureSampling creates a SamplingProcedure (sampling method)
func FakeProcedureSampling() domains.Procedure {
	name := fmt.Sprintf("%s Sampling Procedure", f.Lorem().Word())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.Procedure{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "Standardized sampling procedure for water quality analysis",
		},
		ProcedureType:   domains.ProcedureTypeSampling,
		ProcessType:     "SimpleProcess",
		Lang:            ptrString("en"),
		Keywords:        common_shared.StringArray{"sampling", "collection", "analysis"},
		Identifiers:     FakeIdentifiers(2),
		Classifiers:     FakeClassifiers(1),
		Characteristics: FakeCharacteristicGroupsFull(1),
		Capabilities:    FakeCapabilityGroupsFull(1),
		Contacts:        FakeContactWrappers(1),
		Documentation:   FakeDocumentsFull(2),
		Inputs:          FakeIOListFull(1),
		Outputs:         FakeIOListFull(1),
		Parameters:      FakeIOListFull(2),
		Method:          FakeMethodFull(),
		ValidTime:       FakeValidTimeCurrent(),
		Links:           FakeLinksFull(1),
	}
}

// FakeProcedureActuating creates an ActuatingProcedure (actuation method)
func FakeProcedureActuating() domains.Procedure {
	name := fmt.Sprintf("%s Actuation Procedure", f.Lorem().Word())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.Procedure{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "Control procedure for valve actuation",
		},
		ProcedureType:   domains.ProcedureTypeActuating,
		ProcessType:     "SimpleProcess",
		Lang:            ptrString("en"),
		Keywords:        common_shared.StringArray{"actuation", "control", "automation"},
		Identifiers:     FakeIdentifiers(1),
		Classifiers:     FakeClassifiers(1),
		Characteristics: FakeCharacteristicGroupsFull(1),
		Capabilities:    FakeCapabilityGroupsFull(1),
		Contacts:        FakeContactWrappers(1),
		Documentation:   FakeDocumentsFull(1),
		Inputs:          FakeIOListFull(2),
		Outputs:         FakeIOListFull(1),
		Parameters:      FakeIOListFull(1),
		Method:          FakeMethodFull(),
		ValidTime:       FakeValidTimeCurrent(),
		Links:           FakeLinksFull(1),
	}
}

// FakeProcedureSensorDatasheet creates a Sensor datasheet procedure
func FakeProcedureSensorDatasheet() domains.Procedure {
	name := fmt.Sprintf("%s Sensor Datasheet", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.Procedure{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "Technical specifications and operating parameters for sensor equipment",
		},
		ProcedureType:       domains.ProcedureTypeSensor,
		ProcessType:         "PhysicalComponent",
		Lang:                ptrString("en"),
		Keywords:            common_shared.StringArray{"datasheet", "sensor", "specifications"},
		Identifiers:         FakeIdentifiers(3),
		Classifiers:         FakeClassifiers(2),
		SecurityConstraints: FakeSecurityConstraintsFull(1),
		LegalConstraints:    FakeLegalConstraintsFull(1),
		Characteristics:     FakeCharacteristicGroupsFull(2),
		Capabilities:        FakeCapabilityGroupsFull(2),
		Contacts:            FakeContactWrappers(2),
		Documentation:       FakeDocumentsFull(3),
		History:             FakeHistoryFull(2),
		TypeOf:              &common_shared.Link{Href: f.Internet().URL(), Title: "Parent Datasheet", Rel: "typeOf"},
		Inputs:              FakeIOListFull(1),
		Outputs:             FakeIOListFull(2),
		Parameters:          FakeIOListFull(2),
		ValidTime:           FakeValidTimeCurrent(),
		Links:               FakeLinksFull(2),
	}
}

// FakeProcedureActuatorDatasheet creates an Actuator datasheet procedure
func FakeProcedureActuatorDatasheet() domains.Procedure {
	name := fmt.Sprintf("%s Actuator Datasheet", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.Procedure{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "Technical specifications for actuator equipment",
		},
		ProcedureType:   domains.ProcedureTypeActuator,
		ProcessType:     "PhysicalComponent",
		Lang:            ptrString("en"),
		Keywords:        common_shared.StringArray{"datasheet", "actuator", "control"},
		Identifiers:     FakeIdentifiers(2),
		Classifiers:     FakeClassifiers(2),
		Characteristics: FakeCharacteristicGroupsFull(1),
		Capabilities:    FakeCapabilityGroupsFull(2),
		Contacts:        FakeContactWrappers(1),
		Documentation:   FakeDocumentsFull(2),
		Inputs:          FakeIOListFull(2),
		Outputs:         FakeIOListFull(1),
		ValidTime:       FakeValidTimeCurrent(),
		Links:           FakeLinksFull(1),
	}
}

// FakeProcedurePlatformDatasheet creates a Platform datasheet procedure
func FakeProcedurePlatformDatasheet() domains.Procedure {
	name := fmt.Sprintf("%s Platform Datasheet", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	return domains.Procedure{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "Platform hosting capabilities and specifications",
		},
		ProcedureType:        domains.ProcedureTypePlatform,
		ProcessType:          "PhysicalSystem",
		Lang:                 ptrString("en"),
		Keywords:             common_shared.StringArray{"datasheet", "platform", "hosting"},
		Identifiers:          FakeIdentifiers(2),
		Classifiers:          FakeClassifiers(1),
		Characteristics:      FakeCharacteristicGroupsFull(2),
		Capabilities:         FakeCapabilityGroupsFull(1),
		Contacts:             FakeContactWrappers(2),
		Documentation:        FakeDocumentsFull(2),
		LocalReferenceFrames: []common_shared.SpatialFrame{FakeSpatialFrameFull()},
		LocalTimeFrames:      []common_shared.TemporalFrame{FakeTemporalFrameFull()},
		ValidTime:            FakeValidTimeCurrent(),
		Links:                FakeLinksFull(2),
	}
}

// FakeProcedureAggregateProcess creates an AggregateProcess procedure
func FakeProcedureAggregateProcess() domains.Procedure {
	name := fmt.Sprintf("%s Processing Chain", f.Lorem().Word())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	componentsJSON, _ := json.Marshal([]map[string]interface{}{
		{
			"name": "preprocessor",
			"href": fmt.Sprintf("http://example.org/api/procedures/%s", uuid.New().String()),
		},
		{
			"name": "analyzer",
			"href": fmt.Sprintf("http://example.org/api/procedures/%s", uuid.New().String()),
		},
		{
			"name": "postprocessor",
			"href": fmt.Sprintf("http://example.org/api/procedures/%s", uuid.New().String()),
		},
	})

	connectionsJSON, _ := json.Marshal([]map[string]string{
		{"source": "inputs/rawData", "destination": "components/preprocessor/inputs/data"},
		{"source": "components/preprocessor/outputs/processed", "destination": "components/analyzer/inputs/data"},
		{"source": "components/analyzer/outputs/result", "destination": "components/postprocessor/inputs/data"},
		{"source": "components/postprocessor/outputs/final", "destination": "outputs/result"},
	})

	return domains.Procedure{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "Multi-stage data processing chain",
		},
		ProcedureType:   domains.ProcedureTypeProcedure,
		ProcessType:     "AggregateProcess",
		Lang:            ptrString("en"),
		Keywords:        common_shared.StringArray{"processing", "chain", "aggregate"},
		Identifiers:     FakeIdentifiers(2),
		Classifiers:     FakeClassifiers(1),
		Characteristics: FakeCharacteristicGroupsFull(1),
		Capabilities:    FakeCapabilityGroupsFull(1),
		Contacts:        FakeContactWrappers(1),
		Documentation:   FakeDocumentsFull(1),
		Inputs:          FakeIOListFull(2),
		Outputs:         FakeIOListFull(2),
		Parameters:      FakeIOListFull(1),
		Components:      componentsJSON,
		Connections:     connectionsJSON,
		ValidTime:       FakeValidTimeCurrent(),
		Links:           FakeLinksFull(1),
	}
}

// FakeProcedurePhysicalSystem creates a PhysicalSystem procedure
func FakeProcedurePhysicalSystem() domains.Procedure {
	name := fmt.Sprintf("%s System Datasheet", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	configJSON, _ := json.Marshal(FakeConfigurationSettingsFull())

	return domains.Procedure{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      "Integrated system datasheet with full specifications",
		},
		ProcedureType:        domains.ProcedureTypeSystem,
		ProcessType:          "PhysicalSystem",
		Lang:                 ptrString("en"),
		Keywords:             common_shared.StringArray{"system", "integrated", "datasheet"},
		Identifiers:          FakeIdentifiers(3),
		Classifiers:          FakeClassifiers(2),
		SecurityConstraints:  FakeSecurityConstraintsFull(1),
		LegalConstraints:     FakeLegalConstraintsFull(1),
		Characteristics:      FakeCharacteristicGroupsFull(2),
		Capabilities:         FakeCapabilityGroupsFull(2),
		Contacts:             FakeContactWrappers(2),
		Documentation:        FakeDocumentsFull(3),
		History:              FakeHistoryFull(2),
		TypeOf:               &common_shared.Link{Href: f.Internet().URL(), Title: "Base System", Rel: "typeOf"},
		Configuration:        configJSON,
		FeaturesOfInterest:   FakeLinksFull(1),
		Inputs:               FakeIOListFull(2),
		Outputs:              FakeIOListFull(3),
		Parameters:           FakeIOListFull(2),
		AttachedTo:           &common_shared.Link{Href: f.Internet().URL(), Title: "Platform", Rel: "attachedTo"},
		LocalReferenceFrames: []common_shared.SpatialFrame{FakeSpatialFrameFull()},
		LocalTimeFrames:      []common_shared.TemporalFrame{FakeTemporalFrameFull()},
		ValidTime:            FakeValidTimeCurrent(),
		Links:                FakeLinksFull(2),
	}
}

// FakeProcedureFull creates a fully populated Procedure with all optional fields
func FakeProcedureFull() domains.Procedure {
	name := fmt.Sprintf("%s Complete Procedure", f.Company().Name())
	uid := fmt.Sprintf("urn:uuid:%s", uuid.New().String())

	configJSON, _ := json.Marshal(FakeConfigurationSettingsFull())
	modesJSON, _ := json.Marshal(map[string]interface{}{
		"modes": []map[string]interface{}{
			{
				"id":    "standard",
				"label": "Standard Mode",
				"configuration": map[string]interface{}{
					"setValues": []map[string]interface{}{
						{"ref": "parameters/precision", "value": "normal"},
					},
				},
			},
			{
				"id":    "precision",
				"label": "High Precision Mode",
				"configuration": map[string]interface{}{
					"setValues": []map[string]interface{}{
						{"ref": "parameters/precision", "value": "high"},
					},
				},
			},
		},
	})

	return domains.Procedure{
		Base: domains.Base{ID: uuid.New().String()},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID(uid),
			Name:             name,
			Description:      f.Lorem().Paragraph(2),
		},
		ProcedureType:        domains.ProcedureTypeProcedure,
		ProcessType:          "SimpleProcess",
		Lang:                 ptrString("en"),
		Keywords:             common_shared.StringArray{f.Lorem().Word(), f.Lorem().Word(), f.Lorem().Word()},
		Identifiers:          FakeIdentifiers(3),
		Classifiers:          FakeClassifiers(2),
		SecurityConstraints:  FakeSecurityConstraintsFull(1),
		LegalConstraints:     FakeLegalConstraintsFull(1),
		Characteristics:      FakeCharacteristicGroupsFull(2),
		Capabilities:         FakeCapabilityGroupsFull(2),
		Contacts:             FakeContactWrappers(3),
		Documentation:        FakeDocumentsFull(3),
		History:              FakeHistoryFull(3),
		TypeOf:               &common_shared.Link{Href: f.Internet().URL(), Title: "Base Procedure", Rel: "typeOf"},
		Configuration:        configJSON,
		FeaturesOfInterest:   FakeLinksFull(2),
		Inputs:               FakeIOListFull(2),
		Outputs:              FakeIOListFull(3),
		Parameters:           FakeIOListFull(2),
		Method:               FakeMethodFull(),
		Modes:                modesJSON,
		AttachedTo:           &common_shared.Link{Href: f.Internet().URL(), Title: "Attached Platform", Rel: "attachedTo"},
		LocalReferenceFrames: []common_shared.SpatialFrame{FakeSpatialFrameFull()},
		LocalTimeFrames:      []common_shared.TemporalFrame{FakeTemporalFrameFull()},
		ValidTime:            FakeValidTimeCurrent(),
		Links:                FakeLinksFull(3),
		Properties:           common_shared.Properties{"customField": f.Lorem().Word()},
	}
}

// FakeProcedure returns a standard populated Procedure (backward compatible)
func FakeProcedure() domains.Procedure {
	return FakeProcedureObserving()
}

// FakeProcedureRandom returns a randomly chosen procedure type
func FakeProcedureRandom() domains.Procedure {
	generators := []func() domains.Procedure{
		FakeProcedureMinimal,
		FakeProcedureObserving,
		FakeProcedureSampling,
		FakeProcedureActuating,
		FakeProcedureSensorDatasheet,
		FakeProcedureActuatorDatasheet,
		FakeProcedurePlatformDatasheet,
		FakeProcedureAggregateProcess,
		FakeProcedurePhysicalSystem,
		FakeProcedureFull,
	}
	return generators[rand.Intn(len(generators))]()
}

// FakeProcedureByType returns a procedure of the specified procedure type
func FakeProcedureByType(procedureType string) domains.Procedure {
	switch procedureType {
	case domains.ProcedureTypeObserving:
		return FakeProcedureObserving()
	case domains.ProcedureTypeSampling:
		return FakeProcedureSampling()
	case domains.ProcedureTypeActuating:
		return FakeProcedureActuating()
	case domains.ProcedureTypeSensor:
		return FakeProcedureSensorDatasheet()
	case domains.ProcedureTypeActuator:
		return FakeProcedureActuatorDatasheet()
	case domains.ProcedureTypePlatform:
		return FakeProcedurePlatformDatasheet()
	case domains.ProcedureTypeSystem:
		return FakeProcedurePhysicalSystem()
	default:
		return FakeProcedureObserving()
	}
}

// FakeProcedureByProcessType returns a procedure of the specified process type
func FakeProcedureByProcessType(processType string) domains.Procedure {
	switch processType {
	case "SimpleProcess":
		return FakeProcedureObserving()
	case "AggregateProcess":
		return FakeProcedureAggregateProcess()
	case "PhysicalComponent":
		return FakeProcedureSensorDatasheet()
	case "PhysicalSystem":
		return FakeProcedurePhysicalSystem()
	default:
		return FakeProcedureObserving()
	}
}
