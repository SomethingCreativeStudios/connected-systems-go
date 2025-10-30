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

// PropertyHandler handles Property resource requests
type PropertyHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.PropertyRepository
	sc     *serializers.SerializerCollection[domains.PropertyGeoJSONFeature, *domains.Property]
	fc     model.FeatureCollection[domains.PropertyGeoJSONFeature, *domains.Property]
}

// NewPropertyHandler creates a new PropertyHandler
func NewPropertyHandler(cfg *config.Config, logger *zap.Logger, repo *repository.PropertyRepository, s *serializers.SerializerCollection[domains.PropertyGeoJSONFeature, *domains.Property]) *PropertyHandler {
	return &PropertyHandler{cfg: cfg, logger: logger, repo: repo, sc: s, fc: model.FeatureCollection[domains.PropertyGeoJSONFeature, *domains.Property]{}}
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

	serializer := h.sc.GetSerializer(r.Header.Get("content-type"))
	render.JSON(w, r, h.fc.BuildCollection(properties, serializer, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams))
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

	propertyGeoJSON, err := h.sc.Serialize(r.Header.Get("content-type"), property)

	if err != nil {
		h.logger.Error("Failed to serialize property", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize propety"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, propertyGeoJSON)
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

	propertyGeoJSON, err := h.sc.Serialize(r.Header.Get("content-type"), &property)

	if err != nil {
		h.logger.Error("Failed to serialize property", zap.String("id", property.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize propety"})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, propertyGeoJSON)
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

	propertyGeoJSON, err := h.sc.Serialize(r.Header.Get("content-type"), &property)

	if err != nil {
		h.logger.Error("Failed to serialize property", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize propety"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, propertyGeoJSON)
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
