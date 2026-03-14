package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// SystemEventCollectionResponse follows the collection shape used by dynamic-data resources.
type SystemEventCollectionResponse struct {
	Items []any               `json:"items"`
	Links common_shared.Links `json:"links,omitempty"`
}

// SystemEventHandler handles /systemEvents and /systems/{id}/events resources.
type SystemEventHandler struct {
	cfg        *config.Config
	logger     *zap.Logger
	repo       *repository.SystemEventRepository
	systemRepo *repository.SystemRepository
}

func NewSystemEventHandler(cfg *config.Config, logger *zap.Logger, repo *repository.SystemEventRepository, systemRepo *repository.SystemRepository) *SystemEventHandler {
	return &SystemEventHandler{cfg: cfg, logger: logger, repo: repo, systemRepo: systemRepo}
}

func (h *SystemEventHandler) ListSystemEvents(w http.ResponseWriter, r *http.Request) {
	params := queryparams.SystemEventsQueryParams{}.BuildFromRequest(r)

	events, total, err := h.repo.List(params, nil)
	if err != nil {
		h.logger.Error("Failed to list system events", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	items := make([]any, 0, len(events))
	for _, event := range events {
		items = append(items, event)
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(events))

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, SystemEventCollectionResponse{Items: items, Links: links})
}

func (h *SystemEventHandler) ListEventsBySystem(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemId")
	if systemID == "" {
		systemID = chi.URLParam(r, "id")
	}

	if _, err := h.systemRepo.GetByID(systemID); err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "System not found"})
		return
	}

	params := queryparams.SystemEventsQueryParams{}.BuildFromRequest(r)
	events, total, err := h.repo.List(params, &systemID)
	if err != nil {
		h.logger.Error("Failed to list system events", zap.String("systemId", systemID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	items := make([]any, 0, len(events))
	for _, event := range events {
		items = append(items, event)
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(events))

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, SystemEventCollectionResponse{Items: items, Links: links})
}

func (h *SystemEventHandler) CreateEventBySystem(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemId")
	if systemID == "" {
		systemID = chi.URLParam(r, "id")
	}

	if _, err := h.systemRepo.GetByID(systemID); err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "System not found"})
		return
	}

	// OpenAPI allows either a single event or an array.
	var raw any
	if err := render.DecodeJSON(r.Body, &raw); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	createdIDs := make([]string, 0, 1)
	createOne := func(e *domains.SystemEvent) error {
		e.SystemID = systemID
		if e.Label == "" {
			e.Label = "System Event"
		}
		if err := h.repo.Create(e); err != nil {
			return err
		}
		createdIDs = append(createdIDs, e.ID)
		return nil
	}

	switch v := raw.(type) {
	case map[string]any:
		var evt domains.SystemEvent
		bytes, _ := json.Marshal(v)
		if err := json.Unmarshal(bytes, &evt); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": "Invalid system event payload"})
			return
		}
		if err := createOne(&evt); err != nil {
			h.logger.Error("Failed to create system event", zap.String("systemId", systemID), zap.Error(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to create system event"})
			return
		}
	case []any:
		if len(v) == 0 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": "At least one event is required"})
			return
		}
		for _, item := range v {
			itemObj, ok := item.(map[string]any)
			if !ok {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, map[string]string{"error": "Invalid system event payload"})
				return
			}
			var evt domains.SystemEvent
			bytes, _ := json.Marshal(itemObj)
			if err := json.Unmarshal(bytes, &evt); err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, map[string]string{"error": "Invalid system event payload"})
				return
			}
			if err := createOne(&evt); err != nil {
				h.logger.Error("Failed to create system event", zap.String("systemId", systemID), zap.Error(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, map[string]string{"error": "Failed to create system event"})
				return
			}
		}
	default:
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid system event payload"})
		return
	}

	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/systems/" + systemID + "/events/" + createdIDs[0]
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

func (h *SystemEventHandler) GetEventByID(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemId")
	if systemID == "" {
		systemID = chi.URLParam(r, "id")
	}
	eventID := chi.URLParam(r, "eventId")

	event, err := h.repo.GetByID(systemID, eventID)
	if err != nil {
		h.logger.Error("Failed to get system event", zap.String("systemId", systemID), zap.String("eventId", eventID), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "System event not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, event)
}

func (h *SystemEventHandler) UpdateEventByID(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemId")
	if systemID == "" {
		systemID = chi.URLParam(r, "id")
	}
	eventID := chi.URLParam(r, "eventId")

	existing, err := h.repo.GetByID(systemID, eventID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "System event not found"})
		return
	}

	var event domains.SystemEvent
	if err := render.DecodeJSON(r.Body, &event); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "Invalid request body"})
		return
	}

	event.ID = eventID
	event.SystemID = existing.SystemID
	if err := h.repo.Update(&event); err != nil {
		h.logger.Error("Failed to update system event", zap.String("eventId", eventID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update system event"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SystemEventHandler) DeleteEventByID(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemId")
	if systemID == "" {
		systemID = chi.URLParam(r, "id")
	}
	eventID := chi.URLParam(r, "eventId")

	if _, err := h.repo.GetByID(systemID, eventID); err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "System event not found"})
		return
	}

	if err := h.repo.Delete(systemID, eventID); err != nil {
		h.logger.Error("Failed to delete system event", zap.String("eventId", eventID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete system event"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
