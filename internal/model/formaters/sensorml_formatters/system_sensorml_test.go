package sensorml_formatters

import (
	"context"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

func TestSystemSensorMLSerialize_AssociationLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewSystemSensorMLFormatter(nil)
	parentID := "sys-parent"
	system := &domains.System{
		Base:           domains.Base{ID: "sys-1"},
		ParentSystemID: &parentID,
		Deployments:    []domains.Deployment{{Base: domains.Base{ID: "dep-1"}}},
		Procedures:     []domains.Procedure{{Base: domains.Base{ID: "proc-1"}}},
		Links: common_shared.Links{
			{Href: "/systems/sys-1/subsystems", Rel: common_shared.OGCRel("subsystems")},
			{Href: "/systems/sys-1/samplingFeatures", Rel: common_shared.OGCRel("samplingFeatures")},
			{Href: "/systems/sys-1/deployments", Rel: common_shared.OGCRel("deployments")},
			{Href: "/systems/sys-1/datastreams", Rel: common_shared.OGCRel("datastreams")},
			{Href: "/systems/sys-1/controlstreams", Rel: common_shared.OGCRel("controlstreams")},
			{Href: "/docs/spec", Rel: "alternate"},
		},
	}

	feature, err := formatter.Serialize(context.Background(), system)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	assertHasHref(t, feature.Links, common_shared.OGCRel("parentSystem"), "http://example.test/systems/sys-parent")
	assertHasRel(t, feature.Links, common_shared.OGCRel("subsystems"))
	assertHasRel(t, feature.Links, common_shared.OGCRel("samplingFeatures"))
	assertHasRel(t, feature.Links, common_shared.OGCRel("deployments"))
	assertHasRel(t, feature.Links, common_shared.OGCRel("datastreams"))
	assertHasRel(t, feature.Links, common_shared.OGCRel("controlstreams"))
	assertHasHref(t, feature.Links, common_shared.OGCRel("procedures"), "http://example.test/procedures?id=proc-1")
}

func TestSystemSensorMLDeserialize_AssociationHandling(t *testing.T) {
	formatter := NewSystemSensorMLFormatter(nil)
	payload := `{
		"id": "sys-1",
		"type": "PhysicalSystem",
		"label": "System 1",
		"uniqueId": "urn:system:1",
		"attachedTo": {"href": "/systems/sys-parent", "rel": "ogc-rel:attachedTo"},
		"links": [
			{"href": "/systems/sys-1/subsystems", "rel": "ogc-rel:subsystems"},
			{"href": "/docs/spec", "rel": "alternate"}
		]
	}`

	system, err := formatter.Deserialize(context.Background(), strings.NewReader(payload))
	if err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	if system.ParentSystemID == nil || *system.ParentSystemID != "sys-parent" {
		t.Fatalf("expected parent system id sys-parent, got %+v", system.ParentSystemID)
	}
	if len(system.Links) != 1 || system.Links[0].Rel != "alternate" {
		t.Fatalf("expected only non-association links to remain, got %+v", system.Links)
	}
}
