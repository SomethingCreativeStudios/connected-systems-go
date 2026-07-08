package json_formatters

import (
	"context"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
)

func TestControlStreamJSONFormatter_Serialize_NormalizesRelativeHrefs(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewControlStreamJSONFormatter(nil)

	sysID := "sys-1"
	cs := &domains.ControlStream{
		Base:     domains.Base{ID: "cs-1"},
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

	result, err := formatter.Serialize(context.Background(), cs)
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
func TestControlStreamJSONFormatter_Serialize_SetsTypeOnAllInlineLinks(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewControlStreamJSONFormatter(nil)

	sysID := "sys-1"
	cs := &domains.ControlStream{
		Base:     domains.Base{ID: "cs-1"},
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

	result, err := formatter.Serialize(context.Background(), cs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SystemLink.Type != formaters.GeoJSONContentType {
		t.Errorf("expected SystemLink.Type %q, got %q", formaters.GeoJSONContentType, result.SystemLink.Type)
	}
	if result.ProcedureLink.Type != formaters.SensorMLContentType {
		t.Errorf("expected ProcedureLink.Type %q, got %q", formaters.SensorMLContentType, result.ProcedureLink.Type)
	}
	if result.DeploymentLink.Type != formaters.GeoJSONContentType {
		t.Errorf("expected DeploymentLink.Type %q, got %q", formaters.GeoJSONContentType, result.DeploymentLink.Type)
	}
	if result.FeatureOfInterest.Type != formaters.GeoJSONContentType {
		t.Errorf("expected FeatureOfInterest.Type %q, got %q", formaters.GeoJSONContentType, result.FeatureOfInterest.Type)
	}
	if result.SamplingFeatureLink.Type != formaters.GeoJSONContentType {
		t.Errorf("expected SamplingFeatureLink.Type %q, got %q", formaters.GeoJSONContentType, result.SamplingFeatureLink.Type)
	}
}

// Phase 2: inline @link Title/UID enrichment (nil repos — fields remain empty)
func TestControlStreamJSONFormatter_SerializeAll_EnrichesInlineLinks(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewControlStreamJSONFormatter(nil)

	sysID := "sys-1"
	procID := "proc-1"
	depID := "dep-1"
	sfID := "sf-1"
	controlStreams := []*domains.ControlStream{
		{
			Base:     domains.Base{ID: "cs-1"},
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

	results, err := formatter.SerializeAll(context.Background(), controlStreams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Type is always set (Phase 1)
	if results[0].SystemLink.Type != formaters.GeoJSONContentType {
		t.Errorf("expected SystemLink.Type %q, got %q", formaters.GeoJSONContentType, results[0].SystemLink.Type)
	}
	if results[0].ProcedureLink.Type != formaters.SensorMLContentType {
		t.Errorf("expected ProcedureLink.Type %q, got %q", formaters.SensorMLContentType, results[0].ProcedureLink.Type)
	}
	if results[0].DeploymentLink.Type != formaters.GeoJSONContentType {
		t.Errorf("expected DeploymentLink.Type %q, got %q", formaters.GeoJSONContentType, results[0].DeploymentLink.Type)
	}
	if results[0].SamplingFeatureLink.Type != formaters.GeoJSONContentType {
		t.Errorf("expected SamplingFeatureLink.Type %q, got %q", formaters.GeoJSONContentType, results[0].SamplingFeatureLink.Type)
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
