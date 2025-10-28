package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model"
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
	params := h.parseQueryParams(r)

	properties, total, err := h.repo.List(params)
	if err != nil {
		h.logger.Error("Failed to list properties", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	// Convert to GeoJSON FeatureCollection
	features := make([]model.PropertyGeoJSONFeature, len(properties))
	for i, prop := range properties {
		features[i] = prop.ToGeoJSON()
	}

	totalInt := int(total)
	response := map[string]interface{}{
		"type":           "FeatureCollection",
		"numberMatched":  &totalInt,
		"numberReturned": len(features),
		"features":       features,
		"links":          h.buildCollectionLinks(r, &totalInt, len(features), params.Limit),
	}

	render.JSON(w, r, response)
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
	property, err := h.buildProperty(r, w)

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

	var property model.Property
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

// parseQueryParams parses common query parameters
func (h *PropertyHandler) parseQueryParams(r *http.Request) *repository.PropertiesQueryParams {
	params := &repository.PropertiesQueryParams{
		Limit:  10,
		Offset: 0,
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			params.Limit = val
		}
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if val, err := strconv.Atoi(offset); err == nil {
			params.Offset = val
		}
	}

	if ids := r.URL.Query().Get("id"); ids != "" {
		params.IDs = strings.Split(ids, ",")
	}

	params.Q = r.URL.Query().Get("q")

	if baseProps := r.URL.Query().Get("baseProperty"); baseProps != "" {
		params.BaseProperty = strings.Split(baseProps, ",")
	}

	if objTypes := r.URL.Query().Get("objectType"); objTypes != "" {
		params.ObjectType = strings.Split(objTypes, ",")
	}

	return params
}

// buildCollectionLinks creates pagination links
func (h *PropertyHandler) buildCollectionLinks(r *http.Request, total *int, returned, limit int) model.Links {
	baseURL := h.cfg.API.BaseURL + r.URL.Path

	currentOffsetStr := r.URL.Query().Get("offset")
	currentOffset := 0

	if currentOffsetStr != "" {
		if val, err := strconv.Atoi(currentOffsetStr); err == nil {
			currentOffset = val
		}
	}

	links := model.Links{
		{Href: baseURL + "?" + r.URL.RawQuery, Rel: "self"},
	}

	if (currentOffset + returned) < *total {
		nextLink := r.URL.Query()
		nextLink.Set("offset", strconv.Itoa(currentOffset+returned))

		links = append(links, model.Link{
			Rel:  "next",
			Href: baseURL + "?" + nextLink.Encode(),
		})
	}

	if currentOffset > 0 {
		prevLink := r.URL.Query()
		if currentOffset-limit <= 0 {
			prevLink.Del("offset")
		} else {
			prevLink.Set("offset", strconv.Itoa(currentOffset-limit))
		}

		links = append(links, model.Link{
			Rel:  "prev",
			Href: baseURL + "?" + prevLink.Encode(),
		})
	}

	return links
}

func (h *PropertyHandler) buildProperty(r *http.Request, w http.ResponseWriter) (model.Property, error) {
	// Decode GeoJSON Feature format
	var geoJSON struct {
		Type       string                          `json:"type"`
		ID         string                          `json:"id,omitempty"`
		Properties model.PropertyGeoJSONProperties `json:"properties"`
		Geometry   *model.Geometry                 `json:"geometry,omitempty"`
		Links      model.Links                     `json:"links,omitempty"`
	}

	if err := render.DecodeJSON(r.Body, &geoJSON); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return model.Property{}, err
	}

	// Convert GeoJSON properties to Property model
	property := model.Property{
		Links: geoJSON.Links,
	}

	// Extract properties from the properties object
	property.UniqueIdentifier = model.UniqueID(geoJSON.Properties.UID)

	property.Name = geoJSON.Properties.Name
	property.Description = geoJSON.Properties.Description
	property.Definition = geoJSON.Properties.Definition
	property.PropertyType = geoJSON.Properties.PropertyType
	property.ObjectType = geoJSON.Properties.ObjectType
	property.Definition = geoJSON.Properties.Definition
	property.UnitOfMeasurement = geoJSON.Properties.UnitOfMeasurement

	return property, nil
}
