package common_shared

import (
	"encoding/json"
	"testing"
)

func TestIOItem_UnmarshalJSON_Null(t *testing.T) {
	var item IOItem
	if err := json.Unmarshal([]byte("null"), &item); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Component != nil || item.Observable != nil || item.Raw != nil {
		t.Fatalf("expected zero-value IOItem for null input, got %+v", item)
	}
}

func TestIOItem_MarshalJSON_ZeroValue(t *testing.T) {
	var item IOItem
	b, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != "null" {
		t.Fatalf("expected null, got %s", b)
	}
}

func TestIOItem_RoundTrip(t *testing.T) {
	raw := []byte(`{"type":"Quantity","label":"Temperature","definition":"http://example.com/temp"}`)
	var item IOItem
	if err := json.Unmarshal(raw, &item); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	got, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if string(got) != string(raw) {
		t.Fatalf("round-trip mismatch:\n got  %s\n want %s", got, raw)
	}
}

func TestIOList_Value_Nil(t *testing.T) {
	var l IOList
	v, err := l.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != nil {
		t.Fatalf("expected nil driver.Value for nil IOList, got %v", v)
	}
}

func TestIOList_Scan_Nil(t *testing.T) {
	var l IOList
	if err := l.Scan(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l != nil {
		t.Fatalf("expected nil IOList after scanning nil, got %v", l)
	}
}

func TestIOList_Scan_TypeMismatch(t *testing.T) {
	var l IOList
	if err := l.Scan(42); err == nil {
		t.Fatal("expected error for non-[]byte scan input, got nil")
	}
}

func TestIOList_Scan_Valid(t *testing.T) {
	raw := []byte(`[{"type":"Quantity","label":"Temp"}]`)
	var l IOList
	if err := l.Scan(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(l) != 1 {
		t.Fatalf("expected 1 item, got %d", len(l))
	}
}

func TestIOItem_UnmarshalJSON_NullInList(t *testing.T) {
	raw := []byte(`[{"type":"Quantity","label":"Temp"},null]`)
	var l IOList
	if err := json.Unmarshal(raw, &l); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	// second item should be zero-value — no Component, no Observable
	if l[1].Component != nil || l[1].Observable != nil {
		t.Fatalf("expected zero-value IOItem for null element, got %+v", l[1])
	}
	// round-trip
	b, _ := json.Marshal(l)
	if string(b) != `[{"type":"Quantity","label":"Temp"},null]` {
		t.Fatalf("unexpected round-trip: %s", b)
	}
}
