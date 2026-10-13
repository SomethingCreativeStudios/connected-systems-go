package api

import (
	"errors"
	"net/http"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/queryables"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// canonicalCollectionPaths maps canonical collectionId values to their real API path prefix.
// Requests to /collections/{id}/items[/{featureId}] are transparently redirected to the
// canonical endpoint so that OGC API Features clients work alongside the CS-specific paths.
var canonicalCollectionPaths = map[string]string{
	"systems":          "/systems",
	"deployments":      "/deployments",
	"procedures":       "/procedures",
	"samplingFeatures": "/samplingFeatures",
	"properties":       "/properties",
	"datastreams":      "/datastreams",
	"observations":     "/observations",
	"controlstreams":   "/controlstreams",
	"commands":         "/commands",
	"systemEvents":     "/systemEvents",
}

func redirectToCanonical(w http.ResponseWriter, r *http.Request, target string) {
	if q := r.URL.RawQuery; q != "" {
		target += "?" + q
	}
	http.Redirect(w, r, target, http.StatusTemporaryRedirect)
}

// FeatureHandler handles Feature resource requests (OGC API Features Part 1)
type FeatureHandler struct {
	cfg         *config.Config
	logger      *zap.Logger
	repo        *repository.FeatureRepository
	collections *repository.CollectionRepository
	fc          *formaters.MultiFormatFormatterCollection[*domains.Feature]
}

// NewFeatureHandler creates a new FeatureHandler
func NewFeatureHandler(cfg *config.Config, logger *zap.Logger, repo *repository.FeatureRepository, collections *repository.CollectionRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Feature]) *FeatureHandler {
	return &FeatureHandler{
		cfg:         cfg,
		logger:      logger,
		repo:        repo,
		collections: collections,
		fc:          fc,
	}
}

