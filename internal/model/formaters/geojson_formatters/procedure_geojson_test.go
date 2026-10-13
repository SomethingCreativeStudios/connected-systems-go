package geojson_formatters

import (
	"context"
	"strings"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

func TestProcedureGeoJSONSerialize_AssociationLinks(t *testing.T) {

	formatter := NewProcedureGeoJSONFormatter(&formaters.LinkResolver{BaseURL: testAssociationBaseURL, ProcedureChildren: stubChildren{"proc-1": {"implementingSystems"}}})
	procedure := &domains.Procedure{
		Base: domains.Base{ID: "proc-1"},
		Links: common_shared.Links{
			{Href: "/docs/spec", Rel: "alternate"},
		},
	}

	feature, err := formatter.Serialize(context.Background(), procedure)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	assertHasHref(t, feature.Links, common_shared.OGCRel("implementingSystems"), "http://example.test/systems?procedure=proc-1&recursive=true")
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
			{"href": "/systems?procedure=proc-1&recursive=true", "rel": "ogc-rel:implementingSystems"},
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
