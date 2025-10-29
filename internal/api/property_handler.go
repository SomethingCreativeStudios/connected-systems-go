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

// PropertyHandler handles Property resource requests
type PropertyHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.PropertyRepository
}

// NewPropertyHandler creates a new PropertyHandler
func NewPropertyHandler(cfg *config.Config, logger *zap.Logger, repo *repository.PropertyRepository) *PropertyHandler {
	return &PropertyHandler{cfg: cfg, logger: logger, repo: repo}
}

func (h *PropertyHandler) ListProperties(w http.ResponseWriter, r *http.Request) {
	params := queryparams.PropertiesQueryParams{}.BuildFromRequest(r)

	properties, total, err := h.repo.List(params)
	if err != nil {
		h.logger.Error("Failed to list properties", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	render.JSON(w, r, model.FeatureCollection[domains.PropertyGeoJSONFeature, *domains.Property]{}.BuildCollection(properties, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams))

}

func (h *PropertyHandler) GetProperty(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	property, err := h.repo.GetByID(id)

	if err != nil {
		h.logger.Error("Failed to get property", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Property not found"})
		return
	}

	render.JSON(w, r, property.ToGeoJSON())
}

func (h *PropertyHandler) CreateProperty(w http.ResponseWriter, r *http.Request) {
	property, err := domains.Property{}.BuildFromRequest(r, w)

	if err != nil {
		return // Error already handled in buildSystem
	}

	if err := h.repo.Create(&property); err != nil {
		h.logger.Error("Failed to create property", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create property"})
		return
	}

	render.Status(r, http.StatusCreated)

	render.JSON(w, r, property.ToGeoJSON())
}

func (h *PropertyHandler) UpdateProperty(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var property domains.Property
	if err := render.DecodeJSON(r.Body, &property); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	property.ID = id
	if err := h.repo.Update(&property); err != nil {
		h.logger.Error("Failed to update property", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update system"})
		return
	}

	render.JSON(w, r, property.ToGeoJSON())
}

func (h *PropertyHandler) PatchProperty(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusNotImplemented)
	render.JSON(w, r, map[string]string{"message": "Not implemented"})
}

func (h *PropertyHandler) DeleteProperty(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(id); err != nil {
		h.logger.Error("Failed to delete property", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete property"})
		return
	}

	render.Status(r, http.StatusNoContent)
}
