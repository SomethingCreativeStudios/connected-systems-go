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
