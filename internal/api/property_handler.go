package api

import (
	"net/http"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/go-chi/chi/v5"
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
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	properties, total, err := h.repo.List(r.Context(), params)
	if err != nil {
		h.logger.Error("Failed to list properties", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Use Accept header for content negotiation (not Content-Type)
	writeCollection(w, r, h.logger, h.fc, h.cfg.API.BaseURL, properties, total, params.QueryParams)
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

	property, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Property", "Failed to load property")
		return
	}

	writeResource(w, r, h.logger, h.fc, property)
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
	property, err := h.fc.DeserializeContext(r.Context(), contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize property", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	if err := h.repo.Create(r.Context(), property); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Property", "Failed to create property")
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
	property, err := h.fc.DeserializeContext(r.Context(), contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize property", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	property.ID = id
	if err := h.repo.Update(r.Context(), property); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Property", "Failed to update property")
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

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Property", "Failed to delete property")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
