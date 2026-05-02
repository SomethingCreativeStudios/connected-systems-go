package formaters

import (
	"context"
	"net/url"
	"sort"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

// Content type constants used for association link Type fields.
const (
	GeoJSONContentType  = "application/geo+json"
	SensorMLContentType = "application/sml+json"
	JSONContentType     = "application/json"
)

var associationLinksBaseURL string

// SetAssociationLinksBaseURL configures the absolute base URL used for association link hrefs.
func SetAssociationLinksBaseURL(baseURL string) {
	associationLinksBaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
}

// ResourceCache holds pre-fetched resources for enriching association links.
type ResourceCache struct {
	Systems    map[string]*domains.System
	Procedures map[string]*domains.Procedure
}

// NewResourceCache creates an empty ResourceCache.
func NewResourceCache() *ResourceCache {
	return &ResourceCache{
		Systems:    make(map[string]*domains.System),
		Procedures: make(map[string]*domains.Procedure),
	}
}

// FetchParentSystems loads parent systems for the given system IDs into the cache.
func (c *ResourceCache) FetchParentSystems(ctx context.Context, repo interface {
	GetByIDs(ctx context.Context, ids []string) (map[string]*domains.System, error)
}, systemIDs []string) error {
	if len(systemIDs) == 0 {
		return nil
	}
	systems, err := repo.GetByIDs(ctx, systemIDs)
	if err != nil {
		return err
	}
	for k, v := range systems {
		c.Systems[k] = v
	}
	return nil
}

// FetchProcedures loads procedures for the given IDs into the cache.
func (c *ResourceCache) FetchProcedures(ctx context.Context, repo interface {
	GetByIDs(ctx context.Context, ids []string) (map[string]*domains.Procedure, error)
}, procedureIDs []string) error {
	if len(procedureIDs) == 0 {
		return nil
	}
	procedures, err := repo.GetByIDs(ctx, procedureIDs)
	if err != nil {
		return err
	}
	for k, v := range procedures {
		c.Procedures[k] = v
	}
	return nil
}

var (
	geoJSONSystemAssociationRels = []string{
		common_shared.OGCRel("parentSystem"),
		common_shared.OGCRel("subsystems"),
		common_shared.OGCRel("samplingFeatures"),
		common_shared.OGCRel("deployments"),
		common_shared.OGCRel("procedures"),
		common_shared.OGCRel("datastreams"),
		common_shared.OGCRel("controlstreams"),
	}
	sensorMLSystemAssociationRels = []string{
		common_shared.OGCRel("parentSystem"),
		common_shared.OGCRel("subsystems"),
		common_shared.OGCRel("samplingFeatures"),
		common_shared.OGCRel("deployments"),
		common_shared.OGCRel("procedures"),
		common_shared.OGCRel("datastreams"),
		common_shared.OGCRel("controlstreams"),
	}
	deploymentAssociationRels = []string{
		common_shared.OGCRel("parentDeployment"),
		common_shared.OGCRel("subdeployments"),
		common_shared.OGCRel("featuresOfInterest"),
		common_shared.OGCRel("samplingFeatures"),
		common_shared.OGCRel("datastreams"),
		common_shared.OGCRel("controlstreams"),
	}
	procedureAssociationRels = []string{
		common_shared.OGCRel("implementingSystems"),
	}
	samplingFeatureGeoJSONAssociationRels = []string{
		common_shared.OGCRel("parentSystem"),
		common_shared.OGCRel("sampleOf"),
		common_shared.OGCRel("datastreams"),
		common_shared.OGCRel("controlstreams"),
	}
)

func GeoJSONSystemAssociationLinks(links common_shared.Links) common_shared.Links {
	return resourceAssociationLinks(links, geoJSONSystemAssociationRels)
}

func DeploymentAssociationLinks(links common_shared.Links) common_shared.Links {
	return resourceAssociationLinks(links, deploymentAssociationRels)
}

func SamplingFeatureGeoJSONAssociationLinks(links common_shared.Links) common_shared.Links {
	return resourceAssociationLinks(links, samplingFeatureGeoJSONAssociationRels)
}

func AppendGeoJSONSystemAssociationLinks(system *domains.System, cache *ResourceCache) common_shared.Links {
	if system == nil {
		return nil
	}

	derived := common_shared.Links{}

	if system.ParentSystemID != nil && strings.TrimSpace(*system.ParentSystemID) != "" {
		parentID := strings.TrimSpace(*system.ParentSystemID)
		link := common_shared.Link{
			Rel:  common_shared.OGCRel("parentSystem"),
			Href: "/systems/" + parentID,
			Type: GeoJSONContentType,
		}
		if cache != nil {
			if parent, ok := cache.Systems[parentID]; ok {
				link.Title = parent.Name
				uid := string(parent.UniqueIdentifier)
				if uid != "" {
					link.UID = &uid
				}
			}
		}
		derived = append(derived, link)
	}

	if strings.TrimSpace(system.ID) != "" {
		if hasAssociationLink(system.Links, "subsystems") {
			derived = append(derived, common_shared.Link{
				Rel:  common_shared.OGCRel("subsystems"),
				Href: "/systems/" + system.ID + "/subsystems",
				Type: GeoJSONContentType,
			})
		}
		if hasAssociationLink(system.Links, "samplingFeatures") {
			derived = append(derived, common_shared.Link{
				Rel:  common_shared.OGCRel("samplingFeatures"),
				Href: "/systems/" + system.ID + "/samplingFeatures",
				Type: GeoJSONContentType,
			})
		}
		if len(system.Deployments) > 0 || hasAssociationLink(system.Links, "deployments") {
			derived = append(derived, common_shared.Link{
				Rel:  common_shared.OGCRel("deployments"),
				Href: "/systems/" + system.ID + "/deployments",
				Type: GeoJSONContentType,
			})
		}
		if hasAssociationLink(system.Links, "datastreams") {
			derived = append(derived, common_shared.Link{
				Rel:  common_shared.OGCRel("datastreams"),
				Href: "/systems/" + system.ID + "/datastreams",
				Type: GeoJSONContentType,
			})
		}
		if hasAssociationLink(system.Links, "controlstreams") {
			derived = append(derived, common_shared.Link{
				Rel:  common_shared.OGCRel("controlstreams"),
				Href: "/systems/" + system.ID + "/controlstreams",
				Type: GeoJSONContentType,
			})
		}

		if system.TypeOf != nil {
			link := common_shared.Link{
				Rel:  common_shared.OGCRel("procedures"),
				Href: system.TypeOf.Href,
				Type: GeoJSONContentType,
			}
			// Enrich from cache if we have the procedure
			if cache != nil && system.TypeOfID != nil {
				if proc, ok := cache.Procedures[*system.TypeOfID]; ok {
					link.Title = proc.Name
					uid := string(proc.UniqueIdentifier)
					if uid != "" {
						link.UID = &uid
					}
				}
			}
			derived = append(derived, link)
		}
	}

	return mergeAssociationLinks(common_shared.StripAssociationLinks(system.Links), geoJSONSystemAssociationRels, derived)
}

func AppendSensorMLSystemAssociationLinks(system *domains.System, cache *ResourceCache) common_shared.Links {
	if system == nil {
		return nil
	}

	derived := common_shared.Links{}

	if system.ParentSystemID != nil && strings.TrimSpace(*system.ParentSystemID) != "" {
		parentID := strings.TrimSpace(*system.ParentSystemID)
		link := common_shared.Link{
			Rel:  common_shared.OGCRel("parentSystem"),
			Href: "/systems/" + parentID,
			Type: SensorMLContentType,
		}
		if cache != nil {
			if parent, ok := cache.Systems[parentID]; ok {
				link.Title = parent.Name
				uid := string(parent.UniqueIdentifier)
				if uid != "" {
					link.UID = &uid
				}
			}
		}
		derived = append(derived, link)
	}

	if strings.TrimSpace(system.ID) != "" {
		if hasAssociationLink(system.Links, "subsystems") {
			derived = append(derived, common_shared.Link{
				Rel:  common_shared.OGCRel("subsystems"),
				Href: "/systems/" + system.ID + "/subsystems",
				Type: SensorMLContentType,
			})
		}
		if hasAssociationLink(system.Links, "samplingFeatures") {
			derived = append(derived, common_shared.Link{
				Rel:  common_shared.OGCRel("samplingFeatures"),
				Href: "/systems/" + system.ID + "/samplingFeatures",
				Type: SensorMLContentType,
			})
		}
		if len(system.Deployments) > 0 || hasAssociationLink(system.Links, "deployments") {
			derived = append(derived, common_shared.Link{
				Rel:  common_shared.OGCRel("deployments"),
				Href: "/systems/" + system.ID + "/deployments",
				Type: SensorMLContentType,
			})
		}
		if hasAssociationLink(system.Links, "datastreams") {
			derived = append(derived, common_shared.Link{
				Rel:  common_shared.OGCRel("datastreams"),
				Href: "/systems/" + system.ID + "/datastreams",
				Type: SensorMLContentType,
			})
		}
		if hasAssociationLink(system.Links, "controlstreams") {
			derived = append(derived, common_shared.Link{
				Rel:  common_shared.OGCRel("controlstreams"),
				Href: "/systems/" + system.ID + "/controlstreams",
				Type: SensorMLContentType,
			})
		}

		if system.TypeOf != nil {
			link := common_shared.Link{
				Rel:  common_shared.OGCRel("procedures"),
				Href: system.TypeOf.Href,
				Type: SensorMLContentType,
			}
			if cache != nil && system.TypeOfID != nil {
				if proc, ok := cache.Procedures[*system.TypeOfID]; ok {
					link.Title = proc.Name
					uid := string(proc.UniqueIdentifier)
					if uid != "" {
						link.UID = &uid
					}
				}
			}
			derived = append(derived, link)
		}
	}

	return mergeAssociationLinks(common_shared.StripAssociationLinks(system.Links), sensorMLSystemAssociationRels, derived)
}

func AppendDeploymentAssociationLinks(deployment *domains.Deployment) common_shared.Links {
	if deployment == nil {
		return nil
	}

	derived := common_shared.Links{}

	if deployment.ParentDeploymentID != nil && strings.TrimSpace(*deployment.ParentDeploymentID) != "" {
		derived = append(derived, common_shared.Link{
			Rel:  common_shared.OGCRel("parentDeployment"),
			Href: "/deployments/" + strings.TrimSpace(*deployment.ParentDeploymentID),
			Type: GeoJSONContentType,
		})
	}

	if strings.TrimSpace(deployment.ID) != "" && hasAssociationLink(deployment.Links, "subdeployments") {
		derived = append(derived, common_shared.Link{
			Rel:  common_shared.OGCRel("subdeployments"),
			Href: "/deployments/" + deployment.ID + "/subdeployments",
			Type: GeoJSONContentType,
		})
	}

	return mergeAssociationLinks(common_shared.StripAssociationLinks(deployment.Links), deploymentAssociationRels, derived)
}

func AppendProcedureAssociationLinks(procedure *domains.Procedure) common_shared.Links {
	if procedure == nil {
		return nil
	}

	derived := common_shared.Links{}
	if strings.TrimSpace(procedure.ID) != "" && (len(procedure.Systems) > 0 || hasAssociationLink(procedure.Links, "implementingSystems")) {
		derived = append(derived, common_shared.Link{
			Rel:  common_shared.OGCRel("implementingSystems"),
			Href: "/systems?procedure=" + url.QueryEscape(procedure.ID),
			Type: GeoJSONContentType,
		})
	}

	return mergeAssociationLinks(common_shared.StripAssociationLinks(procedure.Links), procedureAssociationRels, derived)
}

func AppendSamplingFeatureGeoJSONAssociationLinks(sf *domains.SamplingFeature) common_shared.Links {
	if sf == nil {
		return nil
	}

	derived := common_shared.Links{}

	if sf.ParentSystemID != nil && strings.TrimSpace(*sf.ParentSystemID) != "" {
		link := common_shared.Link{
			Rel:  common_shared.OGCRel("parentSystem"),
			Href: "/systems/" + strings.TrimSpace(*sf.ParentSystemID),
			Type: GeoJSONContentType,
		}
		if sf.ParentSystemUID != nil {
			link.UID = sf.ParentSystemUID
		}
		derived = append(derived, link)
	}

	if sf.SampleOf != nil {
		for _, l := range *sf.SampleOf {
			link := l
			if link.Type == "" {
				link.Type = GeoJSONContentType
			}
			derived = append(derived, link)
		}
	}

	return mergeAssociationLinks(common_shared.StripAssociationLinks(sf.Links), samplingFeatureGeoJSONAssociationRels, derived)
}

func ApplyGeoJSONSystemAssociationLinks(system *domains.System, links common_shared.Links) {
	if system == nil {
		return
	}

	for _, link := range links {
		if common_shared.RelEquals(link.Rel, common_shared.OGCRel("parentSystem")) {
			system.ParentSystemID = link.GetId("systems")
		}
	}
}

func ApplyDeploymentAssociationLinks(deployment *domains.Deployment, links common_shared.Links) {
	if deployment == nil {
		return
	}

	for _, link := range links {
		if common_shared.RelEquals(link.Rel, common_shared.OGCRel("parentDeployment")) {
			deployment.ParentDeploymentID = link.GetId("deployments")
		}
	}
}

func ApplySamplingFeatureGeoJSONAssociationLinks(sf *domains.SamplingFeature, links common_shared.Links) {
	if sf == nil || len(links) == 0 {
		return
	}

	sampleIDs := []string{}
	sampleUIDs := []string{}
	sampleOfLinks := common_shared.Links{}

	for _, link := range links {
		if common_shared.RelEquals(link.Rel, common_shared.OGCRel("parentSystem")) {
			sf.ParentSystemID = link.GetId("systems")
			if link.UID != nil {
				sf.ParentSystemUID = link.UID
			}
			continue
		}

		if common_shared.RelEquals(link.Rel, common_shared.OGCRel("sampleOf")) {
			sampleOfLinks = append(sampleOfLinks, link)
			if id := link.GetId("samplingFeatures"); id != nil {
				sampleIDs = append(sampleIDs, *id)
			}
			if link.UID != nil {
				sampleUIDs = append(sampleUIDs, *link.UID)
			}
		}
	}

	if len(sampleOfLinks) > 0 {
		sf.SampleOf = &sampleOfLinks
	}
	if len(sampleIDs) > 0 {
		sf.SampleOfIDs = &sampleIDs
	}
	if len(sampleUIDs) > 0 {
		sf.SampleOfUIDs = &sampleUIDs
	}
}

func mergeAssociationLinks(existing common_shared.Links, allowedRels []string, derived common_shared.Links) common_shared.Links {
	out := common_shared.Links{}
	seen := make(map[string]struct{})
	allowed := make(map[string]struct{}, len(allowedRels))

	for _, rel := range allowedRels {
		allowed[common_shared.CanonicalRel(rel)] = struct{}{}
	}

	add := func(link common_shared.Link, association bool) {
		if strings.TrimSpace(link.Href) == "" {
			return
		}

		canonicalRel := common_shared.CanonicalRel(link.Rel)
		if association {
			if _, ok := allowed[canonicalRel]; !ok {
				return
			}
			link.Rel = common_shared.OGCRel(canonicalRel)
			link.Href = ToFunctionalAssociationHref(link.Href)
		}

		key := canonicalRel + "|" + link.Href
		if _, ok := seen[key]; ok {
			return
		}

		seen[key] = struct{}{}
		out = append(out, link)
	}

	for _, link := range common_shared.StripAssociationLinks(existing) {
		add(link, false)
	}

	for _, link := range resourceAssociationLinks(existing, allowedRels) {
		add(link, true)
	}

	for _, link := range derived {
		add(link, true)
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func resourceAssociationLinks(links common_shared.Links, allowedRels []string) common_shared.Links {
	if len(links) == 0 {
		return nil
	}
	return links.FilterByRels(allowedRels, true)
}

func buildProceduresEndpointHref(procedures []domains.Procedure) string {
	if len(procedures) == 0 {
		return ""
	}

	ids := make([]string, 0, len(procedures))
	seen := make(map[string]struct{}, len(procedures))

	for _, procedure := range procedures {
		id := strings.TrimSpace(procedure.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return ""
	}

	sort.Strings(ids)
	return "/procedures?id=" + url.QueryEscape(strings.Join(ids, ","))
}

func hasAssociationLink(links common_shared.Links, rel string) bool {
	for _, link := range links {
		if common_shared.RelEquals(link.Rel, common_shared.OGCRel(rel)) {
			return true
		}
	}
	return false
}

func ToFunctionalAssociationHref(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return href
	}

	if parsed, err := url.Parse(href); err == nil && parsed.IsAbs() {
		return href
	}

	if associationLinksBaseURL == "" {
		return href
	}

	if strings.HasPrefix(href, "/") {
		return associationLinksBaseURL + href
	}

	return associationLinksBaseURL + "/" + href
}
