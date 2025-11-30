package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// PropertyHandler handles Property resource requests
type PropertyHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.PropertyRepository
	fc     *formaters.MultiFormatFormatterCollection[*domains.Property]
}

// NewPropertyHandler creates a new PropertyHandler
func NewPropertyHandler(cfg *config.Config, logger *zap.Logger, repo *repository.PropertyRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Property]) *PropertyHandler {
	return &PropertyHandler{cfg: cfg, logger: logger, repo: repo, fc: fc}
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

	// Use Accept header for content negotiation (not Content-Type)
	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, properties, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams)

	// Set the response content type based on the serializer used
	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, collection)
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

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, property)
	if err != nil {
		h.logger.Error("Failed to serialize property", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize property"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, serialized)
}

func (h *PropertyHandler) CreateProperty(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	property, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize property", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	if err := h.repo.Create(property); err != nil {
		h.logger.Error("Failed to create property", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create property"})
		return
	}
	// Per conformance behavior, respond with 201 Created and a Location header
	// pointing to the newly created resource. Do not include a response body.
	base := strings.TrimRight(h.cfg.API.BaseURL, "/")
	location := base + "/properties/" + property.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

func (h *PropertyHandler) UpdateProperty(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	property, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize property", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	property.ID = id
	if err := h.repo.Update(property); err != nil {
		h.logger.Error("Failed to update property", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update property"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PropertyHandler) DeleteProperty(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(id); err != nil {
		h.logger.Error("Failed to delete property", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete property"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
