package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/contractvalidation"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters/json_formatters"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"go.uber.org/zap"
)

// DatastreamCollectionResponse follows datastreams-only.yaml collection shape.
type DatastreamCollectionResponse struct {
	Items []any               `json:"items"`
	Links common_shared.Links `json:"links,omitempty"`
}

// DatastreamHandler handles datastream endpoints.
type DatastreamHandler struct {
	cfg       *config.Config
	logger    *zap.Logger
	repo      *repository.DatastreamRepository
	fc        *formaters.MultiFormatFormatterCollection[*domains.Datastream]
	validator *contractvalidation.Validator
	creates   *CreateOperations
}

func NewDatastreamHandler(cfg *config.Config, logger *zap.Logger, repo *repository.DatastreamRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Datastream], validator *contractvalidation.Validator, creates *CreateOperations) *DatastreamHandler {
	return &DatastreamHandler{cfg: cfg, logger: logger, repo: repo, fc: fc, validator: validator, creates: creates}
}

// ListDatastreams returns a paginated list of datastreams
//
// @Summary     List datastreams
// @Description Returns a paginated collection of datastream resources
// @Tags        Datastreams
// @Produce     json
// @Param       limit             query  integer  false  "Maximum number of results"
// @Param       cursor            query  string   false  "Opaque pagination cursor"
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
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	datastreams, _, err := h.repo.List(r.Context(), params, nil)
	if err != nil {
		h.logger.Error("Failed to list datastreams", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeCollection(w, r, h.logger, h.fc, h.cfg.API.BaseURL, datastreams, 0, params.QueryParams)
}

// ListSystemDatastreams returns datastreams for a system
//
// @Summary     List system datastreams
// @Description Returns datastreams associated with a given system
// @Tags        Datastreams
// @Produce     json
// @Param       id                path   string   true   "System ID"
// @Param       limit             query  integer  false  "Maximum number of results"
// @Param       cursor            query  string   false  "Opaque pagination cursor"
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
	systemID := chi.URLParam(r, "id")
	params, err := queryparams.DatastreamsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	datastreams, _, err := h.repo.List(r.Context(), params, &systemID)
	if err != nil {
		h.logger.Error("Failed to list datastreams for system", zap.String("systemId", systemID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeCollection(w, r, h.logger, h.fc, h.cfg.API.BaseURL, datastreams, 0, params.QueryParams)
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

	datastream, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Datastream", "Failed to load datastream")
		return
	}

	writeResource(w, r, h.logger, h.fc, datastream)
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
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/datastreams [post]
func (h *DatastreamHandler) CreateDatastream(w http.ResponseWriter, r *http.Request) {
	h.creates.HTTP(w, r, datastreamCreator, chi.URLParam(r, "id"))
}

// datastreamFormatters builds the datastream representation HTTP serves,
// creates decode and MQTT publishes.
func datastreamFormatters(links *formaters.LinkResolver) *formaters.MultiFormatFormatterCollection[*domains.Datastream] {
	return formaters.NewFormatterCollection(formaters.Adapt(json_formatters.NewDatastreamJSONFormatter(links)))
}

// datastreamCreator creates a datastream under its system. schema is required
// on create per the OGC CS Part 2 dataStream create schema.
var datastreamCreator = resourceCreator{
	Kind: "datastream", Name: "Datastream", Contract: contractvalidation.Datastream, Codec: formatterCodec[*domains.Datastream](datastreamFormatters),
	Create: func(ctx context.Context, tx *repository.Repositories, in createInput) (string, error) {
		if _, err := tx.System.GetByID(ctx, in.Parent); err != nil {
			return "", parentError("system", in.Parent, err)
		}
		datastream, err := datastreamFormatters(in.Links).DeserializeContext(ctx, in.Media, bytes.NewReader(in.Body))
		if err != nil {
			return "", decodeError(err)
		}
		if datastream.Schema == nil {
			return "", invalid(errors.New("schema is required"))
		}
		if err := validateDatastreamSchema(datastream.Schema); err != nil {
			return "", invalid(err)
		}
		datastream.ID = ""
		datastream.SystemID = &in.Parent
		if err := tx.Datastream.Create(ctx, datastream); err != nil {
			return "", err
		}
		return "/datastreams/" + url.PathEscape(datastream.ID), nil
	},
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
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /datastreams/{dataStreamId} [put]
func (h *DatastreamHandler) UpdateDatastream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dataStreamId")
	existing, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Datastream", "Failed to load datastream")
		return
	}

	contentType := r.Header.Get("Content-Type")
	datastream, err := h.fc.DeserializeContext(r.Context(), contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize datastream", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	if err := validateDatastreamSchema(datastream.Schema); err != nil {
		writeValidationError(w, r, err)
		return
	}

	datastream.ID = id
	// system@link is read-only per spec: a datastream cannot be reparented.
	datastream.SystemID = existing.SystemID
	if err := h.repo.Update(r.Context(), datastream); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Datastream", "Failed to update datastream")
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
	if err := h.repo.Delete(r.Context(), id, cascade); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Datastream", "Failed to delete datastream")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetDatastreamSchema returns the schema for a datastream
//
// @Summary     Get datastream schema
// @Description Returns the observation schema for a given datastream. With the
// @Description obsFormat query parameter, returns the schema registered for
// @Description that observation format.
// @Tags        Datastreams
// @Produce     json
// @Param       dataStreamId  path   string  true   "Datastream ID"
// @Param       obsFormat     query  string  false  "Observation format media type"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Router      /datastreams/{dataStreamId}/schema [get]
func (h *DatastreamHandler) GetDatastreamSchema(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dataStreamId")
	schema, schemas, err := h.repo.GetSchema(r.Context(), id)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Datastream", "Failed to load datastream")
		return
	}

	if obsFormat := r.URL.Query().Get("obsFormat"); obsFormat != "" {
		schema = schemas.ForFormat(obsFormat)
		if schema == nil {
			writeError(w, r, http.StatusNotFound, "Datastream has no schema for format "+obsFormat)
			return
		}
	}

	if schema == nil {
		writeError(w, r, http.StatusNotFound, "Datastream schema not found")
		return
	}

	w.Header().Set("Content-Type", mediatype.JSON)
	render.JSON(w, r, schema)
}

// UpdateDatastreamSchema upserts a schema for a datastream
//
// @Summary     Update datastream schema
// @Description Registers the schema for its observation format (replacing any
// @Description existing schema with the same obsFormat, adding a new format
// @Description otherwise) and makes it the datastream's current schema. The
// @Description datastream's formats list reflects all registered schemas.
// @Tags        Datastreams
// @Accept      json
// @Param       dataStreamId  path  string          true  "Datastream ID"
// @Param       schema        body  map[string]any  true  "Datastream schema"
// @Success     204
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     500  {object}  map[string]string
// @Router      /datastreams/{dataStreamId}/schema [put]
func (h *DatastreamHandler) UpdateDatastreamSchema(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "dataStreamId")

	body, err := validateRawRequestBody(r, h.validator, contractvalidation.Datastream)
	if err != nil {
		writeDeserializeError(w, r, err)
		return
	}
	schema, err := decodeStrictJSON[domains.DatastreamSchema](body)
	if err != nil {
		writeDeserializeError(w, r, err)
		return
	}

	if err := validateDatastreamSchema(&schema); err != nil {
		writeValidationError(w, r, err)
		return
	}

	if err := h.repo.UpdateSchema(r.Context(), id, &schema); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Datastream", "Failed to update datastream schema")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
