package common_shared

import (
	"encoding/json"
	"testing"
	"time"
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

func TestNonEmptyTimeRange_EmptyReturnsNil(t *testing.T) {
	if got := NonEmptyTimeRange(&TimeRange{}); got != nil {
		t.Fatalf("expected empty range to be nil, got %#v", got)
	}
}

func TestNonEmptyTimeRange_OpenEndedReturnsRange(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tr := &TimeRange{Start: &start}
	if got := NonEmptyTimeRange(tr); got != tr {
		t.Fatalf("expected bounded range to be preserved, got %#v", got)
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

// A single instant (including "now") selects resources whose time range
// contains that instant, so it bounds both ends rather than opening the
// range towards the future.
func TestParseTimeRangeStrict_InstantBoundsBothEnds(t *testing.T) {
	for _, value := range []string{"2026-03-01T12:00:00Z", "now"} {
		tr, err := ParseTimeRangeStrict([]string{value})
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", value, err)
		}
		if tr.Start == nil || tr.End == nil || !tr.Start.Equal(*tr.End) {
			t.Fatalf("%s: expected Start == End, got %v / %v", value, tr.Start, tr.End)
		}
	}
}

// A string time range in a request body is parsed as strictly as the array
// and object forms: an unparseable bound is an error, not an empty range.
func TestTimeRange_UnmarshalJSON_StringFormIsStrict(t *testing.T) {
	var tr TimeRange
	if err := json.Unmarshal([]byte(`"2026-01-01T00:00:00Z/2026-12-31T00:00:00Z"`), &tr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Start == nil || tr.End == nil {
		t.Fatalf("expected both bounds, got %+v", tr)
	}
	for _, raw := range []string{`"frobnicate"`, `"2026-01-01T00:00:00Z/soon"`} {
		var invalid TimeRange
		if err := json.Unmarshal([]byte(raw), &invalid); err == nil {
			t.Fatalf("%s: expected an error", raw)
		}
	}
}
