package pubsub

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// ChangeResolver resolves a canonical API resource path to the metadata needed
// by a Resource Event. Resolution happens after create/update and before delete
// so parent information is still available for deletions.
type ChangeResolver interface {
	Resolve(ctx context.Context, path string, operation Operation) (Change, error)
}

type publishedKey struct{}

// MarkPublished tells HTTPResourceEventMiddleware that the handler already
// published the request's changes, so it does not resolve them again.
func MarkPublished(ctx context.Context) {
	if published, ok := ctx.Value(publishedKey{}).(*bool); ok {
		*published = true
	}
}

// HTTPResourceEventMiddleware emits individual or Batch Resource Events after
// successful HTTP mutations. Repository methods have completed by the time
// publication occurs.
func HTTPResourceEventMiddleware(publisher *Publisher, resolver ChangeResolver, logger *zap.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			operation, mutation := operationForMethod(r.Method)
			if !mutation || publisher == nil || !publisher.Enabled() || resolver == nil {
				next.ServeHTTP(w, r)
				return
			}

			var change Change
			var resolveErr error
			if operation == OperationDelete {
				change, resolveErr = resolver.Resolve(r.Context(), r.URL.EscapedPath(), operation)
			}

			published := false
			r = r.WithContext(context.WithValue(r.Context(), publishedKey{}, &published))
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			if published || ww.Status() < 200 || ww.Status() >= 300 {
				return
			}

			if operation != OperationDelete {
				resourcePath := r.URL.EscapedPath()
				if operation == OperationCreate {
					resourcePath = ww.Header().Get("Location")
				}
				change, resolveErr = resolver.Resolve(r.Context(), resourcePath, operation)
			}
			if resolveErr != nil {
				logger.Debug("Pub/Sub Resource Event skipped for unsupported or unresolved resource",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Error(resolveErr),
				)
				return
			}
			publisher.PublishChange(change)
		})
	}
}

func operationForMethod(method string) (Operation, bool) {
	switch method {
	case http.MethodPost:
		return OperationCreate, true
	case http.MethodPut, http.MethodPatch:
		return OperationUpdate, true
	case http.MethodDelete:
		return OperationDelete, true
	default:
		return "", false
	}
}

// RepositoryChangeResolver maps the currently supported Pub/Sub Resource Event
// resource types to their canonical URLs and parent resources. The draft's
// event-type mapping annex is still TODO, so this deliberately implements the
// concrete resource tokens listed in Table 2 and advertises that exact set.
type RepositoryChangeResolver struct {
	repos       *repository.Repositories
	apiBasePath string
	represent   Representer
}

// Representer encodes a resolved resource in each representation advertised
// for its resource type, default first. It returns none for resource types
// that carry no Resource Data.
type Representer func(ctx context.Context, resourceType string, resource any) ([]Representation, error)

// NewRepositoryChangeResolver resolves changes against repos. A nil represent
// resolves identities and topics without Resource Data representations.
func NewRepositoryChangeResolver(apiRoot string, repos *repository.Repositories, represent Representer) *RepositoryChangeResolver {
	basePath := ""
	if parsed, err := url.Parse(apiRoot); err == nil {
		basePath = strings.TrimRight(parsed.EscapedPath(), "/")
	}
	return &RepositoryChangeResolver{repos: repos, apiBasePath: basePath, represent: represent}
}

func (r *RepositoryChangeResolver) Resolve(ctx context.Context, rawPath string, operation Operation) (Change, error) {
	change, resource, err := r.resolve(ctx, rawPath, operation)
	if err != nil || operation == OperationDelete || r.represent == nil {
		return change, err
	}
	change.Representations, err = r.represent(ctx, change.ResourceType, resource)
	if err != nil {
		return Change{}, err
	}
	return change, nil
}

// ResolveIdentity resolves a resource's identity, ownership and topic without
// serializing its representations.
func (r *RepositoryChangeResolver) ResolveIdentity(ctx context.Context, rawPath string) (Change, error) {
	change, _, err := r.resolve(ctx, rawPath, OperationCreate)
	return change, err
}

