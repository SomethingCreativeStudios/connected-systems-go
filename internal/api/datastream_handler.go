package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// DatastreamCollectionResponse follows datastreams-only.yaml collection shape.
type DatastreamCollectionResponse struct {
	Items []any               `json:"items"`
	Links common_shared.Links `json:"links,omitempty"`
}

// DatastreamHandler handles datastream endpoints.
type DatastreamHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.DatastreamRepository
	fc     *formaters.MultiFormatFormatterCollection[*domains.Datastream]
}

func NewDatastreamHandler(cfg *config.Config, logger *zap.Logger, repo *repository.DatastreamRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Datastream]) *DatastreamHandler {
	return &DatastreamHandler{cfg: cfg, logger: logger, repo: repo, fc: fc}
}

// ListDatastreams returns a paginated list of datastreams
//
// @Summary     List datastreams
// @Description Returns a paginated collection of datastream resources
// @Tags        Datastreams
// @Produce     json
// @Param       limit             query  integer  false  "Maximum number of results"
// @Param       offset            query  integer  false  "Result offset"
// @Param       id                query  string   false  "Comma-separated resource IDs"
// @Param       q                 query  string   false  "Comma-separated keywords for full-text search"
// @Param       system            query  string   false  "Comma-separated system IDs"
// @Param       foi               query  string   false  "Comma-separated feature of interest IDs"
// @Param       observedProperty  query  string   false  "Comma-separated observed property IDs"
// @Param       phenomenonTime    query  string   false  "Phenomenon time filter (RFC 3339 date-time or interval)"
// @Param       resultTime        query  string   false  "Result time filter (RFC 3339 date-time or interval)"
// @Success     200  {object}  DatastreamCollectionResponse
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /datastreams [get]
func (h *DatastreamHandler) ListDatastreams(w http.ResponseWriter, r *http.Request) {
	params, err := queryparams.DatastreamsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	datastreams, total, err := h.repo.List(params, nil)
	if err != nil {
		h.logger.Error("Failed to list datastreams", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	items, err := h.fc.SerializeAll(acceptHeader, datastreams)
	if err != nil {
		h.logger.Error("Failed to serialize datastreams", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize datastreams"})
		return
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(datastreams))

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, DatastreamCollectionResponse{Items: items, Links: links})
}

// ListSystemDatastreams returns datastreams for a system
//
// @Summary     List system datastreams
// @Description Returns datastreams associated with a given system
// @Tags        Datastreams
// @Produce     json
// @Param       id                path   string   true   "System ID"
// @Param       limit             query  integer  false  "Maximum number of results"
// @Param       offset            query  integer  false  "Result offset"
// @Param       q                 query  string   false  "Comma-separated keywords for full-text search"
// @Param       foi               query  string   false  "Comma-separated feature of interest IDs"
// @Param       observedProperty  query  string   false  "Comma-separated observed property IDs"
// @Param       phenomenonTime    query  string   false  "Phenomenon time filter (RFC 3339 date-time or interval)"
// @Param       resultTime        query  string   false  "Result time filter (RFC 3339 date-time or interval)"
// @Success     200  {object}  DatastreamCollectionResponse
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/datastreams [get]
func (h *DatastreamHandler) ListSystemDatastreams(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemId")
	if systemID == "" {
		systemID = chi.URLParam(r, "id")
	}
	params, err := queryparams.DatastreamsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	datastreams, total, err := h.repo.List(params, &systemID)
	if err != nil {
		h.logger.Error("Failed to list datastreams for system", zap.String("systemId", systemID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	items, err := h.fc.SerializeAll(acceptHeader, datastreams)
	if err != nil {
		h.logger.Error("Failed to serialize datastreams", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize datastreams"})
		return
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(datastreams))

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, DatastreamCollectionResponse{Items: items, Links: links})
}

// GetDatastream returns a single datastream by ID
//
// @Summary     Get datastream
// @Description Returns a single datastream resource by ID
// @Tags        Datastreams
// @Produce     json
// @Param       dataStreamId  path  string  true  "Datastream ID"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /datastreams/{dataStreamId} [get]
func (h *DatastreamHandler) GetDatastream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dataStreamId")

	datastream, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get datastream", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Datastream not found"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, datastream)
	if err != nil {
		h.logger.Error("Failed to serialize datastream", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize datastream"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, serialized)
}

// CreateDatastream creates a new datastream under a system
//
// @Summary     Create datastream
// @Description Creates a new datastream resource under the given system
// @Tags        Datastreams
// @Accept      json
// @Param       id          path  string          true  "System ID"
// @Param       datastream  body  map[string]any  true  "Datastream resource"
// @Success     201
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/datastreams [post]
func (h *DatastreamHandler) CreateDatastream(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemId")
	if systemID == "" {
		systemID = chi.URLParam(r, "id")
	}

	contentType := r.Header.Get("Content-Type")
	datastream, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize datastream", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	if systemID != "" {
		datastream.SystemID = &systemID
	}

	if err := h.repo.Create(datastream); err != nil {
		h.logger.Error("Failed to create datastream", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create datastream"})
		return
	}

	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/datastreams/" + datastream.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

// UpdateDatastream replaces a datastream by ID
//
// @Summary     Update datastream
// @Description Replaces a datastream resource by ID
// @Tags        Datastreams
// @Accept      json
// @Param       dataStreamId  path  string          true  "Datastream ID"
// @Param       datastream    body  map[string]any  true  "Datastream resource"
// @Success     204
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /datastreams/{dataStreamId} [put]
func (h *DatastreamHandler) UpdateDatastream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dataStreamId")
	existing, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get datastream before update", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Datastream not found"})
		return
	}

	contentType := r.Header.Get("Content-Type")
	datastream, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize datastream", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	datastream.ID = id
	if datastream.SystemID == nil {
		datastream.SystemID = existing.SystemID
	}
	if err := h.repo.Update(datastream); err != nil {
		h.logger.Error("Failed to update datastream", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update datastream"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteDatastream deletes a datastream by ID
//
// @Summary     Delete datastream
// @Description Deletes a datastream resource by ID
// @Tags        Datastreams
// @Param       dataStreamId  path   string  true   "Datastream ID"
// @Param       cascade       query  bool    false  "Cascade delete to dependent resources"
// @Success     204
// @Failure     404  {object}  map[string]string
// @Failure     409  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /datastreams/{dataStreamId} [delete]
func (h *DatastreamHandler) DeleteDatastream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dataStreamId")
	cascade := r.URL.Query().Get("cascade") == "true"
	if err := h.repo.Delete(id, cascade); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "Datastream not found"})
		case errors.Is(err, repository.ErrHasChildren):
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, map[string]string{"error": "Datastream has dependent records; use ?cascade=true to delete"})
		default:
			h.logger.Error("Failed to delete datastream", zap.String("id", id), zap.Error(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to delete datastream"})
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetDatastreamSchema returns the schema for a datastream
//
// @Summary     Get datastream schema
// @Description Returns the observation schema for a given datastream
// @Tags        Datastreams
// @Produce     json
// @Param       dataStreamId  path  string  true  "Datastream ID"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Router      /datastreams/{dataStreamId}/schema [get]
func (h *DatastreamHandler) GetDatastreamSchema(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dataStreamId")
	schema, err := h.repo.GetSchema(id)
	if err != nil {
		h.logger.Error("Failed to get datastream schema", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Datastream not found"})
		return
	}

	if schema == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Datastream schema not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, schema)
}

// UpdateDatastreamSchema replaces the schema for a datastream
//
// @Summary     Update datastream schema
// @Description Replaces the observation schema for a given datastream
// @Tags        Datastreams
// @Accept      json
// @Param       dataStreamId  path  string          true  "Datastream ID"
// @Param       schema        body  map[string]any  true  "Datastream schema"
// @Success     204
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /datastreams/{dataStreamId}/schema [put]
func (h *DatastreamHandler) UpdateDatastreamSchema(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dataStreamId")

	var schema domains.DatastreamSchema
	if err := render.DecodeJSON(r.Body, &schema); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	if err := h.repo.UpdateSchema(id, &schema); err != nil {
		h.logger.Error("Failed to update datastream schema", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update datastream schema"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
