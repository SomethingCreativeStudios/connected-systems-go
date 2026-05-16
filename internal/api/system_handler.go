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

// SystemHandler handles System resource requests
type SystemHandler struct {
	cfg         *config.Config
	logger      *zap.Logger
	repo        *repository.SystemRepository
	historyRepo *repository.SystemHistoryRepository
	fc          *formaters.MultiFormatFormatterCollection[*domains.System]
	// deployment dependencies for server-side reuse
	deploymentRepo *repository.DeploymentRepository
	deploymentFC   *formaters.MultiFormatFormatterCollection[*domains.Deployment]
	// procedure dependencies for server-side association endpoint
	procedureRepo *repository.ProcedureRepository
	procedureFC   *formaters.MultiFormatFormatterCollection[*domains.Procedure]
}

// NewSystemHandler creates a new SystemHandler
func NewSystemHandler(cfg *config.Config, logger *zap.Logger, repo *repository.SystemRepository, historyRepo *repository.SystemHistoryRepository, fc *formaters.MultiFormatFormatterCollection[*domains.System], deploymentRepo *repository.DeploymentRepository, deploymentFC *formaters.MultiFormatFormatterCollection[*domains.Deployment], procedureRepo *repository.ProcedureRepository, procedureFC *formaters.MultiFormatFormatterCollection[*domains.Procedure]) *SystemHandler {
	return &SystemHandler{
		cfg:            cfg,
		logger:         logger,
		repo:           repo,
		historyRepo:    historyRepo,
		fc:             fc,
		deploymentRepo: deploymentRepo,
		deploymentFC:   deploymentFC,
		procedureRepo:  procedureRepo,
		procedureFC:    procedureFC,
	}
}

