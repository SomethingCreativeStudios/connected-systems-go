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

// ObservationCollectionResponse follows observation-only.yaml collection shape.
type ObservationCollectionResponse struct {
	Items []any               `json:"items"`
	Links common_shared.Links `json:"links,omitempty"`
}

// ObservationHandler handles Observation resource requests.
type ObservationHandler struct {
	cfg            *config.Config
	logger         *zap.Logger
	repo           *repository.ObservationRepository
	datastreamRepo *repository.DatastreamRepository
	fc             *formaters.MultiFormatFormatterCollection[*domains.Observation]
}

func NewObservationHandler(cfg *config.Config, logger *zap.Logger, repo *repository.ObservationRepository, datastreamRepo *repository.DatastreamRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Observation]) *ObservationHandler {
	return &ObservationHandler{
		cfg:            cfg,
		logger:         logger,
		repo:           repo,
		datastreamRepo: datastreamRepo,
		fc:             fc,
	}
}

func (h *ObservationHandler) ListObservations(w http.ResponseWriter, r *http.Request) {
	params, err := queryparams.ObservationsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	observations, total, err := h.repo.List(params, nil)
	if err != nil {
		h.logger.Error("Failed to list observations", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	items, err := h.fc.SerializeAll(acceptHeader, observations)
	if err != nil {
		h.logger.Error("Failed to serialize observations", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize observations"})
		return
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(observations))

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, ObservationCollectionResponse{Items: items, Links: links})
}

func (h *ObservationHandler) ListDatastreamObservations(w http.ResponseWriter, r *http.Request) {
	datastreamID := chi.URLParam(r, "dataStreamId")
	if _, err := h.datastreamRepo.GetByID(datastreamID); err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Datastream not found"})
		return
	}

	params, err := queryparams.ObservationsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	observations, total, err := h.repo.ListByDatastream(datastreamID, params)
	if err != nil {
		h.logger.Error("Failed to list observations", zap.String("dataStreamId", datastreamID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	items, err := h.fc.SerializeAll(acceptHeader, observations)
	if err != nil {
		h.logger.Error("Failed to serialize observations", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize observations"})
		return
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(observations))

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, ObservationCollectionResponse{Items: items, Links: links})
}

func (h *ObservationHandler) GetObservation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "obsId")

	obs, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get observation", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Observation not found"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, obs)
	if err != nil {
		h.logger.Error("Failed to serialize observation", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize observation"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, serialized)
}

func (h *ObservationHandler) UpdateObservation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "obsId")

	existing, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Observation not found", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Observation not found"})
		return
	}

	obs, err := h.fc.Deserialize(r.Header.Get("Content-Type"), r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize observation", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	datastream, err := h.datastreamRepo.GetByID(existing.DatastreamID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Parent datastream not found"})
		return
	}
	if err := validateObservationAgainstDatastreamSchema(obs, datastream, r.Header.Get("Content-Type")); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Observation does not match parent datastream schema: " + err.Error()})
		return
	}

	obs.ID = id
	obs.DatastreamID = existing.DatastreamID
	if err := h.repo.Update(obs); err != nil {
		h.logger.Error("Failed to update observation", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update observation"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ObservationHandler) DeleteObservation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "obsId")

	if err := h.repo.Delete(id); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "Observation not found"})
		default:
			h.logger.Error("Failed to delete observation", zap.String("id", id), zap.Error(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to delete observation"})
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ObservationHandler) CreateDatastreamObservation(w http.ResponseWriter, r *http.Request) {
	datastreamID := chi.URLParam(r, "dataStreamId")
	datastream, err := h.datastreamRepo.GetByID(datastreamID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Datastream not found"})
		return
	}

	obs, err := h.fc.Deserialize(r.Header.Get("Content-Type"), r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize observation", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	if err := validateObservationAgainstDatastreamSchema(obs, datastream, r.Header.Get("Content-Type")); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Observation does not match parent datastream schema: " + err.Error()})
		return
	}

	obs.DatastreamID = datastreamID
	if err := h.repo.Create(obs); err != nil {
		h.logger.Error("Failed to create observation", zap.String("dataStreamId", datastreamID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create observation"})
		return
	}

	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/observations/" + obs.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}
