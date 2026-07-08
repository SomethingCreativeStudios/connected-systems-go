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

// ListObservations returns a paginated list of observations
//
// @Summary     List observations
// @Description Returns a paginated collection of observation resources
// @Tags        Observations
// @Produce     json
// @Param       limit             query  integer  false  "Maximum number of results"
// @Param       offset            query  integer  false  "Result offset"
// @Param       id                query  string   false  "Comma-separated resource IDs"
// @Param       q                 query  string   false  "Comma-separated keywords for full-text search"
// @Param       dataStream        query  string   false  "Comma-separated datastream IDs"
// @Param       system            query  string   false  "Comma-separated system IDs"
// @Param       foi               query  string   false  "Comma-separated feature of interest IDs"
// @Param       observedProperty  query  string   false  "Comma-separated observed property IDs"
// @Param       phenomenonTime    query  string   false  "Phenomenon time filter (RFC 3339 date-time or interval)"
// @Param       resultTime        query  string   false  "Result time filter (RFC 3339 date-time or interval)"
// @Success     200  {object}  ObservationCollectionResponse
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /observations [get]
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

// ListDatastreamObservations returns observations for a datastream
//
// @Summary     List datastream observations
// @Description Returns observations associated with a given datastream
// @Tags        Observations
// @Produce     json
// @Param       dataStreamId      path   string   true   "Datastream ID"
// @Param       limit             query  integer  false  "Maximum number of results"
// @Param       offset            query  integer  false  "Result offset"
// @Param       q                 query  string   false  "Comma-separated keywords for full-text search"
// @Param       foi               query  string   false  "Comma-separated feature of interest IDs"
// @Param       observedProperty  query  string   false  "Comma-separated observed property IDs"
// @Param       phenomenonTime    query  string   false  "Phenomenon time filter (RFC 3339 date-time or interval)"
// @Param       resultTime        query  string   false  "Result time filter (RFC 3339 date-time or interval)"
// @Success     200  {object}  ObservationCollectionResponse
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /datastreams/{dataStreamId}/observations [get]
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

// GetObservation returns a single observation by ID
//
// @Summary     Get observation
// @Description Returns a single observation resource by ID
// @Tags        Observations
// @Produce     json
// @Param       obsId  path  string  true  "Observation ID"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /observations/{obsId} [get]
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

// UpdateObservation replaces an observation by ID
//
// @Summary     Update observation
// @Description Replaces an observation resource by ID
// @Tags        Observations
// @Accept      json
// @Param       obsId        path  string          true  "Observation ID"
// @Param       observation  body  map[string]any  true  "Observation resource"
// @Success     204
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /observations/{obsId} [put]
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

// DeleteObservation deletes an observation by ID
//
// @Summary     Delete observation
// @Description Deletes an observation resource by ID
// @Tags        Observations
// @Param       obsId  path  string  true  "Observation ID"
// @Success     204
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /observations/{obsId} [delete]
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

// CreateDatastreamObservation creates an observation in a datastream
//
// @Summary     Create observation
// @Description Creates a new observation in the given datastream
// @Tags        Observations
// @Accept      json
// @Param       dataStreamId  path  string          true  "Datastream ID"
// @Param       observation   body  map[string]any  true  "Observation resource"
// @Success     201
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /datastreams/{dataStreamId}/observations [post]
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