// ListSystems retrieves a list of systems
//
// @Summary     List systems
// @Description Returns a paginated collection of system resources
// @Tags        Systems
// @Produce     json
// @Param       limit   query  int     false  "Maximum number of results"
// @Param       offset  query  int     false  "Result offset"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems [get]
func (h *SystemHandler) ListSystems(w http.ResponseWriter, r *http.Request) {
	params, err := queryparams.SystemQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	systems, total, err := h.repo.List(params)
	if err != nil {
		h.logger.Error("Failed to list systems", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	h.populateSystemAssociationLinks(systems)

	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, systems, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, collection)
}

// GetSystem retrieves a single system by ID
//
// @Summary     Get system
// @Description Returns a single system resource by ID
// @Tags        Systems
// @Produce     json
// @Param       id   path  string  true  "System ID"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id} [get]
func (h *SystemHandler) GetSystem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	system, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get system", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "System not found"})
		return
	}

	system.Links = append(system.Links, h.repo.BuildSystemAssociations(id)...)

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
//
// @Summary     Create system
// @Description Creates a new system resource
// @Tags        Systems
// @Accept      json
// @Param       system  body  map[string]any  true  "System resource (GeoJSON or SensorML+JSON)"
// @Success     201
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems [post]
func (h *SystemHandler) CreateSystem(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	system, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize system", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	if err := h.repo.Create(system); err != nil {
		h.logger.Error("Failed to create system", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create system"})
		return
	}

	if _, err := h.historyRepo.CreateFromSystem(system); err != nil {
		h.logger.Warn("Failed to create initial system history snapshot", zap.String("systemId", system.ID), zap.Error(err))
	}

	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/systems/" + system.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

// UpdateSystem updates a system (PUT)
//
// @Summary     Update system
// @Description Replaces a system resource by ID
// @Tags        Systems
// @Accept      json
// @Param       id      path  string          true  "System ID"
// @Param       system  body  map[string]any  true  "System resource"
// @Success     204
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id} [put]
func (h *SystemHandler) UpdateSystem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	system, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize system", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	system.ID = id
	if err := h.repo.Update(system.ID, system); err != nil {
		h.logger.Error("Failed to update system", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update system"})
		return
	}

	if _, err := h.historyRepo.CreateFromSystem(system); err != nil {
		h.logger.Warn("Failed to create system history snapshot after update", zap.String("systemId", system.ID), zap.Error(err))
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteSystem deletes a system
//
// @Summary     Delete system
// @Description Deletes a system resource by ID
// @Tags        Systems
// @Param       id       path   string  true   "System ID"
// @Param       cascade  query  bool    false  "Cascade delete to dependent resources"
// @Success     204
// @Failure     404  {object}  map[string]string
// @Failure     409  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id} [delete]
func (h *SystemHandler) DeleteSystem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cascade := r.URL.Query().Get("cascade") == "true"

	if err := h.repo.Delete(id, cascade); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "System not found"})
		case errors.Is(err, repository.ErrHasChildren):
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, map[string]string{"error": "System has dependent records; use ?cascade=true to delete"})
		default:
			h.logger.Error("Failed to delete system", zap.String("id", id), zap.Error(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to delete system"})
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSubsystems retrieves subsystems of a system
//
// @Summary     List subsystems
// @Description Returns subsystems of a given system
// @Tags        Systems
// @Produce     json
// @Param       id         path   string  true   "System ID"
// @Param       recursive  query  bool    false  "Include nested subsystems recursively"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/subsystems [get]
func (h *SystemHandler) GetSubsystems(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")
	recursive := r.URL.Query().Get("recursive") == "true"
	params, err := queryparams.SystemQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	systems, err := h.repo.GetSubsystems(parentID, recursive)
	if err != nil {
		h.logger.Error("Failed to get subsystems", zap.String("parentID", parentID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to get subsystems"})
		return
	}

	h.populateSystemAssociationLinks(systems)

	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, systems, h.cfg.API.BaseURL+r.URL.Path, len(systems), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, collection)
}

func (h *SystemHandler) populateSystemAssociationLinks(systems []*domains.System) {
	for _, system := range systems {
		if system == nil || strings.TrimSpace(system.ID) == "" {
			continue
		}
		system.Links = append(system.Links, h.repo.BuildSystemAssociations(system.ID)...)
	}
}

// GetDeployments retrieves deployments associated with a system
//
// @Summary     List system deployments
// @Description Returns deployments associated with a given system
// @Tags        Systems
// @Produce     json
// @Param       id  path  string  true  "System ID"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/deployments [get]
func (h *SystemHandler) GetDeployments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Build optional pagination params from request
	params, err := queryparams.DeploymentsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}
	params.System = append(params.System, id)

	// Use deployment repository helper to find deployments associated with this system
	deployments, total, err := h.deploymentRepo.List(params, nil)
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

// GetProcedures retrieves procedures associated with a system.
//
// @Summary     List system procedures
// @Description Returns procedures associated with a given system
// @Tags        Systems
// @Produce     json
// @Param       id  path  string  true  "System ID"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/procedures [get]
func (h *SystemHandler) GetProcedures(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	params, err := queryparams.ProceduresQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	procedures, total, err := h.procedureRepo.ListBySystem(id, params)
	if err != nil {
		h.logger.Error("Failed to get procedures for system", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to get procedures"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	collection := h.procedureFC.BuildCollection(acceptHeader, procedures, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.procedureFC.GetResponseContentType(acceptHeader))
	render.JSON(w, r, collection)
}

// AddSubsystem adds a subsystem to a system
//
// @Summary     Add subsystem
// @Description Creates a new system as a child of the given system
// @Tags        Systems
// @Accept      json
// @Param       id      path  string          true  "Parent system ID"
// @Param       system  body  map[string]any  true  "System resource"
// @Success     201
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/subsystems [post]
func (h *SystemHandler) AddSubsystem(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	system, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize system", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	system.ParentSystemID = &parentID

	if err := h.repo.Create(system); err != nil {
		h.logger.Error("Failed to create subsystem", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create subsystem"})
		return
	}

	if _, err := h.historyRepo.CreateFromSystem(system); err != nil {
		h.logger.Warn("Failed to create subsystem history snapshot", zap.String("systemId", system.ID), zap.Error(err))
	}

	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/systems/" + system.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}
