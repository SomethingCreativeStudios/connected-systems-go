package sensorml_formatters

import (
	"context"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
)

const testAssociationBaseURL = "http://example.test"

func assertHasRel(t *testing.T, links common_shared.Links, rel string) {
	t.Helper()
	for _, link := range links {
		if common_shared.RelEquals(link.Rel, rel) {
			return
		}
	}
	t.Fatalf("expected rel %q in %+v", rel, links)
}

func assertMissingRel(t *testing.T, links common_shared.Links, rel string) {
	t.Helper()
	for _, link := range links {
		if common_shared.RelEquals(link.Rel, rel) {
			t.Fatalf("did not expect rel %q in %+v", rel, links)
		}
	}
}

func assertHasHref(t *testing.T, links common_shared.Links, rel, href string) {
	t.Helper()
	for _, link := range links {
		if common_shared.RelEquals(link.Rel, rel) && link.Href == href {
			return
		}
	}
	t.Fatalf("expected rel %q href %q in %+v", rel, href, links)
}

// stubChildren reports fixed nested collections with members, as a
// repository's ChildCollections would.
type stubChildren map[string][]string

func (s stubChildren) ChildCollections(context.Context, []string) (map[string][]string, error) {
	return s, nil
}
