package api

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/queryables"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/go-chi/chi/v5"
)

// queryablesMiddleware returns the middleware for a list route of the given
// queryables resource: it validates CQL2 before handlers build their query
// objects and emits the standard queryables link. The router attaches it to
// each list route, so the route table is the only place routes are named.
// queryables.Features stands for /collections/{collectionId}/items, whose
// resource depends on the collection.
func queryablesMiddleware(baseURL string, collections *repository.CollectionRepository) func(queryables.Resource) func(http.Handler) http.Handler {
	baseURL = strings.TrimRight(baseURL, "/")
	return func(resource queryables.Resource) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resource := resource
				queryablesURL := baseURL + "/collections/" + string(resource) + "/queryables"
				collectionID := ""
				if resource == queryables.Features && chi.URLParam(r, "collectionId") != "" {
					collectionID = chi.URLParam(r, "collectionId")
					if _, canonical := canonicalCollectionPaths[collectionID]; canonical {
						resource = queryables.Resource(collectionID)
					}
					queryablesURL = baseURL + "/collections/" + url.PathEscape(collectionID) + "/queryables"
				}
				w.Header().Add("Link", "<"+queryablesURL+">; rel=\"http://www.opengis.net/def/rel/ogc/1.0/queryables\"; type=\"application/schema+json\"")
				if filterCRS := r.URL.Query().Get("filter-crs"); filterCRS != "" && filterCRS != "http://www.opengis.net/def/crs/OGC/1.3/CRS84" {
					writeError(w, r, http.StatusBadRequest, "filter-crs is limited to CRS84")
					return
				}
				filter := r.URL.Query().Get("filter")
				if filter == "" {
					next.ServeHTTP(w, r)
					return
				}
				compiled, err := compileRequestCQL(r, resource, collectionID, filter, collections)
				if err != nil {
					writeError(w, r, http.StatusBadRequest, "invalid CQL2 filter: "+err.Error())
					return
				}
				next.ServeHTTP(w, queryables.WithCompiled(r, compiled))
			})
		}
	}
}

func compileRequestCQL(r *http.Request, resource queryables.Resource, collectionID, filter string, collections *repository.CollectionRepository) (queryables.Compiled, error) {
	if resource == queryables.Features && collections != nil {
		collection, err := collections.GetCollectionByID(r.Context(), collectionID)
		if err == nil {
			return queryables.CompileWithCatalog(resource, queryables.CatalogOrDefault(collection.Queryables), filter, r.URL.Query().Get("filter-lang"))
		}
	}
	return queryables.Compile(resource, filter, r.URL.Query().Get("filter-lang"))
}

type headResponseWriter struct{ http.ResponseWriter }

func (headResponseWriter) Write(bytes []byte) (int, error) { return len(bytes), nil }

func head(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { handler(headResponseWriter{w}, r) }
}
