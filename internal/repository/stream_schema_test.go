package repository

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// A child that starts while a schema writer holds the stream lock must be
// validated against the schema committed by that writer, before insertion.
func TestChildWritesValidateAfterStreamLock(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db)
	ds := &domains.Datastream{Name: "Locked observations", Schema: testNumericObservationSchema()}
	require.NoError(t, repos.Datastream.Create(t.Context(), ds))
	cs := &domains.ControlStream{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:locked:commands", Name: "Locked commands"}, Schema: testNumericCommandSchema()}
	require.NoError(t, repos.ControlStream.Create(t.Context(), cs))
	for _, kind := range []string{"observations", "commands"} {
		t.Run(kind, func(t *testing.T) {
			tx := db.Begin()
			require.NoError(t, tx.Error)
			defer tx.Rollback()
			if kind == "observations" {
				require.NoError(t, tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&domains.Datastream{}, "id = ?", ds.ID).Error)
			} else {
				require.NoError(t, tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&domains.ControlStream{}, "id = ?", cs.ID).Error)
			}
			started := make(chan struct{})
			done := make(chan error, 1)
			go func() {
				close(started)
				if kind == "observations" {
					done <- repos.Observation.Create(t.Context(), &domains.Observation{DatastreamID: ds.ID, ResultTime: time.Now().UTC(), Result: []byte(`1`)})
				} else {
					done <- repos.Command.Create(t.Context(), &domains.Command{ControlStreamID: cs.ID, Parameters: []byte(`1`)})
				}
			}()
			<-started
			component := &domains.DatastreamDataComponent{Type: "Text", Definition: "http://example.test/text", Label: "Text"}
			if kind == "observations" {
				schema := domains.DatastreamSchema{ObsFormat: "application/json", ResultSchema: component}
				require.NoError(t, tx.Model(ds).Updates(map[string]any{"schema": schema, "schemas": domains.DatastreamSchemas{schema}}).Error)
			} else {
				schema := domains.ControlStreamSchema{CommandFormat: "application/json", ParametersSchema: component}
				require.NoError(t, tx.Model(cs).Updates(map[string]any{"schema": schema, "schemas": domains.ControlStreamSchemas{schema}}).Error)
			}
			require.NoError(t, tx.Commit().Error)
			select {
			case err := <-done:
				require.ErrorContains(t, err, "must be a string")
			case <-time.After(10 * time.Second):
				t.Fatal("child write did not finish after releasing schema lock")
			}
			var count int64
			column := "datastream_id"
			id := ds.ID
			if kind == "commands" {
				column = "control_stream_id"
				id = cs.ID
			}
			require.NoError(t, db.Table(kind).Where(column+" = ?", id).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestMetadataPUTRetainsLegacySchemaDefault(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db)
	jsonSchema := testNumericObservationSchema()
	ds := &domains.Datastream{Name: "Legacy default", Schema: jsonSchema}
	require.NoError(t, repos.Datastream.Create(t.Context(), ds))
	proto := domains.DatastreamSchema{ObsFormat: "application/x-protobuf"}
	registry := domains.DatastreamSchemas{proto, *jsonSchema}
	require.NoError(t, db.Model(ds).Updates(map[string]any{"schema": nil, "schemas": registry}).Error)
	before, _, err := repos.Datastream.GetSchema(t.Context(), ds.ID)
	require.NoError(t, err)
	require.Equal(t, "application/json", before.ObsFormat)
	require.NoError(t, repos.Datastream.Update(t.Context(), &domains.Datastream{Base: domains.Base{ID: ds.ID}, Name: ds.Name}))
	after, schemas, err := repos.Datastream.GetSchema(t.Context(), ds.ID)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Len(t, schemas, 2)
	cs := &domains.ControlStream{CommonSSN: domains.CommonSSN{UniqueIdentifier: "urn:test:legacy:default", Name: "Legacy control default"}, Schema: testNumericCommandSchema()}
	require.NoError(t, repos.ControlStream.Create(t.Context(), cs))
	controls := domains.ControlStreamSchemas{{CommandFormat: "application/swe+json", RecordSchema: cs.Schema.ParametersSchema}, *cs.Schema}
	require.NoError(t, db.Model(cs).Updates(map[string]any{"schema": nil, "schemas": controls}).Error)
	controlBefore, _, err := repos.ControlStream.GetSchema(t.Context(), cs.ID)
	require.NoError(t, err)
	require.NoError(t, repos.ControlStream.Update(t.Context(), &domains.ControlStream{Base: domains.Base{ID: cs.ID}, CommonSSN: cs.CommonSSN}))
	controlAfter, registered, err := repos.ControlStream.GetSchema(t.Context(), cs.ID)
	require.NoError(t, err)
	require.Equal(t, controlBefore, controlAfter)
	require.Len(t, registered, 2)
}

func TestSchemaEqualityPreservesExactNumbersAndArrayOrder(t *testing.T) {
	for _, test := range []struct {
		a, b  string
		equal bool
	}{
		{`{"constraint":{"values":[1e3,1.00]},"type":"Quantity"}`, `{"type":"Quantity","constraint":{"values":[1000,1]}}`, true},
		{`{"values":[9007199254740992]}`, `{"values":[9007199254740993]}`, false},
		{`{"values":[1,2]}`, `{"values":[2,1]}`, false},
		{`{"value":0}`, `{"value":-0.00e30}`, true},
		{`{"value":1}`, `{"value":"1e0"}`, false},
	} {
		require.Equal(t, test.equal, schemasEqual(json.RawMessage(test.a), json.RawMessage(test.b)), "%s versus %s", test.a, test.b)
	}
}
