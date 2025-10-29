package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// SystemHandler handles System resource requests
type SystemHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.SystemRepository
}

// NewSystemHandler creates a new SystemHandler
func NewSystemHandler(cfg *config.Config, logger *zap.Logger, repo *repository.SystemRepository) *SystemHandler {
	return &SystemHandler{
		cfg:    cfg,
		logger: logger,
		repo:   repo,
	}
}

// ListSystems retrieves a list of systems
func (h *SystemHandler) ListSystems(w http.ResponseWriter, r *http.Request) {
	params := queryparams.SystemQueryParams{}.BuildFromRequest(r)

	systems, total, err := h.repo.List(params)
	if err != nil {
		h.logger.Error("Failed to list systems", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	render.JSON(w, r, model.FeatureCollection[domains.SystemGeoJSONFeature, *domains.System]{}.BuildCollection(systems, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams))
}

// GetSystem retrieves a single system by ID
func (h *SystemHandler) GetSystem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	system, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get system", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "System not found"})
		return
	}

	render.JSON(w, r, system.ToGeoJSON())
}

// CreateSystem creates a new system
func (h *SystemHandler) CreateSystem(w http.ResponseWriter, r *http.Request) {
	system, err := domains.System{}.BuildFromRequest(r, w)

	if err != nil {
		return // Error already handled in buildSystem
	}

	if err := h.repo.Create(&system); err != nil {
		h.logger.Error("Failed to create system", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create system"})
		return
	}

	render.Status(r, http.StatusCreated)

	// Add associations links
	system.Links = append(system.Links, h.repo.BuildSystemAssociations(system.ID)...)

	render.JSON(w, r, system.ToGeoJSON())
}

// UpdateSystem updates a system (PUT)
func (h *SystemHandler) UpdateSystem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var system domains.System
	if err := render.DecodeJSON(r.Body, &system); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	system.ID = id
	if err := h.repo.Update(&system); err != nil {
		h.logger.Error("Failed to update system", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update system"})
		return
	}

	render.JSON(w, r, system.ToGeoJSON())
}

// PatchSystem patches a system (PATCH)
func (h *SystemHandler) PatchSystem(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement PATCH support
	render.Status(r, http.StatusNotImplemented)
	render.JSON(w, r, map[string]string{"error": "PATCH not implemented"})
}

// DeleteSystem deletes a system
func (h *SystemHandler) DeleteSystem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cascade := r.URL.Query().Get("cascade") == "true"

	if err := h.repo.Delete(id, cascade); err != nil {
		h.logger.Error("Failed to delete system", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete system"})
		return
	}

	render.Status(r, http.StatusNoContent)
}

// GetSubsystems retrieves subsystems of a system
func (h *SystemHandler) GetSubsystems(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")
	recursive := r.URL.Query().Get("recursive") == "true"
	params := queryparams.SystemQueryParams{}.BuildFromRequest(r)

	systems, err := h.repo.GetSubsystems(parentID, recursive)
	if err != nil {
		h.logger.Error("Failed to get subsystems", zap.String("parentID", parentID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to get subsystems"})
		return
	}

	render.JSON(w, r, model.FeatureCollection[domains.SystemGeoJSONFeature, *domains.System]{}.BuildCollection(systems, h.cfg.API.BaseURL+r.URL.Path, len(systems), r.URL.Query(), params.QueryParams))
}

// Add subsystem to a system
func (h *SystemHandler) AddSubsystem(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")

	system, err := domains.System{}.BuildFromRequest(r, w)

	if err != nil {
		return // Error already handled in buildSystem
	}

	system.ParentSystemID = &parentID
	system.Links = append(system.Links, common_shared.Link{
		Rel:  "parent",
		Href: h.cfg.API.BaseURL + "/systems/" + parentID,
	})

	if err := h.repo.Create(&system); err != nil {
		h.logger.Error("Failed to create subsystem", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create subsystem"})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, system.ToGeoJSON())
}
