package sensorml_formatters

import (
	"context"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
)

func TestDeploymentSensorMLSerialize_AssociationLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewDeploymentSensorMLFormatter(nil)
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

func TestDeploymentSensorMLDeserialize_AssociationLinks(t *testing.T) {
	formatter := NewDeploymentSensorMLFormatter(nil)
	payload := `{
		"id": "dep-1",
		"type": "Deployment",
		"label": "Deployment 1",
		"uniqueId": "urn:deployment:1",
		"definition": "http://www.w3.org/ns/sosa/Deployment",
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
func TestDeploymentSensorMLSerialize_SetsTypeOnPlatformAndDeployedSystems(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewDeploymentSensorMLFormatter(nil)
	deployment := &domains.Deployment{
		Base: domains.Base{ID: "dep-1"},
		Platform: &domains.DeployedSystemItem{
			System: common_shared.Link{Href: "/systems/sys-platform"},
		},
		DeployedSystems: []domains.DeployedSystemItem{
			{System: common_shared.Link{Href: "/systems/sys-1"}},
		},
	}

	feature, err := formatter.Serialize(context.Background(), deployment)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	if feature.Platform == nil {
		t.Fatal("expected Platform to be non-nil")
	}
	if feature.Platform.System.Type != formaters.GeoJSONContentType {
		t.Errorf("expected Platform.System.Type %q, got %q", formaters.GeoJSONContentType, feature.Platform.System.Type)
	}

	if len(feature.DeployedSystems) != 1 {
		t.Fatalf("expected 1 DeployedSystem, got %d", len(feature.DeployedSystems))
	}
	if feature.DeployedSystems[0].System.Type != formaters.GeoJSONContentType {
		t.Errorf("expected DeployedSystems[0].System.Type %q, got %q", formaters.GeoJSONContentType, feature.DeployedSystems[0].System.Type)
	}
}

// Phase 2: inline @link Title/UID enrichment (nil repos — fields remain empty)
func TestDeploymentSensorMLSerializeAll_EnrichesInlineLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewDeploymentSensorMLFormatter(nil)
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
	if features[0].Platform.System.Type != formaters.GeoJSONContentType {
		t.Errorf("expected Platform.System.Type %q, got %q", formaters.GeoJSONContentType, features[0].Platform.System.Type)
	}

	// Title/UID are empty with nil repos (no cache population)
	if features[0].Platform.System.Title != "" {
		t.Errorf("expected empty Platform.System.Title with nil repos, got %q", features[0].Platform.System.Title)
	}
	if features[0].Platform.System.UID != nil {
		t.Errorf("expected nil Platform.System.UID with nil repos, got %v", features[0].Platform.System.UID)
	}
}
