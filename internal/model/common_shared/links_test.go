package common_shared

import "testing"

func TestStripAssociationLinks(t *testing.T) {
	links := Links{
		{Href: "/systems/1", Rel: "ogc-rel:parentSystem"},
		{Href: "/systems/1", Rel: "parentSystem"},
		{Href: "/samplingFeatures/2", Rel: "ogc-rel:sampleOf"},
		{Href: "/docs/x", Rel: "alternate"},
		{Href: "/docs/y", Rel: "self"},
		{Href: "/custom/z", Rel: "x-custom-rel"},
		{Href: "/nolabel"},
	}

	got := StripAssociationLinks(links)

	if len(got) != 4 {
		t.Fatalf("expected 4 non-association links, got %d", len(got))
	}

	for _, link := range got {
		if CanonicalRel(link.Rel) == "parentSystem" || CanonicalRel(link.Rel) == "sampleOf" {
			t.Fatalf("association link was not stripped: %+v", link)
		}
	}
}