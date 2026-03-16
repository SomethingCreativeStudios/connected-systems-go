package sensorml_formatters

import (
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	formaters "github.com/yourusername/connected-systems-go/internal/model/formaters"
)

const testAssociationBaseURL = "http://example.test"

func useTestAssociationBaseURL(t *testing.T) {
	t.Helper()
	formaters.SetAssociationLinksBaseURL(testAssociationBaseURL)
	t.Cleanup(func() {
		formaters.SetAssociationLinksBaseURL("")
	})
}

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
