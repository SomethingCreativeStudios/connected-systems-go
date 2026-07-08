package geojson_formatters

import (
	"context"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
)

func TestSystemGeoJSONSerialize_AssociationLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewSystemGeoJSONFormatter(nil)
	parentID := "sys-parent"
	system := &domains.System{
		Base:           domains.Base{ID: "sys-1"},
		ParentSystemID: &parentID,
		Deployments:    []domains.Deployment{{Base: domains.Base{ID: "dep-1"}}},
		Procedures:     []domains.Procedure{{Base: domains.Base{ID: "proc-1"}}},
		Links: common_shared.Links{
			{Href: "/docs/spec", Rel: "alternate"},
			{Href: "/systems/sys-1/subsystems", Rel: common_shared.OGCRel("subsystems")},
			{Href: "/systems/sys-1/samplingFeatures", Rel: common_shared.OGCRel("samplingFeatures")},
			{Href: "/systems/sys-1/deployments", Rel: common_shared.OGCRel("deployments")},
			{Href: "/systems/sys-1/datastreams", Rel: common_shared.OGCRel("datastreams")},
			{Href: "/systems/sys-1/controlstreams", Rel: common_shared.OGCRel("controlstreams")},
			{Href: "/features?system=sys-1", Rel: common_shared.OGCRel("featuresOfInterest")},
		},
	}

	feature, err := formatter.Serialize(context.Background(), system)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	assertHasRel(t, feature.Links, common_shared.OGCRel("parentSystem"))
	assertHasRel(t, feature.Links, common_shared.OGCRel("subsystems"))
	assertHasRel(t, feature.Links, common_shared.OGCRel("samplingFeatures"))
	assertHasRel(t, feature.Links, common_shared.OGCRel("deployments"))
	assertHasRel(t, feature.Links, common_shared.OGCRel("datastreams"))
	assertHasRel(t, feature.Links, common_shared.OGCRel("controlstreams"))

	// For these ones i am not sure if this is the best link
	// TO-DO
	//assertHasHref(t, feature.Links, common_shared.OGCRel("procedures"), "http://example.test/procedures?id=proc-1")

	assertMissingRel(t, feature.Links, common_shared.OGCRel("featuresOfInterest"))
}

// Phase 1: inline @link Type enrichment
func TestSystemGeoJSONSerialize_SetsTypeOnSystemKindLink(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewSystemGeoJSONFormatter(nil)
	kindID := "proc-kind"
	system := &domains.System{
		Base:     domains.Base{ID: "sys-1"},
		TypeOfID: &kindID,
	}

	feature, err := formatter.Serialize(context.Background(), system)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	if feature.Properties.SystemKind == nil {
		t.Fatal("expected SystemKind to be non-nil")
	}
	if feature.Properties.SystemKind.Type != formaters.SensorMLContentType {
		t.Errorf("expected SystemKind.Type %q, got %q", formaters.SensorMLContentType, feature.Properties.SystemKind.Type)
	}
}

// Phase 2: inline @link Title/UID enrichment (nil repos — fields remain empty)
func TestSystemGeoJSONSerializeAll_EnrichesInlineLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewSystemGeoJSONFormatter(nil)
	kindID := "proc-kind"
	systems := []*domains.System{
		{
			Base:     domains.Base{ID: "sys-1"},
			TypeOfID: &kindID,
		},
	}

	features, err := formatter.SerializeAll(context.Background(), systems)
	if err != nil {
		t.Fatalf("serializeAll failed: %v", err)
	}

	if len(features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(features))
	}

	// Type is always set (Phase 1)
	if features[0].Properties.SystemKind == nil {
		t.Fatal("expected SystemKind to be non-nil")
	}
	if features[0].Properties.SystemKind.Type != formaters.SensorMLContentType {
		t.Errorf("expected SystemKind.Type %q, got %q", formaters.SensorMLContentType, features[0].Properties.SystemKind.Type)
	}

	// Title/UID are empty with nil repos (no cache population)
	if features[0].Properties.SystemKind.Title != "" {
		t.Errorf("expected empty Title with nil repos, got %q", features[0].Properties.SystemKind.Title)
	}
	if features[0].Properties.SystemKind.UID != nil {
		t.Errorf("expected nil UID with nil repos, got %v", features[0].Properties.SystemKind.UID)
	}
}
