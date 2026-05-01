package api

import (
	"encoding/json"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

func boolPtr(b bool) *bool { return &b }

func quantityComponent(optional bool, nilValues ...domains.DatastreamNilValue) *domains.DatastreamDataComponent {
	return &domains.DatastreamDataComponent{
		Type:      "Quantity",
		Optional:  boolPtr(optional),
		NilValues: nilValues,
	}
}

func recordComponent(fields ...domains.DatastreamNamedComponent) *domains.DatastreamDataComponent {
	return &domains.DatastreamDataComponent{
		Fields: fields,
	}
}

func namedField(name string, c domains.DatastreamDataComponent) domains.DatastreamNamedComponent {
	return domains.DatastreamNamedComponent{Name: name, DatastreamDataComponent: c}
}

// ── Optional null handling (Issue #22) ──────────────────────────────────────

func TestValidateDataComponent_OptionalFieldNull(t *testing.T) {
	comp := recordComponent(
		namedField("temp", domains.DatastreamDataComponent{Type: "Quantity", Optional: boolPtr(true)}),
	)
	value := map[string]any{"temp": nil}
	if err := validateDataComponentValue(comp, value, "result"); err != nil {
		t.Fatalf("expected nil for optional null field, got: %v", err)
	}
}

func TestValidateDataComponent_OptionalFieldMissing(t *testing.T) {
	comp := recordComponent(
		namedField("temp", domains.DatastreamDataComponent{Type: "Quantity", Optional: boolPtr(true)}),
	)
	value := map[string]any{}
	if err := validateDataComponentValue(comp, value, "result"); err != nil {
		t.Fatalf("expected nil for missing optional field, got: %v", err)
	}
}

func TestValidateDataComponent_RequiredFieldNull(t *testing.T) {
	comp := recordComponent(
		namedField("temp", domains.DatastreamDataComponent{Type: "Quantity", Optional: boolPtr(false)}),
	)
	value := map[string]any{"temp": nil}
	if err := validateDataComponentValue(comp, value, "result"); err == nil {
		t.Fatal("expected error for null required field, got nil")
	}
}

func TestValidateDataComponent_RequiredFieldPresent(t *testing.T) {
	comp := recordComponent(
		namedField("temp", domains.DatastreamDataComponent{Type: "Quantity", Optional: boolPtr(false)}),
	)
	value := map[string]any{"temp": float64(21.5)}
	if err := validateDataComponentValue(comp, value, "result"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── nilValues (Issue #21) ────────────────────────────────────────────────────

func nilValue(v string) domains.DatastreamNilValue {
	b, _ := json.Marshal(v)
	return domains.DatastreamNilValue{Reason: "missing", Value: b}
}

func TestValidateDataComponent_NilValueMatch(t *testing.T) {
	comp := quantityComponent(false, nilValue("NaN"))
	// "NaN" is declared as a nilValue — should be accepted for a quantity field
	if err := validateDataComponentValue(comp, "NaN", "result"); err != nil {
		t.Fatalf("expected nil for declared nilValue, got: %v", err)
	}
}

func TestValidateDataComponent_NilValueNoMatch(t *testing.T) {
	comp := quantityComponent(false, nilValue("NaN"))
	// "N/A" is NOT in nilValues — should still fail type check for quantity
	if err := validateDataComponentValue(comp, "N/A", "result"); err == nil {
		t.Fatal("expected error for undeclared nil sentinel, got nil")
	}
}

func TestValidateDataComponent_NilValueMatchInRecord(t *testing.T) {
	nv := nilValue("NaN")
	comp := recordComponent(
		namedField("altitude", domains.DatastreamDataComponent{Type: "Quantity", Optional: boolPtr(false), NilValues: []domains.DatastreamNilValue{nv}}),
	)
	value := map[string]any{"altitude": "NaN"}
	if err := validateDataComponentValue(comp, value, "result"); err != nil {
		t.Fatalf("expected nil for declared nilValue in record field, got: %v", err)
	}
}
