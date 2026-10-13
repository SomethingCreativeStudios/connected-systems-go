package repository

import (
	"testing"
	"time"

	"errors"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/stretchr/testify/require"
)

func TestDatastreamRepository_DeleteCascade_RemovesObservations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	datastreamRepo := NewDatastreamRepository(db)
	observationRepo := NewObservationRepository(db)

	datastream := &domains.Datastream{Schema: testNumericObservationSchema(),
		Name: "Cascade Datastream",
	}
	require.NoError(t, datastreamRepo.Create(t.Context(), datastream))

	observation := &domains.Observation{
		DatastreamID: datastream.ID,
		ResultTime:   time.Now().UTC(),
	}
	require.NoError(t, observationRepo.Create(t.Context(), observation))

	require.NoError(t, datastreamRepo.Delete(t.Context(), datastream.ID, true))

	_, err := datastreamRepo.GetByID(t.Context(), datastream.ID)
	require.Error(t, err)

	_, err = observationRepo.GetByID(t.Context(), observation.ID)
	require.Error(t, err)
}

func TestControlStreamRepository_DeleteCascade_RemovesCommands(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	controlStreamRepo := NewControlStreamRepository(db)
	commandRepo := NewCommandRepository(db)

	controlStream := &domains.ControlStream{Schema: testNumericCommandSchema(),
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: domains.UniqueID("urn:test:cs:cascade:1"),
			Name:             "Cascade Control Stream",
		},
	}
	require.NoError(t, controlStreamRepo.Create(t.Context(), controlStream))

	command := &domains.Command{Parameters: []byte(`1`),
		ControlStreamID: controlStream.ID,
		Sender:          "tester",
	}
	require.NoError(t, commandRepo.Create(t.Context(), command))

	require.NoError(t, controlStreamRepo.Delete(t.Context(), controlStream.ID, true))

	_, err := controlStreamRepo.GetByID(t.Context(), controlStream.ID)
	require.Error(t, err)

	_, err = commandRepo.GetByID(t.Context(), command.ID)
	require.Error(t, err)
}

