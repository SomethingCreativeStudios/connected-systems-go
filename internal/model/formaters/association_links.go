package formaters

import (
	"context"
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
)

// ByIDsReader is the batch lookup the repositories expose.
type ByIDsReader[T any] interface {
	GetByIDs(ctx context.Context, ids []string) (map[string]*T, error)
}

// ChildCollectionReader reports, for each of the given resources, which of
// its nested collections have members.
type ChildCollectionReader interface {
	ChildCollections(ctx context.Context, ids []string) (map[string][]string, error)
}

// LinkResolver builds response links in one place. It makes hrefs absolute
// against the API base URL, and looks up the resources links point to so
// links carry their titles and uids and nested collections are linked only
// when they have members. A nil resolver, or a nil lookup, leaves hrefs
// relative and skips that enrichment.
type DeploymentSystemsReader interface {
	DeployedSystemsByIDs(context.Context, []string) (map[string]domains.DeployedSystemItems, error)
}

type LinkResolver struct {
	DeploymentSystems  DeploymentSystemsReader
	BaseURL            string
	Systems            ByIDsReader[domains.System]
	Procedures         ByIDsReader[domains.Procedure]
	Deployments        ByIDsReader[domains.Deployment]
	SamplingFeatures   ByIDsReader[domains.SamplingFeature]
	SystemChildren     ChildCollectionReader
	DeploymentChildren ChildCollectionReader
	ProcedureChildren  ChildCollectionReader
}

// Href makes a server-relative href absolute.
func (l *LinkResolver) Href(href string) string {
	href = strings.TrimSpace(href)
	if href == "" || l == nil || l.BaseURL == "" {
		return href
	}
	if parsed, err := url.Parse(href); err == nil && parsed.IsAbs() {
		return href
	}
	base := strings.TrimRight(l.BaseURL, "/")
	if strings.HasPrefix(href, "/") {
		return base + href
	}
	return base + "/" + href
}

// References are the IDs of the resources a page of results links to, and
// of the resources whose nested collections are checked for members.
type References struct {
	Systems            []string
	Procedures         []string
	Deployments        []string
	SamplingFeatures   []string
	SystemChildren     []string
	DeploymentChildren []string
	ProcedureChildren  []string
}

// ResourceCache holds what a LinkResolver looked up for one page of results.
type ResourceCache struct {
	DeploymentSystems  map[string]domains.DeployedSystemItems
	Systems            map[string]*domains.System
	Procedures         map[string]*domains.Procedure
	Deployments        map[string]*domains.Deployment
	SamplingFeatures   map[string]*domains.SamplingFeature
	SystemChildren     map[string][]string
	DeploymentChildren map[string][]string
	ProcedureChildren  map[string][]string
}

// NewResourceCache creates an empty ResourceCache.
func NewResourceCache() *ResourceCache {
	return &ResourceCache{
		Systems:            make(map[string]*domains.System),
		Procedures:         make(map[string]*domains.Procedure),
		Deployments:        make(map[string]*domains.Deployment),
		SamplingFeatures:   make(map[string]*domains.SamplingFeature),
		SystemChildren:     make(map[string][]string),
		DeploymentChildren: make(map[string][]string),
		ProcedureChildren:  make(map[string][]string),
	}
}

// Lookup batch-loads references. Failures are returned so a cancelled request
// or database error cannot silently remove associations from a response.
func (l *LinkResolver) Lookup(ctx context.Context, refs References) (*ResourceCache, error) {
	cache := NewResourceCache()
	if l == nil {
		return cache, nil
	}
	if l.DeploymentSystems != nil && len(refs.DeploymentChildren) > 0 {
		var err error
		cache.DeploymentSystems, err = l.DeploymentSystems.DeployedSystemsByIDs(ctx, compactIDs(refs.DeploymentChildren))
		if err != nil {
			return nil, err
		}
		for _, items := range cache.DeploymentSystems {
			for _, item := range items {
				refs.Systems = append(refs.Systems, trimmedID(l.LocalID(item.System, "systems")))
			}
		}
	}
	lookups := []func() error{
		func() error { return fetchInto(ctx, cache.Systems, l.Systems, refs.Systems) },
		func() error { return fetchInto(ctx, cache.Procedures, l.Procedures, refs.Procedures) },
		func() error { return fetchInto(ctx, cache.Deployments, l.Deployments, refs.Deployments) },
		func() error { return fetchInto(ctx, cache.SamplingFeatures, l.SamplingFeatures, refs.SamplingFeatures) },
		func() error { return fetchChildren(ctx, cache.SystemChildren, l.SystemChildren, refs.SystemChildren) },
		func() error {
			return fetchChildren(ctx, cache.DeploymentChildren, l.DeploymentChildren, refs.DeploymentChildren)
		},
		func() error {
			return fetchChildren(ctx, cache.ProcedureChildren, l.ProcedureChildren, refs.ProcedureChildren)
		},
	}
	for _, lookup := range lookups {
		if err := lookup(); err != nil {
			return nil, err
		}
	}
	return cache, nil
}

