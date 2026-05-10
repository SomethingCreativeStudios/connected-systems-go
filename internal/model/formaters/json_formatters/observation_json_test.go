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

func TestObservationJSONFormatter_Serialize_NormalizesRelativeHrefs(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewObservationJSONFormatter()

	obs := &domains.Observation{
		Base: domains.Base{ID: "obs-1"},
		ProcedureLink: &common_shared.Link{
			Href: "/procedures/proc-1",
		},
		ResultLink: &common_shared.Link{
			Href: "/results/result-1",
		},
	}

	result, err := formatter.Serialize(context.Background(), obs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProcedureLink == nil {
		t.Fatal("expected ProcedureLink to be non-nil")
	}
	if !strings.HasPrefix(result.ProcedureLink.Href, "http") {
		t.Errorf("expected absolute ProcedureLink href, got %q", result.ProcedureLink.Href)
	}

	if result.ResultLink == nil {
		t.Fatal("expected ResultLink to be non-nil")
	}
	if !strings.HasPrefix(result.ResultLink.Href, "http") {
		t.Errorf("expected absolute ResultLink href, got %q", result.ResultLink.Href)
	}
}

func TestObservationJSONFormatter_Serialize_PreservesAbsoluteHrefs(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewObservationJSONFormatter()

	obs := &domains.Observation{
		Base: domains.Base{ID: "obs-1"},
		ProcedureLink: &common_shared.Link{
			Href: "https://other.example/procedures/proc-1",
		},
		ResultLink: &common_shared.Link{
			Href: "https://other.example/results/result-1",
		},
	}

	result, err := formatter.Serialize(context.Background(), obs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProcedureLink.Href != "https://other.example/procedures/proc-1" {
		t.Errorf("expected unchanged ProcedureLink href, got %q", result.ProcedureLink.Href)
	}
	if result.ResultLink.Href != "https://other.example/results/result-1" {
		t.Errorf("expected unchanged ResultLink href, got %q", result.ResultLink.Href)
	}
}

func TestObservationJSONFormatter_Serialize_NilLinks(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewObservationJSONFormatter()

	obs := &domains.Observation{
		Base: domains.Base{ID: "obs-1"},
	}

	result, err := formatter.Serialize(context.Background(), obs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProcedureLink != nil {
		t.Error("expected ProcedureLink to be nil")
	}
	if result.ResultLink != nil {
		t.Error("expected ResultLink to be nil")
	}
}

func TestObservationJSONFormatter_Serialize_EmptyHrefs(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewObservationJSONFormatter()

	obs := &domains.Observation{
		Base: domains.Base{ID: "obs-1"},
		ProcedureLink: &common_shared.Link{
			Href: "",
		},
		ResultLink: &common_shared.Link{
			Href: "",
		},
	}

	result, err := formatter.Serialize(context.Background(), obs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProcedureLink.Href != "" {
		t.Errorf("expected empty ProcedureLink href to remain empty, got %q", result.ProcedureLink.Href)
	}
	if result.ResultLink.Href != "" {
		t.Errorf("expected empty ResultLink href to remain empty, got %q", result.ResultLink.Href)
	}
}

func TestObservationJSONFormatter_SerializeAll_NormalizesLinks(t *testing.T) {
	formaters.SetAssociationLinksBaseURL("http://example.test")
	t.Cleanup(func() { formaters.SetAssociationLinksBaseURL("") })

	formatter := NewObservationJSONFormatter()

	observations := []*domains.Observation{
		{
			Base: domains.Base{ID: "obs-1"},
			ProcedureLink: &common_shared.Link{
				Href: "/procedures/proc-1",
			},
			ResultLink: &common_shared.Link{
				Href: "/results/result-1",
			},
		},
		{
			Base: domains.Base{ID: "obs-2"},
			ProcedureLink: &common_shared.Link{
				Href: "/procedures/proc-2",
			},
		},
	}

	results, err := formatter.SerializeAll(context.Background(), observations)
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
			t.Errorf("result[%d]: expected absolute ProcedureLink href, got %q", i, r.ProcedureLink.Href)
		}
	}

	if results[0].ResultLink == nil {
		t.Error("result[0]: expected ResultLink to be non-nil")
	} else if !strings.HasPrefix(results[0].ResultLink.Href, "http") {
		t.Errorf("result[0]: expected absolute ResultLink href, got %q", results[0].ResultLink.Href)
	}

	if results[1].ResultLink != nil {
		t.Error("result[1]: expected ResultLink to be nil")
	}
}

func TestObservationJSONFormatter_Deserialize(t *testing.T) {
	formatter := NewObservationJSONFormatter()

	payload := map[string]interface{}{
		"id":              "obs-1",
		"datastream@id":   "ds-1",
		"resultTime":      "2026-01-01T00:00:00Z",
		"procedure@link": map[string]interface{}{
			"href": "/procedures/proc-1",
		},
		"result@link": map[string]interface{}{
			"href": "/results/result-1",
		},
		"result": map[string]interface{}{
			"temperature": 21.4,
		},
	}

	body, _ := json.Marshal(payload)
	reader := bytes.NewReader(body)

	result, err := formatter.Deserialize(context.Background(), reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "obs-1" {
		t.Errorf("expected ID 'obs-1', got %q", result.ID)
	}
	if result.DatastreamID != "ds-1" {
		t.Errorf("expected DatastreamID 'ds-1', got %q", result.DatastreamID)
	}
	if result.ProcedureLink == nil {
		t.Fatal("expected ProcedureLink to be non-nil")
	}
	if result.ProcedureLink.Href != "/procedures/proc-1" {
		t.Errorf("expected ProcedureLink href '/procedures/proc-1', got %q", result.ProcedureLink.Href)
	}
	if result.ResultLink == nil {
		t.Fatal("expected ResultLink to be non-nil")
	}
	if result.ResultLink.Href != "/results/result-1" {
		t.Errorf("expected ResultLink href '/results/result-1', got %q", result.ResultLink.Href)
	}
}