func TestSystemRepository_DeleteCascade_RemovesAssociatedResourcesRecursively(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	systemRepo := NewSystemRepository(db)
	historyRepo := NewSystemHistoryRepository(db)
	systemEventRepo := NewSystemEventRepository(db)
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
	require.NoError(t, systemRepo.Create(t.Context(), parent))

	child := &domains.System{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:sys:cascade:child", Name: "Cascade Child"},
		SystemType:     domains.SystemTypeSensor,
		ParentSystemID: &parent.ID,
	}
	require.NoError(t, systemRepo.Create(t.Context(), child))

	require.NoError(t, samplingFeatureRepo.Create(t.Context(), &domains.SamplingFeature{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:sf:cascade:parent", Name: "Parent SF"},
		FeatureType:    domains.SamplingFeatureTypeSample,
		ParentSystemID: &parent.ID,
	}))
	require.NoError(t, samplingFeatureRepo.Create(t.Context(), &domains.SamplingFeature{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:sf:cascade:child", Name: "Child SF"},
		FeatureType:    domains.SamplingFeatureTypeSample,
		ParentSystemID: &child.ID,
	}))

	parentDS := &domains.Datastream{Schema: testNumericObservationSchema(),
		Name:     "Parent DS",
		SystemID: &parent.ID,
	}
	require.NoError(t, datastreamRepo.Create(t.Context(), parentDS))

	childDS := &domains.Datastream{Schema: testNumericObservationSchema(),
		Name:     "Child DS",
		SystemID: &child.ID,
	}
	require.NoError(t, datastreamRepo.Create(t.Context(), childDS))

	require.NoError(t, observationRepo.Create(t.Context(), &domains.Observation{DatastreamID: parentDS.ID, ResultTime: time.Now().UTC()}))
	require.NoError(t, observationRepo.Create(t.Context(), &domains.Observation{DatastreamID: childDS.ID, ResultTime: time.Now().UTC()}))

	parentCS := &domains.ControlStream{Schema: testNumericCommandSchema(),
		CommonSSN: domains.CommonSSN{UniqueIdentifier: domains.UniqueID("urn:test:cs:cascade:parent"), Name: "Parent CS"},
		SystemID:  &parent.ID,
	}
	require.NoError(t, controlStreamRepo.Create(t.Context(), parentCS))

	childCS := &domains.ControlStream{Schema: testNumericCommandSchema(),
		CommonSSN: domains.CommonSSN{UniqueIdentifier: domains.UniqueID("urn:test:cs:cascade:child"), Name: "Child CS"},
		SystemID:  &child.ID,
	}
	require.NoError(t, controlStreamRepo.Create(t.Context(), childCS))

	require.NoError(t, commandRepo.Create(t.Context(), &domains.Command{Parameters: []byte(`1`), ControlStreamID: parentCS.ID, Sender: "tester"}))
	require.NoError(t, commandRepo.Create(t.Context(), &domains.Command{Parameters: []byte(`1`), ControlStreamID: childCS.ID, Sender: "tester"}))

	_, err := historyRepo.CreateFromSystem(t.Context(), parent)
	require.NoError(t, err)
	_, err = historyRepo.CreateFromSystem(t.Context(), child)
	require.NoError(t, err)
	require.NoError(t, systemEventRepo.Create(t.Context(), &domains.SystemEvent{
		SystemID: parent.ID,
		Label:    "Parent Event",
		Time:     common_shared.HistoryTime{Instant: new(time.Now().UTC())},
	}))
	require.NoError(t, systemEventRepo.Create(t.Context(), &domains.SystemEvent{
		SystemID: child.ID,
		Label:    "Child Event",
		Time:     common_shared.HistoryTime{Instant: new(time.Now().UTC())},
	}))
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
	require.NoError(t, deploymentRepo.Create(t.Context(), deployment))

	require.NoError(t, systemRepo.Delete(t.Context(), parent.ID, true))

	_, err = systemRepo.GetByID(t.Context(), parent.ID)
	require.Error(t, err)
	_, err = systemRepo.GetByID(t.Context(), child.ID)
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

	var eventCount int64
	require.NoError(t, db.Model(&domains.SystemEvent{}).Where("system_id IN ?", []string{parent.ID, child.ID}).Count(&eventCount).Error)
	require.Equal(t, int64(0), eventCount)

	updatedDeployment, err := deploymentRepo.GetByID(t.Context(), deployment.ID)
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

func TestSystemRepository_Delete_ReturnsErrNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := NewSystemRepository(db).Delete(t.Context(), "nonexistent-id", false)
	require.True(t, errors.Is(err, ErrNotFound))
}

func TestSystemRepository_Delete_ReturnsErrHasChildren(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	systemRepo := NewSystemRepository(db)

	parent := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sys:err:parent", Name: "Error Parent"},
		SystemType: domains.SystemTypePlatform,
	}
	require.NoError(t, systemRepo.Create(t.Context(), parent))

	child := &domains.System{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:sys:err:child", Name: "Error Child"},
		SystemType:     domains.SystemTypeSensor,
		ParentSystemID: &parent.ID,
	}
	require.NoError(t, systemRepo.Create(t.Context(), child))

	err := systemRepo.Delete(t.Context(), parent.ID, false)
	require.True(t, errors.Is(err, ErrHasChildren))
}

func TestDatastreamRepository_Delete_ReturnsErrNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := NewDatastreamRepository(db).Delete(t.Context(), "nonexistent-id", false)
	require.True(t, errors.Is(err, ErrNotFound))
}

func TestDatastreamRepository_Delete_ReturnsErrHasChildren(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	datastreamRepo := NewDatastreamRepository(db)
	observationRepo := NewObservationRepository(db)

	ds := &domains.Datastream{Schema: testNumericObservationSchema(), Name: "Has Children DS"}
	require.NoError(t, datastreamRepo.Create(t.Context(), ds))
	require.NoError(t, observationRepo.Create(t.Context(), &domains.Observation{
		DatastreamID: ds.ID,
		ResultTime:   time.Now().UTC(),
	}))

	err := datastreamRepo.Delete(t.Context(), ds.ID, false)
	require.True(t, errors.Is(err, ErrHasChildren))
}

