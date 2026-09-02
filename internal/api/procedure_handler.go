package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// ProcedureHandler handles Procedure resource requests
type ProcedureHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.ProcedureRepository
	fc     *formaters.MultiFormatFormatterCollection[*domains.Procedure]
}

// NewProcedureHandler creates a new ProcedureHandler
func NewProcedureHandler(cfg *config.Config, logger *zap.Logger, repo *repository.ProcedureRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Procedure]) *ProcedureHandler {
	return &ProcedureHandler{cfg: cfg, logger: logger, repo: repo, fc: fc}
}

// ListProcedures returns a paginated list of procedures
//
// @Summary     List procedures
// @Description Returns a paginated collection of procedure resources
// @Tags        Procedures
// @Produce     json
// @Param       limit               query  integer  false  "Maximum number of results"
// @Param       cursor              query  string   false  "Opaque pagination cursor"
// @Param       id                  query  string   false  "Comma-separated resource IDs"
// @Param       q                   query  string   false  "Comma-separated keywords for full-text search"
// @Param       dateTime            query  string   false  "Date-time or interval (RFC 3339), e.g. 2023-01-01T00:00:00Z/2023-12-31T23:59:59Z"
// @Param       observedProperty    query  string   false  "Comma-separated observed property IDs"
// @Param       controlledProperty  query  string   false  "Comma-separated controlled property IDs"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /procedures [get]
func (h *ProcedureHandler) ListProcedures(w http.ResponseWriter, r *http.Request) {
	params, err := queryparams.ProceduresQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	procedures, total, err := h.repo.List(params)
	if err != nil {
		h.logger.Error("Failed to list procedures", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, procedures, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusOK)
	json.NewEncoder(w).Encode(collection)
}

// GetProcedure returns a single procedure by ID
//
// @Summary     Get procedure
// @Description Returns a single procedure resource by ID
// @Tags        Procedures
// @Produce     json
// @Param       id  path  string  true  "Procedure ID"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /procedures/{id} [get]
func (h *ProcedureHandler) GetProcedure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	procedure, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get procedure", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Procedure not found"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, procedure)
	if err != nil {
		h.logger.Error("Failed to serialize procedure", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize procedure"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusOK)
	json.NewEncoder(w).Encode(serialized)
}

// CreateProcedure creates a new procedure
//
// @Summary     Create procedure
// @Description Creates a new procedure resource
// @Tags        Procedures
// @Accept      json
// @Param       procedure  body  map[string]any  true  "Procedure resource"
// @Success     201
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     500  {object}  map[string]string
// @Router      /procedures [post]
func (h *ProcedureHandler) CreateProcedure(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	procedure, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize procedure", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	if err := h.repo.Create(procedure); err != nil {
		h.logger.Error("Failed to create procedure", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create procedure"})
		return
	}

	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/procedures/" + procedure.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

// UpdateProcedure replaces a procedure by ID
//
// @Summary     Update procedure
// @Description Replaces a procedure resource by ID
// @Tags        Procedures
// @Accept      json
// @Param       id         path  string          true  "Procedure ID"
// @Param       procedure  body  map[string]any  true  "Procedure resource"
// @Success     204
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     500  {object}  map[string]string
// @Router      /procedures/{id} [put]
func (h *ProcedureHandler) UpdateProcedure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	procedure, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize procedure", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	procedure.ID = id
	if err := h.repo.Update(procedure); err != nil {
		h.logger.Error("Failed to update procedure", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update procedure"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteProcedure deletes a procedure by ID
//
// @Summary     Delete procedure
// @Description Deletes a procedure resource by ID
// @Tags        Procedures
// @Param       id  path  string  true  "Procedure ID"
// @Success     204
// @Failure     404  {object}  map[string]string
// @Failure     409  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /procedures/{id} [delete]
func (h *ProcedureHandler) DeleteProcedure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(id); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "Procedure not found"})
		case errors.Is(err, repository.ErrHasChildren):
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, map[string]string{"error": "Procedure has dependent records"})
		default:
			h.logger.Error("Failed to delete procedure", zap.String("id", id), zap.Error(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to delete procedure"})
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
