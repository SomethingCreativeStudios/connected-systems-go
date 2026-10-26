package repository

import (
	"context"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository/testutil"
	"github.com/stretchr/testify/require"
)

// A database created before versioned migrations, still carrying the
// duplicate relationship tables and the deployment closure triggers, is
// brought to the latest version; running Migrate again changes nothing.
func TestMigrate_UpgradesLegacyDatabase(t *testing.T) {
	ctx := context.Background()
	container := testutil.StartPostGISContainer(ctx, t)
	db := testutil.OpenTestDB(t, container.DSN, testutil.OpenTestDBOptions{Models: testutil.DefaultSystemModels()})
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		container.Terminate(ctx)
	})

	for _, statement := range []string{
		`CREATE TABLE system_procedures (system_id varchar(255), procedure_id varchar(255))`,
		`CREATE TABLE system_datastreams (system_id varchar(255), datastream_id varchar(255))`,
		`CREATE TABLE system_controlstreams (system_id varchar(255), control_stream_id varchar(255))`,
		`CREATE TABLE deployment_closures (ancestor_id varchar(255), descendant_id varchar(255), depth int)`,
		`CREATE FUNCTION deployments_closure_after_insert() RETURNS trigger AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql`,
		`CREATE TRIGGER trg_deployments_closure_insert AFTER INSERT ON deployments FOR EACH ROW EXECUTE FUNCTION deployments_closure_after_insert()`,
	} {
		require.NoError(t, db.Exec(statement).Error)
	}

	system := &domains.System{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:legacy:system", Name: "Legacy"}, SystemType: domains.SystemTypeSensor}
	require.NoError(t, db.Create(system).Error)
	procedure := &domains.Procedure{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:legacy:procedure", Name: "Legacy procedure"}}
	require.NoError(t, db.Create(procedure).Error)
	ds := &domains.Datastream{Name: "Legacy observations", Schema: testNumericObservationSchema()}
	require.NoError(t, db.Create(ds).Error)
	cs := &domains.ControlStream{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:legacy:control", Name: "Legacy commands"}, Schema: testNumericCommandSchema()}
	require.NoError(t, db.Create(cs).Error)
	parent := &domains.Deployment{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:legacy:parent", Name: "Parent"}}
	child := &domains.Deployment{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:legacy:child", Name: "Child"}}
	require.NoError(t, db.Create(parent).Error)
	require.NoError(t, db.Create(child).Error)
	require.NoError(t, db.Exec("INSERT INTO system_datastreams VALUES (?, ?)", system.ID, ds.ID).Error)
	require.NoError(t, db.Exec("INSERT INTO system_controlstreams VALUES (?, ?)", system.ID, cs.ID).Error)
	require.NoError(t, db.Exec("INSERT INTO system_procedures VALUES (?, ?)", system.ID, procedure.ID).Error)
	require.NoError(t, db.Exec("INSERT INTO deployment_closures VALUES (?, ?, 1)", parent.ID, child.ID).Error)

	require.NoError(t, Migrate(ctx, db))
	require.NoError(t, Migrate(ctx, db))

	require.NoError(t, db.First(ds, "id = ?", ds.ID).Error)
	require.Equal(t, &system.ID, ds.SystemID)
	require.NotNil(t, ds.ObservedProperties)
	require.Len(t, ds.Schemas, 1)
	require.NoError(t, db.First(cs, "id = ?", cs.ID).Error)
	require.Equal(t, &system.ID, cs.SystemID)
	require.NotNil(t, cs.ControlledProperties)
	require.Len(t, cs.Schemas, 1)
	require.NoError(t, db.First(system, "id = ?", system.ID).Error)
	require.Equal(t, &procedure.ID, system.TypeOfID)
	require.NoError(t, db.First(child, "id = ?", child.ID).Error)
	require.Equal(t, &parent.ID, child.ParentDeploymentID)

	var version int64
	require.NoError(t, db.Raw(`SELECT MAX(version_id) FROM goose_db_version WHERE is_applied`).Scan(&version).Error)
	require.EqualValues(t, 4, version)
	require.True(t, db.Migrator().HasTable("mqtt_create_operations"))
	require.True(t, db.Migrator().HasColumn(&MQTTCreateOperation{}, "response"))

	for _, table := range []string{"system_procedures", "system_datastreams", "system_controlstreams", "deployment_closures"} {
		require.False(t, db.Migrator().HasTable(table), table)
	}
	var triggers int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM pg_trigger WHERE tgname LIKE 'trg_deployments_closure_%'`).Scan(&triggers).Error)
	require.Zero(t, triggers)

	// The application tables are untouched, and inserting a deployment no
	// longer fires a closure trigger.
	require.NoError(t, NewRepositories(db).Deployment.Create(ctx, &domains.Deployment{
		CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:migrated", Name: "Migrated"},
	}))
}

func TestMigrate_RejectsConflictingLegacyRelationships(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db)
	first := &domains.System{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:migration:first", Name: "First"}, SystemType: domains.SystemTypeSensor}
	second := &domains.System{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:migration:second", Name: "Second"}, SystemType: domains.SystemTypeSensor}
	require.NoError(t, repos.System.Create(t.Context(), first))
	require.NoError(t, repos.System.Create(t.Context(), second))
	ds := &domains.Datastream{Name: "Owned", SystemID: &first.ID, Schema: testNumericObservationSchema()}
	require.NoError(t, repos.Datastream.Create(t.Context(), ds))
	require.NoError(t, db.Exec("CREATE TABLE system_datastreams (system_id varchar(255), datastream_id varchar(255))").Error)
	require.NoError(t, db.Exec("INSERT INTO system_datastreams VALUES (?, ?)", second.ID, ds.ID).Error)
	err := Migrate(t.Context(), db)
	require.ErrorContains(t, err, "Conflicting legacy relationships")
	require.True(t, db.Migrator().HasTable("system_datastreams"))
	require.NoError(t, db.Exec("INSERT INTO system_datastreams VALUES (?, ?)", first.ID, ds.ID).Error)
	err = Migrate(t.Context(), db)
	require.ErrorContains(t, err, "Ambiguous legacy relationships")
	require.True(t, db.Migrator().HasTable("system_datastreams"))

	stored, err := repos.Datastream.GetByID(t.Context(), ds.ID)
	require.NoError(t, err)
	require.Equal(t, &first.ID, stored.SystemID)
}
