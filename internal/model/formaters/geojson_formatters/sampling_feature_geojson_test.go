package geojson_formatters

import (
	"context"
	"strings"
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
)

func TestSamplingFeatureDeserialize_StripsOnlyAssociationLinks(t *testing.T) {
	formatter := NewSamplingFeatureGeoJSONFormatter(nil)

	payload := `{
		"type": "Feature",
		"properties": {
			"uid": "urn:test:sf:1",
			"name": "SF 1",
			"featureType": "http://www.w3.org/ns/sosa/Sample"
		},
		"links": [
			{"href": "/systems/abc", "rel": "ogc-rel:parentSystem"},
			{"href": "/samplingFeatures/xyz", "rel": "sampleOf"},
			{"href": "/docs/spec", "rel": "alternate"}
		]
	}`

	sf, err := formatter.Deserialize(context.Background(), strings.NewReader(payload))
	if err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	if sf.ParentSystemID == nil || *sf.ParentSystemID != "abc" {
		t.Fatalf("expected parent system association to be mapped from link")
	}

	if sf.SampleOf == nil || len(*sf.SampleOf) != 1 {
		t.Fatalf("expected sampleOf association to be mapped from link")
	}

	if sf.SampleOfIDs == nil || len(*sf.SampleOfIDs) != 1 || (*sf.SampleOfIDs)[0] != "xyz" {
		t.Fatalf("expected sampleOf IDs to be extracted from link")
	}

	if len(sf.Links) != 1 {
		t.Fatalf("expected 1 non-association link to remain, got %d", len(sf.Links))
	}

	if sf.Links[0].Rel != "alternate" {
		t.Fatalf("expected non-association link to be preserved, got rel=%q", sf.Links[0].Rel)
	}
}

func TestSamplingFeatureSerialize_ShowsAssociationLinksToEndUser(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewSamplingFeatureGeoJSONFormatter(nil)

	parentID := "sys-123"
	sampleOf := common_shared.Links{
		{Href: "/samplingFeatures/parent-sample", Rel: common_shared.OGCRel("sampleOf")},
	}

	sf := &domains.SamplingFeature{
		Base: domains.Base{ID: "sf-123"},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: "urn:test:sf:123",
			Name:             "Sample Node",
		},
		FeatureType:    domains.SamplingFeatureTypeSample,
		ParentSystemID: &parentID,
		SampleOf:       &sampleOf,
		Links: common_shared.Links{
			{Href: "/docs/spec", Rel: "alternate"},
		},
	}

	out, err := formatter.Serialize(context.Background(), sf)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	foundParent := false
	foundSampleOf := false
	foundCustom := false
	for _, link := range out.Links {
		if common_shared.RelEquals(link.Rel, common_shared.OGCRel("parentSystem")) {
			foundParent = true
		}
		if common_shared.RelEquals(link.Rel, common_shared.OGCRel("sampleOf")) {
			foundSampleOf = true
		}
		if link.Rel == "alternate" {
			foundCustom = true
		}
	}

	if !foundParent {
		t.Fatalf("expected parentSystem association link in serialized output")
	}
	if !foundSampleOf {
		t.Fatalf("expected sampleOf association link in serialized output")
	}
	if !foundCustom {
		t.Fatalf("expected custom non-association link in serialized output")
	}
}

// Phase 1: inline @link Type enrichment
func TestSamplingFeatureGeoJSONSerialize_SetsTypeOnSampledFeatureLink(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewSamplingFeatureGeoJSONFormatter(nil)

	sf := &domains.SamplingFeature{
		Base: domains.Base{ID: "sf-1"},
		CommonSSN: domains.CommonSSN{
			UniqueIdentifier: "urn:test:sf:1",
			Name:             "Sample 1",
		},
		FeatureType: domains.SamplingFeatureTypeSample,
		SampledFeatureLink: &common_shared.Link{
			Href: "/features/feat-1",
		},
	}

	out, err := formatter.Serialize(context.Background(), sf)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	if out.Properties.SampledFeatureLink == nil {
		t.Fatal("expected SampledFeatureLink to be non-nil")
	}
	if out.Properties.SampledFeatureLink.Type != formaters.GeoJSONContentType {
		t.Errorf("expected SampledFeatureLink.Type %q, got %q", formaters.GeoJSONContentType, out.Properties.SampledFeatureLink.Type)
	}
}

// Phase 2: inline @link Title/UID enrichment (nil repos — fields remain empty)
func TestSamplingFeatureGeoJSONSerializeAll_EnrichesInlineLinks(t *testing.T) {
	useTestAssociationBaseURL(t)

	formatter := NewSamplingFeatureGeoJSONFormatter(nil)

	sfs := []*domains.SamplingFeature{
		{
			Base: domains.Base{ID: "sf-1"},
			CommonSSN: domains.CommonSSN{
				UniqueIdentifier: "urn:test:sf:1",
				Name:             "Sample 1",
			},
			FeatureType: domains.SamplingFeatureTypeSample,
			SampledFeatureLink: &common_shared.Link{
				Href: "/features/feat-1",
			},
		},
	}

	features, err := formatter.SerializeAll(context.Background(), sfs)
	if err != nil {
		t.Fatalf("serializeAll failed: %v", err)
	}

	if len(features) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(features))
	}

	// Type is always set (Phase 1)
	if features[0].Properties.SampledFeatureLink.Type != formaters.GeoJSONContentType {
		t.Errorf("expected SampledFeatureLink.Type %q, got %q", formaters.GeoJSONContentType, features[0].Properties.SampledFeatureLink.Type)
	}

	// Title/UID are empty with nil repos (no cache population)
	if features[0].Properties.SampledFeatureLink.Title != "" {
		t.Errorf("expected empty Title with nil repos, got %q", features[0].Properties.SampledFeatureLink.Title)
	}
	if features[0].Properties.SampledFeatureLink.UID != nil {
		t.Errorf("expected nil UID with nil repos, got %v", features[0].Properties.SampledFeatureLink.UID)
	}
}
