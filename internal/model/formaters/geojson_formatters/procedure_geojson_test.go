package geojson_formatters

import (
	"context"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

func TestProcedureGeoJSONSerialize_AssociationLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewProcedureGeoJSONFormatter(nil)
	procedure := &domains.Procedure{
		Base: domains.Base{ID: "proc-1"},
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

func TestProcedureGeoJSONDeserialize_StripsAssociationLinks(t *testing.T) {
	formatter := NewProcedureGeoJSONFormatter(nil)
	payload := `{
		"type": "Feature",
		"properties": {
			"uid": "urn:procedure:1",
			"name": "Procedure 1",
			"featureType": "http://www.w3.org/ns/sosa/Procedure"
		},
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