func fetchInto[T any](ctx context.Context, into map[string]*T, reader ByIDsReader[T], ids []string) error {
	ids = compactIDs(ids)
	if reader == nil || len(ids) == 0 {
		return nil
	}
	found, err := reader.GetByIDs(ctx, ids)
	if err != nil {
		return err
	}
	maps.Copy(into, found)
	return nil
}

func fetchChildren(ctx context.Context, into map[string][]string, reader ChildCollectionReader, ids []string) error {
	ids = compactIDs(ids)
	if reader == nil || len(ids) == 0 {
		return nil
	}
	found, err := reader.ChildCollections(ctx, ids)
	if err != nil {
		return err
	}
	maps.Copy(into, found)
	return nil
}

// compactIDs drops blank IDs and duplicates.
func compactIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

func trimmedID(id *string) string {
	if id == nil {
		return ""
	}
	return strings.TrimSpace(*id)
}

func (c *ResourceCache) hasChildren(children map[string][]string, id, rel string) bool {
	return c != nil && slices.Contains(children[id], rel)
}

// SystemReferences lists what a page of systems links to: their procedures
// (system kinds), their parents, and their own nested collections.
func SystemReferences(systems []*domains.System) References {
	var refs References
	for _, system := range systems {
		if system == nil {
			continue
		}
		refs.SystemChildren = append(refs.SystemChildren, system.ID)
		refs.Procedures = append(refs.Procedures, trimmedID(system.TypeOfID))
		refs.Systems = append(refs.Systems, trimmedID(system.ParentSystemID))
	}
	return refs
}

// systemChildCollections are the nested collections a system links to when
// they have members.
var systemChildCollections = []string{"subsystems", "samplingFeatures", "deployments", "datastreams", "controlstreams", "procedures"}

// SystemLinks builds a system's links for the given representation: the
// client's own links plus the server-generated association links, typed with
// that representation's media type.
func (l *LinkResolver) SystemLinks(system *domains.System, cache *ResourceCache, contentType string) common_shared.Links {
	if system == nil {
		return nil
	}

	derived := common_shared.Links{}

	if parentID := trimmedID(system.ParentSystemID); parentID != "" {
		link := common_shared.Link{
			Rel:  common_shared.OGCRel("parentSystem"),
			Href: "/systems/" + parentID,
			Type: contentType,
		}
		if cache != nil {
			if parent, ok := cache.Systems[parentID]; ok {
				link.Describe(parent.Name, string(parent.UniqueIdentifier))
			}
		}
		derived = append(derived, link)
	}

	if strings.TrimSpace(system.ID) != "" {
		for _, rel := range systemChildCollections {
			if cache.hasChildren(cache.SystemChildren, system.ID, rel) || (rel == "deployments" && len(system.Deployments) > 0) {
				targetType := contentType
				if rel == "datastreams" || rel == "controlstreams" {
					targetType = mediatype.JSON
				}
				derived = append(derived, common_shared.Link{
					Rel:  common_shared.OGCRel(rel),
					Href: "/systems/" + system.ID + "/" + rel,
					Type: targetType,
				})
			}
		}

		if system.TypeOf != nil {
			link := common_shared.Link{
				Rel:  common_shared.OGCRel("procedures"),
				Href: system.TypeOf.Href,
				Type: contentType,
			}
			if cache != nil {
				if proc, ok := cache.Procedures[trimmedID(system.TypeOfID)]; ok {
					link.Describe(proc.Name, string(proc.UniqueIdentifier))
				}
			}
			derived = append(derived, link)
		}
	}

	return l.withAssociationLinks(system.Links, derived)
}

// DeploymentReferences lists what a page of deployments links to: their
// platforms and deployed systems, and their own nested collections.
func DeploymentReferences(deployments []*domains.Deployment) References {
	var refs References
	for _, deployment := range deployments {
		if deployment == nil {
			continue
		}
		refs.DeploymentChildren = append(refs.DeploymentChildren, deployment.ID)
		if deployment.Platform != nil {
			refs.Systems = append(refs.Systems, trimmedID(deployment.Platform.System.GetId("systems")))
		}
		for _, deployed := range deployment.DeployedSystems {
			refs.Systems = append(refs.Systems, trimmedID(deployed.System.GetId("systems")))
		}
	}
	return refs
}

// DeploymentLinks builds a deployment's links: the client's own links plus
// its parent and subdeployments.
func (l *LinkResolver) DeploymentLinks(deployment *domains.Deployment, cache *ResourceCache) common_shared.Links {
	if deployment == nil {
		return nil
	}

	derived := common_shared.Links{}

	if parentID := trimmedID(deployment.ParentDeploymentID); parentID != "" {
		derived = append(derived, common_shared.Link{
			Rel:  common_shared.OGCRel("parentDeployment"),
			Href: "/deployments/" + parentID,
			Type: mediatype.GeoJSON,
		})
	}

	if strings.TrimSpace(deployment.ID) != "" {
		for _, rel := range []string{"subdeployments", "samplingFeatures", "featuresOfInterest", "datastreams", "controlstreams"} {
			if cache.hasChildren(cache.DeploymentChildren, deployment.ID, rel) {
				targetType := mediatype.GeoJSON
				if rel == "datastreams" || rel == "controlstreams" {
					targetType = mediatype.JSON
				}
				derived = append(derived, common_shared.Link{Rel: common_shared.OGCRel(rel), Href: "/deployments/" + deployment.ID + "/" + rel, Type: targetType})
			}
		}
	}

	return l.withAssociationLinks(deployment.Links, derived)
}

