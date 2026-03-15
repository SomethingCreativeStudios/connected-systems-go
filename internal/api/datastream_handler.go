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

func (h *DatastreamHandler) ListDatastreams(w http.ResponseWriter, r *http.Request) {
	params := queryparams.DatastreamsQueryParams{}.BuildFromRequest(r)

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

func (h *DatastreamHandler) ListSystemDatastreams(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemId")
	if systemID == "" {
		systemID = chi.URLParam(r, "id")
	}
	params := queryparams.DatastreamsQueryParams{}.BuildFromRequest(r)

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

func (h *DatastreamHandler) CreateDatastream(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemId")
	if systemID == "" {
		systemID = chi.URLParam(r, "id")
	}

	contentType := r.Header.Get("Content-Type")
	datastream, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize datastream", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	if systemID != "" {
		datastream.SystemID = &systemID
		if datastream.SystemLink == nil {
			datastream.SystemLink = &common_shared.Link{Href: "systems/" + systemID}
		}
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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	datastream.ID = id
	if datastream.SystemLink == nil {
		datastream.SystemLink = existing.SystemLink
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

func (h *DatastreamHandler) DeleteDatastream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dataStreamId")
	cascade := r.URL.Query().Get("cascade") == "true"
	if err := h.repo.Delete(id, cascade); err != nil {
		h.logger.Error("Failed to delete datastream", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete datastream"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

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
