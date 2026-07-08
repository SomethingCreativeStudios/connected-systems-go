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

// ControlStreamCollectionResponse follows the collection shape used by other dynamic-data resources.
type ControlStreamCollectionResponse struct {
	Items []any               `json:"items"`
	Links common_shared.Links `json:"links,omitempty"`
}

// ControlStreamHandler handles control stream endpoints.
type ControlStreamHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.ControlStreamRepository
	fc     *formaters.MultiFormatFormatterCollection[*domains.ControlStream]
}

func NewControlStreamHandler(
	cfg *config.Config,
	logger *zap.Logger,
	repo *repository.ControlStreamRepository,
	fc *formaters.MultiFormatFormatterCollection[*domains.ControlStream],
) *ControlStreamHandler {
	return &ControlStreamHandler{cfg: cfg, logger: logger, repo: repo, fc: fc}
}

// ListControlStreams handles GET /controlstreams
//
// @Summary     List control streams
// @Description Returns a paginated collection of control stream resources
// @Tags        Control Streams
// @Produce     json
// @Param       limit               query  integer  false  "Maximum number of results"
// @Param       offset              query  integer  false  "Result offset"
// @Param       id                  query  string   false  "Comma-separated resource IDs"
// @Param       q                   query  string   false  "Comma-separated keywords for full-text search"
// @Param       system              query  string   false  "Comma-separated system IDs"
// @Param       foi                 query  string   false  "Comma-separated feature of interest IDs"
// @Param       controlledProperty  query  string   false  "Comma-separated controlled property IDs"
// @Param       issueTime           query  string   false  "Issue time filter (RFC 3339 date-time or interval)"
// @Param       executionTime       query  string   false  "Execution time filter (RFC 3339 date-time or interval)"
// @Success     200  {object}  ControlStreamCollectionResponse
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /controlstreams [get]
func (h *ControlStreamHandler) ListControlStreams(w http.ResponseWriter, r *http.Request) {
	params, err := queryparams.ControlStreamsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	controlStreams, total, err := h.repo.List(params, nil)
	if err != nil {
		h.logger.Error("Failed to list control streams", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	items, err := h.fc.SerializeAll(acceptHeader, controlStreams)
	if err != nil {
		h.logger.Error("Failed to serialize control streams", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize control streams"})
		return
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(controlStreams))

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, ControlStreamCollectionResponse{Items: items, Links: links})
}

// ListSystemControlStreams handles GET /systems/{id}/controlstreams
//
// @Summary     List system control streams
// @Description Returns control streams associated with a given system
// @Tags        Control Streams
// @Produce     json
// @Param       id                  path   string   true   "System ID"
// @Param       limit               query  integer  false  "Maximum number of results"
// @Param       offset              query  integer  false  "Result offset"
// @Param       q                   query  string   false  "Comma-separated keywords for full-text search"
// @Param       foi                 query  string   false  "Comma-separated feature of interest IDs"
// @Param       controlledProperty  query  string   false  "Comma-separated controlled property IDs"
// @Param       issueTime           query  string   false  "Issue time filter (RFC 3339 date-time or interval)"
// @Param       executionTime       query  string   false  "Execution time filter (RFC 3339 date-time or interval)"
// @Success     200  {object}  ControlStreamCollectionResponse
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/controlstreams [get]
func (h *ControlStreamHandler) ListSystemControlStreams(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemId")
	if systemID == "" {
		systemID = chi.URLParam(r, "id")
	}
	params, err := queryparams.ControlStreamsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	controlStreams, total, err := h.repo.List(params, &systemID)
	if err != nil {
		h.logger.Error("Failed to list control streams for system", zap.String("systemId", systemID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	items, err := h.fc.SerializeAll(acceptHeader, controlStreams)
	if err != nil {
		h.logger.Error("Failed to serialize control streams", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize control streams"})
		return
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(controlStreams))

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, ControlStreamCollectionResponse{Items: items, Links: links})
}

// GetControlStream handles GET /controlstreams/{id}
//
// @Summary     Get control stream
// @Description Returns a single control stream resource by ID
// @Tags        Control Streams
// @Produce     json
// @Param       controlStreamId  path  string  true  "Control Stream ID"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /controlstreams/{controlStreamId} [get]
func (h *ControlStreamHandler) GetControlStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "controlStreamId")

	cs, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get control stream", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Control stream not found"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, cs)
	if err != nil {
		h.logger.Error("Failed to serialize control stream", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize control stream"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	writeNegotiated(w, serialized)
}

// CreateControlStream handles POST /systems/{id}/controlstreams
//
// @Summary     Create control stream
// @Description Creates a new control stream under the given system
// @Tags        Control Streams
// @Accept      json
// @Param       id             path  string          true  "System ID"
// @Param       controlStream  body  map[string]any  true  "Control Stream resource"
// @Success     201
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/controlstreams [post]
func (h *ControlStreamHandler) CreateControlStream(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemId")
	if systemID == "" {
		systemID = chi.URLParam(r, "id")
	}

	contentType := r.Header.Get("Content-Type")
	cs, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize control stream", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	if systemID != "" {
		cs.SystemID = &systemID
	}

	if err := h.repo.Create(cs); err != nil {
		h.logger.Error("Failed to create control stream", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create control stream"})
		return
	}

	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/controlstreams/" + cs.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

// UpdateControlStream handles PUT /controlstreams/{id}
//
// @Summary     Update control stream
// @Description Replaces a control stream resource by ID
// @Tags        Control Streams
// @Accept      json
// @Param       controlStreamId  path  string          true  "Control Stream ID"
// @Param       controlStream    body  map[string]any  true  "Control Stream resource"
// @Success     204
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /controlstreams/{controlStreamId} [put]
func (h *ControlStreamHandler) UpdateControlStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "controlStreamId")
	existing, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get control stream before update", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Control stream not found"})
		return
	}

	contentType := r.Header.Get("Content-Type")
	cs, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize control stream", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	cs.ID = id
	if cs.SystemID == nil {
		cs.SystemID = existing.SystemID
	}
	if err := h.repo.Update(cs); err != nil {
		h.logger.Error("Failed to update control stream", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update control stream"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteControlStream handles DELETE /controlstreams/{id}
//
// @Summary     Delete control stream
// @Description Deletes a control stream resource by ID
// @Tags        Control Streams
// @Param       controlStreamId  path   string  true   "Control Stream ID"
// @Param       cascade          query  bool    false  "Cascade delete to dependent resources"
// @Success     204
// @Failure     404  {object}  map[string]string
// @Failure     409  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /controlstreams/{controlStreamId} [delete]
func (h *ControlStreamHandler) DeleteControlStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "controlStreamId")
	cascade := r.URL.Query().Get("cascade") == "true"
	if err := h.repo.Delete(id, cascade); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "Control stream not found"})
		case errors.Is(err, repository.ErrHasChildren):
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, map[string]string{"error": "Control stream has dependent records; use ?cascade=true to delete"})
		default:
			h.logger.Error("Failed to delete control stream", zap.String("id", id), zap.Error(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to delete control stream"})
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetControlStreamSchema handles GET /controlstreams/{id}/schema
//
// @Summary     Get control stream schema
// @Description Returns the command schema for a given control stream
// @Tags        Control Streams
// @Produce     json
// @Param       controlStreamId  path  string  true  "Control Stream ID"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Router      /controlstreams/{controlStreamId}/schema [get]
func (h *ControlStreamHandler) GetControlStreamSchema(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "controlStreamId")
	schema, err := h.repo.GetSchema(id)
	if err != nil {
		h.logger.Error("Failed to get control stream schema", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Control stream not found"})
		return
	}

	if schema == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Control stream schema not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, schema)
}

// UpdateControlStreamSchema handles PUT /controlstreams/{id}/schema
//
// @Summary     Update control stream schema
// @Description Replaces the command schema for a given control stream
// @Tags        Control Streams
// @Accept      json
// @Param       controlStreamId  path  string          true  "Control Stream ID"
// @Param       schema           body  map[string]any  true  "Control Stream schema"
// @Success     204
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /controlstreams/{controlStreamId}/schema [put]
func (h *ControlStreamHandler) UpdateControlStreamSchema(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "controlStreamId")

	var schema domains.ControlStreamSchema
	if err := render.DecodeJSON(r.Body, &schema); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	if err := h.repo.UpdateSchema(id, &schema); err != nil {
		h.logger.Error("Failed to update control stream schema", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update control stream schema"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
