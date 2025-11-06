package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// FeatureHandler handles Feature resource requests (OGC API Features Part 1)
type FeatureHandler struct {
	cfg                  *config.Config
	logger               *zap.Logger
	repo                 *repository.FeatureRepository
	serializerCollection *serializers.SerializerCollection[domains.FeatureGeoJSONFeature, *domains.Feature]
}

// NewFeatureHandler creates a new FeatureHandler
func NewFeatureHandler(cfg *config.Config, logger *zap.Logger, repo *repository.FeatureRepository, s *serializers.SerializerCollection[domains.FeatureGeoJSONFeature, *domains.Feature]) *FeatureHandler {
	return &FeatureHandler{
		cfg:                  cfg,
		logger:               logger,
		repo:                 repo,
		serializerCollection: s,
	}
}

// ListFeatures retrieves features from a collection (OGC path: /collections/{collectionId}/items)
func (h *FeatureHandler) ListFeatures(w http.ResponseWriter, r *http.Request) {
	collectionID := chi.URLParam(r, "collectionId")
	params := queryparams.FeatureQueryParams{}.BuildFromRequest(r)
	params.CollectionID = collectionID

	features, total, err := h.repo.ListByCollection(collectionID, params)
	if err != nil {
		h.logger.Error("Failed to list features", zap.String("collectionId", collectionID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	serializer := h.serializerCollection.GetSerializer(r.Header.Get("content-type"))
	render.JSON(w, r, model.FeatureCollection[domains.FeatureGeoJSONFeature, *domains.Feature]{}.BuildCollection(
		features,
		serializer,
		h.cfg.API.BaseURL+r.URL.Path,
		int(total),
		r.URL.Query(),
		params.QueryParams,
	))
}

// GetFeature retrieves a single feature by ID (OGC path: /collections/{collectionId}/items/{featureId})
func (h *FeatureHandler) GetFeature(w http.ResponseWriter, r *http.Request) {
	collectionID := chi.URLParam(r, "collectionId")
	featureID := chi.URLParam(r, "featureId")

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

	serializer := h.serializerCollection.GetSerializer(r.Header.Get("content-type"))
	json, err := serializer.Serialize(context.Background(), feature)
	if err != nil {
		render.JSON(w, r, map[string]string{"error": "Failed to serialize feature"})
		return
	}
	render.JSON(w, r, json)
}

// CreateFeature creates a new feature in a collection
func (h *FeatureHandler) CreateFeature(w http.ResponseWriter, r *http.Request) {
	collectionID := chi.URLParam(r, "collectionId")

	feature, err := domains.Feature{}.BuildFromRequest(r, w)
	if err != nil {
		h.logger.Error("Failed to decode feature", zap.Error(err))
		return // BuildFromRequest already wrote error response
	}

	// Set collection ID from path
	feature.CollectionID = collectionID

	if err := h.repo.Create(&feature); err != nil {
		h.logger.Error("Failed to create feature", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create feature"})
		return
	}

	render.Status(r, http.StatusCreated)
	serializer := h.serializerCollection.GetSerializer(r.Header.Get("content-type"))
	json, _ := serializer.Serialize(context.Background(), &feature)
	render.JSON(w, r, json)
}

// UpdateFeature updates an existing feature
func (h *FeatureHandler) UpdateFeature(w http.ResponseWriter, r *http.Request) {
	collectionID := chi.URLParam(r, "collectionId")
	featureID := chi.URLParam(r, "featureId")

	existing, err := h.repo.GetByCollectionAndID(collectionID, featureID)
	if err != nil {
		h.logger.Error("Feature not found",
			zap.String("collectionId", collectionID),
			zap.String("featureId", featureID))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Feature not found"})
		return
	}

	updated, err := domains.Feature{}.BuildFromRequest(r, w)
	if err != nil {
		h.logger.Error("Failed to decode feature", zap.Error(err))
		return // BuildFromRequest already wrote error response
	}

	// Preserve ID and collection
	updated.ID = existing.ID
	updated.CollectionID = collectionID

	if err := h.repo.Update(&updated); err != nil {
		h.logger.Error("Failed to update feature", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update feature"})
		return
	}

	serializer := h.serializerCollection.GetSerializer(r.Header.Get("content-type"))
	json, _ := serializer.Serialize(context.Background(), &updated)
	render.JSON(w, r, json)
}

// DeleteFeature deletes a feature
func (h *FeatureHandler) DeleteFeature(w http.ResponseWriter, r *http.Request) {
	collectionID := chi.URLParam(r, "collectionId")
	featureID := chi.URLParam(r, "featureId")

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
		h.logger.Error("Failed to delete feature", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete feature"})
		return
	}

	render.Status(r, http.StatusNoContent)
	w.Write(nil)
}
