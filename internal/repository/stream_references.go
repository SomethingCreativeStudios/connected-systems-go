package repository

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"gorm.io/gorm"
)

type linkBaseURLKey struct{}

func WithLinkBaseURL(ctx context.Context, base string) context.Context {
	return context.WithValue(ctx, linkBaseURLKey{}, base)
}

// localLinkPath recognizes this API's namespace. External links are retained
// as URLs and never converted into local foreign keys or fetched remotely.
func localLinkPath(ctx context.Context, link *common_shared.Link) (string, bool, error) {
	if link == nil {
		return "", false, nil
	}
	u, err := url.Parse(strings.TrimSpace(link.Href))
	if err != nil || u.String() == "" {
		return "", false, invalidReference("link href must be a valid URI")
	}
	if u.IsAbs() || u.Host != "" {
		base, _ := url.Parse(baseURLFromContext(ctx))
		if base == nil || base.Host == "" || !strings.EqualFold(u.Host, base.Host) || u.Scheme != base.Scheme {
			return "", false, nil
		}
		prefix := strings.TrimRight(base.Path, "/")
		if prefix != "" && !strings.HasPrefix(u.Path, prefix+"/") {
			return "", false, nil
		}
		u.Path = strings.TrimPrefix(u.Path, prefix)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", true, invalidReference("local resource links cannot contain a query or fragment")
	}
	return strings.Trim(u.Path, "/"), true, nil
}
func baseURLFromContext(ctx context.Context) string {
	s, _ := ctx.Value(linkBaseURLKey{}).(string)
	return s
}

func resolveResourceLink(ctx context.Context, tx *gorm.DB, link *common_shared.Link, role string, model any) (*string, error) {
	path, local, err := localLinkPath(ctx, link)
	if err != nil {
		return nil, err
	}
	if !local {
		return nil, nil
	}
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] != role || parts[1] == "" {
		return nil, invalidReference("link must identify a %s resource", role)
	}
	exists, err := hasRows(tx, model, "id = ?", parts[1])
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, invalidReference("%s link target %q does not exist", role, parts[1])
	}
	return &parts[1], nil
}
func resolveFOILink(ctx context.Context, tx *gorm.DB, link *common_shared.Link) (*string, error) {
	path, local, err := localLinkPath(ctx, link)
	if err != nil || !local {
		return nil, err
	}
	parts := strings.Split(path, "/")
	if len(parts) == 4 && parts[0] == "collections" && parts[2] == "items" {
		exists, err := hasRows(tx, &domains.Feature{}, "collection_id = ? AND id = ?", parts[1], parts[3])
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, invalidReference("featureOfInterest link target does not exist")
		}
		return &parts[3], nil
	}
	if len(parts) == 2 {
		models := map[string]any{"systems": &domains.System{}, "samplingFeatures": &domains.SamplingFeature{}, "features": &domains.Feature{}, "deployments": &domains.Deployment{}, "procedures": &domains.Procedure{}}
		if model := models[parts[0]]; model != nil {
			id, err := resolveResourceLink(ctx, tx, link, parts[0], model)
			if err != nil {
				return nil, err
			}
			if parts[0] == "features" {
				var feature domains.Feature
				if err := tx.Select("id", "collection_id").Where("id = ?", *id).First(&feature).Error; err != nil {
					return nil, err
				}
				link.Href = "/collections/" + url.PathEscape(feature.CollectionID) + "/items/" + url.PathEscape(feature.ID)
			}
			return id, nil
		}
	}
	return nil, invalidReference("featureOfInterest link must identify a local feature resource")
}

func resolveStreamReferences(ctx context.Context, tx *gorm.DB, procedure, deployment, sampling, foi *common_shared.Link) (procID, depID, sfID, foiID *string, err error) {
	if procID, err = resolveResourceLink(ctx, tx, procedure, "procedures", &domains.Procedure{}); err != nil {
		return
	}
	if depID, err = resolveResourceLink(ctx, tx, deployment, "deployments", &domains.Deployment{}); err != nil {
		return
	}
	if sfID, err = resolveResourceLink(ctx, tx, sampling, "samplingFeatures", &domains.SamplingFeature{}); err != nil {
		return
	}
	foiID, err = resolveFOILink(ctx, tx, foi)
	return
}

func invalidReference(format string, args ...any) error {
	return &ValidationError{Err: fmt.Errorf(format, args...)}
}
