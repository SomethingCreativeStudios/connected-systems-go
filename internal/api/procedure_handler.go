package api

import (
	"net/http"

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

func (h *ProcedureHandler) ListProcedures(w http.ResponseWriter, r *http.Request) {
	params := queryparams.ProceduresQueryParams{}.BuildFromRequest(r)

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
	render.JSON(w, r, collection)
}

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
	render.JSON(w, r, serialized)
}

func (h *ProcedureHandler) CreateProcedure(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	procedure, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize procedure", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	if err := h.repo.Create(procedure); err != nil {
		h.logger.Error("Failed to create procedure", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create procedure"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, procedure)
	if err != nil {
		h.logger.Error("Failed to serialize procedure", zap.String("id", procedure.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize procedure"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serialized)
}

func (h *ProcedureHandler) UpdateProcedure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	procedure, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize procedure", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	procedure.ID = id
	if err := h.repo.Update(procedure); err != nil {
		h.logger.Error("Failed to update procedure", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update procedure"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, procedure)
	if err != nil {
		h.logger.Error("Failed to serialize procedure", zap.String("id", procedure.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize procedure"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, serialized)
}

func (h *ProcedureHandler) DeleteProcedure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(id); err != nil {
		h.logger.Error("Failed to delete procedure", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete procedure"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