func (r *RepositoryChangeResolver) resolve(ctx context.Context, rawPath string, operation Operation) (Change, any, error) {
	if r == nil || r.repos == nil {
		return Change{}, nil, fmt.Errorf("resource resolver is not configured")
	}
	resourcePath, err := r.relativeResourcePath(rawPath)
	if err != nil {
		return Change{}, nil, err
	}
	parts := splitPath(resourcePath)
	change := Change{Operation: operation, SubjectPath: resourcePath}
	var resolvedResource any

	switch {
	case len(parts) == 2 && parts[0] == "systems":
		resource, err := r.repos.System.GetByID(ctx, parts[1])
		if err != nil {
			return Change{}, nil, err
		}
		resolvedResource = resource
		change.ResourceType = "system"
		change.ResourceID = resource.ID
		change.CollectionPath = "/systems"
		if resource.ParentSystemID != nil && *resource.ParentSystemID != "" {
			change.ParentPath = "/systems/" + url.PathEscape(*resource.ParentSystemID)
			change.CollectionPath = change.ParentPath + "/subsystems"
		}

	case len(parts) == 2 && parts[0] == "deployments":
		resource, err := r.repos.Deployment.GetByID(ctx, parts[1])
		if err != nil {
			return Change{}, nil, err
		}
		resolvedResource = resource
		change.ResourceType = "deployment"
		change.ResourceID = resource.ID
		change.CollectionPath = "/deployments"
		if resource.ParentDeploymentID != nil && *resource.ParentDeploymentID != "" {
			change.ParentPath = "/deployments/" + url.PathEscape(*resource.ParentDeploymentID)
		}

	case len(parts) == 2 && parts[0] == "procedures":
		resource, err := r.repos.Procedure.GetByID(ctx, parts[1])
		if err != nil {
			return Change{}, nil, err
		}
		resolvedResource = resource
		change.ResourceType, change.ResourceID = "procedure", resource.ID
		change.CollectionPath = "/procedures"

	case len(parts) == 2 && parts[0] == "properties":
		resource, err := r.repos.Property.GetByID(ctx, parts[1])
		if err != nil {
			return Change{}, nil, err
		}
		resolvedResource = resource
		change.ResourceType, change.ResourceID = "property", resource.ID
		change.CollectionPath = "/properties"

	case len(parts) == 2 && parts[0] == "samplingFeatures":
		resource, err := r.repos.SamplingFeature.GetByID(ctx, parts[1])
		if err != nil {
			return Change{}, nil, err
		}
		resolvedResource = resource
		change.ResourceType, change.ResourceID = "samplingfeature", resource.ID
		change.CollectionPath = "/samplingFeatures"
		if resource.ParentSystemID != nil && *resource.ParentSystemID != "" {
			change.ParentPath = "/systems/" + url.PathEscape(*resource.ParentSystemID)
			change.CollectionPath = change.ParentPath + "/samplingFeatures"
		}

	case len(parts) == 2 && parts[0] == "datastreams":
		resource, err := r.repos.Datastream.GetByID(ctx, parts[1])
		if err != nil {
			return Change{}, nil, err
		}
		resolvedResource = resource
		change.ResourceType, change.ResourceID = "datastream", resource.ID
		change.CollectionPath = "/datastreams"
		if resource.SystemID != nil && *resource.SystemID != "" {
			change.ParentPath = "/systems/" + url.PathEscape(*resource.SystemID)
			change.CollectionPath = change.ParentPath + "/datastreams"
		}

	case len(parts) == 2 && parts[0] == "observations":
		resource, err := r.repos.Observation.GetByID(ctx, parts[1])
		if err != nil {
			return Change{}, nil, err
		}
		resolvedResource = resource
		change.ResourceType, change.ResourceID = "observation", resource.ID
		change.ParentPath = "/datastreams/" + url.PathEscape(resource.DatastreamID)
		change.CollectionPath = change.ParentPath + "/observations"

	case len(parts) == 2 && parts[0] == "controlstreams":
		resource, err := r.repos.ControlStream.GetByID(ctx, parts[1])
		if err != nil {
			return Change{}, nil, err
		}
		resolvedResource = resource
		change.ResourceType, change.ResourceID = "controlstream", resource.ID
		change.CollectionPath = "/controlstreams"
		if resource.SystemID != nil && *resource.SystemID != "" {
			change.ParentPath = "/systems/" + url.PathEscape(*resource.SystemID)
			change.CollectionPath = change.ParentPath + "/controlstreams"
		}

	case len(parts) == 2 && parts[0] == "commands":
		resource, err := r.repos.Command.GetByID(ctx, parts[1])
		if err != nil {
			return Change{}, nil, err
		}
		resolvedResource = resource
		change.ResourceType, change.ResourceID = "command", resource.ID
		change.ParentPath = "/controlstreams/" + url.PathEscape(resource.ControlStreamID)
		change.CollectionPath = change.ParentPath + "/commands"

	case len(parts) == 4 && parts[0] == "commands" && parts[2] == "status":
		resource, err := r.repos.Command.GetStatusByID(ctx, parts[1], parts[3])
		if err != nil {
			return Change{}, nil, err
		}
		resolvedResource = resource
		change.ResourceType, change.ResourceID = "commandstatus", resource.ID
		change.ParentPath = "/commands/" + url.PathEscape(parts[1])
		change.CollectionPath = change.ParentPath + "/status"

	case len(parts) == 4 && parts[0] == "commands" && parts[2] == "result":
		resource, err := r.repos.Command.GetResultByID(ctx, parts[1], parts[3])
		if err != nil {
			return Change{}, nil, err
		}
		resolvedResource = resource
		change.ResourceType, change.ResourceID = "commandresult", resource.ID
		change.ParentPath = "/commands/" + url.PathEscape(parts[1])
		change.CollectionPath = change.ParentPath + "/result"

	case len(parts) == 4 && parts[0] == "systems" && parts[2] == "events":
		resource, err := r.repos.SystemEvent.GetByID(ctx, parts[1], parts[3])
		if err != nil {
			return Change{}, nil, err
		}
		resolvedResource = resource
		change.ResourceType, change.ResourceID = "systemevent", resource.ID
		change.ParentPath = "/systems/" + url.PathEscape(parts[1])
		change.CollectionPath = change.ParentPath + "/events"

	default:
		return Change{}, nil, fmt.Errorf("resource path %q is not in the advertised Pub/Sub Resource Event set", resourcePath)
	}

	// Emit the canonical HTTP spelling even when the request used a URI escape alias.
	canonical := append([]string(nil), parts...)
	for i := 1; i < len(canonical); i += 2 {
		canonical[i] = url.PathEscape(canonical[i])
	}
	change.SubjectPath = "/" + strings.Join(canonical, "/")
	change.Data = summaryForResolvedResource(resolvedResource)
	change.TopicPath, err = r.topicPath(ctx, change)
	if err != nil {
		return Change{}, nil, err
	}
	change.CollectionTopicPath = collectionTopic(change.TopicPath)
	return change, resolvedResource, nil
}

