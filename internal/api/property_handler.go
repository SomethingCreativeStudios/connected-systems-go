package api

import (
	"errors"
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

// ListProperties returns a paginated list of properties
//
// @Summary     List properties
// @Description Returns a paginated collection of property resources
// @Tags        Properties
// @Produce     json
// @Param       limit         query  integer  false  "Maximum number of results"
// @Param       cursor        query  string   false  "Opaque pagination cursor"
// @Param       id            query  string   false  "Comma-separated resource IDs"
// @Param       q             query  string   false  "Comma-separated keywords for full-text search"
// @Param       baseProperty  query  string   false  "Comma-separated base property IDs"
// @Param       objectType    query  string   false  "Comma-separated object type URIs"
// @Success     200  {object}  map[string]any
// @Failure     500  {object}  map[string]string
// @Router      /properties [get]
func (h *PropertyHandler) ListProperties(w http.ResponseWriter, r *http.Request) {
	params, err := queryparams.PropertiesQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

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
	writeNegotiated(w, collection)
}

// GetProperty returns a single property by ID
//
// @Summary     Get property
// @Description Returns a single property resource by ID
// @Tags        Properties
// @Produce     json
// @Param       id  path  string  true  "Property ID"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /properties/{id} [get]
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
	writeNegotiated(w, serialized)
}

// CreateProperty creates a new property
//
// @Summary     Create property
// @Description Creates a new property resource
// @Tags        Properties
// @Accept      json
// @Param       property  body  map[string]any  true  "Property resource"
// @Success     201
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     500  {object}  map[string]string
// @Router      /properties [post]
func (h *PropertyHandler) CreateProperty(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	property, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize property", zap.Error(err))
		writeDeserializeError(w, r, err)
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

// UpdateProperty replaces a property by ID
//
// @Summary     Update property
// @Description Replaces a property resource by ID
// @Tags        Properties
// @Accept      json
// @Param       id        path  string          true  "Property ID"
// @Param       property  body  map[string]any  true  "Property resource"
// @Success     204
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     500  {object}  map[string]string
// @Router      /properties/{id} [put]
func (h *PropertyHandler) UpdateProperty(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	property, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize property", zap.Error(err))
		writeDeserializeError(w, r, err)
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

// DeleteProperty deletes a property by ID
//
// @Summary     Delete property
// @Description Deletes a property resource by ID
// @Tags        Properties
// @Param       id  path  string  true  "Property ID"
// @Success     204
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /properties/{id} [delete]
func (h *PropertyHandler) DeleteProperty(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(id); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "Property not found"})
		default:
			h.logger.Error("Failed to delete property", zap.String("id", id), zap.Error(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to delete property"})
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
