package geojson_formatters

import (
	"context"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
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
	assertHasRel(t, feature.Links, common_shared.OGCRel("samplingFeatures"))
	assertHasRel(t, feature.Links, common_shared.OGCRel("featuresOfInterest"))
	assertHasRel(t, feature.Links, common_shared.OGCRel("datastreams"))
	assertHasRel(t, feature.Links, common_shared.OGCRel("controlstreams"))
	assertMissingRel(t, feature.Links, common_shared.OGCRel("deployedSystems"))
}

func TestDeploymentGeoJSONDeserialize_AssociationLinks(t *testing.T) {
	formatter := NewDeploymentGeoJSONFormatter(nil)
	payload := `{
		"type": "Feature",
		"properties": {
			"uid": "urn:deployment:1",
			"name": "Deployment 1",
			"featureType": "http://www.w3.org/ns/sosa/Deployment"
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
