package sensorml_formatters

import (
	"context"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	"github.com/stretchr/testify/require"
)

// TestProcedureSensorMLFormatter_SerializeAll_EnrichesInlineLinksFromDB verifies that
// SerializeAll populates Title and UID on typeOf and attachedTo.
func TestProcedureSensorMLFormatter_SerializeAll_EnrichesInlineLinksFromDB(t *testing.T) {

	repos, cleanup := setupFormatterTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Seed parent procedure (typeOf target)
	parentProc := &domains.Procedure{
		CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:proc:parent", Name: "Parent Procedure"},
		ProcessType: "SimpleProcess",
	}
	require.NoError(t, repos.Procedure.Create(t.Context(), parentProc))

	// Seed attached system (attachedTo target)
	attachedSys := &domains.System{
		CommonSSN:  domains.CommonSSN{UniqueIdentifier: "urn:test:sys:attached", Name: "Attached System"},
		SystemType: domains.SystemTypePlatform,
	}
	require.NoError(t, repos.System.Create(t.Context(), attachedSys))

	// Create procedure with typeOf and attachedTo
	procedure := &domains.Procedure{
		Base:        domains.Base{ID: "proc-1"},
		CommonSSN:   domains.CommonSSN{UniqueIdentifier: "urn:test:proc:1", Name: "Test Procedure"},
		ProcessType: "SimpleProcess",
		TypeOf: &common_shared.Link{
			Href: "/procedures/" + parentProc.ID,
		},
		AttachedTo: &common_shared.Link{
			Href: "/systems/" + attachedSys.ID,
		},
	}

	formatter := NewProcedureSensorMLFormatter(&formaters.LinkResolver{BaseURL: "http://example.test", Systems: repos.System, Procedures: repos.Procedure, Deployments: repos.Deployment, SamplingFeatures: repos.SamplingFeature, SystemChildren: repos.System, DeploymentChildren: repos.Deployment, ProcedureChildren: repos.Procedure})
	features, err := formatter.SerializeAll(ctx, []*domains.Procedure{procedure})
	require.NoError(t, err)
	require.Len(t, features, 1)

	f := features[0]

	// typeOf enrichment
	require.NotNil(t, f.TypeOf)
	require.Equal(t, mediatype.SensorML, f.TypeOf.Type)
	require.Equal(t, "Parent Procedure", f.TypeOf.Title)
	require.NotNil(t, f.TypeOf.UID)
	require.Equal(t, "urn:test:proc:parent", *f.TypeOf.UID)

	// attachedTo enrichment
	require.NotNil(t, f.AttachedTo)
	require.Equal(t, mediatype.GeoJSON, f.AttachedTo.Type)
	require.Equal(t, "Attached System", f.AttachedTo.Title)
	require.NotNil(t, f.AttachedTo.UID)
	require.Equal(t, "urn:test:sys:attached", *f.AttachedTo.UID)
}
