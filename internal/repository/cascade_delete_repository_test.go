package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

func TestDatastreamRepository_DeleteCascade_RemovesObservations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	datastreamRepo := NewDatastreamRepository(db)
	observationRepo := NewObservationRepository(db)

	datastream := &domains.Datastream{
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID("urn:test:ds:cascade:1"),
			Name:             "Cascade Datastream",
		},
	}
	require.NoError(t, datastreamRepo.Create(datastream))

	observation := &domains.Observation{
		DatastreamID: datastream.ID,
		ResultTime:   time.Now().UTC(),
	}
	require.NoError(t, observationRepo.Create(observation))

	require.NoError(t, datastreamRepo.Delete(datastream.ID, true))

	_, err := datastreamRepo.GetByID(datastream.ID)
	require.Error(t, err)

	_, err = observationRepo.GetByID(observation.ID)
	require.Error(t, err)
}

func TestControlStreamRepository_DeleteCascade_RemovesCommands(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	controlStreamRepo := NewControlStreamRepository(db)
	commandRepo := NewCommandRepository(db)

	controlStream := &domains.ControlStream{
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID("urn:test:cs:cascade:1"),
			Name:             "Cascade Control Stream",
		},
	}
	require.NoError(t, controlStreamRepo.Create(controlStream))

	command := &domains.Command{
		ControlStreamID: controlStream.ID,
		Sender:          "tester",
	}
	require.NoError(t, commandRepo.Create(command))

	require.NoError(t, controlStreamRepo.Delete(controlStream.ID, true))

	_, err := controlStreamRepo.GetByID(controlStream.ID)
	require.Error(t, err)

	_, err = commandRepo.GetByID(command.ID)
	require.Error(t, err)
}