// ListFeatures retrieves features from a collection (OGC path: /collections/{collectionId}/items)
//
// @Summary     List features
// @Description Returns a feature collection. Canonical Connected Systems collections are served directly by the typed resource handler.
// @Tags        Features
// @Produce     json
// @Param       collectionId  path   string  true   "Collection ID"
// @Param       limit         query  int     false  "Maximum number of results"
// @Param       cursor        query  string  false  "Opaque pagination cursor"
// @Param       filter        query  string  false  "CQL2 Text filter (default) or URL-encoded CQL2 JSON"
// @Param       filter-lang   query  string  false  "CQL2 language: cql2-text or cql2-json"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /collections/{collectionId}/items [get]
func (h *FeatureHandler) ListFeatures(w http.ResponseWriter, r *http.Request) {
	collectionID := chi.URLParam(r, "collectionId")
	if basePath, ok := canonicalCollectionPaths[collectionID]; ok {
		redirectToCanonical(w, r, basePath)
		return
	}

	params, err := queryparams.FeatureQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	params.CollectionID = collectionID

	features, total, err := h.repo.ListByCollection(r.Context(), collectionID, params)
	if err != nil {
		h.logger.Error("Failed to list features", zap.String("collectionId", collectionID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeCollection(w, r, h.logger, h.fc, h.cfg.API.BaseURL, features, total, params.QueryParams)
}

// GetFeature retrieves a single feature by ID (OGC path: /collections/{collectionId}/items/{featureId})
//
// @Summary     Get feature
// @Description Returns a single feature from a collection by ID
// @Tags        Features
// @Produce     json
// @Param       collectionId  path  string  true  "Collection ID"
// @Param       featureId     path  string  true  "Feature ID"
// @Success     200  {object}  map[string]any
// @Success     307  {object}  nil  "Redirect to canonical resource endpoint"
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /collections/{collectionId}/items/{featureId} [get]
func (h *FeatureHandler) GetFeature(w http.ResponseWriter, r *http.Request) {
	collectionID := chi.URLParam(r, "collectionId")
	featureID := chi.URLParam(r, "featureId")
	if basePath, ok := canonicalCollectionPaths[collectionID]; ok {
		redirectToCanonical(w, r, basePath+"/"+featureID)
		return
	}

	feature, err := h.repo.GetByCollectionAndID(r.Context(), collectionID, featureID)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Feature", "Failed to load feature")
		return
	}

	writeResource(w, r, h.logger, h.fc, feature)
}

// CreateFeature creates a new feature in a collection
//
// @Summary     Create feature
// @Description Creates a new feature in a collection
// @Tags        Features
// @Accept      json
// @Param       collectionId  path  string          true  "Collection ID"
// @Param       feature       body  map[string]any  true  "Feature resource"
// @Success     201  {object}  map[string]any
// @Success     307  {object}  nil  "Redirect to canonical resource endpoint"
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /collections/{collectionId}/items [post]
func (h *FeatureHandler) CreateFeature(w http.ResponseWriter, r *http.Request) {
	collectionID := chi.URLParam(r, "collectionId")
	if basePath, ok := canonicalCollectionPaths[collectionID]; ok {
		redirectToCanonical(w, r, basePath)
		return
	}

	contentType := r.Header.Get("Content-Type")

	feature, err := h.fc.DeserializeContext(r.Context(), contentType, r.Body)

	if err != nil {
		h.logger.Error("Failed to decode feature", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	// Set collection ID from path
	feature.CollectionID = collectionID
	if err := h.validateCollectionFeature(r, collectionID, feature); err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.repo.Create(r.Context(), feature); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Feature", "Failed to create feature")
		return
	}

	w.Header().Set("Location", h.cfg.API.BaseURL+r.URL.Path+"/"+feature.ID)
	w.WriteHeader(http.StatusCreated)
}

// UpdateFeature updates an existing feature
//
// @Summary     Update feature
// @Description Replaces an existing feature in a collection
// @Tags        Features
// @Accept      json
// @Param       collectionId  path  string          true  "Collection ID"
// @Param       featureId     path  string          true  "Feature ID"
// @Param       feature       body  map[string]any  true  "Feature resource"
// @Success     200  {object}  map[string]any
// @Success     307  {object}  nil  "Redirect to canonical resource endpoint"
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /collections/{collectionId}/items/{featureId} [put]
func (h *FeatureHandler) UpdateFeature(w http.ResponseWriter, r *http.Request) {
	collectionID := chi.URLParam(r, "collectionId")
	featureID := chi.URLParam(r, "featureId")
	if basePath, ok := canonicalCollectionPaths[collectionID]; ok {
		redirectToCanonical(w, r, basePath+"/"+featureID)
		return
	}

	existing, err := h.repo.GetByCollectionAndID(r.Context(), collectionID, featureID)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Feature", "Failed to load feature")
		return
	}

	updated, err := h.fc.DeserializeContext(r.Context(), r.Header.Get("content-type"), r.Body)
	if err != nil {
		h.logger.Error("Failed to decode feature", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	// Preserve ID and collection
	updated.ID = existing.ID
	updated.CollectionID = collectionID
	if err := h.validateCollectionFeature(r, collectionID, updated); err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.repo.Update(r.Context(), updated); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Feature", "Failed to update feature")
		return
	}

	writeResource(w, r, h.logger, h.fc, updated)
}

func (h *FeatureHandler) validateCollectionFeature(r *http.Request, collectionID string, feature *domains.Feature) error {
	collection, err := h.collections.GetCollectionByID(r.Context(), collectionID)
	if err != nil {
		return errors.New("collection not found")
	}
	return queryables.ValidateFeatureProperties(queryables.CatalogOrDefault(collection.Queryables), feature.Properties)
}

// DeleteFeature deletes a feature
//
// @Summary     Delete feature
// @Description Deletes a feature from a collection
// @Tags        Features
// @Param       collectionId  path  string  true  "Collection ID"
// @Param       featureId     path  string  true  "Feature ID"
// @Success     204
// @Success     307  {object}  nil  "Redirect to canonical resource endpoint"
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /collections/{collectionId}/items/{featureId} [delete]
func (h *FeatureHandler) DeleteFeature(w http.ResponseWriter, r *http.Request) {
	collectionID := chi.URLParam(r, "collectionId")
	featureID := chi.URLParam(r, "featureId")
	if basePath, ok := canonicalCollectionPaths[collectionID]; ok {
		redirectToCanonical(w, r, basePath+"/"+featureID)
		return
	}

	_, err := h.repo.GetByCollectionAndID(r.Context(), collectionID, featureID)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Feature", "Failed to load feature")
		return
	}

	if err := h.repo.Delete(r.Context(), featureID); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Feature", "Failed to delete feature")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
