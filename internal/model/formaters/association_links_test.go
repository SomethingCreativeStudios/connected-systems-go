package formaters

import (
	"reflect"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

const testAssociationBaseURL = "http://example.test"

func useTestAssociationBaseURL(t *testing.T) {
	t.Helper()
	SetAssociationLinksBaseURL(testAssociationBaseURL)
	t.Cleanup(func() {
		SetAssociationLinksBaseURL("")
	})
}

func TestAppendGeoJSONSystemAssociationLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	parentID := "parent-1"
	system := &domains.System{
		Base:           domains.Base{ID: "sys-1"},
		ParentSystemID: &parentID,
		Deployments:    []domains.Deployment{{Base: domains.Base{ID: "dep-1"}}},
		Procedures: []domains.Procedure{
			{Base: domains.Base{ID: "proc-2"}},
			{Base: domains.Base{ID: "proc-1"}},
		},
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

	links := AppendGeoJSONSystemAssociationLinks(system)

	assertHasRel(t, links, "alternate")
	assertHasRel(t, links, common_shared.OGCRel("parentSystem"))
	assertHasRel(t, links, common_shared.OGCRel("subsystems"))
	assertHasRel(t, links, common_shared.OGCRel("samplingFeatures"))
	assertHasRel(t, links, common_shared.OGCRel("deployments"))
	assertHasRel(t, links, common_shared.OGCRel("datastreams"))
	assertHasRel(t, links, common_shared.OGCRel("controlstreams"))
	assertHasHref(t, links, common_shared.OGCRel("procedures"), "http://example.test/procedures?id=proc-1%2Cproc-2")
	assertMissingRel(t, links, common_shared.OGCRel("featuresOfInterest"))
}

func TestAppendGeoJSONSystemAssociationLinks_DedupesDerivedAndExisting(t *testing.T) {
	useTestAssociationBaseURL(t)

	parentID := "parent-1"
	system := &domains.System{
		Base:           domains.Base{ID: "sys-1"},
		ParentSystemID: &parentID,
		Links: common_shared.Links{
			{Href: "/systems/parent-1", Rel: "parentSystem"},
			{Href: "/systems/sys-1/subsystems", Rel: common_shared.OGCRel("subsystems")},
		},
	}

	links := AppendGeoJSONSystemAssociationLinks(system)

	assertRelCount(t, links, common_shared.OGCRel("parentSystem"), 1)
	assertRelCount(t, links, common_shared.OGCRel("subsystems"), 1)
}

func TestAppendSensorMLSystemAssociationLinksExcludesParentSystem(t *testing.T) {
	useTestAssociationBaseURL(t)

	parentID := "parent-1"
	system := &domains.System{
		Base:           domains.Base{ID: "sys-1"},
		ParentSystemID: &parentID,
		Links: common_shared.Links{
			{Href: "/systems/parent-1", Rel: common_shared.OGCRel("parentSystem")},
			{Href: "/systems/sys-1/subsystems", Rel: common_shared.OGCRel("subsystems")},
			{Href: "/systems/sys-1/samplingFeatures", Rel: common_shared.OGCRel("samplingFeatures")},
			{Href: "/systems/sys-1/deployments", Rel: common_shared.OGCRel("deployments")},
			{Href: "/systems/sys-1/datastreams", Rel: common_shared.OGCRel("datastreams")},
			{Href: "/systems/sys-1/controlstreams", Rel: common_shared.OGCRel("controlstreams")},
			{Href: "/docs/spec", Rel: "alternate"},
		},
	}

	links := AppendSensorMLSystemAssociationLinks(system)

	assertHasRel(t, links, "alternate")
	assertHasRel(t, links, common_shared.OGCRel("samplingFeatures"))
	assertHasRel(t, links, common_shared.OGCRel("deployments"))
	assertHasRel(t, links, common_shared.OGCRel("datastreams"))
	assertHasRel(t, links, common_shared.OGCRel("controlstreams"))
	assertHasRel(t, links, common_shared.OGCRel("subsystems"))
	assertMissingRel(t, links, common_shared.OGCRel("parentSystem"))
}

func TestAppendDeploymentAssociationLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

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

	links := AppendDeploymentAssociationLinks(deployment)

	assertHasRel(t, links, "alternate")
	assertHasHref(t, links, common_shared.OGCRel("parentDeployment"), "http://example.test/deployments/dep-parent")
	assertHasHref(t, links, common_shared.OGCRel("subdeployments"), "http://example.test/deployments/dep-1/subdeployments")
	assertHasHref(t, links, common_shared.OGCRel("samplingFeatures"), "http://example.test/samplingFeatures?deployment=dep-1")
	assertHasHref(t, links, common_shared.OGCRel("featuresOfInterest"), "http://example.test/features?deployment=dep-1")
	assertHasHref(t, links, common_shared.OGCRel("datastreams"), "http://example.test/datastreams?deployment=dep-1")
	assertHasHref(t, links, common_shared.OGCRel("controlstreams"), "http://example.test/controlStreams?deployment=dep-1")
	assertMissingRel(t, links, common_shared.OGCRel("deployedSystems"))
}

func TestAppendProcedureAssociationLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	procedure := &domains.Procedure{
		Base: domains.Base{ID: "proc-1"},
		Systems: []domains.System{
			{Base: domains.Base{ID: "sys-1"}},
		},
		Links: common_shared.Links{
			{Href: "/docs/spec", Rel: "alternate"},
		},
	}

	links := AppendProcedureAssociationLinks(procedure)

	assertHasRel(t, links, "alternate")
	assertHasHref(t, links, common_shared.OGCRel("implementingSystems"), "http://example.test/systems?procedure=proc-1")
}

func TestAppendSamplingFeatureGeoJSONAssociationLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	parentID := "sys-10"
	parentUID := "urn:system:10"
	sampleOf := common_shared.Links{
		{Href: "/samplingFeatures/sf-parent-1", Rel: common_shared.OGCRel("sampleOf"), UID: ptrString("urn:sample:1")},
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

	links := AppendSamplingFeatureGeoJSONAssociationLinks(sf)

	assertHasRel(t, links, "alternate")
	assertHasHref(t, links, common_shared.OGCRel("parentSystem"), "http://example.test/systems/sys-10")
	assertHasHref(t, links, common_shared.OGCRel("sampleOf"), "http://example.test/samplingFeatures/sf-parent-1")
	assertHasHref(t, links, common_shared.OGCRel("datastreams"), "http://example.test/datastreams?samplingFeature=sf-10")
	assertHasHref(t, links, common_shared.OGCRel("controlstreams"), "http://example.test/controlStreams?samplingFeature=sf-10")
	assertHasRel(t, links, common_shared.OGCRel("attachedTo"))
}

func TestApplyGeoJSONSystemAssociationLinks(t *testing.T) {
	system := &domains.System{}
	links := common_shared.Links{
		{Href: "/systems/parent-9", Rel: common_shared.OGCRel("parentSystem")},
		{Href: "/systems/sys-9/subsystems", Rel: common_shared.OGCRel("subsystems")},
	}

	ApplyGeoJSONSystemAssociationLinks(system, links)

	if system.ParentSystemID == nil || *system.ParentSystemID != "parent-9" {
		t.Fatalf("expected parentSystem id parent-9, got %+v", system.ParentSystemID)
	}
}

func TestApplyDeploymentAssociationLinks(t *testing.T) {
	deployment := &domains.Deployment{}
	links := common_shared.Links{
		{Href: "/deployments/parent-4", Rel: common_shared.OGCRel("parentDeployment")},
		{Href: "/deployments/dep-4/subdeployments", Rel: common_shared.OGCRel("subdeployments")},
	}

	ApplyDeploymentAssociationLinks(deployment, links)

	if deployment.ParentDeploymentID == nil || *deployment.ParentDeploymentID != "parent-4" {
		t.Fatalf("expected parentDeployment id parent-4, got %+v", deployment.ParentDeploymentID)
	}
}

func TestApplySamplingFeatureGeoJSONAssociationLinks(t *testing.T) {
	sf := &domains.SamplingFeature{}
	links := common_shared.Links{
		{Href: "/systems/sys-2", Rel: common_shared.OGCRel("parentSystem"), UID: ptrString("urn:system:2")},
		{Href: "/samplingFeatures/sample-1", Rel: common_shared.OGCRel("sampleOf"), UID: ptrString("urn:sample:1")},
		{Href: "/samplingFeatures/sample-2", Rel: common_shared.OGCRel("sampleOf"), UID: ptrString("urn:sample:2")},
	}

	ApplySamplingFeatureGeoJSONAssociationLinks(sf, links)

	if sf.ParentSystemID == nil || *sf.ParentSystemID != "sys-2" {
		t.Fatalf("expected parentSystem id sys-2, got %+v", sf.ParentSystemID)
	}
	if sf.ParentSystemUID == nil || *sf.ParentSystemUID != "urn:system:2" {
		t.Fatalf("expected parentSystem uid urn:system:2, got %+v", sf.ParentSystemUID)
	}
	if sf.SampleOf == nil || len(*sf.SampleOf) != 2 {
		t.Fatalf("expected 2 sampleOf links, got %+v", sf.SampleOf)
	}
	if sf.SampleOfIDs == nil || !reflect.DeepEqual(*sf.SampleOfIDs, []string{"sample-1", "sample-2"}) {
		t.Fatalf("unexpected sampleOf IDs: %+v", sf.SampleOfIDs)
	}
	if sf.SampleOfUIDs == nil || !reflect.DeepEqual(*sf.SampleOfUIDs, []string{"urn:sample:1", "urn:sample:2"}) {
		t.Fatalf("unexpected sampleOf UIDs: %+v", sf.SampleOfUIDs)
	}
}

func TestBuildProceduresEndpointHref(t *testing.T) {
	procedures := []domains.Procedure{
		{Base: domains.Base{ID: "proc-c"}},
		{Base: domains.Base{ID: "proc-a"}},
		{Base: domains.Base{ID: "proc-a"}},
		{Base: domains.Base{ID: ""}},
		{Base: domains.Base{ID: "proc-b"}},
	}

	href := buildProceduresEndpointHref(procedures)
	if href != "/procedures?id=proc-a%2Cproc-b%2Cproc-c" {
		t.Fatalf("unexpected href: %s", href)
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

func ptrString(v string) *string {
	return &v
}
