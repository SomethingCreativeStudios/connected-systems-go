package common_shared

import (
	"encoding/json"
	"testing"
)

// ── UnmarshalJSON ────────────────────────────────────────────────────────────

func TestTimeRange_UnmarshalJSON_ValidArray_OK(t *testing.T) {
	var tr TimeRange
	if err := json.Unmarshal([]byte(`["2026-01-01T00:00:00Z", null]`), &tr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Start == nil {
		t.Fatal("expected Start to be set")
	}
	if tr.End != nil {
		t.Fatal("expected End to be nil")
	}
}

func TestTimeRange_UnmarshalJSON_DotDot_OK(t *testing.T) {
	var tr TimeRange
	if err := json.Unmarshal([]byte(`["..", "2026-12-31T23:59:59Z"]`), &tr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Start != nil {
		t.Fatal("expected Start to be nil for \"..\"")
	}
	if tr.End == nil {
		t.Fatal("expected End to be set")
	}
}

func TestTimeRange_UnmarshalJSON_NumericElement_ReturnsError(t *testing.T) {
	var tr TimeRange
	if err := json.Unmarshal([]byte(`[1773100000, null]`), &tr); err == nil {
		t.Fatal("expected error for numeric array element, got nil")
	}
}

func TestTimeRange_UnmarshalJSON_InvalidString_ReturnsError(t *testing.T) {
	var tr TimeRange
	if err := json.Unmarshal([]byte(`["notadate", null]`), &tr); err == nil {
		t.Fatal("expected error for invalid date string, got nil")
	}
}

func TestTimeRange_UnmarshalJSON_ObjectInvalidStart_ReturnsError(t *testing.T) {
	var tr TimeRange
	if err := json.Unmarshal([]byte(`{"start":"notadate"}`), &tr); err == nil {
		t.Fatal("expected error for invalid start in object form, got nil")
	}
}

func TestTimeRange_UnmarshalJSON_Null_OK(t *testing.T) {
	var tr TimeRange
	if err := json.Unmarshal([]byte(`null`), &tr); err != nil {
		t.Fatalf("unexpected error for null: %v", err)
	}
}

// ── ParseTimeRangeStrict ─────────────────────────────────────────────────────

func TestParseTimeRangeStrict_ValidRange_OK(t *testing.T) {
	tr, err := ParseTimeRangeStrict([]string{"2026-01-01T00:00:00Z/2026-12-31T23:59:59Z"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Start == nil || tr.End == nil {
		t.Fatal("expected both Start and End to be set")
	}
}

func TestParseTimeRangeStrict_TwoElementSlice_OK(t *testing.T) {
	tr, err := ParseTimeRangeStrict([]string{"2026-01-01T00:00:00Z", "2026-12-31T23:59:59Z"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Start == nil || tr.End == nil {
		t.Fatal("expected both Start and End to be set")
	}
}

func TestParseTimeRangeStrict_Garbage_ReturnsError(t *testing.T) {
	if _, err := ParseTimeRangeStrict([]string{"frobnicate"}); err == nil {
		t.Fatal("expected error for garbage input, got nil")
	}
}

func TestParseTimeRangeStrict_GarbageInRange_ReturnsError(t *testing.T) {
	if _, err := ParseTimeRangeStrict([]string{"frobnicate/2026-12-31T23:59:59Z"}); err == nil {
		t.Fatal("expected error for garbage start in range, got nil")
	}
}

func TestParseTimeRangeStrict_LatestKeyword_SetsLatestFlag(t *testing.T) {
	tr, err := ParseTimeRangeStrict([]string{"latest"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tr.Latest {
		t.Fatal("expected Latest=true for 'latest' keyword")
	}
	if tr.Start != nil || tr.End != nil {
		t.Fatal("expected Start/End to be nil when Latest=true")
	}
}

func TestToTimeRange_LatestKeyword_SetsLatestFlag(t *testing.T) {
	tr := ToTimeRange("latest")
	if !tr.Latest {
		t.Fatal("expected Latest=true for 'latest' keyword")
	}
}

func TestParseTimeRangeStrict_DotDot_OK(t *testing.T) {
	// ".." as a standalone value is treated the same as "latest"/"now" — accepted without error.
	_, err := ParseTimeRangeStrict([]string{".."})
	if err != nil {
		t.Fatalf("unexpected error for \"..\" keyword: %v", err)
	}
}

func TestParseTimeRangeStrict_OpenEndedRange_OK(t *testing.T) {
	tr, err := ParseTimeRangeStrict([]string{"2026-01-01T00:00:00Z/.."})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Start == nil {
		t.Fatal("expected Start to be set")
	}
	if tr.End != nil {
		t.Fatal("expected End to be nil for \"..\"")
	}
}
