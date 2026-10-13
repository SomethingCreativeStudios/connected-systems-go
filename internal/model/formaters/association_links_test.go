package formaters

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
)

const testAssociationBaseURL = "http://example.test"

var testLinks = &LinkResolver{BaseURL: testAssociationBaseURL}

// childrenCache reports the given nested collections of the system as
// having members, as the repositories' ChildCollections would.
func childrenCache(systemID string, rels ...string) *ResourceCache {
	cache := NewResourceCache()
	cache.SystemChildren[systemID] = rels
	return cache
}

func TestSystemLinks(t *testing.T) {
	parentID := "parent-1"
	system := &domains.System{
		Base:           domains.Base{ID: "sys-1"},
		ParentSystemID: &parentID,
		Deployments:    []domains.Deployment{{Base: domains.Base{ID: "dep-1"}}},
		Links: common_shared.Links{
			{Href: "/docs/spec", Rel: "alternate"},
			{Href: "/systems/sys-1/subsystems", Rel: common_shared.OGCRel("subsystems")},
			{Href: "/systems/sys-1/samplingFeatures", Rel: common_shared.OGCRel("samplingFeatures")},
			{Href: "/systems/sys-1/deployments", Rel: common_shared.OGCRel("deployments")},
			{Href: "/systems/sys-1/datastreams", Rel: common_shared.OGCRel("datastreams")},
			{Href: "/systems/sys-1/controlstreams", Rel: common_shared.OGCRel("controlstreams")},
			{Href: "/features?system=sys-1", Rel: common_shared.OGCRel("featuresOfInterest")},
		},
	}

	cache := childrenCache("sys-1", "subsystems", "samplingFeatures", "datastreams", "controlstreams")
	links := testLinks.SystemLinks(system, cache, mediatype.GeoJSON)

	assertHasRel(t, links, "alternate")
	assertHasRel(t, links, common_shared.OGCRel("parentSystem"))
	assertHasHref(t, links, common_shared.OGCRel("subsystems"), "http://example.test/systems/sys-1/subsystems")
	assertHasRel(t, links, common_shared.OGCRel("samplingFeatures"))
	assertHasRel(t, links, common_shared.OGCRel("deployments"))
	assertHasRel(t, links, common_shared.OGCRel("datastreams"))
	assertHasRel(t, links, common_shared.OGCRel("controlstreams"))
	// Association links are derived, never echoed from stored links.
	assertMissingRel(t, links, common_shared.OGCRel("featuresOfInterest"))
	assertMissingRel(t, links, common_shared.OGCRel("procedures"))
}

func TestSystemLinks_DedupesDerivedAndExisting(t *testing.T) {
	parentID := "parent-1"
	system := &domains.System{
		Base:           domains.Base{ID: "sys-1"},
		ParentSystemID: &parentID,
		Links: common_shared.Links{
			{Href: "/systems/parent-1", Rel: "parentSystem"},
			{Href: "/systems/sys-1/subsystems", Rel: common_shared.OGCRel("subsystems")},
		},
	}

	links := testLinks.SystemLinks(system, childrenCache("sys-1", "subsystems"), mediatype.GeoJSON)

	assertRelCount(t, links, common_shared.OGCRel("parentSystem"), 1)
	assertRelCount(t, links, common_shared.OGCRel("subsystems"), 1)
}

func TestSystemLinks_EnrichesFromCache(t *testing.T) {
	parentID := "parent-1"
	parentName := "Parent System"
	parentUID := "urn:system:parent-1"

	system := &domains.System{
		Base:           domains.Base{ID: "sys-1"},
		ParentSystemID: &parentID,
	}

	cache := NewResourceCache()
	cache.Systems[parentID] = &domains.System{
		Base:      domains.Base{ID: parentID},
		CommonSSN: domains.CommonSSN{Name: parentName, UniqueIdentifier: domains.UniqueID(parentUID)},
	}

	links := testLinks.SystemLinks(system, cache, mediatype.GeoJSON)

	// Find the parentSystem link and verify enrichment
	var parentLink *common_shared.Link
	for i := range links {
		if common_shared.RelEquals(links[i].Rel, common_shared.OGCRel("parentSystem")) {
			parentLink = &links[i]
			break
		}
	}
	if parentLink == nil {
		t.Fatal("expected parentSystem link")
	}
	if parentLink.Title != parentName {
		t.Fatalf("expected title %q, got %q", parentName, parentLink.Title)
	}
	if parentLink.UID == nil || *parentLink.UID != parentUID {
		t.Fatalf("expected uid %q, got %v", parentUID, parentLink.UID)
	}
	if parentLink.Type != mediatype.GeoJSON {
		t.Fatalf("expected type %q, got %q", mediatype.GeoJSON, parentLink.Type)
	}
	if parentLink.Href != "http://example.test/systems/parent-1" {
		t.Fatalf("expected absolute href, got %q", parentLink.Href)
	}
}

