package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/contractvalidation"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
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
	validator  *contractvalidation.Validator
	creates    *CreateOperations
}

func NewSystemEventHandler(cfg *config.Config, logger *zap.Logger, repo *repository.SystemEventRepository, systemRepo *repository.SystemRepository, validator *contractvalidation.Validator, creates *CreateOperations) *SystemEventHandler {
	return &SystemEventHandler{cfg: cfg, logger: logger, repo: repo, systemRepo: systemRepo, validator: validator, creates: creates}
}

// validateSystemEvent enforces the required root fields of the OGC CS Part 2
// systemEvent schema (definition, label, time).
func validateSystemEvent(e *domains.SystemEvent) error {
	if e.Definition == "" {
		return fmt.Errorf("definition is required")
	}
	if e.Label == "" {
		return fmt.Errorf("label is required")
	}
	if e.Time.Instant == nil && e.Time.Range == nil {
		return fmt.Errorf("time is required")
	}
	return nil
}

func decodeSystemEvent(value map[string]any) (*domains.SystemEvent, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	event, err := common_shared.DecodeWithFieldErrors[domains.SystemEvent](body)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// ListSystemEvents returns a paginated list of system events
//
// @Summary     List system events
// @Description Returns a paginated collection of system event resources
// @Tags        System Events
// @Produce     json
// @Param       limit      query  integer  false  "Maximum number of results"
// @Param       cursor     query  string   false  "Opaque pagination cursor"
// @Param       id         query  string   false  "Comma-separated resource IDs"
// @Param       q          query  string   false  "Comma-separated keywords for full-text search"
// @Param       system     query  string   false  "Comma-separated system IDs"
// @Param       eventType  query  string   false  "Comma-separated event type values"
// @Param       keyword    query  string   false  "Comma-separated keywords to match against event fields"
// @Param       datetime   query  string   false  "Date-time or interval (RFC 3339); also accepted as eventTime"
// @Success     200  {object}  SystemEventCollectionResponse
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systemEvents [get]
func (h *SystemEventHandler) ListSystemEvents(w http.ResponseWriter, r *http.Request) {
	params, err := queryparams.SystemEventsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	events, _, err := h.repo.List(r.Context(), params, nil)
	if err != nil {
		h.logger.Error("Failed to list system events", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	items := make([]any, 0, len(events))
	for _, event := range events {
		items = append(items, event)
	}

	links := params.BuildPaginationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query())

	w.Header().Set("Content-Type", mediatype.JSON)
	render.JSON(w, r, SystemEventCollectionResponse{Items: items, Links: links})
}

// ListEventsBySystem returns events for a system
//
// @Summary     List system events by system
// @Description Returns events associated with a given system
// @Tags        System Events
// @Produce     json
// @Param       id         path   string   true   "System ID"
// @Param       limit      query  integer  false  "Maximum number of results"
// @Param       cursor     query  string   false  "Opaque pagination cursor"
// @Param       q          query  string   false  "Comma-separated keywords for full-text search"
// @Param       eventType  query  string   false  "Comma-separated event type values"
// @Param       keyword    query  string   false  "Comma-separated keywords to match against event fields"
// @Param       datetime   query  string   false  "Date-time or interval (RFC 3339); also accepted as eventTime"
// @Success     200  {object}  SystemEventCollectionResponse
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/events [get]
func (h *SystemEventHandler) ListEventsBySystem(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "id")

	if _, err := h.systemRepo.GetByID(r.Context(), systemID); err != nil {
		writeRepositoryError(w, r, h.logger, err, "System", "Failed to load system")
		return
	}

	params, err := queryparams.SystemEventsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	events, _, err := h.repo.List(r.Context(), params, &systemID)
	if err != nil {
		h.logger.Error("Failed to list system events", zap.String("systemId", systemID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	items := make([]any, 0, len(events))
	for _, event := range events {
		items = append(items, event)
	}

	links := params.BuildPaginationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query())

	w.Header().Set("Content-Type", mediatype.JSON)
	render.JSON(w, r, SystemEventCollectionResponse{Items: items, Links: links})
}

// CreateEventBySystem creates a system event
//
// @Summary     Create system event
// @Description Creates one or more system events for a given system
// @Tags        System Events
// @Accept      json
// @Param       id     path  string  true  "System ID"
// @Param       event  body  map[string]any  true  "System event or array of events"
// @Success     201
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/events [post]
func (h *SystemEventHandler) CreateEventBySystem(w http.ResponseWriter, r *http.Request) {
	h.creates.HTTP(w, r, systemEventCreator, chi.URLParam(r, "id"))
}

// systemEventCreator creates system events; one HTTP request may carry an
// array of them, created in one transaction.
var systemEventCreator = resourceCreator{
	Kind: "systemevent", Name: "System event", Contract: contractvalidation.SystemEvent, Codec: jsonCodec[*domains.SystemEvent]{}, Batch: true,
	Create: func(ctx context.Context, tx *repository.Repositories, in createInput) (string, error) {
		if _, err := tx.System.GetByID(ctx, in.Parent); err != nil {
			return "", parentError("system", in.Parent, err)
		}
		event, err := decodeSystemEvent(in.Value)
		if err != nil {
			return "", decodeError(err)
		}
		if err := validateSystemEvent(event); err != nil {
			return "", invalid(err)
		}
		event.ID = ""
		event.SystemID = in.Parent
		if err := tx.SystemEvent.Create(ctx, event); err != nil {
			return "", err
		}
		return "/systems/" + url.PathEscape(in.Parent) + "/events/" + url.PathEscape(event.ID), nil
	},
}

// GetEventByID returns a single system event by ID
//
// @Summary     Get system event
// @Description Returns a single system event resource by ID
// @Tags        System Events
// @Produce     json
// @Param       id       path  string  true  "System ID"
// @Param       eventId  path  string  true  "Event ID"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Router      /systems/{id}/events/{eventId} [get]
func (h *SystemEventHandler) GetEventByID(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "id")
	eventID := chi.URLParam(r, "eventId")

	event, err := h.repo.GetByID(r.Context(), systemID, eventID)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "System event", "Failed to load system event")
		return
	}

	w.Header().Set("Content-Type", mediatype.JSON)
	render.JSON(w, r, event)
}

