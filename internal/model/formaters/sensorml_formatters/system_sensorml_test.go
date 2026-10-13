package sensorml_formatters

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	"github.com/stretchr/testify/require"
)

func TestSystemSensorMLSerialize_AssociationLinks(t *testing.T) {

	formatter := NewSystemSensorMLFormatter(&formaters.LinkResolver{BaseURL: testAssociationBaseURL, SystemChildren: stubChildren{"sys-1": {"subsystems", "samplingFeatures", "datastreams", "controlstreams"}}})
	parentID := "sys-parent"
	system := &domains.System{
		Base:           domains.Base{ID: "sys-1"},
		ParentSystemID: &parentID,
		Deployments:    []domains.Deployment{{Base: domains.Base{ID: "dep-1"}}},
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

	// For these ones i am not sure if this is the best link
	// TO-DO
	//assertHasHref(t, feature.Links, common_shared.OGCRel("procedures"), "http://example.test/procedures?id=proc-1")
}

func TestSystemSensorMLSerialize_EmptyValidTimeOmitted(t *testing.T) {
	formatter := NewSystemSensorMLFormatter(nil)
	system := &domains.System{
		Base:      domains.Base{ID: "sys-1"},
		ValidTime: &common_shared.TimeRange{},
	}

	feature, err := formatter.Serialize(context.Background(), system)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	if feature.ValidTime != nil {
		t.Fatalf("expected empty validTime to be nil, got %#v", feature.ValidTime)
	}

	data, err := json.Marshal(feature)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(data), "validTime") {
		t.Fatalf("expected validTime to be omitted, got %s", data)
	}
}

func TestSystemSensorMLDeserialize_AssociationHandling(t *testing.T) {
	formatter := NewSystemSensorMLFormatter(nil)
	payload := `{
		"id": "sys-1",
		"type": "PhysicalSystem",
		"label": "System 1",
		"uniqueId": "urn:system:1",
		"definition": "http://www.w3.org/ns/sosa/Sensor",
		"attachedTo": {"href": "/systems/sys-parent", "rel": "ogc-rel:attachedTo"},
		"links": [
			{"href": "/systems/sys-other", "rel": "ogc-rel:parentSystem"},
			{"href": "/systems/sys-1/subsystems", "rel": "ogc-rel:subsystems"},
			{"href": "/docs/spec", "rel": "alternate"}
		]
	}`

	system, err := formatter.Deserialize(context.Background(), strings.NewReader(payload))
	if err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	// attachedTo and ogc-rel links are server-generated: the parent system is
	// only set through /systems/{id}/subsystems, never from the request body.
	if system.ParentSystemID != nil {
		t.Fatalf("expected attachedTo and parentSystem link to be ignored, got %+v", *system.ParentSystemID)
	}
	if len(system.Links) != 1 || system.Links[0].Rel != "alternate" {
		t.Fatalf("expected only non-association links to remain, got %+v", system.Links)
	}
}

// Phase 1: inline @link Type enrichment
func TestSystemSensorMLSerialize_SetsTypeOnAttachedToAndTypeOf(t *testing.T) {

	formatter := NewSystemSensorMLFormatter(&formaters.LinkResolver{BaseURL: testAssociationBaseURL})
	parentID := "sys-parent"
	kindID := "proc-kind"
	system := &domains.System{
		Base:           domains.Base{ID: "sys-1"},
		ParentSystemID: &parentID,
		TypeOfID:       &kindID,
	}

	feature, err := formatter.Serialize(context.Background(), system)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	if feature.AttachedTo == nil {
		t.Fatal("expected AttachedTo to be non-nil")
	}
	if feature.AttachedTo.Type != mediatype.GeoJSON {
		t.Errorf("expected AttachedTo.Type %q, got %q", mediatype.GeoJSON, feature.AttachedTo.Type)
	}

	if feature.TypeOf == nil {
		t.Fatal("expected TypeOf to be non-nil")
	}
	if feature.TypeOf.Type != mediatype.SensorML {
		t.Errorf("expected TypeOf.Type %q, got %q", mediatype.SensorML, feature.TypeOf.Type)
	}
}

// Phase 2: inline @link Title/UID enrichment (nil repos — fields remain empty)
func TestSystemSensorMLSerializeAll_EnrichesInlineLinks(t *testing.T) {

	formatter := NewSystemSensorMLFormatter(&formaters.LinkResolver{BaseURL: testAssociationBaseURL})
	parentID := "sys-parent"
	kindID := "proc-kind"
	systems := []*domains.System{
		{
			Base:           domains.Base{ID: "sys-1"},
			ParentSystemID: &parentID,
			TypeOfID:       &kindID,
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
	if features[0].AttachedTo.Type != mediatype.GeoJSON {
		t.Errorf("expected AttachedTo.Type %q, got %q", mediatype.GeoJSON, features[0].AttachedTo.Type)
	}
	if features[0].TypeOf.Type != mediatype.SensorML {
		t.Errorf("expected TypeOf.Type %q, got %q", mediatype.SensorML, features[0].TypeOf.Type)
	}

	// Title/UID are empty with nil repos (no cache population)
	if features[0].AttachedTo.Title != "" {
		t.Errorf("expected empty AttachedTo.Title with nil repos, got %q", features[0].AttachedTo.Title)
	}
	if features[0].AttachedTo.UID != nil {
		t.Errorf("expected nil AttachedTo.UID with nil repos, got %v", features[0].AttachedTo.UID)
	}
	if features[0].TypeOf.Title != "" {
		t.Errorf("expected empty TypeOf.Title with nil repos, got %q", features[0].TypeOf.Title)
	}
	if features[0].TypeOf.UID != nil {
		t.Errorf("expected nil TypeOf.UID with nil repos, got %v", features[0].TypeOf.UID)
	}
}

// A GeoJSON position becomes the system geometry, with its altitude; other
// SensorML position encodings are kept as raw position only, and a malformed
// GeoJSON position is a client error rather than a silently missing geometry.
func TestSystemSensorMLDeserialize_Position(t *testing.T) {
	decode := func(position string) (*domains.System, error) {
		body := `{"type": "PhysicalSystem", "uniqueId": "urn:test:pos", "label": "Pos",
			"definition": "http://www.w3.org/ns/sosa/Sensor", "position": ` + position + `}`
		return NewSystemSensorMLFormatter(nil).Deserialize(context.Background(), strings.NewReader(body))
	}

	system, err := decode(`{"type": "Point", "coordinates": [-117.1, 32.7, 15]}`)
	require.NoError(t, err)
	require.Equal(t, []float64{-117.1, 32.7, 15}, system.Geometry.T.FlatCoords())

	system, err = decode(`{"type": "Vector", "referenceFrame": "http://www.opengis.net/def/crs/EPSG/0/4979", "coordinates": []}`)
	require.NoError(t, err)
	require.Nil(t, system.Geometry)
	require.NotEmpty(t, system.Position)

	_, err = decode(`{"type": "Point", "coordinates": [1]}`)
	require.ErrorContains(t, err, "position: invalid geometry")
}
