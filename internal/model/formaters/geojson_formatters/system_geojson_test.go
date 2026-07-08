package geojson_formatters

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
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

func TestSystemGeoJSONSerialize_EmptyValidTimeOmitted(t *testing.T) {
	formatter := NewSystemGeoJSONFormatter(nil)
	system := &domains.System{
		Base:      domains.Base{ID: "sys-1"},
		ValidTime: &common_shared.TimeRange{},
	}

	feature, err := formatter.Serialize(context.Background(), system)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	if feature.Properties.ValidTime != nil {
		t.Fatalf("expected empty validTime to be nil, got %#v", feature.Properties.ValidTime)
	}

	data, err := json.Marshal(feature)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(data), "validTime") {
		t.Fatalf("expected validTime to be omitted, got %s", data)
	}
}
