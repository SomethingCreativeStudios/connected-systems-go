package json_formatters

import (
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
)

// streamRefs are the resources a datastream or control stream links to: the
// stored IDs used to look them up and the inline links to resolve.
type streamRefs struct {
	SystemID          *string
	ProcedureID       *string
	DeploymentID      *string
	SamplingFeatureID *string

	Procedure         *common_shared.Link
	Deployment        *common_shared.Link
	FeatureOfInterest *common_shared.Link
	SamplingFeature   *common_shared.Link
}

// streamLinks are the resolved inline links of a datastream or control stream.
type streamLinks struct {
	System            *common_shared.Link
	Procedure         *common_shared.Link
	Deployment        *common_shared.Link
	FeatureOfInterest *common_shared.Link
	SamplingFeature   *common_shared.Link
}

// streamReferences lists what a page of streams links to.
func streamReferences(refs []streamRefs) formaters.References {
	var out formaters.References
	for _, ref := range refs {
		out.Systems = appendID(out.Systems, ref.SystemID)
		out.Procedures = appendID(out.Procedures, ref.ProcedureID)
		out.Deployments = appendID(out.Deployments, ref.DeploymentID)
		out.SamplingFeatures = appendID(out.SamplingFeatures, ref.SamplingFeatureID)
	}
	return out
}

// resolve builds the inline links with absolute hrefs, default media types,
// and the titles and uids of the linked resources.
func (ref streamRefs) resolve(resolver *formaters.LinkResolver, cache *formaters.ResourceCache) streamLinks {
	links := streamLinks{
		Procedure:         inlineLink(resolver, ref.Procedure, mediatype.SensorML),
		Deployment:        inlineLink(resolver, ref.Deployment, mediatype.GeoJSON),
		FeatureOfInterest: inlineLink(resolver, ref.FeatureOfInterest, mediatype.GeoJSON),
		SamplingFeature:   inlineLink(resolver, ref.SamplingFeature, mediatype.GeoJSON),
	}
	if ref.SystemID != nil {
		links.System = inlineLink(resolver, &common_shared.Link{Href: "/systems/" + *ref.SystemID}, mediatype.GeoJSON)
	}
	if system, ok := lookup(cache.Systems, ref.SystemID); ok && hasHref(links.System) {
		links.System.Describe(system.Name, string(system.UniqueIdentifier))
	}
	if procedure, ok := lookup(cache.Procedures, ref.ProcedureID); ok && hasHref(links.Procedure) {
		links.Procedure.Describe(procedure.Name, string(procedure.UniqueIdentifier))
	}
	if deployment, ok := lookup(cache.Deployments, ref.DeploymentID); ok && hasHref(links.Deployment) {
		links.Deployment.Describe(deployment.Name, string(deployment.UniqueIdentifier))
	}
	if samplingFeature, ok := lookup(cache.SamplingFeatures, ref.SamplingFeatureID); ok && hasHref(links.SamplingFeature) {
		links.SamplingFeature.Describe(samplingFeature.Name, string(samplingFeature.UniqueIdentifier))
	}
	return links
}

// inlineLink copies an inline link, making its href absolute and giving it
// mediaType when it has none. The copy leaves the domain object untouched. A
// link without an href is copied as is.
func inlineLink(resolver *formaters.LinkResolver, link *common_shared.Link, mediaType string) *common_shared.Link {
	if link == nil {
		return nil
	}
	resolved := *link
	if resolved.Href == "" {
		return &resolved
	}
	resolved.Href = resolver.Href(resolved.Href)
	if resolved.Type == "" {
		resolved.Type = mediaType
	}
	return &resolved
}

// describeProcedure titles a resolved procedure@link from the cache.
func describeProcedure(link *common_shared.Link, cache *formaters.ResourceCache) {
	if !hasHref(link) {
		return
	}
	if procedure, ok := lookup(cache.Procedures, link.GetId("procedures")); ok {
		link.Describe(procedure.Name, string(procedure.UniqueIdentifier))
	}
}

func hasHref(link *common_shared.Link) bool {
	return link != nil && link.Href != ""
}

func lookup[T any](resources map[string]*T, id *string) (*T, bool) {
	if id == nil {
		return nil, false
	}
	resource, ok := resources[*id]
	return resource, ok
}

func appendID(ids []string, id *string) []string {
	if id == nil || *id == "" {
		return ids
	}
	return append(ids, *id)
}
