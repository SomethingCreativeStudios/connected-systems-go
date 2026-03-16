package queryparams

import (
	"net/url"
	"testing"
)

func TestBuildPagintationLinks_SelfWithoutQueryHasNoTrailingQuestionMark(t *testing.T) {
	qp := &QueryParams{Limit: 10}
	params := url.Values{}
	total := 1

	links := qp.BuildPagintationLinks("http://localhost:8080/systems", params, &total, 1)
	if len(links) == 0 {
		t.Fatalf("expected at least one pagination link")
	}

	if links[0].Rel != "self" {
		t.Fatalf("expected first link to be self, got %q", links[0].Rel)
	}

	if links[0].Href != "http://localhost:8080/systems" {
		t.Fatalf("expected self href without trailing '?', got %q", links[0].Href)
	}
}

func TestBuildPagintationLinks_WithQueryBuildsNextAndPrev(t *testing.T) {
	qp := &QueryParams{Limit: 10}
	params := url.Values{}
	params.Set("limit", "10")
	params.Set("offset", "10")
	total := 35

	links := qp.BuildPagintationLinks("http://localhost:8080/systems", params, &total, 10)

	findRel := func(rel string) string {
		for _, link := range links {
			if link.Rel == rel {
				return link.Href
			}
		}
		return ""
	}

	self := findRel("self")
	next := findRel("next")
	prev := findRel("prev")

	if self != "http://localhost:8080/systems?limit=10&offset=10" {
		t.Fatalf("unexpected self href: %q", self)
	}
	if next != "http://localhost:8080/systems?limit=10&offset=20" {
		t.Fatalf("unexpected next href: %q", next)
	}
	if prev != "http://localhost:8080/systems?limit=10" {
		t.Fatalf("unexpected prev href: %q", prev)
	}

	if params.Get("offset") != "10" {
		t.Fatalf("expected original params offset to remain unchanged, got %q", params.Get("offset"))
	}
}
