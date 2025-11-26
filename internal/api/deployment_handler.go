package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/model/serializers"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// DeploymentHandler handles Deployment resource requests
type DeploymentHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.DeploymentRepository
	fc     *serializers.MultiFormatFormatterCollection[*domains.Deployment]
}

// NewDeploymentHandler creates a new DeploymentHandler
func NewDeploymentHandler(cfg *config.Config, logger *zap.Logger, repo *repository.DeploymentRepository, fc *serializers.MultiFormatFormatterCollection[*domains.Deployment]) *DeploymentHandler {
	return &DeploymentHandler{cfg: cfg, logger: logger, repo: repo, fc: fc}
}

func (h *DeploymentHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	params := queryparams.DeploymentsQueryParams{}.BuildFromRequest(r)

	deployments, total, err := h.repo.List(params)
	if err != nil {
		h.logger.Error("Failed to list deployments", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, deployments, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, collection)
}

func (h *DeploymentHandler) GetDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	deployment, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get deployment", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Deployment not found"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, deployment)
	if err != nil {
		h.logger.Error("Failed to serialize deployment", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize deployment"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, serialized)
}

func (h *DeploymentHandler) CreateDeployment(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	deployment, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize deployment", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	if err := h.repo.Create(deployment); err != nil {
		h.logger.Error("Failed to create deployment", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create deployment"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, deployment)
	if err != nil {
		h.logger.Error("Failed to serialize deployment", zap.String("id", deployment.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize deployment"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serialized)
}

func (h *DeploymentHandler) UpdateDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	deployment, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize deployment", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	deployment.ID = id
	if err := h.repo.Update(deployment); err != nil {
		h.logger.Error("Failed to update deployment", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update deployment"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, deployment)
	if err != nil {
		h.logger.Error("Failed to serialize deployment", zap.String("id", deployment.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize deployment"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, serialized)
}

func (h *DeploymentHandler) DeleteDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(id); err != nil {
		h.logger.Error("Failed to delete deployment", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete deployment"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List all subdeployments
func (h *DeploymentHandler) ListSubdeployments(w http.ResponseWriter, r *http.Request) {
	params := queryparams.DeploymentsQueryParams{}.BuildFromRequest(r)

	deployments, total, err := h.repo.List(params)
	if err != nil {
		h.logger.Error("Failed to list subdeployments", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, deployments, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, collection)
}

// Add subdeployment to a deployment
func (h *DeploymentHandler) AddSubdeployment(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	subdeployment, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize subdeployment", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	subdeployment.ParentDeploymentID = &parentID

	if err := h.repo.Create(subdeployment); err != nil {
		h.logger.Error("Failed to create subdeployment", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create subdeployment"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, subdeployment)
	if err != nil {
		h.logger.Error("Failed to serialize subdeployment", zap.String("id", subdeployment.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize subdeployment"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serialized)
}
