package repository

import (
	"testing"
	"time"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"github.com/stretchr/testify/require"
)

// Association filters reach into other tables. They must neither fail on
// column names the joined tables share (id, name, ...) nor return a resource
// once per matching child.
func TestAssociationFilters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db)

	weatherStation := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:station", Name: "Weather station"},
		SystemType: domains.SystemTypeSensor,
	}
	require.NoError(t, repos.System.Create(t.Context(), weatherStation))
	other := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:other", Name: "Other station"},
		SystemType: domains.SystemTypeSensor,
	}
	require.NoError(t, repos.System.Create(t.Context(), other))

	lake := "lake-1"
	for i, uid := range []string{"urn:test:sf:north", "urn:test:sf:south"} {
		require.NoError(t, repos.SamplingFeature.Create(t.Context(), &domains.SamplingFeature{
			CommonSSN:        domains.CommonSSN{UniqueIdentifier: domains.UniqueID(uid), Name: []string{"North shore", "South shore"}[i]},
			ParentSystemID:   &weatherStation.ID,
			SampledFeatureID: &lake,
		}))
	}
	river := "river-1"
	require.NoError(t, repos.SamplingFeature.Create(t.Context(), &domains.SamplingFeature{
		CommonSSN:        domains.CommonSSN{UniqueIdentifier: "urn:test:sf:river", Name: "River gauge"},
		ParentSystemID:   &other.ID,
		SampledFeatureID: &river,
	}))

	temperature := domains.DatastreamObservedProperties{{Definition: "http://example.org/def/temperature"}}
	datastreamIDs := make([]string, 0, 2)
	for _, name := range []string{"Air temperature", "Water temperature"} {
		ds := &domains.Datastream{Name: name, SystemID: &weatherStation.ID, Schema: &domains.DatastreamSchema{ObsFormat: "application/json", ResultSchema: &domains.DatastreamDataComponent{Type: "Quantity", Definition: temperature[0].Definition, Label: temperature[0].Label, UOM: &domains.DatastreamUOM{Code: "Cel"}}}}
		require.NoError(t, repos.Datastream.Create(t.Context(), ds))
		datastreamIDs = append(datastreamIDs, ds.ID)
	}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 3 {
		require.NoError(t, repos.Observation.Create(t.Context(), &domains.Observation{
			DatastreamID: datastreamIDs[i%2],
			ResultTime:   start.Add(time.Duration(i) * time.Hour),
			Result:       []byte(`1`),
		}))
	}

	t.Run("sampling features by foi", func(t *testing.T) {
		params := &queryparams.SamplingFeatureQueryParams{QueryParams: queryparams.QueryParams{Limit: 10}, FOI: []string{lake}}
		features, total, err := repos.SamplingFeature.List(t.Context(), params)
		require.NoError(t, err)
		require.EqualValues(t, 2, total)
		require.Len(t, features, 2)
	})

	t.Run("sampling feature keywords match any keyword", func(t *testing.T) {
		params := &queryparams.SamplingFeatureQueryParams{QueryParams: queryparams.QueryParams{Limit: 10, Q: []string{"north", "river"}}}
		features, total, err := repos.SamplingFeature.List(t.Context(), params)
		require.NoError(t, err)
		require.EqualValues(t, 2, total)
		names := []string{features[0].Name, features[1].Name}
		require.ElementsMatch(t, []string{"North shore", "River gauge"}, names)
	})

	t.Run("systems by foi are returned once", func(t *testing.T) {
		params := &queryparams.SystemQueryParams{QueryParams: queryparams.QueryParams{Limit: 10}, FOI: []string{lake}}
		systems, total, err := repos.System.List(t.Context(), params)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		require.Len(t, systems, 1)
		require.Equal(t, weatherStation.ID, systems[0].ID)
	})

	t.Run("systems by foi combined with id and keyword filters", func(t *testing.T) {
		params := &queryparams.SystemQueryParams{
			QueryParams: queryparams.QueryParams{Limit: 10, IDs: []string{weatherStation.ID}, Q: []string{"weather"}},
			FOI:         []string{lake},
		}
		systems, total, err := repos.System.List(t.Context(), params)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		require.Len(t, systems, 1)
	})

	t.Run("systems by observed property are returned once", func(t *testing.T) {
		params := &queryparams.SystemQueryParams{
			QueryParams:      queryparams.QueryParams{Limit: 10},
			ObservedProperty: []string{"http://example.org/def/temperature"},
		}
		systems, total, err := repos.System.List(t.Context(), params)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		require.Len(t, systems, 1)
	})

	t.Run("observations by system page with a cursor", func(t *testing.T) {
		params := &queryparams.ObservationsQueryParams{
			QueryParams: queryparams.QueryParams{Limit: 2},
			System:      []string{weatherStation.ID},
		}
		observations, total, err := repos.Observation.List(t.Context(), params, nil)
		require.NoError(t, err)
		require.EqualValues(t, 3, total)
		require.Len(t, observations, 2)
		require.True(t, params.Page.HasNext)

		// The next page applies the cursor predicate, which compares the
		// observation id with the anchor.
		next := &queryparams.ObservationsQueryParams{
			QueryParams: queryparams.QueryParams{Limit: 2, Cursor: &queryparams.Cursor{
				Version: 1, Kind: queryparams.CursorKindTimeDesc, Direction: "after", Values: params.Anchors.Last,
			}},
			System: []string{weatherStation.ID},
		}
		rest, _, err := repos.Observation.List(t.Context(), next, nil)
		require.NoError(t, err)
		require.Len(t, rest, 1)
	})

	t.Run("observations by system and foi", func(t *testing.T) {
		params := &queryparams.ObservationsQueryParams{
			QueryParams: queryparams.QueryParams{Limit: 10, IDs: []string{}},
			System:      []string{weatherStation.ID},
			FOI:         []string{"none"},
		}
		observations, _, err := repos.Observation.List(t.Context(), params, nil)
		require.NoError(t, err)
		require.Empty(t, observations)
	})

}