func TestSystemRepository_DeleteCascade_RemovesAssociatedResourcesRecursively(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	systemRepo := NewSystemRepository(db)
	historyRepo := NewSystemHistoryRepository(db)
	samplingFeatureRepo := NewSamplingFeatureRepository(db)
	datastreamRepo := NewDatastreamRepository(db)
	observationRepo := NewObservationRepository(db)
	controlStreamRepo := NewControlStreamRepository(db)
	commandRepo := NewCommandRepository(db)
	deploymentRepo := NewDeploymentRepository(db)

	parent := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sys:cascade:parent", Name: "Cascade Parent"},
		SystemType: domains.SystemTypePlatform,
	}
	require.NoError(t, systemRepo.Create(parent))

	child := &domains.System{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:sys:cascade:child", Name: "Cascade Child"},
		SystemType:     domains.SystemTypeSensor,
		ParentSystemID: &parent.ID,
	}
	require.NoError(t, systemRepo.Create(child))

	require.NoError(t, samplingFeatureRepo.Create(&domains.SamplingFeature{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:sf:cascade:parent", Name: "Parent SF"},
		FeatureType:    domains.SamplingFeatureTypeSample,
		ParentSystemID: &parent.ID,
	}))
	require.NoError(t, samplingFeatureRepo.Create(&domains.SamplingFeature{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:sf:cascade:child", Name: "Child SF"},
		FeatureType:    domains.SamplingFeatureTypeSample,
		ParentSystemID: &child.ID,
	}))

	parentDS := &domains.Datastream{
		CommonSSN: domains.CommonSSN{UniqueIdentifier: domains.UniqueID("urn:test:ds:cascade:parent"), Name: "Parent DS"},
		SystemID:  &parent.ID,
	}
	require.NoError(t, datastreamRepo.Create(parentDS))

	childDS := &domains.Datastream{
		CommonSSN: domains.CommonSSN{UniqueIdentifier: domains.UniqueID("urn:test:ds:cascade:child"), Name: "Child DS"},
		SystemID:  &child.ID,
	}
	require.NoError(t, datastreamRepo.Create(childDS))

	require.NoError(t, observationRepo.Create(&domains.Observation{DatastreamID: parentDS.ID, ResultTime: time.Now().UTC()}))
	require.NoError(t, observationRepo.Create(&domains.Observation{DatastreamID: childDS.ID, ResultTime: time.Now().UTC()}))

	parentCS := &domains.ControlStream{
		CommonSSN: domains.CommonSSN{UniqueIdentifier: domains.UniqueID("urn:test:cs:cascade:parent"), Name: "Parent CS"},
		SystemID:  &parent.ID,
	}
	require.NoError(t, controlStreamRepo.Create(parentCS))

	childCS := &domains.ControlStream{
		CommonSSN: domains.CommonSSN{UniqueIdentifier: domains.UniqueID("urn:test:cs:cascade:child"), Name: "Child CS"},
		SystemID:  &child.ID,
	}
	require.NoError(t, controlStreamRepo.Create(childCS))

	require.NoError(t, commandRepo.Create(&domains.Command{ControlStreamID: parentCS.ID, Sender: "tester"}))
	require.NoError(t, commandRepo.Create(&domains.Command{ControlStreamID: childCS.ID, Sender: "tester"}))

	_, err := historyRepo.CreateFromSystem(parent)
	require.NoError(t, err)
	_, err = historyRepo.CreateFromSystem(child)
	require.NoError(t, err)

	systemIDs := common_shared.StringArray{parent.ID, child.ID}
	deployment := &domains.Deployment{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:dep:cascade:1", Name: "Cascade Deployment"},
		DeploymentType: domains.DeploymentTypeDeployment,
		SystemIds:      &systemIDs,
		DeployedSystems: domains.DeployedSystemItems{
			{System: common_shared.Link{Href: "/systems/" + parent.ID}},
			{System: common_shared.Link{Href: "/systems/" + child.ID}},
		},
		Platform:   &domains.DeployedSystemItem{System: common_shared.Link{Href: "/systems/" + parent.ID}},
		PlatformID: &parent.ID,
	}
	require.NoError(t, deploymentRepo.Create(deployment))

	require.NoError(t, systemRepo.Delete(parent.ID, true))

	_, err = systemRepo.GetByID(parent.ID)
	require.Error(t, err)
	_, err = systemRepo.GetByID(child.ID)
	require.Error(t, err)

	var sfCount int64
	require.NoError(t, db.Model(&domains.SamplingFeature{}).Where("parent_system_id IN ?", []string{parent.ID, child.ID}).Count(&sfCount).Error)
	require.Equal(t, int64(0), sfCount)

	var dsCount int64
	require.NoError(t, db.Model(&domains.Datastream{}).Where("system_id IN ?", []string{parent.ID, child.ID}).Count(&dsCount).Error)
	require.Equal(t, int64(0), dsCount)

	var obsCount int64
	require.NoError(t, db.Model(&domains.Observation{}).Where("datastream_id IN ?", []string{parentDS.ID, childDS.ID}).Count(&obsCount).Error)
	require.Equal(t, int64(0), obsCount)

	var csCount int64
	require.NoError(t, db.Model(&domains.ControlStream{}).Where("system_id IN ?", []string{parent.ID, child.ID}).Count(&csCount).Error)
	require.Equal(t, int64(0), csCount)

	var cmdCount int64
	require.NoError(t, db.Model(&domains.Command{}).Where("control_stream_id IN ?", []string{parentCS.ID, childCS.ID}).Count(&cmdCount).Error)
	require.Equal(t, int64(0), cmdCount)

	var histCount int64
	require.NoError(t, db.Model(&domains.SystemHistoryRevision{}).Where("system_id IN ?", []string{parent.ID, child.ID}).Count(&histCount).Error)
	require.Equal(t, int64(0), histCount)

	updatedDeployment, err := deploymentRepo.GetByID(deployment.ID)
	require.NoError(t, err)

	if updatedDeployment.SystemIds != nil {
		for _, id := range *updatedDeployment.SystemIds {
			require.NotEqual(t, parent.ID, id)
			require.NotEqual(t, child.ID, id)
		}
	}
	for _, item := range updatedDeployment.DeployedSystems {
		id := item.System.GetId("systems")
		if id != nil {
			require.NotEqual(t, parent.ID, *id)
			require.NotEqual(t, child.ID, *id)
		}
	}
	require.Nil(t, updatedDeployment.Platform)
	require.Nil(t, updatedDeployment.PlatformID)
}
