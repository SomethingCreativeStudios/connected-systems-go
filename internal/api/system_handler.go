package api

import (
	"net/http"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/go-chi/chi/v5"
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
// @Param       limit               query  integer  false  "Maximum number of results"
// @Param       cursor              query  string   false  "Opaque pagination cursor"
// @Param       id                  query  string   false  "Comma-separated resource IDs"
// @Param       q                   query  string   false  "Comma-separated keywords for full-text search"
// @Param       bbox                query  string   false  "Bounding box filter: minx,miny[,minz],maxx,maxy[,maxz]"
// @Param       datetime            query  string   false  "Date-time or interval (RFC 3339)"
// @Param       dateTime            query  string   false  "Compatibility alias for datetime"
// @Param       filter              query  string   false  "CQL2 Text filter (default) or URL-encoded CQL2 JSON"
// @Param       filter-lang         query  string   false  "CQL2 language: cql2-text or cql2-json"
// @Param       geom                query  string   false  "WKT geometry for spatial intersection"
// @Param       parent              query  string   false  "Comma-separated parent system IDs"
// @Param       procedure           query  string   false  "Comma-separated procedure IDs"
// @Param       foi                 query  string   false  "Comma-separated feature of interest IDs"
// @Param       observedProperty    query  string   false  "Comma-separated observed property IDs"
// @Param       controlledProperty  query  string   false  "Comma-separated controlled property IDs"
// @Param       recursive           query  boolean  false  "Include sub-resources recursively"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems [get]
func (h *SystemHandler) ListSystems(w http.ResponseWriter, r *http.Request) {
	params, err := queryparams.SystemQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	systems, total, err := h.repo.List(r.Context(), params)
	if err != nil {
		h.logger.Error("Failed to list systems", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeCollection(w, r, h.logger, h.fc, h.cfg.API.BaseURL, systems, total, params.QueryParams)
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

	system, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "System", "Failed to load system")
		return
	}

	writeResource(w, r, h.logger, h.fc, system)
}

// CreateSystem creates a new system
//
// @Summary     Create system
// @Description Creates a new system resource
// @Tags        Systems
// @Accept      json
// @Param       system  body  map[string]any  true  "System resource (GeoJSON or SensorML+JSON)"
// @Success     201
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     500  {object}  map[string]string
// @Router      /systems [post]
func (h *SystemHandler) CreateSystem(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	system, err := h.fc.DeserializeContext(r.Context(), contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize system", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	if err := h.repo.Create(r.Context(), system); err != nil {
		writeRepositoryError(w, r, h.logger, err, "System", "Failed to create system")
		return
	}

	if _, err := h.historyRepo.CreateFromSystem(r.Context(), system); err != nil {
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
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id} [put]
func (h *SystemHandler) UpdateSystem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	system, err := h.fc.DeserializeContext(r.Context(), contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize system", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	system.ID = id
	if err := h.repo.Update(r.Context(), system.ID, system); err != nil {
		writeRepositoryError(w, r, h.logger, err, "System", "Failed to update system")
		return
	}

	if _, err := h.historyRepo.CreateFromSystem(r.Context(), system); err != nil {
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

	if err := h.repo.Delete(r.Context(), id, cascade); err != nil {
		writeRepositoryError(w, r, h.logger, err, "System", "Failed to delete system")
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
// @Param       id                  path   string   true   "System ID"
// @Param       limit               query  integer  false  "Maximum number of results"
// @Param       cursor              query  string   false  "Opaque pagination cursor"
// @Param       id                  query  string   false  "Comma-separated resource IDs"
// @Param       q                   query  string   false  "Comma-separated keywords for full-text search"
// @Param       bbox                query  string   false  "Bounding box filter: minx,miny[,minz],maxx,maxy[,maxz]"
// @Param       dateTime            query  string   false  "Date-time or interval (RFC 3339)"
// @Param       geom                query  string   false  "WKT geometry for spatial intersection"
// @Param       procedure           query  string   false  "Comma-separated procedure IDs"
// @Param       foi                 query  string   false  "Comma-separated feature of interest IDs"
// @Param       observedProperty    query  string   false  "Comma-separated observed property IDs"
// @Param       controlledProperty  query  string   false  "Comma-separated controlled property IDs"
// @Param       recursive           query  boolean  false  "Include nested subsystems recursively"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/subsystems [get]
func (h *SystemHandler) GetSubsystems(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")
	params, err := queryparams.SystemQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	systems, total, err := h.repo.ListSubsystems(r.Context(), parentID, params)
	if err != nil {
		h.logger.Error("Failed to get subsystems", zap.String("parentID", parentID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Failed to get subsystems")
		return
	}

	writeCollection(w, r, h.logger, h.fc, h.cfg.API.BaseURL, systems, total, params.QueryParams)
}

// GetDeployments retrieves deployments associated with a system
//
// @Summary     List system deployments
// @Description Returns deployments associated with a given system
// @Tags        Systems
// @Produce     json
// @Param       id                  path   string   true   "System ID"
// @Param       limit               query  integer  false  "Maximum number of results"
// @Param       cursor              query  string   false  "Opaque pagination cursor"
// @Param       q                   query  string   false  "Comma-separated keywords for full-text search"
// @Param       bbox                query  string   false  "Bounding box filter: minx,miny[,minz],maxx,maxy[,maxz]"
// @Param       dateTime            query  string   false  "Date-time or interval (RFC 3339)"
// @Param       geom                query  string   false  "WKT geometry for spatial intersection"
// @Param       parent              query  string   false  "Comma-separated parent deployment IDs"
// @Param       foi                 query  string   false  "Comma-separated feature of interest IDs"
// @Param       observedProperty    query  string   false  "Comma-separated observed property IDs"
// @Param       controlledProperty  query  string   false  "Comma-separated controlled property IDs"
// @Param       recursive           query  boolean  false  "Include sub-deployments recursively"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/deployments [get]
func (h *SystemHandler) GetDeployments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Build optional pagination params from request
	params, err := queryparams.DeploymentsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	params.System = append(params.System, id)

	// Use deployment repository helper to find deployments associated with this system
	deployments, total, err := h.deploymentRepo.List(r.Context(), params, nil)
	if err != nil {
		h.logger.Error("Failed to get deployments for system", zap.String("id", id), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Failed to get deployments")
		return
	}

	writeCollection(w, r, h.logger, h.deploymentFC, h.cfg.API.BaseURL, deployments, total, params.QueryParams)
}

// GetProcedures retrieves procedures associated with a system.
//
// @Summary     List system procedures
// @Description Returns procedures associated with a given system
// @Tags        Systems
// @Produce     json
// @Param       id                  path   string   true   "System ID"
// @Param       limit               query  integer  false  "Maximum number of results"
// @Param       cursor              query  string   false  "Opaque pagination cursor"
// @Param       q                   query  string   false  "Comma-separated keywords for full-text search"
// @Param       dateTime            query  string   false  "Date-time or interval (RFC 3339)"
// @Param       observedProperty    query  string   false  "Comma-separated observed property IDs"
// @Param       controlledProperty  query  string   false  "Comma-separated controlled property IDs"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/procedures [get]
func (h *SystemHandler) GetProcedures(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	params, err := queryparams.ProceduresQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	procedures, total, err := h.procedureRepo.ListBySystem(r.Context(), id, params)
	if err != nil {
		h.logger.Error("Failed to get procedures for system", zap.String("id", id), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Failed to get procedures")
		return
	}

	writeCollection(w, r, h.logger, h.procedureFC, h.cfg.API.BaseURL, procedures, total, params.QueryParams)
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
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/subsystems [post]
func (h *SystemHandler) AddSubsystem(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	system, err := h.fc.DeserializeContext(r.Context(), contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize system", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	system.ParentSystemID = &parentID

	if err := h.repo.Create(r.Context(), system); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Subsystem", "Failed to create subsystem")
		return
	}

	if _, err := h.historyRepo.CreateFromSystem(r.Context(), system); err != nil {
		h.logger.Warn("Failed to create subsystem history snapshot", zap.String("systemId", system.ID), zap.Error(err))
	}

	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/systems/" + system.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}