func summaryForResolvedResource(resource any) map[string]any {
	switch resource := resource.(type) {
	case *domains.System:
		return BuildResourceEventSummary(resource.Name, resource.Description, string(resource.UniqueIdentifier))
	case *domains.Deployment:
		return BuildResourceEventSummary(resource.Name, resource.Description, string(resource.UniqueIdentifier))
	case *domains.Procedure:
		return BuildResourceEventSummary(resource.Name, resource.Description, string(resource.UniqueIdentifier))
	case *domains.Property:
		return BuildResourceEventSummary(resource.Name, resource.Description, string(resource.UniqueIdentifier))
	case *domains.SamplingFeature:
		return BuildResourceEventSummary(resource.Name, resource.Description, string(resource.UniqueIdentifier))
	case *domains.Datastream:
		return BuildResourceEventSummary(resource.Name, resource.Description, "")
	case *domains.ControlStream:
		return BuildResourceEventSummary(resource.Name, resource.Description, string(resource.UniqueIdentifier))
	case *domains.SystemEvent:
		// System Events call their human-readable name "label" in Part 2.
		// Resource Event summaries normalize it to the Part 3 "name" field.
		return BuildResourceEventSummary(resource.Label, resource.Description, "")
	default:
		return nil
	}
}

func (r *RepositoryChangeResolver) relativeResourcePath(rawPath string) (string, error) {
	if strings.TrimSpace(rawPath) == "" {
		return "", fmt.Errorf("resource path is empty")
	}
	parsed, err := url.Parse(rawPath)
	if err != nil {
		return "", fmt.Errorf("parse resource path: %w", err)
	}
	path := parsed.EscapedPath()
	if r.apiBasePath != "" && (path == r.apiBasePath || strings.HasPrefix(path, r.apiBasePath+"/")) {
		path = strings.TrimPrefix(path, r.apiBasePath)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/"), nil
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "/")
	for i, part := range parts {
		if decoded, err := url.PathUnescape(part); err == nil {
			parts[i] = decoded
		}
	}
	return parts
}