// ProcedureReferences lists the procedures whose implementing systems are
// checked for.
func ProcedureReferences(procedures []*domains.Procedure) References {
	var refs References
	for _, procedure := range procedures {
		if procedure != nil {
			refs.ProcedureChildren = append(refs.ProcedureChildren, procedure.ID)
		}
	}
	return refs
}

// ProcedureLinks builds a procedure's links: the client's own links plus the
// systems that implement it.
func (l *LinkResolver) ProcedureLinks(procedure *domains.Procedure, cache *ResourceCache) common_shared.Links {
	if procedure == nil {
		return nil
	}

	derived := common_shared.Links{}
	if strings.TrimSpace(procedure.ID) != "" && cache.hasChildren(cache.ProcedureChildren, procedure.ID, "implementingSystems") {
		derived = append(derived, common_shared.Link{
			Rel:  common_shared.OGCRel("implementingSystems"),
			Href: "/systems?procedure=" + url.QueryEscape(procedure.ID) + "&recursive=true",
			Type: mediatype.GeoJSON,
		})
	}

	return l.withAssociationLinks(procedure.Links, derived)
}

// SamplingFeatureLinks builds a sampling feature's links: the client's own
// links plus its parent system and the features it is a sample of.
func (l *LinkResolver) SamplingFeatureLinks(sf *domains.SamplingFeature) common_shared.Links {
	if sf == nil {
		return nil
	}

	derived := common_shared.Links{}

	if parentID := trimmedID(sf.ParentSystemID); parentID != "" {
		link := common_shared.Link{
			Rel:  common_shared.OGCRel("parentSystem"),
			Href: "/systems/" + parentID,
			Type: mediatype.GeoJSON,
		}
		if sf.ParentSystemUID != nil {
			link.UID = sf.ParentSystemUID
		}
		derived = append(derived, link)
	}

	// sampleOf is a body property, so whatever rel the client gave its links
	// is irrelevant: they are always published as ogc-rel:sampleOf.
	if sf.SampleOf != nil {
		for _, link := range *sf.SampleOf {
			link.Rel = common_shared.OGCRel("sampleOf")
			if link.Type == "" {
				link.Type = mediatype.GeoJSON
			}
			derived = append(derived, link)
		}
	}

	return l.withAssociationLinks(sf.Links, derived)
}

// withAssociationLinks returns the client's own links followed by the
// server-generated association links, with absolute hrefs and without
// duplicates. Association links a client sent are never stored, so the
// derived ones are the only association links published.
func (l *LinkResolver) withAssociationLinks(own, derived common_shared.Links) common_shared.Links {
	out := common_shared.Links{}
	seen := make(map[string]struct{})

	add := func(link common_shared.Link, association bool) {
		if strings.TrimSpace(link.Href) == "" {
			return
		}
		canonicalRel := common_shared.CanonicalRel(link.Rel)
		if association {
			link.Rel = common_shared.OGCRel(canonicalRel)
			link.Href = l.Href(link.Href)
		}
		key := canonicalRel + "|" + link.Href
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, link)
	}

	for _, link := range common_shared.StripAssociationLinks(own) {
		add(link, false)
	}
	for _, link := range derived {
		add(link, true)
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

// DeploymentSystems returns the recursive read projection without altering
// the deployment's directly owned membership records.
func (c *ResourceCache) DeployedSystems(d *domains.Deployment) domains.DeployedSystemItems {
	if c != nil && c.DeploymentSystems != nil {
		return c.DeploymentSystems[d.ID]
	}
	return d.DeployedSystems
}

// LocalID returns an ID only for a canonical resource in this API namespace.
// An external resource with the same final path segment remains external.
func (l *LinkResolver) LocalID(link common_shared.Link, collection string) *string {
	parsed, err := url.Parse(link.Href)
	if err != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil
	}
	if parsed.IsAbs() || parsed.Host != "" {
		if l == nil || l.BaseURL == "" {
			return nil
		}
		base, err := url.Parse(l.BaseURL)
		if err != nil || parsed.Host != base.Host || parsed.Scheme != base.Scheme {
			return nil
		}
		prefix := strings.TrimRight(base.Path, "/")
		if prefix != "" && !strings.HasPrefix(parsed.Path, prefix+"/") {
			return nil
		}
		parsed.Path = strings.TrimPrefix(parsed.Path, prefix)
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != collection || parts[1] == "" {
		return nil
	}
	return &parts[1]
}
