package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// CollectionItemsHandler is the non-redirecting Features collection facade for
// canonical Connected Systems resources. The typed handlers remain the single
// authority for serialisation, legacy parameters and repository access.
type CollectionItemsHandler struct {
	features  *FeatureHandler
	canonical map[string]http.HandlerFunc
}

func NewCollectionItemsHandler(features *FeatureHandler, canonical map[string]http.HandlerFunc) *CollectionItemsHandler {
	return &CollectionItemsHandler{features: features, canonical: canonical}
}

func (h *CollectionItemsHandler) List(w http.ResponseWriter, r *http.Request) {
	if handler, ok := h.canonical[chi.URLParam(r, "collectionId")]; ok {
		handler(w, r)
		return
	}
	h.features.ListFeatures(w, r)
}

// Head returns the collection response headers without a representation body.
//
// @Summary     Head collection items
// @Description Returns response headers for canonical or generic collection items
// @Tags        Features
// @Param       collectionId path string true "Collection ID"
// @Param       filter query string false "CQL2 Text filter (default) or URL-encoded CQL2 JSON"
// @Param       filter-lang query string false "CQL2 language: cql2-text or cql2-json"
// @Success     200
// @Failure     400 {object} map[string]string
// @Router      /collections/{collectionId}/items [head]
func (h *CollectionItemsHandler) Head(w http.ResponseWriter, r *http.Request) {
	head(h.List)(w, r)
}
