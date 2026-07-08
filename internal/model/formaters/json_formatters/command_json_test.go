package json_formatters

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
)

func TestCommandJSONFormatter_Serialize_NormalizesRelativeHref(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewCommandJSONFormatter(nil)

	cmd := &domains.Command{
		Base: domains.Base{ID: "cmd-1"},
		ProcedureLink: &common_shared.Link{
			Href: "/procedures/proc-1",
		},
	}

	result, err := formatter.Serialize(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProcedureLink == nil {
		t.Fatal("expected ProcedureLink to be non-nil")
	}
	if !strings.HasPrefix(result.ProcedureLink.Href, "http") {
		t.Errorf("expected absolute href, got %q", result.ProcedureLink.Href)
	}
}

func TestCommandJSONFormatter_Serialize_PreservesAbsoluteHref(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewCommandJSONFormatter(nil)

	cmd := &domains.Command{
		Base: domains.Base{ID: "cmd-1"},
		ProcedureLink: &common_shared.Link{
			Href: "https://other.example/procedures/proc-1",
		},
	}

	result, err := formatter.Serialize(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProcedureLink == nil {
		t.Fatal("expected ProcedureLink to be non-nil")
	}
	if result.ProcedureLink.Href != "https://other.example/procedures/proc-1" {
		t.Errorf("expected unchanged absolute href, got %q", result.ProcedureLink.Href)
	}
}

func TestCommandJSONFormatter_Serialize_NilLink(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewCommandJSONFormatter(nil)

	cmd := &domains.Command{
		Base: domains.Base{ID: "cmd-1"},
	}

	result, err := formatter.Serialize(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProcedureLink != nil {
		t.Error("expected ProcedureLink to be nil")
	}
}

func TestCommandJSONFormatter_Serialize_EmptyHref(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewCommandJSONFormatter(nil)

	cmd := &domains.Command{
		Base: domains.Base{ID: "cmd-1"},
		ProcedureLink: &common_shared.Link{
			Href: "",
		},
	}

	result, err := formatter.Serialize(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProcedureLink == nil {
		t.Fatal("expected ProcedureLink to be non-nil")
	}
	if result.ProcedureLink.Href != "" {
		t.Errorf("expected empty href to remain empty, got %q", result.ProcedureLink.Href)
	}
}

func TestCommandJSONFormatter_SerializeAll_NormalizesLinks(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewCommandJSONFormatter(nil)

	commands := []*domains.Command{
		{
			Base: domains.Base{ID: "cmd-1"},
			ProcedureLink: &common_shared.Link{
				Href: "/procedures/proc-1",
			},
		},
		{
			Base: domains.Base{ID: "cmd-2"},
			ProcedureLink: &common_shared.Link{
				Href: "/procedures/proc-2",
			},
		},
	}

	results, err := formatter.SerializeAll(context.Background(), commands)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for i, r := range results {
		if r.ProcedureLink == nil {
			t.Errorf("result[%d]: expected ProcedureLink to be non-nil", i)
			continue
		}
		if !strings.HasPrefix(r.ProcedureLink.Href, "http") {
			t.Errorf("result[%d]: expected absolute href, got %q", i, r.ProcedureLink.Href)
		}
	}
}

func TestCommandJSONFormatter_Deserialize(t *testing.T) {
	formatter := NewCommandJSONFormatter(nil)

	payload := map[string]interface{}{
		"id":               "cmd-1",
		"controlstream@id": "cs-1",
		"procedure@link": map[string]interface{}{
			"href": "/procedures/proc-1",
		},
		"parameters": map[string]interface{}{
			"setPoint": 22.5,
		},
	}

	body, _ := json.Marshal(payload)
	reader := bytes.NewReader(body)

	result, err := formatter.Deserialize(context.Background(), reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "cmd-1" {
		t.Errorf("expected ID 'cmd-1', got %q", result.ID)
	}
	if result.ControlStreamID != "cs-1" {
		t.Errorf("expected ControlStreamID 'cs-1', got %q", result.ControlStreamID)
	}
	if result.ProcedureLink == nil {
		t.Fatal("expected ProcedureLink to be non-nil")
	}
	if result.ProcedureLink.Href != "/procedures/proc-1" {
		t.Errorf("expected href '/procedures/proc-1', got %q", result.ProcedureLink.Href)
	}
}

// Phase 1: inline @link Type enrichment
func TestCommandJSONFormatter_Serialize_SetsTypeOnProcedureLink(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewCommandJSONFormatter(nil)

	cmd := &domains.Command{
		Base: domains.Base{ID: "cmd-1"},
		ProcedureLink: &common_shared.Link{
			Href: "/procedures/proc-1",
		},
	}

	result, err := formatter.Serialize(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProcedureLink == nil {
		t.Fatal("expected ProcedureLink to be non-nil")
	}
	if result.ProcedureLink.Type != formaters.SensorMLContentType {
		t.Errorf("expected ProcedureLink.Type %q, got %q", formaters.SensorMLContentType, result.ProcedureLink.Type)
	}
}

// Phase 2: inline @link Title/UID enrichment (nil repos — fields remain empty)
func TestCommandJSONFormatter_SerializeAll_EnrichesInlineLinks(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewCommandJSONFormatter(nil)

	commands := []*domains.Command{
		{
			Base: domains.Base{ID: "cmd-1"},
			ProcedureLink: &common_shared.Link{
				Href: "/procedures/proc-1",
			},
		},
	}

	results, err := formatter.SerializeAll(context.Background(), commands)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Type is always set (Phase 1)
	if results[0].ProcedureLink.Type != formaters.SensorMLContentType {
		t.Errorf("expected ProcedureLink.Type %q, got %q", formaters.SensorMLContentType, results[0].ProcedureLink.Type)
	}

	// Title/UID are empty with nil repos (no cache population)
	if results[0].ProcedureLink.Title != "" {
		t.Errorf("expected empty Title with nil repos, got %q", results[0].ProcedureLink.Title)
	}
	if results[0].ProcedureLink.UID != nil {
		t.Errorf("expected nil UID with nil repos, got %v", results[0].ProcedureLink.UID)
	}
}
