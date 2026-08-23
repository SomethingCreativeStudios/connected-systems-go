package resourcevalidation

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

// ── Constraint validation (Issue #8) ─────────────────────────────────────────

func intPtr(n int) *int { return &n }

func constraintComponent(compType string, c *domains.DatastreamConstraint) *domains.DatastreamDataComponent {
	return &domains.DatastreamDataComponent{
		Type:       compType,
		Constraint: c,
	}
}

func TestValidateDataComponent_Constraint_NoConstraint(t *testing.T) {
	comp := &domains.DatastreamDataComponent{Type: "Quantity"}
	if err := validateDataComponentValue(comp, float64(999), "result"); err != nil {
		t.Fatalf("expected nil for component with no constraint, got: %v", err)
	}
}

func TestValidateDataComponent_Constraint_QuantityInInterval(t *testing.T) {
	intervals, _ := json.Marshal([][]float64{{0, 100}})
	comp := constraintComponent("Quantity", &domains.DatastreamConstraint{Intervals: intervals})
	if err := validateDataComponentValue(comp, float64(50), "result"); err != nil {
		t.Fatalf("expected nil for value within interval, got: %v", err)
	}
}

func TestValidateDataComponent_Constraint_QuantityOutsideInterval(t *testing.T) {
	intervals, _ := json.Marshal([][]float64{{0, 100}})
	comp := constraintComponent("Quantity", &domains.DatastreamConstraint{Intervals: intervals})
	if err := validateDataComponentValue(comp, float64(150), "result"); err == nil {
		t.Fatal("expected error for value outside interval, got nil")
	}
}

func TestValidateDataComponent_Constraint_QuantityInValues(t *testing.T) {
	values, _ := json.Marshal([]float64{1, 2, 3, 5, 8})
	comp := constraintComponent("Quantity", &domains.DatastreamConstraint{Values: values})
	if err := validateDataComponentValue(comp, float64(3), "result"); err != nil {
		t.Fatalf("expected nil for value in allowed set, got: %v", err)
	}
}

func TestValidateDataComponent_Constraint_QuantityNotInValues(t *testing.T) {
	values, _ := json.Marshal([]float64{1, 2, 3, 5, 8})
	comp := constraintComponent("Quantity", &domains.DatastreamConstraint{Values: values})
	if err := validateDataComponentValue(comp, float64(4), "result"); err == nil {
		t.Fatal("expected error for value not in allowed set, got nil")
	}
}

func TestValidateDataComponent_Constraint_TextMatchesPattern(t *testing.T) {
	comp := constraintComponent("Text", &domains.DatastreamConstraint{Pattern: `^[A-Z]{3}-\d{4}$`})
	if err := validateDataComponentValue(comp, "ABC-1234", "result"); err != nil {
		t.Fatalf("expected nil for matching pattern, got: %v", err)
	}
}

func TestValidateDataComponent_Constraint_TextFailsPattern(t *testing.T) {
	comp := constraintComponent("Text", &domains.DatastreamConstraint{Pattern: `^[A-Z]{3}-\d{4}$`})
	if err := validateDataComponentValue(comp, "not-matching", "result"); err == nil {
		t.Fatal("expected error for non-matching pattern, got nil")
	}
}

func TestValidateDataComponent_Constraint_TextInValues(t *testing.T) {
	values, _ := json.Marshal([]string{"low", "medium", "high"})
	comp := constraintComponent("Category", &domains.DatastreamConstraint{Values: values})
	if err := validateDataComponentValue(comp, "medium", "result"); err != nil {
		t.Fatalf("expected nil for allowed token, got: %v", err)
	}
}

func TestValidateDataComponent_Constraint_TextNotInValues(t *testing.T) {
	values, _ := json.Marshal([]string{"low", "medium", "high"})
	comp := constraintComponent("Category", &domains.DatastreamConstraint{Values: values})
	if err := validateDataComponentValue(comp, "extreme", "result"); err == nil {
		t.Fatal("expected error for disallowed token, got nil")
	}
}

func TestValidateDataComponent_Constraint_BooleanWithConstraint(t *testing.T) {
	intervals, _ := json.Marshal([][]float64{{0, 1}})
	comp := constraintComponent("Boolean", &domains.DatastreamConstraint{Intervals: intervals})
	if err := validateDataComponentValue(comp, true, "result"); err != nil {
		t.Fatalf("expected nil for boolean with constraint (no-op), got: %v", err)
	}
}

func TestValidateDataComponent_Constraint_SignificantFigures(t *testing.T) {
	comp := constraintComponent("Quantity", &domains.DatastreamConstraint{SignificantFigures: intPtr(3)})
	if err := validateDataComponentValue(comp, float64(12.3), "result"); err != nil {
		t.Fatalf("expected nil for value within sig figs, got: %v", err)
	}
}

func TestValidateDataComponent_Constraint_SignificantFiguresViolated(t *testing.T) {
	comp := constraintComponent("Quantity", &domains.DatastreamConstraint{SignificantFigures: intPtr(3)})
	if err := validateDataComponentValue(comp, float64(12.345), "result"); err == nil {
		t.Fatal("expected error for too many sig figs, got nil")
	}
}

func TestValidateDataComponent_Constraint_CountInInterval(t *testing.T) {
	intervals, _ := json.Marshal([][]float64{{0, 255}})
	comp := constraintComponent("Count", &domains.DatastreamConstraint{Intervals: intervals})
	if err := validateDataComponentValue(comp, float64(128), "result"); err != nil {
		t.Fatalf("expected nil for count within interval, got: %v", err)
	}
}

func TestValidateDataComponent_Constraint_TimeInValues(t *testing.T) {
	values, _ := json.Marshal([]string{"2024-01-01T00:00:00Z", "2024-06-01T00:00:00Z"})
	comp := constraintComponent("Time", &domains.DatastreamConstraint{Values: values})
	if err := validateDataComponentValue(comp, "2024-01-01T00:00:00Z", "result"); err != nil {
		t.Fatalf("expected nil for allowed time value, got: %v", err)
	}
}

func TestValidateDataComponent_Constraint_TimeNotInValues(t *testing.T) {
	values, _ := json.Marshal([]string{"2024-01-01T00:00:00Z", "2024-06-01T00:00:00Z"})
	comp := constraintComponent("Time", &domains.DatastreamConstraint{Values: values})
	if err := validateDataComponentValue(comp, "2025-01-01T00:00:00Z", "result"); err == nil {
		t.Fatal("expected error for disallowed time value, got nil")
	}
}

func TestValidateDataComponent_Constraint_NilValueBypassesConstraint(t *testing.T) {
	intervals, _ := json.Marshal([][]float64{{0, 100}})
	nv := nilValue("NaN")
	comp := &domains.DatastreamDataComponent{
		Type:       "Quantity",
		Constraint: &domains.DatastreamConstraint{Intervals: intervals},
		NilValues:  []domains.DatastreamNilValue{nv},
	}
	// "NaN" is outside the interval but is a declared nilValue — should pass
	if err := validateDataComponentValue(comp, "NaN", "result"); err != nil {
		t.Fatalf("expected nil for nilValue bypassing constraint, got: %v", err)
	}
}
