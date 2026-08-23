package api

import (
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

// DeploymentHandler handles Deployment resource requests
type DeploymentHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.DeploymentRepository
	fc     *formaters.MultiFormatFormatterCollection[*domains.Deployment]
}

// NewDeploymentHandler creates a new DeploymentHandler
func NewDeploymentHandler(cfg *config.Config, logger *zap.Logger, repo *repository.DeploymentRepository, fc *formaters.MultiFormatFormatterCollection[*domains.Deployment]) *DeploymentHandler {
	return &DeploymentHandler{cfg: cfg, logger: logger, repo: repo, fc: fc}
}

// ListDeployments returns a paginated list of deployments
//
// @Summary     List deployments
// @Description Returns a paginated collection of deployment resources
// @Tags        Deployments
// @Produce     json
// @Param       limit               query  integer  false  "Maximum number of results"
// @Param       cursor              query  string   false  "Opaque pagination cursor"
// @Param       id                  query  string   false  "Comma-separated resource IDs"
// @Param       q                   query  string   false  "Comma-separated keywords for full-text search"
// @Param       bbox                query  string   false  "Bounding box filter: minx,miny[,minz],maxx,maxy[,maxz]"
// @Param       dateTime            query  string   false  "Date-time or interval (RFC 3339), e.g. 2023-01-01T00:00:00Z/2023-12-31T23:59:59Z"
// @Param       geom                query  string   false  "WKT geometry for spatial intersection"
// @Param       parent              query  string   false  "Comma-separated parent deployment IDs"
// @Param       system              query  string   false  "Comma-separated system IDs"
// @Param       foi                 query  string   false  "Comma-separated feature of interest IDs"
// @Param       observedProperty    query  string   false  "Comma-separated observed property IDs"
// @Param       controlledProperty  query  string   false  "Comma-separated controlled property IDs"
// @Param       recursive           query  boolean  false  "Include sub-deployments recursively"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /deployments [get]
func (h *DeploymentHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	params, err := queryparams.DeploymentsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	deployments, total, err := h.repo.List(params, nil)
	if err != nil {
		h.logger.Error("Failed to list deployments", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, deployments, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	writeNegotiated(w, collection)
}

// GetDeployment returns a single deployment by ID
//
// @Summary     Get deployment
// @Description Returns a single deployment resource by ID
// @Tags        Deployments
// @Produce     json
// @Param       id  path  string  true  "Deployment ID"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /deployments/{id} [get]
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
	writeNegotiated(w, serialized)
}

// CreateDeployment creates a new deployment
//
// @Summary     Create deployment
// @Description Creates a new deployment resource
// @Tags        Deployments
// @Accept      json
// @Param       deployment  body  map[string]any  true  "Deployment resource"
// @Success     201
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /deployments [post]
func (h *DeploymentHandler) CreateDeployment(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	deployment, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize deployment", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	if err := h.repo.Create(deployment); err != nil {
		h.logger.Error("Failed to create deployment", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create deployment"})
		return
	}

	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/deployments/" + deployment.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

// UpdateDeployment replaces a deployment by ID
//
// @Summary     Update deployment
// @Description Replaces a deployment resource by ID
// @Tags        Deployments
// @Accept      json
// @Param       id          path  string          true  "Deployment ID"
// @Param       deployment  body  map[string]any  true  "Deployment resource"
// @Success     204
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /deployments/{id} [put]
func (h *DeploymentHandler) UpdateDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	deployment, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize deployment", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	deployment.ID = id
	if err := h.repo.Update(deployment); err != nil {
		h.logger.Error("Failed to update deployment", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update deployment"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteDeployment deletes a deployment by ID
//
// @Summary     Delete deployment
// @Description Deletes a deployment resource by ID
// @Tags        Deployments
// @Param       id       path   string  true   "Deployment ID"
// @Param       cascade  query  bool    false  "Cascade delete to dependent resources"
// @Success     204
// @Failure     404  {object}  map[string]string
// @Failure     409  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /deployments/{id} [delete]
func (h *DeploymentHandler) DeleteDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cascade := r.URL.Query().Get("cascade") == "true"

	if err := h.repo.Delete(id, cascade); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "Deployment not found"})
		case errors.Is(err, repository.ErrHasChildren):
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, map[string]string{"error": "Deployment has dependent records; use ?cascade=true to delete"})
		default:
			h.logger.Error("Failed to delete deployment", zap.String("id", id), zap.Error(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to delete deployment"})
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListSubdeployments lists subdeployments of a deployment
//
// @Summary     List subdeployments
// @Description Returns child deployments of the given deployment
// @Tags        Deployments
// @Produce     json
// @Param       id                  path   string   true   "Parent deployment ID"
// @Param       limit               query  integer  false  "Maximum number of results"
// @Param       cursor              query  string   false  "Opaque pagination cursor"
// @Param       q                   query  string   false  "Comma-separated keywords for full-text search"
// @Param       bbox                query  string   false  "Bounding box filter: minx,miny[,minz],maxx,maxy[,maxz]"
// @Param       dateTime            query  string   false  "Date-time or interval (RFC 3339)"
// @Param       geom                query  string   false  "WKT geometry for spatial intersection"
// @Param       system              query  string   false  "Comma-separated system IDs"
// @Param       foi                 query  string   false  "Comma-separated feature of interest IDs"
// @Param       observedProperty    query  string   false  "Comma-separated observed property IDs"
// @Param       controlledProperty  query  string   false  "Comma-separated controlled property IDs"
// @Param       recursive           query  boolean  false  "Include nested sub-deployments recursively"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /deployments/{id}/subdeployments [get]
func (h *DeploymentHandler) ListSubdeployments(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")
	params, err := queryparams.DeploymentsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	deployments, total, err := h.repo.List(params, &parentID)
	if err != nil {
		h.logger.Error("Failed to list subdeployments", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, deployments, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	writeNegotiated(w, collection)
}

// AddSubdeployment adds a subdeployment to a deployment
//
// @Summary     Add subdeployment
// @Description Creates a new deployment as a child of the given deployment
// @Tags        Deployments
// @Accept      json
// @Param       id          path  string          true  "Parent deployment ID"
// @Param       deployment  body  map[string]any  true  "Deployment resource"
// @Success     201
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /deployments/{id}/subdeployments [post]
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

	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/deployments/" + subdeployment.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}