// A resource without a validTime matches every datetime filter, and a missing
// start or end bound is open-ended.
func TestDatetimeFilter_ValidTimeSemantics(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db)

	at := func(year int) *time.Time {
		v := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		return &v
	}
	systems := map[string]*common_shared.TimeRange{
		"timeless": nil,
		"retired":  {Start: at(2020), End: at(2021)},
		"ongoing":  {Start: at(2025)},
		"legacy":   {End: at(2019)},
	}
	ids := map[string]string{}
	for name, validTime := range systems {
		system := &domains.System{
			CommonSSN:  domains.CommonSSN{UniqueIdentifier: domains.UniqueID("urn:test:datetime:" + name), Name: name},
			SystemType: domains.SystemTypeSensor,
			ValidTime:  validTime,
		}
		require.NoError(t, repos.System.Create(t.Context(), system))
		ids[system.ID] = name
	}

	cases := map[string]struct {
		datetime *common_shared.TimeRange
		want     []string
	}{
		"instant in the ongoing range": {&common_shared.TimeRange{Start: at(2026), End: at(2026)}, []string{"timeless", "ongoing"}},
		"range overlapping retired":    {&common_shared.TimeRange{Start: at(2018), End: at(2020)}, []string{"timeless", "retired", "legacy"}},
		"open end":                     {&common_shared.TimeRange{Start: at(2022)}, []string{"timeless", "ongoing"}},
		"open start":                   {&common_shared.TimeRange{End: at(2018)}, []string{"timeless", "legacy"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			params := &queryparams.SystemQueryParams{QueryParams: queryparams.QueryParams{Limit: 10}, Datetime: tc.datetime}
			found, total, err := repos.System.List(t.Context(), params)
			require.NoError(t, err)
			names := make([]string, 0, len(found))
			for _, system := range found {
				names = append(names, ids[system.ID])
			}
			require.ElementsMatch(t, tc.want, names)
			require.EqualValues(t, len(tc.want), total)
		})
	}
}
