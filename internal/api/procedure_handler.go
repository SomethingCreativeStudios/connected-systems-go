package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// ProcedureHandler handles Procedure resource requests
type ProcedureHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.ProcedureRepository
}

// NewProcedureHandler creates a new ProcedureHandler
func NewProcedureHandler(cfg *config.Config, logger *zap.Logger, repo *repository.ProcedureRepository) *ProcedureHandler {
	return &ProcedureHandler{cfg: cfg, logger: logger, repo: repo}
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

	render.JSON(w, r, model.FeatureCollection[domains.ProcedureGeoJSONFeature, *domains.Procedure]{}.BuildCollection(procedures, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams))

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

	render.JSON(w, r, procedure.ToGeoJSON())
}

func (h *ProcedureHandler) CreateProcedure(w http.ResponseWriter, r *http.Request) {
	procedure, err := domains.Procedure{}.BuildFromRequest(r, w)

	if err != nil {
		return // Error already handled in buildProcedure
	}

	if err := h.repo.Create(&procedure); err != nil {
		h.logger.Error("Failed to create procedure", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create procedure"})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, procedure.ToGeoJSON())
}

func (h *ProcedureHandler) UpdateProcedure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var procedure domains.Procedure
	if err := render.DecodeJSON(r.Body, &procedure); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	procedure.ID = id
	if err := h.repo.Update(&procedure); err != nil {
		h.logger.Error("Failed to update procedure", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update procedure"})
		return
	}

	render.JSON(w, r, procedure.ToGeoJSON())
}

func (h *ProcedureHandler) PatchProcedure(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusNotImplemented)
	render.JSON(w, r, map[string]string{"message": "Not implemented"})
}

func (h *ProcedureHandler) DeleteProcedure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(id); err != nil {
		h.logger.Error("Failed to delete procedure", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete procedure"})
		return
	}

	render.Status(r, http.StatusNoContent)
}
