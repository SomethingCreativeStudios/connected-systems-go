package api

import (
	"fmt"
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

// SamplingFeatureHandler handles SamplingFeature resource requests
type SamplingFeatureHandler struct {
	cfg        *config.Config
	logger     *zap.Logger
	repo       *repository.SamplingFeatureRepository
	systemRepo *repository.SystemRepository
	fc         *formaters.MultiFormatFormatterCollection[*domains.SamplingFeature]
}

// NewSamplingFeatureHandler creates a new SamplingFeatureHandler
func NewSamplingFeatureHandler(cfg *config.Config, logger *zap.Logger, repo *repository.SamplingFeatureRepository, systemRepo *repository.SystemRepository, fc *formaters.MultiFormatFormatterCollection[*domains.SamplingFeature]) *SamplingFeatureHandler {
	return &SamplingFeatureHandler{cfg: cfg, logger: logger, repo: repo, systemRepo: systemRepo, fc: fc}
}

// ListSamplingFeatures returns a paginated list of sampling features
//
// @Summary     List sampling features
// @Description Returns a paginated collection of sampling feature resources
// @Tags        Sampling Features
// @Produce     json
// @Param       limit               query  integer  false  "Maximum number of results"
// @Param       cursor              query  string   false  "Opaque pagination cursor"
// @Param       id                  query  string   false  "Comma-separated resource IDs"
// @Param       q                   query  string   false  "Comma-separated keywords for full-text search"
// @Param       bbox                query  string   false  "Bounding box filter: minx,miny[,minz],maxx,maxy[,maxz]"
// @Param       dateTime            query  string   false  "Date-time or interval (RFC 3339), e.g. 2023-01-01T00:00:00Z/2023-12-31T23:59:59Z"
// @Param       geom                query  string   false  "WKT geometry for spatial intersection"
// @Param       foi                 query  string   false  "Comma-separated feature of interest IDs"
// @Param       observedProperty    query  string   false  "Comma-separated observed property IDs"
// @Param       controlledProperty  query  string   false  "Comma-separated controlled property IDs"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /samplingFeatures [get]
func (h *SamplingFeatureHandler) ListSamplingFeatures(w http.ResponseWriter, r *http.Request) {
	params, err := queryparams.SamplingFeatureQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)

	if err != nil {
		h.logger.Error("Failed to parse query parameters", zap.Error(err))
		writeError(w, r, http.StatusBadRequest, "Invalid query parameters")
		return
	}

	sampledFeatures, total, err := h.repo.List(r.Context(), params)
	if err != nil {
		h.logger.Error("Failed to list sampling features", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeCollection(w, r, h.logger, h.fc, h.cfg.API.BaseURL, sampledFeatures, total, params.QueryParams)
}

// GetSamplingFeature returns a single sampling feature by ID
//
// @Summary     Get sampling feature
// @Description Returns a single sampling feature resource by ID
// @Tags        Sampling Features
// @Produce     json
// @Param       id  path  string  true  "Sampling Feature ID"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /samplingFeatures/{id} [get]
func (h *SamplingFeatureHandler) GetSamplingFeature(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	samplingFeature, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Sampling Feature", "Failed to load sampling Feature")
		return
	}

	writeResource(w, r, h.logger, h.fc, samplingFeature)
}

func validateSamplingFeature(sf *domains.SamplingFeature) error {
	if sf.Name == "" {
		return fmt.Errorf("name is required")
	}
	if sf.FeatureType == "" {
		return fmt.Errorf("featureType is required")
	}
	if sf.SampledFeatureLink == nil || sf.SampledFeatureLink.Href == "" {
		return fmt.Errorf("sampledFeature@link (with href) is required")
	}
	return nil
}

// CreateSamplingFeature creates a new sampling feature
//
// @Summary     Create sampling feature
// @Description Creates a new sampling feature resource (also available under /systems/{id}/samplingFeatures)
// @Tags        Sampling Features
// @Accept      json
// @Param       id               path  string          true   "Parent system ID"
// @Param       samplingFeature  body  map[string]any  true   "Sampling Feature resource"
// @Success     201
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/samplingFeatures [post]
func (h *SamplingFeatureHandler) CreateSamplingFeature(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	sampledFeature, err := h.fc.DeserializeContext(r.Context(), contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize sampling feature", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	if err := validateSamplingFeature(sampledFeature); err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// If this request is scoped under a system (POST /systems/{id}/samplingFeatures)
	// set the ParentSystemID from the URL param so the created sampling feature
	// is associated with the parent system.
	if parentID := chi.URLParam(r, "id"); parentID != "" {
		// Validate the parent system exists before associating
		if _, err := h.systemRepo.GetByID(r.Context(), parentID); err != nil {
			writeRepositoryError(w, r, h.logger, err, "System", "Failed to load system")
			return
		}
		sampledFeature.ParentSystemID = &parentID
	}

	if err := h.repo.Create(r.Context(), sampledFeature); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Sampling feature", "Failed to create sampling feature")
		return
	}

	// Per spec: return 201 Created with Location header and no response body
	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/samplingFeatures/" + sampledFeature.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

// UpdateSamplingFeature replaces a sampling feature by ID
//
// @Summary     Update sampling feature
// @Description Replaces a sampling feature resource by ID
// @Tags        Sampling Features
// @Accept      json
// @Param       id               path  string          true  "Sampling Feature ID"
// @Param       samplingFeature  body  map[string]any  true  "Sampling Feature resource"
// @Success     204
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     500  {object}  map[string]string
// @Router      /samplingFeatures/{id} [put]
func (h *SamplingFeatureHandler) UpdateSamplingFeature(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	sampledFeature, err := h.fc.DeserializeContext(r.Context(), contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize sampling feature", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	sampledFeature.ID = id
	if err := validateSamplingFeature(sampledFeature); err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.repo.Update(r.Context(), sampledFeature); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Sampling feature", "Failed to update sampling feature")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteSamplingFeature deletes a sampling feature by ID
//
// @Summary     Delete sampling feature
// @Description Deletes a sampling feature resource by ID
// @Tags        Sampling Features
// @Param       id  path  string  true  "Sampling Feature ID"
// @Success     204
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /samplingFeatures/{id} [delete]
func (h *SamplingFeatureHandler) DeleteSamplingFeature(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Sampling feature", "Failed to delete sampling feature")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSystemSamplingFeatures returns sampling features belonging to a system
//
// @Summary     List system sampling features
// @Description Returns sampling features associated with a given system
// @Tags        Sampling Features
// @Produce     json
// @Param       id                  path   string   true   "System ID"
// @Param       limit               query  integer  false  "Maximum number of results"
// @Param       cursor              query  string   false  "Opaque pagination cursor"
// @Param       q                   query  string   false  "Comma-separated keywords for full-text search"
// @Param       bbox                query  string   false  "Bounding box filter: minx,miny[,minz],maxx,maxy[,maxz]"
// @Param       dateTime            query  string   false  "Date-time or interval (RFC 3339)"
// @Param       geom                query  string   false  "WKT geometry for spatial intersection"
// @Param       foi                 query  string   false  "Comma-separated feature of interest IDs"
// @Param       observedProperty    query  string   false  "Comma-separated observed property IDs"
// @Param       controlledProperty  query  string   false  "Comma-separated controlled property IDs"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/samplingFeatures [get]
func (h *SamplingFeatureHandler) GetSystemSamplingFeatures(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "id")

	params, err := queryparams.SamplingFeatureQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		h.logger.Error("Failed to parse query parameters", zap.Error(err))
		writeError(w, r, http.StatusBadRequest, "Invalid query parameters")
		return
	}

	sampledFeatures, total, err := h.repo.ListSystem(r.Context(), params, &systemID)
	if err != nil {
		h.logger.Error("Failed to list sampling features", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeCollection(w, r, h.logger, h.fc, h.cfg.API.BaseURL, sampledFeatures, total, params.QueryParams)

}
