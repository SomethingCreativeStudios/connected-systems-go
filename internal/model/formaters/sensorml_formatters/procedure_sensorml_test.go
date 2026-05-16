package sensorml_formatters

import (
	"context"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
)

func TestProcedureSensorMLSerialize_AssociationLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewProcedureSensorMLFormatter(nil)
	procedure := &domains.Procedure{
		Base:        domains.Base{ID: "proc-1"},
		ProcessType: "SimpleProcess",
		Systems: []domains.System{
			{Base: domains.Base{ID: "sys-1"}},
		},
		Links: common_shared.Links{
			{Href: "/docs/spec", Rel: "alternate"},
		},
	}

	feature, err := formatter.Serialize(context.Background(), procedure)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	assertHasHref(t, feature.Links, common_shared.OGCRel("implementingSystems"), "http://example.test/systems?procedure=proc-1")
	assertHasRel(t, feature.Links, "alternate")
}

func TestProcedureSensorMLDeserialize_StripsAssociationLinks(t *testing.T) {
	formatter := NewProcedureSensorMLFormatter(nil)
	payload := `{
		"id": "proc-1",
		"type": "SimpleProcess",
		"label": "Procedure 1",
		"uniqueId": "urn:procedure:1",
		"links": [
			{"href": "/systems?procedure=proc-1", "rel": "ogc-rel:implementingSystems"},
			{"href": "/docs/spec", "rel": "alternate"}
		]
	}`

	procedure, err := formatter.Deserialize(context.Background(), strings.NewReader(payload))
	if err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	if len(procedure.Links) != 1 || procedure.Links[0].Rel != "alternate" {
		t.Fatalf("expected only non-association links to remain, got %+v", procedure.Links)
	}
}

// Phase 1: inline @link Type enrichment
func TestProcedureSensorMLSerialize_SetsTypeOnTypeOfAndAttachedTo(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewProcedureSensorMLFormatter(nil)
	procedure := &domains.Procedure{
		Base:        domains.Base{ID: "proc-1"},
		ProcessType: "SimpleProcess",
		TypeOf: &common_shared.Link{
			Href: "/procedures/proc-parent",
		},
		AttachedTo: &common_shared.Link{
			Href: "/systems/sys-1",
		},
	}

	feature, err := formatter.Serialize(context.Background(), procedure)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	if feature.TypeOf == nil {
		t.Fatal("expected TypeOf to be non-nil")
	}
	if feature.TypeOf.Type != formaters.SensorMLContentType {
		t.Errorf("expected TypeOf.Type %q, got %q", formaters.SensorMLContentType, feature.TypeOf.Type)
	}

	if feature.AttachedTo == nil {
		t.Fatal("expected AttachedTo to be non-nil")
	}
	if feature.AttachedTo.Type != formaters.GeoJSONContentType {
		t.Errorf("expected AttachedTo.Type %q, got %q", formaters.GeoJSONContentType, feature.AttachedTo.Type)
	}
}

// Phase 2: inline @link Title/UID enrichment (nil repos — fields remain empty)
func TestProcedureSensorMLSerializeAll_EnrichesInlineLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewProcedureSensorMLFormatter(nil)
	procedures := []*domains.Procedure{
		{
			Base:        domains.Base{ID: "proc-1"},
			ProcessType: "SimpleProcess",
			TypeOf: &common_shared.Link{
				Href: "/procedures/proc-parent",
			},
			AttachedTo: &common_shared.Link{
				Href: "/systems/sys-1",
			},
		},
	}

	features, err := formatter.SerializeAll(context.Background(), procedures)
	if err != nil {
		t.Fatalf("serializeAll failed: %v", err)
	}

	if len(features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(features))
	}

	// Type is always set (Phase 1)
	if features[0].TypeOf.Type != formaters.SensorMLContentType {
		t.Errorf("expected TypeOf.Type %q, got %q", formaters.SensorMLContentType, features[0].TypeOf.Type)
	}
	if features[0].AttachedTo.Type != formaters.GeoJSONContentType {
		t.Errorf("expected AttachedTo.Type %q, got %q", formaters.GeoJSONContentType, features[0].AttachedTo.Type)
	}

	// Title/UID are empty with nil repos (no cache population)
	if features[0].TypeOf.Title != "" {
		t.Errorf("expected empty TypeOf.Title with nil repos, got %q", features[0].TypeOf.Title)
	}
	if features[0].TypeOf.UID != nil {
		t.Errorf("expected nil TypeOf.UID with nil repos, got %v", features[0].TypeOf.UID)
	}
	if features[0].AttachedTo.Title != "" {
		t.Errorf("expected empty AttachedTo.Title with nil repos, got %q", features[0].AttachedTo.Title)
	}
	if features[0].AttachedTo.UID != nil {
		t.Errorf("expected nil AttachedTo.UID with nil repos, got %v", features[0].AttachedTo.UID)
	}
}
