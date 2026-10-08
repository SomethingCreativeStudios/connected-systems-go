package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/queryables"
	"github.com/yourusername/connected-systems-go/internal/repository"
)

// queryablesMiddleware validates CQL2 before handlers construct their legacy
// query objects and emits the standard queryables link for every list route.
func queryablesMiddleware(baseURL string, collections *repository.CollectionRepository) func(http.Handler) http.Handler {
	baseURL = strings.TrimRight(baseURL, "/")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resource, ok := queryablesResourceForPath(r.URL.Path)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			queryablesURL := baseURL + "/collections/" + string(resource) + "/queryables"
			if resource == queryables.Features {
				collectionPath := strings.TrimSuffix(strings.TrimRight(r.URL.EscapedPath(), "/"), "/items")
				queryablesURL = baseURL + collectionPath + "/queryables"
			}
			w.Header().Add("Link", "<"+queryablesURL+">; rel=\"http://www.opengis.net/def/rel/ogc/1.0/queryables\"; type=\"application/schema+json\"")
			if filterCRS := r.URL.Query().Get("filter-crs"); filterCRS != "" && filterCRS != "http://www.opengis.net/def/crs/OGC/1.3/CRS84" {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, map[string]string{"error": "filter-crs is limited to CRS84"})
				return
			}
			filter := r.URL.Query().Get("filter")
			if filter == "" {
				next.ServeHTTP(w, r)
				return
			}
			compiled, err := compileRequestCQL(r, resource, filter, collections)
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, map[string]string{"error": "invalid CQL2 filter: " + err.Error()})
				return
			}
			next.ServeHTTP(w, queryables.WithCompiled(r, compiled))
		})
	}
}

func compileRequestCQL(r *http.Request, resource queryables.Resource, filter string, collections *repository.CollectionRepository) (queryables.Compiled, error) {
	if resource == queryables.Features && collections != nil {
		parts := strings.FieldsFunc(strings.Trim(r.URL.Path, "/"), func(r rune) bool { return r == '/' })
		if len(parts) >= 2 && parts[0] == "collections" {
			collection, err := collections.GetCollectionByID(r.Context(), parts[1])
			if err == nil {
				return queryables.CompileWithCatalog(resource, queryables.CatalogOrDefault(collection.Queryables), filter, r.URL.Query().Get("filter-lang"))
			}
		}
	}
	return queryables.Compile(resource, filter, r.URL.Query().Get("filter-lang"))
}

func queryablesResourceForPath(path string) (queryables.Resource, bool) {
	parts := strings.FieldsFunc(strings.Trim(path, "/"), func(r rune) bool { return r == '/' })
	if len(parts) == 0 {
		return "", false
	}
	if parts[0] == "collections" && len(parts) == 3 && parts[2] == "items" {
		if _, ok := canonicalCollectionPaths[parts[1]]; ok {
			return queryables.Resource(parts[1]), true
		}
		return queryables.Features, true
	}
	switch parts[0] {
	case "systems":
		if len(parts) == 1 {
			return queryables.Systems, true
		}
		if len(parts) == 3 {
			switch parts[2] {
			case "deployments":
				return queryables.Deployments, true
			case "procedures":
				return queryables.Procedures, true
			case "samplingFeatures":
				return queryables.SamplingFeatures, true
			case "datastreams":
				return queryables.Datastreams, true
			case "controlstreams":
				return queryables.ControlStreams, true
			case "events":
				return queryables.SystemEvents, true
			case "history":
				return queryables.SystemHistory, true
			case "subsystems":
				return queryables.Systems, true
			}
		}
		return "", false
	case "deployments":
		if len(parts) == 1 || (len(parts) == 3 && parts[2] == "subdeployments") {
			return queryables.Deployments, true
		}
		return "", false
	case "procedures":
		if len(parts) == 1 {
			return queryables.Procedures, true
		}
		return "", false
	case "samplingFeatures":
		if len(parts) == 1 {
			return queryables.SamplingFeatures, true
		}
		return "", false
	case "properties":
		if len(parts) == 1 {
			return queryables.Properties, true
		}
		return "", false
	case "datastreams":
		if len(parts) == 1 {
			return queryables.Datastreams, true
		}
		if len(parts) == 3 && parts[2] == "observations" {
			return queryables.Observations, true
		}
		return "", false
	case "observations":
		if len(parts) == 1 {
			return queryables.Observations, true
		}
		return "", false
	case "controlstreams":
		if len(parts) == 1 {
			return queryables.ControlStreams, true
		}
		if len(parts) == 3 && parts[2] == "commands" {
			return queryables.Commands, true
		}
		return "", false
	case "commands":
		if len(parts) == 1 {
			return queryables.Commands, true
		}
		if len(parts) == 3 && parts[2] == "status" {
			return queryables.CommandStatuses, true
		}
		if len(parts) == 3 && parts[2] == "result" {
			return queryables.CommandResults, true
		}
		return "", false
	case "systemEvents":
		if len(parts) == 1 {
			return queryables.SystemEvents, true
		}
		return "", false
	default:
		return "", false
	}
}

type headResponseWriter struct{ http.ResponseWriter }

func (headResponseWriter) Write(bytes []byte) (int, error) { return len(bytes), nil }

func head(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { handler(headResponseWriter{w}, r) }
}
