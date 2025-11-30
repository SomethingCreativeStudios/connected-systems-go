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

// SamplingFeatureHandler handles SamplingFeature resource requests
type SamplingFeatureHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	repo   *repository.SamplingFeatureRepository
	fc     *formaters.MultiFormatFormatterCollection[*domains.SamplingFeature]
}

// NewSamplingFeatureHandler creates a new SamplingFeatureHandler
func NewSamplingFeatureHandler(cfg *config.Config, logger *zap.Logger, repo *repository.SamplingFeatureRepository, fc *formaters.MultiFormatFormatterCollection[*domains.SamplingFeature]) *SamplingFeatureHandler {
	return &SamplingFeatureHandler{cfg: cfg, logger: logger, repo: repo, fc: fc}
}

func (h *SamplingFeatureHandler) ListSamplingFeatures(w http.ResponseWriter, r *http.Request) {
	params, err := queryparams.SamplingFeatureQueryParams{}.BuildFromRequest(r)

	if err != nil {
		h.logger.Error("Failed to parse query parameters", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid query parameters"})
		return
	}

	sampledFeatures, total, err := h.repo.List(params)
	if err != nil {
		h.logger.Error("Failed to list sampling features", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, sampledFeatures, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, collection)
}

func (h *SamplingFeatureHandler) GetSamplingFeature(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	samplingFeature, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get sampling feature", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Sampling Feature not found"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, samplingFeature)
	if err != nil {
		h.logger.Error("Failed to serialize sampling feature", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize sampling feature"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, serialized)
}

func (h *SamplingFeatureHandler) CreateSamplingFeature(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	sampledFeature, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize sampling feature", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	// If this request is scoped under a system (POST /systems/{id}/samplingFeatures)
	// set the ParentSystemID from the URL param so the created sampling feature
	// is associated with the parent system.
	if parentID := chi.URLParam(r, "id"); parentID != "" {
		sampledFeature.ParentSystemID = &parentID

		// add a parentSystem link so the serialized response reflects the relationship
		parentLink := common_shared.Link{
			Rel:  "parentSystem",
			Href: "systems/" + parentID,
			Type: "application/geo+json",
		}
		sampledFeature.Links = append(sampledFeature.Links, parentLink)
	}

	if err := h.repo.Create(sampledFeature); err != nil {
		h.logger.Error("Failed to create sampling feature", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create sampling feature"})
		return
	}

	// Per spec: return 201 Created with Location header and no response body
	w.Header().Set("Location", h.cfg.API.BaseURL+"/samplingFeatures/"+sampledFeature.ID)
	w.WriteHeader(http.StatusCreated)
}

func (h *SamplingFeatureHandler) UpdateSamplingFeature(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	contentType := r.Header.Get("Content-Type")
	sampledFeature, err := h.fc.Deserialize(contentType, r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize sampling feature", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	sampledFeature.ID = id
	if err := h.repo.Update(sampledFeature); err != nil {
		h.logger.Error("Failed to update sampling feature", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update sampling feature"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, sampledFeature)
	if err != nil {
		h.logger.Error("Failed to serialize sampling feature", zap.String("id", sampledFeature.ID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize sampling feature"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.Status(r, http.StatusOK)
	render.JSON(w, r, serialized)
}

func (h *SamplingFeatureHandler) DeleteSamplingFeature(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.repo.Delete(id); err != nil {
		h.logger.Error("Failed to delete sampling feature", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete sampling feature"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SamplingFeatureHandler) GetSystemSamplingFeatures(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "id")

	params, err := queryparams.SamplingFeatureQueryParams{}.BuildFromRequest(r)
	if err != nil {
		h.logger.Error("Failed to parse query parameters", zap.Error(err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid query parameters"})
		return
	}

	sampledFeatures, total, err := h.repo.ListSystem(params, &systemID)
	if err != nil {
		h.logger.Error("Failed to list sampling features", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	collection := h.fc.BuildCollection(acceptHeader, sampledFeatures, h.cfg.API.BaseURL+r.URL.Path, int(total), r.URL.Query(), params.QueryParams)

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, collection)

}
