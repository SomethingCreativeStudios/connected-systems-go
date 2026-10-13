package json_formatters

import (
	"context"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	"github.com/stretchr/testify/require"
)

// Formatting a resource must not rewrite the domain object it was given, and a
// single resource must come out exactly as it would inside a collection.
func TestSerialize_LeavesInputUntouchedAndMatchesSerializeAll(t *testing.T) {
	ctx := context.Background()

	t.Run("datastream", func(t *testing.T) {
		ds := &domains.Datastream{Name: "Temperature", ProcedureLink: &common_shared.Link{Href: "/procedures/p-1"}}
		formatter := NewDatastreamJSONFormatter(&formaters.LinkResolver{BaseURL: "http://example.test"})
		one, err := formatter.Serialize(ctx, ds)
		require.NoError(t, err)
		all, err := formatter.SerializeAll(ctx, []*domains.Datastream{ds})
		require.NoError(t, err)

		require.Equal(t, "/procedures/p-1", ds.ProcedureLink.Href)
		require.Equal(t, all[0], one)
		require.Equal(t, "http://example.test/procedures/p-1", one.ProcedureLink.Href)
	})

	t.Run("control stream", func(t *testing.T) {
		cs := &domains.ControlStream{DeploymentLink: &common_shared.Link{Href: "/deployments/d-1"}}
		formatter := NewControlStreamJSONFormatter(&formaters.LinkResolver{BaseURL: "http://example.test"})
		one, err := formatter.Serialize(ctx, cs)
		require.NoError(t, err)
		all, err := formatter.SerializeAll(ctx, []*domains.ControlStream{cs})
		require.NoError(t, err)

		require.Equal(t, "/deployments/d-1", cs.DeploymentLink.Href)
		require.Equal(t, all[0], one)
	})

	t.Run("command", func(t *testing.T) {
		cmd := &domains.Command{ProcedureLink: &common_shared.Link{Href: "/procedures/p-2"}}
		formatter := NewCommandJSONFormatter(&formaters.LinkResolver{BaseURL: "http://example.test"})
		one, err := formatter.Serialize(ctx, cmd)
		require.NoError(t, err)
		all, err := formatter.SerializeAll(ctx, []*domains.Command{cmd})
		require.NoError(t, err)

		require.Equal(t, "/procedures/p-2", cmd.ProcedureLink.Href)
		require.Equal(t, all[0], one)
	})

	t.Run("observation", func(t *testing.T) {
		obs := &domains.Observation{ResultLink: &common_shared.Link{Href: "/results/r-1"}}
		formatter := NewObservationJSONFormatter(&formaters.LinkResolver{BaseURL: "http://example.test"})
		one, err := formatter.Serialize(ctx, obs)
		require.NoError(t, err)
		all, err := formatter.SerializeAll(ctx, []*domains.Observation{obs})
		require.NoError(t, err)

		require.Equal(t, "/results/r-1", obs.ResultLink.Href)
		require.Equal(t, all[0], one)
	})
}