func TestControlStreamRepository_Delete_ReturnsErrNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := NewControlStreamRepository(db).Delete(t.Context(), "nonexistent-id", false)
	require.True(t, errors.Is(err, ErrNotFound))
}

func TestControlStreamRepository_Delete_ReturnsErrHasChildren(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	controlStreamRepo := NewControlStreamRepository(db)
	commandRepo := NewCommandRepository(db)

	cs := &domains.ControlStream{Schema: testNumericCommandSchema(),
		CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:cs:err:1", Name: "Has Children CS"},
	}
	require.NoError(t, controlStreamRepo.Create(t.Context(), cs))
	require.NoError(t, commandRepo.Create(t.Context(), &domains.Command{Parameters: []byte(`1`),
		ControlStreamID: cs.ID,
		Sender:          "tester",
	}))

	err := controlStreamRepo.Delete(t.Context(), cs.ID, false)
	require.True(t, errors.Is(err, ErrHasChildren))
}

func TestDeploymentRepository_Delete_ReturnsErrNotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := NewDeploymentRepository(db).Delete(t.Context(), "nonexistent-id", false)
	require.True(t, errors.Is(err, ErrNotFound))
}

func TestDeploymentRepository_Delete_ReturnsErrHasChildren(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	deploymentRepo := NewDeploymentRepository(db)

	parent := &domains.Deployment{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:dep:err:parent", Name: "Error Parent Dep"},
		DeploymentType: domains.DeploymentTypeDeployment,
	}
	require.NoError(t, deploymentRepo.Create(t.Context(), parent))

	child := &domains.Deployment{
		CommonSSN:          domains.CommonSSN{UniqueIdentifier: "urn:test:dep:err:child", Name: "Error Child Dep"},
		DeploymentType:     domains.DeploymentTypeDeployment,
		ParentDeploymentID: &parent.ID,
	}
	require.NoError(t, deploymentRepo.Create(t.Context(), child))

	err := deploymentRepo.Delete(t.Context(), parent.ID, false)
	require.True(t, errors.Is(err, ErrHasChildren))
}

func TestDeploymentRepository_DeleteCascade_RemovesSubdeployments(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	deploymentRepo := NewDeploymentRepository(db)

	parent := &domains.Deployment{
		CommonSSN:      domains.CommonSSN{UniqueIdentifier: "urn:test:dep:cascade:parent", Name: "Cascade Parent Dep"},
		DeploymentType: domains.DeploymentTypeDeployment,
	}
	require.NoError(t, deploymentRepo.Create(t.Context(), parent))

	child := &domains.Deployment{
		CommonSSN:          domains.CommonSSN{UniqueIdentifier: "urn:test:dep:cascade:child", Name: "Cascade Child Dep"},
		DeploymentType:     domains.DeploymentTypeDeployment,
		ParentDeploymentID: &parent.ID,
	}
	require.NoError(t, deploymentRepo.Create(t.Context(), child))

	grandchild := &domains.Deployment{
		CommonSSN:          domains.CommonSSN{UniqueIdentifier: "urn:test:dep:cascade:grandchild", Name: "Cascade Grandchild Dep"},
		DeploymentType:     domains.DeploymentTypeDeployment,
		ParentDeploymentID: &child.ID,
	}
	require.NoError(t, deploymentRepo.Create(t.Context(), grandchild))

	require.NoError(t, deploymentRepo.Delete(t.Context(), parent.ID, true))

	_, err := deploymentRepo.GetByID(t.Context(), parent.ID)
	require.Error(t, err)
	_, err = deploymentRepo.GetByID(t.Context(), child.ID)
	require.Error(t, err)
	_, err = deploymentRepo.GetByID(t.Context(), grandchild.ID)
	require.Error(t, err)
}
