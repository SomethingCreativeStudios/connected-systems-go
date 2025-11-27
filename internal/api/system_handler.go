package api

import (
	"net/http"

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

// SystemHandler handles System resource requests
type SystemHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.SystemRepository
	fc     *formaters.MultiFormatFormatterCollection[*domains.System]
	// deployment dependencies for server-side reuse
	deploymentRepo *repository.DeploymentRepository
	deploymentFC   *formaters.MultiFormatFormatterCollection[*domains.Deployment]
}

// NewSystemHandler creates a new SystemHandler
func NewSystemHandler(cfg *config.Config, logger *zap.Logger, repo *repository.SystemRepository, fc *formaters.MultiFormatFormatterCollection[*domains.System], deploymentRepo *repository.DeploymentRepository, deploymentFC *formaters.MultiFormatFormatterCollection[*domains.Deployment]) *SystemHandler {
	return &SystemHandler{
		cfg:            cfg,
		logger:         logger,
		repo:           repo,
		fc:             fc,
		deploymentRepo: deploymentRepo,
		deploymentFC:   deploymentFC,
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

	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, systems, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, collection)
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

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, system)
	if err != nil {
		h.logger.Error("Failed to serialize system", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize system"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, serialized)
}

// CreateSystem creates a new system
func (h *SystemHandler) CreateSystem(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	system, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize system", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	if err := h.repo.Create(system); err != nil {
		h.logger.Error("Failed to create system", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create system"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, system)
	if err != nil {
		h.logger.Error("Failed to serialize system", zap.String("id", system.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize system"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serialized)
}

// UpdateSystem updates a system (PUT)
func (h *SystemHandler) UpdateSystem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	system, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize system", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	system.ID = id
	if err := h.repo.Update(system.ID, system); err != nil {
		h.logger.Error("Failed to update system", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update system"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, system)
	if err != nil {
		h.logger.Error("Failed to serialize system", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize system"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, serialized)
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

	w.WriteHeader(http.StatusNoContent)
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

	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, systems, h.cfg.API.BaseURL+r.URL.Path, len(systems), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, collection)
}

// GetDeployments retrieves deployments associated with a system
func (h *SystemHandler) GetDeployments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Build optional pagination params from request
	params := queryparams.DeploymentsQueryParams{}.BuildFromRequest(r)

	// Use deployment repository helper to find deployments associated with this system
	deployments, total, err := h.deploymentRepo.List(params)
	if err != nil {
		h.logger.Error("Failed to get deployments for system", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to get deployments"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	collection := h.deploymentFC.BuildCollection(acceptHeader, deployments, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.deploymentFC.GetResponseContentType(acceptHeader))
	render.JSON(w, r, collection)
}

// Add subsystem to a system
func (h *SystemHandler) AddSubsystem(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	system, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize system", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	system.ParentSystemID = &parentID
	system.Links = append(system.Links, common_shared.Link{
		Rel:  "parent",
		Href: h.cfg.API.BaseURL + "/systems/" + parentID,
	})

	if err := h.repo.Create(system); err != nil {
		h.logger.Error("Failed to create subsystem", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create subsystem"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, system)
	if err != nil {
		h.logger.Error("Failed to serialize system", zap.String("id", system.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize system"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serialized)
}
