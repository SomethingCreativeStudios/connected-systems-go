package geojson_formatters

import (
	"context"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
)

func TestDeploymentGeoJSONSerialize_AssociationLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewDeploymentGeoJSONFormatter(nil)
	parentID := "dep-parent"
	deployment := &domains.Deployment{
		Base:               domains.Base{ID: "dep-1"},
		ParentDeploymentID: &parentID,
		Links: common_shared.Links{
			{Href: "/deployments/dep-1/subdeployments", Rel: common_shared.OGCRel("subdeployments")},
			{Href: "/samplingFeatures?deployment=dep-1", Rel: common_shared.OGCRel("samplingFeatures")},
			{Href: "/features?deployment=dep-1", Rel: common_shared.OGCRel("featuresOfInterest")},
			{Href: "/datastreams?deployment=dep-1", Rel: common_shared.OGCRel("datastreams")},
			{Href: "/controlStreams?deployment=dep-1", Rel: common_shared.OGCRel("controlstreams")},
			{Href: "/systems?id=s1,s2", Rel: common_shared.OGCRel("deployedSystems")},
			{Href: "/docs/spec", Rel: "alternate"},
		},
	}

	feature, err := formatter.Serialize(context.Background(), deployment)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	assertHasHref(t, feature.Links, common_shared.OGCRel("parentDeployment"), "http://example.test/deployments/dep-parent")
	assertHasHref(t, feature.Links, common_shared.OGCRel("subdeployments"), "http://example.test/deployments/dep-1/subdeployments")

	// For these ones i am not sure if this is the best link
	// TO-DO
	// assertHasRel(t, feature.Links, common_shared.OGCRel("samplingFeatures"))
	// assertHasRel(t, feature.Links, common_shared.OGCRel("featuresOfInterest"))
	// assertHasRel(t, feature.Links, common_shared.OGCRel("datastreams"))
	// assertHasRel(t, feature.Links, common_shared.OGCRel("controlstreams"))
	assertMissingRel(t, feature.Links, common_shared.OGCRel("deployedSystems"))
}

func TestDeploymentGeoJSONDeserialize_AssociationLinks(t *testing.T) {
	formatter := NewDeploymentGeoJSONFormatter(nil)
	payload := `{
		"type": "Feature",
		"properties": {
			"uid": "urn:deployment:1",
			"name": "Deployment 1",
			"featureType": "http://www.w3.org/ns/sosa/Deployment",
			"validTime": ["2026-01-01T00:00:00Z", "2026-12-31T23:59:59Z"]
		},
		"links": [
			{"href": "/deployments/dep-parent", "rel": "ogc-rel:parentDeployment"},
			{"href": "/docs/spec", "rel": "alternate"}
		]
	}`

	deployment, err := formatter.Deserialize(context.Background(), strings.NewReader(payload))
	if err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	if deployment.ParentDeploymentID == nil || *deployment.ParentDeploymentID != "dep-parent" {
		t.Fatalf("expected parent deployment id dep-parent, got %+v", deployment.ParentDeploymentID)
	}
	if len(deployment.Links) != 1 || deployment.Links[0].Rel != "alternate" {
		t.Fatalf("expected only non-association links to remain, got %+v", deployment.Links)
	}
}

// Phase 1: inline @link Type enrichment
func TestDeploymentGeoJSONSerialize_SetsTypeOnPlatformAndDeployedSystems(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewDeploymentGeoJSONFormatter(nil)
	deployment := &domains.Deployment{
		Base: domains.Base{ID: "dep-1"},
		Platform: &domains.DeployedSystemItem{
			System: common_shared.Link{Href: "/systems/sys-platform"},
		},
		DeployedSystems: []domains.DeployedSystemItem{
			{System: common_shared.Link{Href: "/systems/sys-1"}},
			{System: common_shared.Link{Href: "/systems/sys-2"}},
		},
	}

	feature, err := formatter.Serialize(context.Background(), deployment)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	if feature.Properties.Platform == nil {
		t.Fatal("expected Platform to be non-nil")
	}
	if feature.Properties.Platform.Type != formaters.GeoJSONContentType {
		t.Errorf("expected Platform.Type %q, got %q", formaters.GeoJSONContentType, feature.Properties.Platform.Type)
	}

	if len(feature.Properties.DeployedSystems) != 2 {
		t.Fatalf("expected 2 DeployedSystems, got %d", len(feature.Properties.DeployedSystems))
	}
	for i, ds := range feature.Properties.DeployedSystems {
		if ds.Type != formaters.GeoJSONContentType {
			t.Errorf("DeployedSystems[%d].Type: expected %q, got %q", i, formaters.GeoJSONContentType, ds.Type)
		}
	}
}

// Phase 2: inline @link Title/UID enrichment (nil repos — fields remain empty)
func TestDeploymentGeoJSONSerializeAll_EnrichesInlineLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewDeploymentGeoJSONFormatter(nil)
	deployments := []*domains.Deployment{
		{
			Base: domains.Base{ID: "dep-1"},
			Platform: &domains.DeployedSystemItem{
				System: common_shared.Link{Href: "/systems/sys-platform"},
			},
			DeployedSystems: []domains.DeployedSystemItem{
				{System: common_shared.Link{Href: "/systems/sys-1"}},
			},
		},
	}

	features, err := formatter.SerializeAll(context.Background(), deployments)
	if err != nil {
		t.Fatalf("serializeAll failed: %v", err)
	}

	if len(features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(features))
	}

	// Type is always set (Phase 1)
	if features[0].Properties.Platform.Type != formaters.GeoJSONContentType {
		t.Errorf("expected Platform.Type %q, got %q", formaters.GeoJSONContentType, features[0].Properties.Platform.Type)
	}

	// Title/UID are empty with nil repos (no cache population)
	if features[0].Properties.Platform.Title != "" {
		t.Errorf("expected empty Platform.Title with nil repos, got %q", features[0].Properties.Platform.Title)
	}
	if features[0].Properties.Platform.UID != nil {
		t.Errorf("expected nil Platform.UID with nil repos, got %v", features[0].Properties.Platform.UID)
	}
}
