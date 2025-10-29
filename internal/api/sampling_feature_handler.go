package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// SamplingFeatureHandler handles SamplingFeature resource requests
type SamplingFeatureHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.SamplingFeatureRepository
}

// NewSamplingFeatureHandler creates a new SamplingFeatureHandler
func NewSamplingFeatureHandler(cfg *config.Config, logger *zap.Logger, repo *repository.SamplingFeatureRepository) *SamplingFeatureHandler {
	return &SamplingFeatureHandler{cfg: cfg, logger: logger, repo: repo}
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

	render.JSON(w, r, model.FeatureCollection[domains.SamplingFeatureGeoJSONFeature, *domains.SamplingFeature]{}.BuildCollection(sampledFeatures, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams))

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

	render.JSON(w, r, system.ToGeoJSON())
}

func (h *SamplingFeatureHandler) CreateSamplingFeature(w http.ResponseWriter, r *http.Request) {
	system, err := domains.SamplingFeature{}.BuildFromRequest(r, w)

	if err != nil {
		return // Error already handled in BuildFromRequest
	}

	if err := h.repo.Create(&system); err != nil {
		h.logger.Error("Failed to create sampling feature", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create sampling feature"})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, system.ToGeoJSON())
}

func (h *SamplingFeatureHandler) UpdateSamplingFeature(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var system domains.SamplingFeature
	if err := render.DecodeJSON(r.Body, &system); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	system.ID = id
	if err := h.repo.Update(&system); err != nil {
		h.logger.Error("Failed to update sampling feature", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update sampling feature"})
		return
	}

	render.JSON(w, r, system.ToGeoJSON())
}

func (h *SamplingFeatureHandler) PatchSamplingFeature(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusNotImplemented)
	render.JSON(w, r, map[string]string{"message": "Not implemented"})
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
