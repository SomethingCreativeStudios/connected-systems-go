package common_shared

import (
	"errors"
	"testing"
)

type tcInner struct {
	Definition string `json:"definition,omitempty"`
	Label      string `json:"label,omitempty"`
}

type tcOuter struct {
	Name               string    `json:"name"`
	ObservedProperties []tcInner `json:"observedProperties,omitempty"`
}

func TestDecodeWithFieldErrors_TopLevelUnknownField(t *testing.T) {
	data := []byte(`{"name":"x","bogus":1}`)
	_, err := DecodeWithFieldErrors[tcOuter](data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ufErr *UnknownFieldError
	if !errors.As(err, &ufErr) {
		t.Fatalf("expected UnknownFieldError, got %T: %v", err, err)
	}
	if ufErr.Field != "bogus" {
		t.Fatalf("Field=%q, want %q", ufErr.Field, "bogus")
	}
	if ufErr.Path != "" {
		t.Fatalf("Path=%q, want empty", ufErr.Path)
	}
}

func TestDecodeWithFieldErrors_NestedSliceUnknownField(t *testing.T) {
	data := []byte(`{"name":"x","observedProperties":[{"definitions":"http://example.org/x","label":"X"}]}`)
	_, err := DecodeWithFieldErrors[tcOuter](data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ufErr *UnknownFieldError
	if !errors.As(err, &ufErr) {
		t.Fatalf("expected UnknownFieldError, got %T: %v", err, err)
	}
	if ufErr.Field != "definitions" {
		t.Fatalf("Field=%q, want %q", ufErr.Field, "definitions")
	}
	if ufErr.Path != "observedProperties[0]" {
		t.Fatalf("Path=%q, want %q", ufErr.Path, "observedProperties[0]")
	}
}

func TestDecodeWithFieldErrors_KnownFieldsPass(t *testing.T) {
	data := []byte(`{"name":"x","observedProperties":[{"definition":"http://example.org/x","label":"X"}]}`)
	out, err := DecodeWithFieldErrors[tcOuter](data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "x" || len(out.ObservedProperties) != 1 || out.ObservedProperties[0].Definition != "http://example.org/x" {
		t.Fatalf("decoded value mismatch: %+v", out)
	}
}

func TestDecodeWithFieldErrors_IOItemRawTolerated(t *testing.T) {
	// IOItem has a custom UnmarshalJSON that captures Raw. Strict mode should
	// not poke into its payload, since it's intentionally polymorphic.
	type wrapper struct {
		Inputs IOList `json:"inputs"`
	}
	data := []byte(`{"inputs":[{"type":"Quantity","label":"Temp","weirdvendorfield":42}]}`)
	if _, err := DecodeWithFieldErrors[wrapper](data); err != nil {
		t.Fatalf("expected IOItem to tolerate unknown nested field, got: %v", err)
	}
}

func TestDecodeWithFieldErrors_EmbeddedFieldRecognized(t *testing.T) {
	type Base struct {
		ID string `json:"id,omitempty"`
	}
	type Wire struct {
		Base
		Name string `json:"name"`
	}
	if _, err := DecodeWithFieldErrors[Wire]([]byte(`{"id":"abc","name":"x"}`)); err != nil {
		t.Fatalf("embedded id should be recognized, got: %v", err)
	}
	_, err := DecodeWithFieldErrors[Wire]([]byte(`{"id":"abc","name":"x","what":1}`))
	var ufErr *UnknownFieldError
	if !errors.As(err, &ufErr) || ufErr.Field != "what" {
		t.Fatalf("expected UnknownFieldError on 'what', got: %v", err)
	}
}

func TestDecodeWithFieldErrors_TimeRangeFieldErrorPrecedence(t *testing.T) {
	type Wire struct {
		ValidTime TimeRange `json:"validTime"`
	}
	// Bad time range value — should produce field-level TimeRange error,
	// not an UnknownFieldError.
	data := []byte(`{"validTime":["not-a-date","also-bad"]}`)
	_, err := DecodeWithFieldErrors[Wire](data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ufErr *UnknownFieldError
	if errors.As(err, &ufErr) {
		t.Fatalf("did not expect UnknownFieldError; got %v", err)
	}
}
