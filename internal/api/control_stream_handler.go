package api

import (
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
func (h *ControlStreamHandler) ListControlStreams(w http.ResponseWriter, r *http.Request) {
	params := queryparams.ControlStreamsQueryParams{}.BuildFromRequest(r)

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
func (h *ControlStreamHandler) ListSystemControlStreams(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemId")
	if systemID == "" {
		systemID = chi.URLParam(r, "id")
	}
	params := queryparams.ControlStreamsQueryParams{}.BuildFromRequest(r)

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
	render.JSON(w, r, serialized)
}

// CreateControlStream handles POST /systems/{id}/controlstreams
func (h *ControlStreamHandler) CreateControlStream(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemId")
	if systemID == "" {
		systemID = chi.URLParam(r, "id")
	}

	contentType := r.Header.Get("Content-Type")
	cs, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize control stream", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	if systemID != "" {
		cs.SystemID = &systemID
		if cs.SystemLink == nil {
			cs.SystemLink = &common_shared.Link{Href: "systems/" + systemID}
		}
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
func (h *ControlStreamHandler) UpdateControlStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "controlStreamId")

	contentType := r.Header.Get("Content-Type")
	cs, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize control stream", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	cs.ID = id
	if err := h.repo.Update(cs); err != nil {
		h.logger.Error("Failed to update control stream", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update control stream"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteControlStream handles DELETE /controlstreams/{id}
func (h *ControlStreamHandler) DeleteControlStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "controlStreamId")
	if err := h.repo.Delete(id); err != nil {
		h.logger.Error("Failed to delete control stream", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete control stream"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetControlStreamSchema handles GET /controlstreams/{id}/schema
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
