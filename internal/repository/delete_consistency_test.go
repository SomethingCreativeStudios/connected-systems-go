package repository

import (
	"testing"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/stretchr/testify/require"
)

// A refused or failed delete must leave everything as it was, and a cascade
// delete must not leave rows behind that point at the deleted resources.
func TestDelete_Consistency(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db)

	newSystem := func(t *testing.T, uid string) *domains.System {
		t.Helper()
		system := &domains.System{
			CommonSSN:  domains.CommonSSN{UniqueIdentifier: domains.UniqueID(uid), Name: uid},
			SystemType: domains.SystemTypeSensor,
		}
		require.NoError(t, repos.System.Create(t.Context(), system))
		return system
	}
	rows := func(t *testing.T, table, id string) int64 {
		t.Helper()
		var count int64
		require.NoError(t, db.Table(table).Where("id = ?", id).Count(&count).Error)
		return count
	}

	t.Run("refused datastream delete keeps the datastream", func(t *testing.T) {
		system := newSystem(t, "urn:test:delete:ds")
		ds := &domains.Datastream{Schema: testNumericObservationSchema(), Name: "Temperature", SystemID: &system.ID}
		require.NoError(t, repos.Datastream.Create(t.Context(), ds))
		require.NoError(t, repos.Observation.Create(t.Context(), &domains.Observation{
			DatastreamID: ds.ID, ResultTime: time.Now().UTC(), Result: []byte(`1`),
		}))

		require.ErrorIs(t, repos.Datastream.Delete(t.Context(), ds.ID, false), ErrHasChildren)
		require.EqualValues(t, 1, rows(t, "datastreams", ds.ID))

		require.ErrorIs(t, repos.Datastream.Delete(t.Context(), "missing-datastream", false), ErrNotFound)
	})

	t.Run("refused control stream delete keeps the control stream", func(t *testing.T) {
		system := newSystem(t, "urn:test:delete:cs")
		cs := &domains.ControlStream{Schema: testNumericCommandSchema(), CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:cs:pantilt", Name: "Pan/tilt"}, SystemID: &system.ID}
		require.NoError(t, repos.ControlStream.Create(t.Context(), cs))
		require.NoError(t, repos.Command.Create(t.Context(), &domains.Command{Parameters: []byte(`1`), ControlStreamID: cs.ID}))

		require.ErrorIs(t, repos.ControlStream.Delete(t.Context(), cs.ID, false), ErrHasChildren)
		require.EqualValues(t, 1, rows(t, "control_streams", cs.ID))

		require.ErrorIs(t, repos.ControlStream.Delete(t.Context(), "missing-control-stream", true), ErrNotFound)
	})

	t.Run("cascade system delete removes command status reports and results", func(t *testing.T) {
		system := newSystem(t, "urn:test:delete:cascade")
		cs := &domains.ControlStream{Schema: testNumericCommandSchema(), CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:cs:zoom", Name: "Zoom"}, SystemID: &system.ID}
		require.NoError(t, repos.ControlStream.Create(t.Context(), cs))
		cmd := &domains.Command{Parameters: []byte(`1`), ControlStreamID: cs.ID}
		require.NoError(t, repos.Command.Create(t.Context(), cmd))
		require.NoError(t, repos.Command.CreateStatus(t.Context(), &domains.CommandStatusReport{
			CommandID: cmd.ID, ReportTime: time.Now().UTC(), StatusCode: "ACCEPTED",
		}))
		require.NoError(t, repos.Command.CreateResult(t.Context(), &domains.CommandResult{CommandID: cmd.ID, Data: []byte(`{}`)}))

		require.NoError(t, repos.System.Delete(t.Context(), system.ID, true))

		for _, model := range []any{&domains.Command{}, &domains.CommandStatusReport{}, &domains.CommandResult{}} {
			var count int64
			query := db.Model(model)
			if _, isCommand := model.(*domains.Command); isCommand {
				query = query.Where("id = ?", cmd.ID)
			} else {
				query = query.Where("command_id = ?", cmd.ID)
			}
			require.NoError(t, query.Count(&count).Error)
			require.Zero(t, count, "%T rows left after cascade delete", model)
		}
	})

	t.Run("refused deployment delete keeps its system links", func(t *testing.T) {
		system := newSystem(t, "urn:test:delete:dep")
		parent := &domains.Deployment{
			CommonSSN:       domains.CommonSSN{UniqueIdentifier: "urn:test:delete:dep:parent", Name: "Campaign"},
			DeployedSystems: domains.DeployedSystemItems{{System: common_shared.Link{Href: "/systems/" + system.ID}}},
		}
		require.NoError(t, repos.Deployment.Create(t.Context(), parent))
		require.NoError(t, repos.Deployment.Create(t.Context(), &domains.Deployment{
			CommonSSN:          domains.CommonSSN{UniqueIdentifier: "urn:test:delete:dep:child", Name: "Leg"},
			ParentDeploymentID: &parent.ID,
		}))

		require.ErrorIs(t, repos.Deployment.Delete(t.Context(), parent.ID, false), ErrHasChildren)
		var links int64
		require.NoError(t, db.Table("system_deployments").Where("deployment_id = ?", parent.ID).Count(&links).Error)
		require.EqualValues(t, 1, links)
	})

}

// An observation without a resultTime gets "now", and an observation without
// a phenomenonTime takes its resultTime; neither may fall back to year 1.
func TestObservationCreate_DefaultsTimes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db)

	ds := &domains.Datastream{Schema: testNumericObservationSchema(), Name: "Defaults"}
	require.NoError(t, repos.Datastream.Create(t.Context(), ds))

	obs := &domains.Observation{DatastreamID: ds.ID, Result: []byte(`1`)}
	require.NoError(t, repos.Observation.Create(t.Context(), obs))
	require.False(t, obs.ResultTime.IsZero())
	require.NotNil(t, obs.PhenomenonTime)
	require.True(t, obs.PhenomenonTime.Equal(obs.ResultTime), "phenomenonTime %v, resultTime %v", obs.PhenomenonTime, obs.ResultTime)
}
