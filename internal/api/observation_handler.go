package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/contractvalidation"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters/json_formatters"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/resourcevalidation"
	"github.com/go-chi/chi/v5"
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
	creates        *CreateOperations
}

func NewObservationHandler(cfg *config.Config, logger *zap.Logger, repo *repository.ObservationRepository, datastreamRepo *repository.DatastreamRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Observation], creates *CreateOperations) *ObservationHandler {
	return &ObservationHandler{
		cfg:            cfg,
		logger:         logger,
		repo:           repo,
		datastreamRepo: datastreamRepo,
		fc:             fc,
		creates:        creates,
	}
}

// ListObservations returns a paginated list of observations
//
// @Summary     List observations
// @Description Returns a paginated collection of observation resources
// @Tags        Observations
// @Produce     json
// @Param       limit             query  integer  false  "Maximum number of results"
// @Param       cursor            query  string   false  "Opaque pagination cursor"
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
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	observations, _, err := h.repo.List(r.Context(), params, nil)
	if err != nil {
		h.logger.Error("Failed to list observations", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeCollection(w, r, h.logger, h.fc, h.cfg.API.BaseURL, observations, 0, params.QueryParams)
}

// ListDatastreamObservations returns observations for a datastream
//
// @Summary     List datastream observations
// @Description Returns observations associated with a given datastream
// @Tags        Observations
// @Produce     json
// @Param       dataStreamId      path   string   true   "Datastream ID"
// @Param       limit             query  integer  false  "Maximum number of results"
// @Param       cursor            query  string   false  "Opaque pagination cursor"
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
	if _, err := h.datastreamRepo.GetByID(r.Context(), datastreamID); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Datastream", "Failed to load datastream")
		return
	}

	params, err := queryparams.ObservationsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	observations, _, err := h.repo.ListByDatastream(r.Context(), datastreamID, params)
	if err != nil {
		h.logger.Error("Failed to list observations", zap.String("dataStreamId", datastreamID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeCollection(w, r, h.logger, h.fc, h.cfg.API.BaseURL, observations, 0, params.QueryParams)
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

	obs, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Observation", "Failed to load observation")
		return
	}

	writeResource(w, r, h.logger, h.fc, obs)
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
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /observations/{obsId} [put]
func (h *ObservationHandler) UpdateObservation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "obsId")

	existing, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Observation", "Failed to load observation")
		return
	}

	obs, err := h.fc.DeserializeContext(r.Context(), r.Header.Get("Content-Type"), r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize observation", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	datastream, err := h.datastreamRepo.GetByID(r.Context(), existing.DatastreamID)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Parent datastream", "Failed to load parent datastream")
		return
	}
	if err := resourcevalidation.ValidateObservationAgainstDatastreamSchema(obs, datastream, r.Header.Get("Content-Type")); err != nil {
		writeValidationError(w, r, fmt.Errorf("observation does not match parent datastream schema: %w", err))
		return
	}

	obs.ID = id
	obs.DatastreamID = existing.DatastreamID
	if err := h.repo.Update(r.Context(), obs); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Observation", "Failed to update observation")
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

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Observation", "Failed to delete observation")
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
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /datastreams/{dataStreamId}/observations [post]
func (h *ObservationHandler) CreateDatastreamObservation(w http.ResponseWriter, r *http.Request) {
	h.creates.HTTP(w, r, observationCreator, chi.URLParam(r, "dataStreamId"))
}

// observationFormatters builds the observation representation HTTP serves,
// creates decode and MQTT publishes.
func observationFormatters(links *formaters.LinkResolver) *formaters.MultiFormatFormatterCollection[*domains.Observation] {
	return formaters.NewFormatterCollection(formaters.Adapt(json_formatters.NewObservationJSONFormatter(links)))
}

// observationCreator creates an observation in its datastream. The repository
// validates the result against the datastream schema.
var observationCreator = resourceCreator{
	Kind: "observation", Name: "Observation", Contract: contractvalidation.Observation, Codec: formatterCodec[*domains.Observation](observationFormatters), ParentField: "datastream@id",
	Create: func(ctx context.Context, tx *repository.Repositories, in createInput) (string, error) {
		if _, err := tx.Datastream.GetByID(ctx, in.Parent); err != nil {
			return "", parentError("datastream", in.Parent, err)
		}
		observation, err := observationFormatters(in.Links).DeserializeContext(ctx, in.Media, bytes.NewReader(in.Body))
		if err != nil {
			return "", decodeError(err)
		}
		observation.ID = ""
		observation.DatastreamID = in.Parent
		if err := tx.Observation.Create(ctx, observation); err != nil {
			return "", err
		}
		return "/observations/" + url.PathEscape(observation.ID), nil
	},
}