func TestSystemLinks_SensorMLIncludesParentSystem(t *testing.T) {
	parentID := "parent-1"
	system := &domains.System{
		Base:           domains.Base{ID: "sys-1"},
		ParentSystemID: &parentID,
		Links: common_shared.Links{
			{Href: "/systems/sys-1/subsystems", Rel: common_shared.OGCRel("subsystems")},
			{Href: "/systems/sys-1/samplingFeatures", Rel: common_shared.OGCRel("samplingFeatures")},
			{Href: "/systems/sys-1/deployments", Rel: common_shared.OGCRel("deployments")},
			{Href: "/systems/sys-1/datastreams", Rel: common_shared.OGCRel("datastreams")},
			{Href: "/systems/sys-1/controlstreams", Rel: common_shared.OGCRel("controlstreams")},
			{Href: "/docs/spec", Rel: "alternate"},
		},
	}

	cache := childrenCache("sys-1", "subsystems", "samplingFeatures", "deployments", "datastreams", "controlstreams")
	links := testLinks.SystemLinks(system, cache, mediatype.SensorML)

	assertHasRel(t, links, "alternate")
	assertHasRel(t, links, common_shared.OGCRel("samplingFeatures"))
	assertHasRel(t, links, common_shared.OGCRel("deployments"))
	assertHasRel(t, links, common_shared.OGCRel("datastreams"))
	assertHasRel(t, links, common_shared.OGCRel("controlstreams"))
	assertHasRel(t, links, common_shared.OGCRel("subsystems"))
	assertHasHref(t, links, common_shared.OGCRel("parentSystem"), "http://example.test/systems/parent-1")
}

func TestDeploymentLinks(t *testing.T) {
	parentID := "dep-parent"
	deployment := &domains.Deployment{
		Base:               domains.Base{ID: "dep-1"},
		ParentDeploymentID: &parentID,
		Links: common_shared.Links{
			{Href: "/deployments/dep-1/subdeployments", Rel: common_shared.OGCRel("subdeployments")},
			{Href: "/samplingFeatures?deployment=dep-1", Rel: common_shared.OGCRel("samplingFeatures")},
			{Href: "/features?deployment=dep-1", Rel: common_shared.OGCRel("featuresOfInterest")},
			{Href: "/datastreams?deployment=dep-1", Rel: common_shared.OGCRel("datastreams")},
			{Href: "/controlStreams?deployment=dep-1", Rel: common_shared.OGCRel("controlstreams")},
			{Href: "/systems?id=s1,s2", Rel: common_shared.OGCRel("deployedSystems")},
			{Href: "/docs/spec", Rel: "alternate"},
		},
	}

	cache := NewResourceCache()
	cache.DeploymentChildren["dep-1"] = []string{"subdeployments"}
	links := testLinks.DeploymentLinks(deployment, cache)

	assertHasRel(t, links, "alternate")
	assertHasHref(t, links, common_shared.OGCRel("parentDeployment"), "http://example.test/deployments/dep-parent")
	assertHasHref(t, links, common_shared.OGCRel("subdeployments"), "http://example.test/deployments/dep-1/subdeployments")
	// These will need a little bit more work as i am not sure the best link for any of theme
	// assertHasHref(t, links, common_shared.OGCRel("samplingFeatures"), "http://example.test/samplingFeatures?deployment=dep-1")
	// assertHasHref(t, links, common_shared.OGCRel("featuresOfInterest"), "http://example.test/features?deployment=dep-1")
	// assertHasHref(t, links, common_shared.OGCRel("datastreams"), "http://example.test/datastreams?deployment=dep-1")
	// assertHasHref(t, links, common_shared.OGCRel("controlstreams"), "http://example.test/controlStreams?deployment=dep-1")
	assertMissingRel(t, links, common_shared.OGCRel("deployedSystems"))
}

func TestProcedureLinks(t *testing.T) {
	procedure := &domains.Procedure{
		Base: domains.Base{ID: "proc-1"},
		Links: common_shared.Links{
			{Href: "/docs/spec", Rel: "alternate"},
		},
	}

	cache := NewResourceCache()
	cache.ProcedureChildren["proc-1"] = []string{"implementingSystems"}
	links := testLinks.ProcedureLinks(procedure, cache)

	assertHasRel(t, links, "alternate")
	assertHasHref(t, links, common_shared.OGCRel("implementingSystems"), "http://example.test/systems?procedure=proc-1&recursive=true")
}

