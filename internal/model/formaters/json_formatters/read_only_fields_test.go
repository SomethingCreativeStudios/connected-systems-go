package json_formatters

import (
	"context"
	"strings"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/stretchr/testify/require"
)

// A fetched datastream can be sent back as is: the read-only fields and the
// server-generated association links it carries are accepted but not kept,
// so a GET/PUT round trip neither stores them nor duplicates the links.
func TestDatastreamDeserialize_IgnoresReadOnlyFields(t *testing.T) {
	body := `{
		"id": "client-chosen",
		"name": "Air temperature",
		"system@link": {"href": "http://example.test/systems/other"},
		"formats": ["application/x-made-up"],
		"observedProperties": [{"definition": "http://example.test/made-up"}],
		"phenomenonTime": ["1999-01-01T00:00:00Z", "1999-01-02T00:00:00Z"],
		"resultTime": ["1999-01-01T00:00:00Z", "1999-01-02T00:00:00Z"],
		"resultType": "coverage",
		"schema": {"obsFormat": "application/json", "resultSchema": {"type": "Quantity", "definition": "http://example.test/temperature"}},
		"links": [
			{"rel": "ogc-rel:observations", "href": "http://example.test/datastreams/x/observations"},
			{"rel": "alternate", "href": "http://example.test/elsewhere"}
		]
	}`
	datastream, err := NewDatastreamJSONFormatter(nil).Deserialize(context.Background(), strings.NewReader(body))
	require.NoError(t, err)

	require.Empty(t, datastream.ID)
	require.Nil(t, datastream.SystemID)
	require.Nil(t, datastream.PhenomenonTime)
	require.Nil(t, datastream.ResultTime)
	require.Equal(t, common_shared.StringArray{"application/json"}, datastream.Formats)
	require.Equal(t, "http://example.test/temperature", (*datastream.ObservedProperties)[0].Definition)
	require.Equal(t, "measure", *datastream.ResultType)
	require.Equal(t, common_shared.Links{{Rel: "alternate", Href: "http://example.test/elsewhere"}}, datastream.Links)
}

func TestCommandDeserialize_IgnoresServerOwnedFields(t *testing.T) {
	body := `{
		"id": "client-chosen",
		"controlstream@id": "other",
		"issueTime": "2026-01-01T00:00:00Z",
		"executionTime": ["1999-01-01T00:00:00Z", "1999-01-02T00:00:00Z"],
		"sender": "someone-else",
		"currentStatus": "COMPLETED",
		"parameters": {"setPoint": 20}
	}`
	command, err := NewCommandJSONFormatter(nil).Deserialize(context.Background(), strings.NewReader(body))
	require.NoError(t, err)

	require.Empty(t, command.ID)
	require.Empty(t, command.ControlStreamID)
	require.Empty(t, command.Sender)
	require.Empty(t, command.CurrentStatus)
	require.Nil(t, command.ExecutionTime)
	require.NotNil(t, command.IssueTime, "issueTime is honored on create")
}
