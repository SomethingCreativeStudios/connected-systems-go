package json_formatters

import (
	"context"
	"strings"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

func TestDatastreamJSONFormatter_Serialize_NormalizesRelativeHrefs(t *testing.T) {

	formatter := NewDatastreamJSONFormatter(&formaters.LinkResolver{BaseURL: "http://example.test"})

	sysID := "sys-1"
	ds := &domains.Datastream{
		Base:     domains.Base{ID: "ds-1"},
		SystemID: &sysID,
		ProcedureLink: &common_shared.Link{
			Href: "/procedures/proc-1",
		},
		DeploymentLink: &common_shared.Link{
			Href: "/deployments/dep-1",
		},
		FeatureOfInterest: &common_shared.Link{
			Href: "/features/feat-1",
		},
		SamplingFeatureLink: &common_shared.Link{
			Href: "/samplingFeatures/sf-1",
		},
	}

	result, err := formatter.Serialize(context.Background(), ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SystemLink == nil {
		t.Fatal("expected SystemLink to be non-nil")
	}
	if !strings.HasPrefix(result.SystemLink.Href, "http") {
		t.Errorf("expected absolute SystemLink href, got %q", result.SystemLink.Href)
	}

	if result.ProcedureLink == nil {
		t.Fatal("expected ProcedureLink to be non-nil")
	}
	if !strings.HasPrefix(result.ProcedureLink.Href, "http") {
		t.Errorf("expected absolute ProcedureLink href, got %q", result.ProcedureLink.Href)
	}

	if result.DeploymentLink == nil {
		t.Fatal("expected DeploymentLink to be non-nil")
	}
	if !strings.HasPrefix(result.DeploymentLink.Href, "http") {
		t.Errorf("expected absolute DeploymentLink href, got %q", result.DeploymentLink.Href)
	}

	if result.FeatureOfInterest == nil {
		t.Fatal("expected FeatureOfInterest to be non-nil")
	}
	if !strings.HasPrefix(result.FeatureOfInterest.Href, "http") {
		t.Errorf("expected absolute FeatureOfInterest href, got %q", result.FeatureOfInterest.Href)
	}

	if result.SamplingFeatureLink == nil {
		t.Fatal("expected SamplingFeatureLink to be non-nil")
	}
	if !strings.HasPrefix(result.SamplingFeatureLink.Href, "http") {
		t.Errorf("expected absolute SamplingFeatureLink href, got %q", result.SamplingFeatureLink.Href)
	}
}

// Phase 1: inline @link Type enrichment
func TestDatastreamJSONFormatter_Serialize_SetsTypeOnAllInlineLinks(t *testing.T) {

	formatter := NewDatastreamJSONFormatter(&formaters.LinkResolver{BaseURL: "http://example.test"})

	sysID := "sys-1"
	ds := &domains.Datastream{
		Base:     domains.Base{ID: "ds-1"},
		SystemID: &sysID,
		ProcedureLink: &common_shared.Link{
			Href: "/procedures/proc-1",
		},
		DeploymentLink: &common_shared.Link{
			Href: "/deployments/dep-1",
		},
		FeatureOfInterest: &common_shared.Link{
			Href: "/features/feat-1",
		},
		SamplingFeatureLink: &common_shared.Link{
			Href: "/samplingFeatures/sf-1",
		},
	}

	result, err := formatter.Serialize(context.Background(), ds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SystemLink.Type != mediatype.GeoJSON {
		t.Errorf("expected SystemLink.Type %q, got %q", mediatype.GeoJSON, result.SystemLink.Type)
	}
	if result.ProcedureLink.Type != mediatype.SensorML {
		t.Errorf("expected ProcedureLink.Type %q, got %q", mediatype.SensorML, result.ProcedureLink.Type)
	}
	if result.DeploymentLink.Type != mediatype.GeoJSON {
		t.Errorf("expected DeploymentLink.Type %q, got %q", mediatype.GeoJSON, result.DeploymentLink.Type)
	}
	if result.FeatureOfInterest.Type != mediatype.GeoJSON {
		t.Errorf("expected FeatureOfInterest.Type %q, got %q", mediatype.GeoJSON, result.FeatureOfInterest.Type)
	}
	if result.SamplingFeatureLink.Type != mediatype.GeoJSON {
		t.Errorf("expected SamplingFeatureLink.Type %q, got %q", mediatype.GeoJSON, result.SamplingFeatureLink.Type)
	}
}

// Phase 2: inline @link Title/UID enrichment (nil repos — fields remain empty)
func TestDatastreamJSONFormatter_SerializeAll_EnrichesInlineLinks(t *testing.T) {

	formatter := NewDatastreamJSONFormatter(&formaters.LinkResolver{BaseURL: "http://example.test"})

	sysID := "sys-1"
	procID := "proc-1"
	depID := "dep-1"
	sfID := "sf-1"
	datastreams := []*domains.Datastream{
		{
			Base:     domains.Base{ID: "ds-1"},
			SystemID: &sysID,
			ProcedureLink: &common_shared.Link{
				Href: "/procedures/proc-1",
			},
			DeploymentLink: &common_shared.Link{
				Href: "/deployments/dep-1",
			},
			FeatureOfInterest: &common_shared.Link{
				Href: "/features/feat-1",
			},
			SamplingFeatureLink: &common_shared.Link{
				Href: "/samplingFeatures/sf-1",
			},
			ProcedureID:       &procID,
			DeploymentID:      &depID,
			SamplingFeatureID: &sfID,
		},
	}

	results, err := formatter.SerializeAll(context.Background(), datastreams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Type is always set (Phase 1)
	if results[0].SystemLink.Type != mediatype.GeoJSON {
		t.Errorf("expected SystemLink.Type %q, got %q", mediatype.GeoJSON, results[0].SystemLink.Type)
	}
	if results[0].ProcedureLink.Type != mediatype.SensorML {
		t.Errorf("expected ProcedureLink.Type %q, got %q", mediatype.SensorML, results[0].ProcedureLink.Type)
	}
	if results[0].DeploymentLink.Type != mediatype.GeoJSON {
		t.Errorf("expected DeploymentLink.Type %q, got %q", mediatype.GeoJSON, results[0].DeploymentLink.Type)
	}
	if results[0].SamplingFeatureLink.Type != mediatype.GeoJSON {
		t.Errorf("expected SamplingFeatureLink.Type %q, got %q", mediatype.GeoJSON, results[0].SamplingFeatureLink.Type)
	}

	// Title/UID are empty with nil repos (no cache population)
	if results[0].SystemLink.Title != "" {
		t.Errorf("expected empty SystemLink.Title with nil repos, got %q", results[0].SystemLink.Title)
	}
	if results[0].SystemLink.UID != nil {
		t.Errorf("expected nil SystemLink.UID with nil repos, got %v", results[0].SystemLink.UID)
	}
	if results[0].ProcedureLink.Title != "" {
		t.Errorf("expected empty ProcedureLink.Title with nil repos, got %q", results[0].ProcedureLink.Title)
	}
	if results[0].ProcedureLink.UID != nil {
		t.Errorf("expected nil ProcedureLink.UID with nil repos, got %v", results[0].ProcedureLink.UID)
	}
	if results[0].DeploymentLink.Title != "" {
		t.Errorf("expected empty DeploymentLink.Title with nil repos, got %q", results[0].DeploymentLink.Title)
	}
	if results[0].DeploymentLink.UID != nil {
		t.Errorf("expected nil DeploymentLink.UID with nil repos, got %v", results[0].DeploymentLink.UID)
	}
	if results[0].SamplingFeatureLink.Title != "" {
		t.Errorf("expected empty SamplingFeatureLink.Title with nil repos, got %q", results[0].SamplingFeatureLink.Title)
	}
	if results[0].SamplingFeatureLink.UID != nil {
		t.Errorf("expected nil SamplingFeatureLink.UID with nil repos, got %v", results[0].SamplingFeatureLink.UID)
	}
}