func TestSamplingFeatureLinks(t *testing.T) {
	parentID := "sys-10"
	parentUID := "urn:system:10"
	sampleOf := common_shared.Links{
		{Href: "/samplingFeatures/sf-parent-1", Rel: common_shared.OGCRel("sampleOf"), UID: new("urn:sample:1")},
	}

	sf := &domains.SamplingFeature{
		Base:            domains.Base{ID: "sf-10"},
		ParentSystemID:  &parentID,
		ParentSystemUID: &parentUID,
		SampleOf:        &sampleOf,
		Links: common_shared.Links{
			{Href: "/datastreams?samplingFeature=sf-10", Rel: common_shared.OGCRel("datastreams")},
			{Href: "/controlStreams?samplingFeature=sf-10", Rel: common_shared.OGCRel("controlstreams")},
			{Href: "/docs/spec", Rel: "alternate"},
			{Href: "/systems/sys-10", Rel: common_shared.OGCRel("attachedTo")},
		},
	}

	links := testLinks.SamplingFeatureLinks(sf)

	assertHasRel(t, links, "alternate")
	assertHasHref(t, links, common_shared.OGCRel("parentSystem"), "http://example.test/systems/sys-10")
	assertHasHref(t, links, common_shared.OGCRel("sampleOf"), "http://example.test/samplingFeatures/sf-parent-1")

	// For these ones i am not sure if this is the best link
	// TO-DO
	// assertHasHref(t, links, common_shared.OGCRel("datastreams"), "http://example.test/datastreams?samplingFeature=sf-10")
	// assertHasHref(t, links, common_shared.OGCRel("controlstreams"), "http://example.test/controlStreams?samplingFeature=sf-10")
	// A stored ogc-rel link the server does not generate is never echoed back.
	assertMissingRel(t, links, common_shared.OGCRel("attachedTo"))
}

// ogc-rel links are server-generated, so writes drop them in every spelling
// while keeping the client's own links.
func TestStripAssociationLinks_DropsEveryOGCRel(t *testing.T) {
	links := common_shared.StripAssociationLinks(common_shared.Links{
		{Href: "/systems/a", Rel: common_shared.OGCRel("parentSystem")},
		{Href: "/systems/a/events", Rel: common_shared.OGCRel("systemEvents")},
		{Href: "/collections/x/queryables", Rel: "http://www.opengis.net/def/rel/ogc/1.0/queryables"},
		{Href: "/systems/a", Rel: "parentSystem"},
		{Href: "/docs/spec", Rel: "alternate"},
	})
	if len(links) != 1 || links[0].Rel != "alternate" {
		t.Fatalf("expected only the alternate link to remain, got %+v", links)
	}
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

func assertRelCount(t *testing.T, links common_shared.Links, rel string, expected int) {
	t.Helper()
	count := 0
	for _, link := range links {
		if common_shared.RelEquals(link.Rel, rel) {
			count++
		}
	}
	if count != expected {
		t.Fatalf("expected %d links for rel %q, got %d in %+v", expected, rel, count, links)
	}
}

// A nil resolver leaves hrefs relative.
func TestLinkResolverHref(t *testing.T) {
	var nilResolver *LinkResolver
	if got := nilResolver.Href("/systems/a"); got != "/systems/a" {
		t.Fatalf("expected relative href, got %q", got)
	}
	resolver := &LinkResolver{BaseURL: "http://example.test/api/"}
	for href, want := range map[string]string{
		"/systems/a":                 "http://example.test/api/systems/a",
		"systems/a":                  "http://example.test/api/systems/a",
		"https://elsewhere.test/x/y": "https://elsewhere.test/x/y",
		"":                           "",
	} {
		if got := resolver.Href(href); got != want {
			t.Fatalf("Href(%q) = %q, want %q", href, got, want)
		}
	}
}

type failingChildren struct {
	err error
	ctx context.Context
}

func (f *failingChildren) ChildCollections(ctx context.Context, _ []string) (map[string][]string, error) {
	f.ctx = ctx
	return nil, f.err
}
func TestLookupReturnsFailureAndUsesRequestContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	reader := &failingChildren{err: context.Canceled}
	resolver := &LinkResolver{SystemChildren: reader}
	cache, err := resolver.Lookup(ctx, References{SystemChildren: []string{"system"}})
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, cache)
	require.Equal(t, ctx, reader.ctx)
}

func TestLocalIDDoesNotResolveRemoteCollisions(t *testing.T) {
	resolver := &LinkResolver{BaseURL: "https://local.test/api"}
	require.Nil(t, resolver.LocalID(common_shared.Link{Href: "https://remote.test/api/systems/id"}, "systems"))
	require.Nil(t, resolver.LocalID(common_shared.Link{Href: "https://local.test/other/systems/id"}, "systems"))
	require.Equal(t, new("id"), resolver.LocalID(common_shared.Link{Href: "https://local.test/api/systems/id"}, "systems"))
	require.Equal(t, new("id"), resolver.LocalID(common_shared.Link{Href: "/systems/id"}, "systems"))
}
