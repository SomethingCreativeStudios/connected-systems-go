package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
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
}

func NewObservationHandler(cfg *config.Config, logger *zap.Logger, repo *repository.ObservationRepository, datastreamRepo *repository.DatastreamRepository) *ObservationHandler {
	return &ObservationHandler{
		cfg:            cfg,
		logger:         logger,
		repo:           repo,
		datastreamRepo: datastreamRepo,
	}
}

func (h *ObservationHandler) ListObservations(w http.ResponseWriter, r *http.Request) {
	params := queryparams.ObservationsQueryParams{}.BuildFromRequest(r)

	observations, total, err := h.repo.List(params, nil)
	if err != nil {
		h.logger.Error("Failed to list observations", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	items := make([]any, 0, len(observations))
	for _, obs := range observations {
		items = append(items, obs)
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(observations))

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, ObservationCollectionResponse{Items: items, Links: links})
}

func (h *ObservationHandler) ListDatastreamObservations(w http.ResponseWriter, r *http.Request) {
	datastreamID := chi.URLParam(r, "dataStreamId")
	if _, err := h.datastreamRepo.GetByID(datastreamID); err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Datastream not found"})
		return
	}

	params := queryparams.ObservationsQueryParams{}.BuildFromRequest(r)

	observations, total, err := h.repo.ListByDatastream(datastreamID, params)
	if err != nil {
		h.logger.Error("Failed to list observations", zap.String("dataStreamId", datastreamID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	items := make([]any, 0, len(observations))
	for _, obs := range observations {
		items = append(items, obs)
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(observations))

	w.Header().Set("Content-Type", "application/json")
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

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, obs)
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

	obs, err := decodeObservationPayload(r)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
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

	if _, err := h.repo.GetByID(id); err != nil {
		h.logger.Error("Observation not found", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Observation not found"})
		return
	}

	if err := h.repo.Delete(id); err != nil {
		h.logger.Error("Failed to delete observation", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete observation"})
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

	obs, err := decodeObservationPayload(r)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
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

func decodeObservationPayload(r *http.Request) (*domains.Observation, error) {
	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}

	obs := &domains.Observation{}

	if sfID, ok := raw["samplingFeature@id"].(string); ok && sfID != "" {
		obs.SamplingFeatureID = &sfID
	}

	if procRaw, ok := raw["procedure@link"]; ok {
		procBytes, _ := json.Marshal(procRaw)
		var procLink common_shared.Link
		if err := json.Unmarshal(procBytes, &procLink); err != nil || procLink.Href == "" {
			return nil, &decodeError{msg: "Invalid procedure@link payload"}
		}
		obs.ProcedureLink = &procLink
	}

	if resultTimeRaw, ok := raw["resultTime"].(string); ok && resultTimeRaw != "" {
		resultTime, err := time.Parse(time.RFC3339, resultTimeRaw)
		if err != nil {
			return nil, &decodeError{msg: "Invalid resultTime format"}
		}
		obs.ResultTime = resultTime
	}

	if phenomenonTimeRaw, ok := raw["phenomenonTime"].(string); ok && phenomenonTimeRaw != "" {
		phenomenonTime, err := time.Parse(time.RFC3339, phenomenonTimeRaw)
		if err != nil {
			return nil, &decodeError{msg: "Invalid phenomenonTime format"}
		}
		obs.PhenomenonTime = &phenomenonTime
	}

	if parametersRaw, exists := raw["parameters"]; exists {
		if paramsObj, ok := parametersRaw.(map[string]any); ok {
			obs.Parameters = common_shared.Properties(paramsObj)
		} else {
			return nil, &decodeError{msg: "Invalid parameters payload"}
		}
	}

	if resultRaw, exists := raw["result"]; exists {
		b, err := json.Marshal(resultRaw)
		if err != nil {
			return nil, &decodeError{msg: "Invalid result payload"}
		}
		obs.Result = b
	}

	if resultLinkRaw, exists := raw["result@link"]; exists {
		b, _ := json.Marshal(resultLinkRaw)
		var resultLink common_shared.Link
		if err := json.Unmarshal(b, &resultLink); err != nil || resultLink.Href == "" {
			return nil, &decodeError{msg: "Invalid result@link payload"}
		}
		obs.ResultLink = &resultLink
	}

	if len(obs.Result) == 0 && obs.ResultLink == nil {
		return nil, &decodeError{msg: "Either result or result@link is required"}
	}
	if obs.ResultTime.IsZero() {
		return nil, &decodeError{msg: "resultTime is required"}
	}

	return obs, nil
}

type decodeError struct {
	msg string
}

func (e *decodeError) Error() string {
	return e.msg
}
