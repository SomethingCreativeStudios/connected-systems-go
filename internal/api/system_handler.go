package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model"
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
	params := h.parseQueryParams(r)

	systems, total, err := h.repo.List(params)
	if err != nil {
		h.logger.Error("Failed to list systems", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	// Convert to GeoJSON FeatureCollection
	features := make([]model.SystemGeoJSONFeature, len(systems))
	for i, sys := range systems {

		// Add associations links
		sys.Links = append(sys.Links, h.repo.BuildSystemAssociations(sys.ID)...)

		features[i] = sys.ToGeoJSON()
	}

	totalInt := int(total)
	response := map[string]interface{}{
		"type":           "FeatureCollection",
		"numberMatched":  &totalInt,
		"numberReturned": len(features),
		"features":       features,
		"links":          h.buildCollectionLinks(r, &totalInt, len(features), params.Limit),
	}

	render.JSON(w, r, response)
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
	system, err := h.buildSystem(r, w)

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

	var system model.System
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

	systems, err := h.repo.GetSubsystems(parentID, recursive)
	if err != nil {
		h.logger.Error("Failed to get subsystems", zap.String("parentID", parentID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to get subsystems"})
		return
	}

	features := make([]model.SystemGeoJSONFeature, len(systems))
	for i, sys := range systems {
		features[i] = sys.ToGeoJSON()
	}

	response := map[string]interface{}{
		"type":           "FeatureCollection",
		"numberReturned": len(features),
		"features":       features,
	}

	render.JSON(w, r, response)
}

// Add subsystem to a system
func (h *SystemHandler) AddSubsystem(w http.ResponseWriter, r *http.Request) {
	parentID := chi.URLParam(r, "id")

	system, err := h.buildSystem(r, w)

	if err != nil {
		return // Error already handled in buildSystem
	}

	system.ParentSystemID = &parentID
	system.Links = append(system.Links, model.Link{
		Rel:  "ogc-rel:parent",
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

// parseQueryParams parses common query parameters
func (h *SystemHandler) parseQueryParams(r *http.Request) *repository.QueryParams {
	params := &repository.QueryParams{
		Limit:  10,
		Offset: 0,
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			params.Limit = val
		}
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if val, err := strconv.Atoi(offset); err == nil {
			params.Offset = val
		}
	}

	if ids := r.URL.Query().Get("id"); ids != "" {
		params.IDs = strings.Split(ids, ",")
	}

	params.Q = r.URL.Query().Get("q")
	params.Recursive = r.URL.Query().Get("recursive") == "true"

	if parent := r.URL.Query().Get("parent"); parent != "" {
		params.Parent = strings.Split(parent, ",")
	}

	return params
}

// buildCollectionLinks creates pagination links
func (h *SystemHandler) buildCollectionLinks(r *http.Request, total *int, returned, limit int) model.Links {
	baseURL := h.cfg.API.BaseURL + r.URL.Path

	currentOffsetStr := r.URL.Query().Get("offset")
	currentOffset := 0

	if currentOffsetStr != "" {
		if val, err := strconv.Atoi(currentOffsetStr); err == nil {
			currentOffset = val
		}
	}

	links := model.Links{
		{Href: baseURL + "?" + r.URL.RawQuery, Rel: "self"},
	}

	if (currentOffset + returned) < *total {
		nextLink := r.URL.Query()
		nextLink.Set("offset", strconv.Itoa(currentOffset+returned))

		links = append(links, model.Link{
			Rel:  "next",
			Href: baseURL + "?" + nextLink.Encode(),
		})
	}

	if currentOffset > 0 {
		prevLink := r.URL.Query()
		if currentOffset-limit <= 0 {
			prevLink.Del("offset")
		} else {
			prevLink.Set("offset", strconv.Itoa(currentOffset-limit))
		}

		links = append(links, model.Link{
			Rel:  "prev",
			Href: baseURL + "?" + prevLink.Encode(),
		})
	}

	return links
}

func (h *SystemHandler) buildSystem(r *http.Request, w http.ResponseWriter) (model.System, error) {
	// Decode GeoJSON Feature format
	var geoJSON struct {
		Type       string                 `json:"type"`
		ID         string                 `json:"id,omitempty"`
		Properties map[string]interface{} `json:"properties"`
		Geometry   *model.Geometry        `json:"geometry,omitempty"`
		Links      model.Links            `json:"links,omitempty"`
	}

	if err := render.DecodeJSON(r.Body, &geoJSON); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return model.System{}, err
	}

	// Convert GeoJSON properties to System model
	system := model.System{
		Geometry: geoJSON.Geometry,
		Links:    geoJSON.Links,
	}

	// Extract properties from the properties object
	if uid, ok := geoJSON.Properties["uid"].(string); ok {
		system.UniqueIdentifier = model.UniqueID(uid)
	}
	if name, ok := geoJSON.Properties["name"].(string); ok {
		system.Name = name
	}
	if desc, ok := geoJSON.Properties["description"].(string); ok {
		system.Description = desc
	}
	if featureType, ok := geoJSON.Properties["featureType"].(string); ok {
		system.SystemType = featureType
	}
	if assetType, ok := geoJSON.Properties["assetType"].(string); ok {
		system.AssetType = &assetType
	}

	// Handle validTime if present
	if validTimeMap, ok := geoJSON.Properties["validTime"].(map[string]interface{}); ok {
		system.ValidTime = &model.TimeRange{}
		if startStr, ok := validTimeMap["start"].(string); ok && startStr != "" {
			startTime, _ := time.Parse(time.RFC3339, startStr)
			system.ValidTime.Start = &startTime
		}
		if endStr, ok := validTimeMap["end"].(string); ok && endStr != "" {
			endTime, _ := time.Parse(time.RFC3339, endStr)
			system.ValidTime.End = &endTime
		}
	}

	return system, nil
}