// UpdateEventByID replaces a system event by ID
//
// @Summary     Update system event
// @Description Replaces a system event resource by ID
// @Tags        System Events
// @Accept      json
// @Param       id       path  string          true  "System ID"
// @Param       eventId  path  string          true  "Event ID"
// @Param       event    body  map[string]any  true  "System event resource"
// @Success     204
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/events/{eventId} [put]
func (h *SystemEventHandler) UpdateEventByID(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "id")
	eventID := chi.URLParam(r, "eventId")

	existing, err := h.repo.GetByID(r.Context(), systemID, eventID)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "System event", "Failed to load system event")
		return
	}

	body, err := validateRawRequestBody(r, h.validator, contractvalidation.SystemEvent)
	if err != nil {
		writeDeserializeError(w, r, err)
		return
	}
	event, err := decodeStrictJSON[domains.SystemEvent](body)
	if err != nil {
		writeDeserializeError(w, r, err)
		return
	}

	if err := validateSystemEvent(&event); err != nil {
		writeValidationError(w, r, err)
		return
	}

	event.ID = eventID
	event.SystemID = existing.SystemID
	if err := h.repo.Update(r.Context(), &event); err != nil {
		writeRepositoryError(w, r, h.logger, err, "System event", "Failed to update system event")
		return
	}

	w.WriteHeader(http.StatusNoContent)

}

// DeleteEventByID deletes a system event by ID
//
// @Summary     Delete system event
// @Description Deletes a system event resource by ID
// @Tags        System Events
// @Param       id       path  string  true  "System ID"
// @Param       eventId  path  string  true  "Event ID"
// @Success     204
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/events/{eventId} [delete]
func (h *SystemEventHandler) DeleteEventByID(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "id")
	eventID := chi.URLParam(r, "eventId")

	if _, err := h.repo.GetByID(r.Context(), systemID, eventID); err != nil {
		writeRepositoryError(w, r, h.logger, err, "System event", "Failed to load system event")
		return
	}

	if err := h.repo.Delete(r.Context(), systemID, eventID); err != nil {
		h.logger.Error("Failed to delete system event", zap.String("eventId", eventID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Failed to delete system event")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
