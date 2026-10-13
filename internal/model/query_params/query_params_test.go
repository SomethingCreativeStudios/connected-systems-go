package queryparams

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBuildPaginationLinks_PreservesFiltersAndUsesCursors(t *testing.T) {
	qp := &QueryParams{
		Limit: 10,
		Kind:  CursorKindIDAsc,
		Path:  "/systems",
		Page:  PageInfo{HasNext: true, HasPrev: true},
		Anchors: CursorAnchors{
			First: []string{"first"},
			Last:  []string{"last"},
		},
	}
	params := url.Values{"limit": {"10"}, "q": {"weather"}}
	links := qp.BuildPaginationLinks("http://localhost:8080/systems", params)

	byRel := map[string]string{}
	for _, link := range links {
		byRel[link.Rel] = link.Href
	}
	if byRel["self"] != "http://localhost:8080/systems?limit=10&q=weather" {
		t.Fatalf("unexpected self href: %q", byRel["self"])
	}
	for _, rel := range []string{"next", "prev"} {
		href := byRel[rel]
		if href == "" || strings.Contains(href, "offset=") || !strings.Contains(href, "cursor=") || !strings.Contains(href, "q=weather") {
			t.Fatalf("%s link did not preserve filters with a cursor: %q", rel, href)
		}
	}
	if params.Get("cursor") != "" {
		t.Fatalf("expected original params to remain unchanged")
	}
}

func TestBuildFromRequest_RejectsOffsetAndInvalidCursor(t *testing.T) {
	for _, target := range []string{"/systems?offset=1", "/systems?cursor=not-a-token"} {
		req := httptest.NewRequest("GET", target, nil)
		_, err := QueryParams{}.BuildFromRequest(req, 10, CursorKindIDAsc)
		if err == nil {
			t.Fatalf("expected %s to fail", target)
		}
	}
}

func TestBuildFromRequest_RejectsCursorForOtherRouteOrFilter(t *testing.T) {
	qp := &QueryParams{
		Kind:    CursorKindIDAsc,
		Path:    "/systems",
		Page:    PageInfo{HasNext: true},
		Anchors: CursorAnchors{Last: []string{"last"}, First: []string{"first"}},
	}
	links := qp.BuildPaginationLinks("http://localhost:8080/systems", url.Values{"q": {"weather"}})
	var next string
	for _, link := range links {
		if link.Rel == "next" {
			next = link.Href
		}
	}
	parsed, err := url.Parse(next)
	if err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{
		"/deployments?" + parsed.RawQuery,
		"/systems?cursor=" + url.QueryEscape(parsed.Query().Get("cursor")) + "&q=other",
	} {
		req := httptest.NewRequest("GET", target, nil)
		_, err := QueryParams{}.BuildFromRequest(req, 10, CursorKindIDAsc)
		if err == nil || err.Error() != invalidCursorError {
			t.Fatalf("expected invalid cursor for %q, got %v", target, err)
		}
	}
}

// limit must be a positive integer; larger values are clamped to MaxLimit
// rather than disabling the LIMIT altogether.
func TestBuildFromRequest_ValidatesLimit(t *testing.T) {
	for _, raw := range []string{"-1", "0", "ten", "1.5"} {
		req := httptest.NewRequest("GET", "/systems?limit="+raw, nil)
		if _, err := (QueryParams{}).BuildFromRequest(req, 10, CursorKindIDAsc); err == nil {
			t.Fatalf("limit=%s: expected an error", raw)
		}
	}

	cases := map[string]int{"": 10, "limit=25": 25, "limit=100000": MaxLimit}
	for query, want := range cases {
		req := httptest.NewRequest("GET", "/systems?"+query, nil)
		params, err := (QueryParams{}).BuildFromRequest(req, 10, CursorKindIDAsc)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", query, err)
		}
		if params.Limit != want {
			t.Fatalf("%q: limit = %d, want %d", query, params.Limit, want)
		}
	}

	// An unset default limit is bounded rather than unlimited.
	params, err := (QueryParams{}).BuildFromRequest(httptest.NewRequest("GET", "/systems", nil), 0, CursorKindIDAsc)
	if err != nil || params.Limit != MaxLimit {
		t.Fatalf("default 0: limit = %v (err %v), want %d", params, err, MaxLimit)
	}
}
