package api

import (
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

// SamplingFeatureHandler handles SamplingFeature resource requests
type SamplingFeatureHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.SamplingFeatureRepository
	sc     *serializers.SerializerCollection[domains.SamplingFeatureGeoJSONFeature, *domains.SamplingFeature]
	fc     model.FeatureCollection[domains.SamplingFeatureGeoJSONFeature, *domains.SamplingFeature]
}

// NewSamplingFeatureHandler creates a new SamplingFeatureHandler
func NewSamplingFeatureHandler(cfg *config.Config, logger *zap.Logger, repo *repository.SamplingFeatureRepository, s *serializers.SerializerCollection[domains.SamplingFeatureGeoJSONFeature, *domains.SamplingFeature]) *SamplingFeatureHandler {
	return &SamplingFeatureHandler{cfg: cfg, logger: logger, repo: repo, sc: s, fc: model.FeatureCollection[domains.SamplingFeatureGeoJSONFeature, *domains.SamplingFeature]{}}
}

func (h *SamplingFeatureHandler) ListSamplingFeatures(w http.ResponseWriter, r *http.Request) {
	params, err := queryparams.SamplingFeatureQueryParams{}.BuildFromRequest(r)

	if err != nil {
		h.logger.Error("Failed to parse query parameters", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid query parameters"})
		return
	}

	sampledFeatures, total, err := h.repo.List(params)
	if err != nil {
		h.logger.Error("Failed to list sampling features", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	serializer := h.sc.GetSerializer(r.Header.Get("content-type"))
	render.JSON(w, r, h.fc.BuildCollection(sampledFeatures, serializer, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams))
}

func (h *SamplingFeatureHandler) GetSamplingFeature(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	system, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get sampling feature", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Sampleing Feature not found"})
		return
	}

	sampledFeatureGeoJSON, err := h.sc.Serialize(r.Header.Get("content-type"), system)

	if err != nil {
		h.logger.Error("Failed to serialize sampling feature", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize sampling feature"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, sampledFeatureGeoJSON)
}

func (h *SamplingFeatureHandler) CreateSamplingFeature(w http.ResponseWriter, r *http.Request) {
	sampledFeature, err := domains.SamplingFeature{}.BuildFromRequest(r, w)

	if err != nil {
		return // Error already handled in BuildFromRequest
	}

	if err := h.repo.Create(&sampledFeature); err != nil {
		h.logger.Error("Failed to create sampling feature", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create sampling feature"})
		return
	}

	systemGeoJSON, err := h.sc.Serialize(r.Header.Get("content-type"), &sampledFeature)

	if err != nil {
		h.logger.Error("Failed to serialize sampling feature", zap.String("id", sampledFeature.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize sampling feature"})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, systemGeoJSON)
}

func (h *SamplingFeatureHandler) UpdateSamplingFeature(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var sampledFeature domains.SamplingFeature
	if err := render.DecodeJSON(r.Body, &sampledFeature); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	sampledFeature.ID = id
	if err := h.repo.Update(&sampledFeature); err != nil {
		h.logger.Error("Failed to update sampling feature", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update sampling feature"})
		return
	}

	systemGeoJSON, err := h.sc.Serialize(r.Header.Get("content-type"), &sampledFeature)

	if err != nil {
		h.logger.Error("Failed to serialize sampling feature", zap.String("id", sampledFeature.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize sampling feature"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, systemGeoJSON)
}

func (h *SamplingFeatureHandler) DeleteSamplingFeature(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(id); err != nil {
		h.logger.Error("Failed to delete sampling feature", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete sampling feature"})
		return
	}

	render.Status(r, http.StatusNoContent)
}

func (h *SamplingFeatureHandler) GetSystemSamplingFeatures(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusNotImplemented)
	render.JSON(w, r, map[string]string{"message": "Not implemented"})
}
