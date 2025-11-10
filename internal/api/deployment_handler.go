package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model"
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
	sc     *serializers.SerializerCollection[domains.DeploymentGeoJSONFeature, *domains.Deployment]
	fc     model.FeatureCollection[domains.DeploymentGeoJSONFeature, *domains.Deployment]
}

// NewDeploymentHandler creates a new DeploymentHandler
func NewDeploymentHandler(cfg *config.Config, logger *zap.Logger, repo *repository.DeploymentRepository, s *serializers.SerializerCollection[domains.DeploymentGeoJSONFeature, *domains.Deployment]) *DeploymentHandler {
	return &DeploymentHandler{cfg: cfg, logger: logger, repo: repo, sc: s, fc: model.FeatureCollection[domains.DeploymentGeoJSONFeature, *domains.Deployment]{}}
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

	serializer := h.sc.GetSerializer(r.Header.Get("content-type"))
	render.JSON(w, r, h.fc.BuildCollection(deployments, serializer, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams))
}

func (h *DeploymentHandler) GetDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	deployment, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get deployment", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Procedure not found"})
		return
	}

	deploymentGeoJSON, err := h.sc.Serialize(r.Header.Get("content-type"), deployment)

	if err != nil {
		h.logger.Error("Failed to serialize deployment", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize deployment"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, deploymentGeoJSON)
}

func (h *DeploymentHandler) CreateDeployment(w http.ResponseWriter, r *http.Request) {
	deployment, err := domains.Deployment{}.BuildFromRequest(r, w)

	if err != nil {
		return // Error already handled in buildProcedure
	}

	if err := h.repo.Create(&deployment); err != nil {
		h.logger.Error("Failed to create deployment", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create deployment"})
		return
	}

	deploymentGeoJSON, err := h.sc.Serialize(r.Header.Get("content-type"), &deployment)

	if err != nil {
		h.logger.Error("Failed to serialize deployment", zap.String("id", deployment.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize deployment"})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, deploymentGeoJSON)
}

func (h *DeploymentHandler) UpdateDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var deployment domains.Deployment
	if err := render.DecodeJSON(r.Body, &deployment); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	deployment.ID = id
	if err := h.repo.Update(&deployment); err != nil {
		h.logger.Error("Failed to update procedure", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update procedure"})
		return
	}

	deploymentGeoJSON, err := h.sc.Serialize(r.Header.Get("content-type"), &deployment)

	if err != nil {
		h.logger.Error("Failed to serialize deployment", zap.String("id", deployment.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize deployment"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, deploymentGeoJSON)
}

func (h *DeploymentHandler) DeleteDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(id); err != nil {
		h.logger.Error("Failed to delete deployment", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete deployment"})
		return
	}

	render.Status(r, http.StatusNoContent)
}

// List all subdeployments
func (h *DeploymentHandler) ListSubdeployments(w http.ResponseWriter, r *http.Request) {
	//parentID := chi.URLParam(r, "id")
	params := queryparams.DeploymentsQueryParams{}.BuildFromRequest(r)

	deployments, total, err := h.repo.List(params)
	if err != nil {
		h.logger.Error("Failed to list subdeployments", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	serializer := h.sc.GetSerializer(r.Header.Get("content-type"))
	render.JSON(w, r, h.fc.BuildCollection(deployments, serializer, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams))
}

// Add subdeployment to a deployment
func (h *DeploymentHandler) AddSubdeployment(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")

	subdeployment, err := domains.Deployment{}.BuildFromRequest(r, w)
	if err != nil {
		return // Error already handled in buildDeployment
	}

	subdeployment.ParentDeploymentID = &parentID

	if err := h.repo.Create(&subdeployment); err != nil {
		h.logger.Error("Failed to create subdeployment", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create subdeployment"})
		return
	}

	subdeploymentGeoJSON, err := h.sc.Serialize(r.Header.Get("content-type"), &subdeployment)

	if err != nil {
		h.logger.Error("Failed to serialize subdeployment", zap.String("id", subdeployment.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize subdeployment"})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, subdeploymentGeoJSON)
}
