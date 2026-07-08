package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository"
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
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.FeatureRepository
	fc     *formaters.MultiFormatFormatterCollection[*domains.Feature]
}

// NewFeatureHandler creates a new FeatureHandler
func NewFeatureHandler(cfg *config.Config, logger *zap.Logger, repo *repository.FeatureRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Feature]) *FeatureHandler {
	return &FeatureHandler{
		cfg:    cfg,
		logger: logger,
		repo:   repo,
		fc:     fc,
	}
}

// ListFeatures retrieves features from a collection (OGC path: /collections/{collectionId}/items)
//
// @Summary     List features
// @Description Returns features from a collection; canonical collection IDs redirect to their resource endpoint
// @Tags        Features
// @Produce     json
// @Param       collectionId  path   string  true   "Collection ID"
// @Param       limit         query  int     false  "Maximum number of results"
// @Param       offset        query  int     false  "Result offset"
// @Success     200  {object}  map[string]any
// @Success     307  {object}  nil  "Redirect to canonical resource endpoint"
// @Failure     500  {object}  map[string]string
// @Router      /collections/{collectionId}/items [get]
func (h *FeatureHandler) ListFeatures(w http.ResponseWriter, r *http.Request) {
	collectionID := chi.URLParam(r, "collectionId")
	if basePath, ok := canonicalCollectionPaths[collectionID]; ok {
		redirectToCanonical(w, r, basePath)
		return
	}

	params := queryparams.FeatureQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	params.CollectionID = collectionID

	features, total, err := h.repo.ListByCollection(collectionID, params)
	if err != nil {
		h.logger.Error("Failed to list features", zap.String("collectionId", collectionID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, features, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	writeNegotiated(w, collection)
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

	feature, err := h.repo.GetByCollectionAndID(collectionID, featureID)
	if err != nil {
		h.logger.Error("Failed to get feature",
			zap.String("collectionId", collectionID),
			zap.String("featureId", featureID),
			zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Feature not found"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	json, err := h.fc.Serialize(acceptHeader, feature)

	if err != nil {
		h.logger.Error("Failed to serialize feature",
			zap.String("collectionId", collectionID),
			zap.String("featureId", featureID),
			zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize feature"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	writeNegotiated(w, json)
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

	feature, err := h.fc.Deserialize(contentType, r.Body)

	if err != nil {
		h.logger.Error("Failed to decode feature", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	// Set collection ID from path
	feature.CollectionID = collectionID

	if err := h.repo.Create(feature); err != nil {
		h.logger.Error("Failed to create feature", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create feature"})
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

	existing, err := h.repo.GetByCollectionAndID(collectionID, featureID)
	if err != nil {
		h.logger.Error("Feature not found",
			zap.String("collectionId", collectionID),
			zap.String("featureId", featureID))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Feature not found"})
		return
	}

	updated, err := h.fc.Deserialize(r.Header.Get("content-type"), r.Body)
	if err != nil {
		h.logger.Error("Failed to decode feature", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	// Preserve ID and collection
	updated.ID = existing.ID
	updated.CollectionID = collectionID

	if err := h.repo.Update(updated); err != nil {
		h.logger.Error("Failed to update feature", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update feature"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	json, _ := h.fc.Serialize(acceptHeader, updated)
	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	writeNegotiated(w, json)
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

	_, err := h.repo.GetByCollectionAndID(collectionID, featureID)
	if err != nil {
		h.logger.Error("Feature not found",
			zap.String("collectionId", collectionID),
			zap.String("featureId", featureID))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Feature not found"})
		return
	}

	if err := h.repo.Delete(featureID); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "Feature not found"})
		default:
			h.logger.Error("Failed to delete feature", zap.Error(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to delete feature"})
		}
		return
	}

	render.Status(r, http.StatusNoContent)
	w.Write(nil)
}
