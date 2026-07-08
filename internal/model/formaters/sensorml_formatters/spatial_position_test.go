package sensorml_formatters

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

func mustTestGoGeom(t *testing.T, raw string) *common_shared.GoGeom {
	t.Helper()
	var geom common_shared.GoGeom
	if err := json.Unmarshal([]byte(raw), &geom); err != nil {
		t.Fatalf("unmarshal geometry failed: %v", err)
	}
	return &geom
}

func requirePointJSON(t *testing.T, raw []byte, lon, lat float64) {
	t.Helper()
	var point map[string]interface{}
	if err := json.Unmarshal(raw, &point); err != nil {
		t.Fatalf("unmarshal point failed: %v", err)
	}
	if point["type"] != "Point" {
		t.Fatalf("expected Point, got %#v", point["type"])
	}
	coords, ok := point["coordinates"].([]interface{})
	if !ok || len(coords) != 2 {
		t.Fatalf("expected two point coordinates, got %#v", point["coordinates"])
	}
	if coords[0] != lon || coords[1] != lat {
		t.Fatalf("expected coordinates [%v %v], got %#v", lon, lat, coords)
	}
}

func requirePointGeom(t *testing.T, geom *common_shared.GoGeom, lon, lat float64) {
	t.Helper()
	if geom == nil {
		t.Fatal("expected geometry")
	}
	data, err := json.Marshal(geom)
	if err != nil {
		t.Fatalf("marshal geometry failed: %v", err)
	}
	requirePointJSON(t, data, lon, lat)
}

func TestSystemSensorMLSerialize_PositionDerivedFromGeometry(t *testing.T) {
	formatter := NewSystemSensorMLFormatter(nil)
	system := &domains.System{
		Base:     domains.Base{ID: "sys-1"},
		Geometry: mustTestGoGeom(t, `{"type":"Point","coordinates":[-117.1625,32.715]}`),
	}

	feature, err := formatter.Serialize(context.Background(), system)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	if len(feature.Position) == 0 {
		t.Fatal("expected SensorML position derived from geometry")
	}
	requirePointJSON(t, feature.Position, -117.1625, 32.715)
}

func TestSystemSensorMLSerialize_PositionPrefersExplicitSensorMLPosition(t *testing.T) {
	formatter := NewSystemSensorMLFormatter(nil)
	explicitPosition := json.RawMessage(`{"type":"DataRecord","label":"richer-position"}`)
	system := &domains.System{
		Base:     domains.Base{ID: "sys-1"},
		Geometry: mustTestGoGeom(t, `{"type":"Point","coordinates":[-117.1625,32.715]}`),
		Position: explicitPosition,
	}

	feature, err := formatter.Serialize(context.Background(), system)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	if string(feature.Position) != string(explicitPosition) {
		t.Fatalf("expected explicit position to be preserved, got %s", feature.Position)
	}
}

func TestSamplingFeatureSensorMLSerialize_PositionDerivedFromGeometry(t *testing.T) {
	formatter := NewSamplingFeatureSensorMLFormatter(nil)
	samplingFeature := &domains.SamplingFeature{
		Base:     domains.Base{ID: "sf-1"},
		Geometry: mustTestGoGeom(t, `{"type":"Point","coordinates":[-119,35]}`),
	}

	feature, err := formatter.Serialize(context.Background(), samplingFeature)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	requirePointGeom(t, feature.Position, -119, 35)
}

func TestSamplingFeatureSensorMLDeserialize_PositionBecomesGeometry(t *testing.T) {
	formatter := NewSamplingFeatureSensorMLFormatter(nil)
	payload := `{
		"type": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
		"label": "Sampling Feature",
		"uniqueId": "urn:sf:1",
		"definition": "http://www.opengis.net/def/samplingFeatureType/OGC-OM/2.0/SF_SamplingPoint",
		"sampledFeature": {"href": "http://example.org/features/test"},
		"position": {"type":"Point","coordinates":[-119,35]}
	}`

	samplingFeature, err := formatter.Deserialize(context.Background(), strings.NewReader(payload))
	if err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	requirePointGeom(t, samplingFeature.Geometry, -119, 35)
}
