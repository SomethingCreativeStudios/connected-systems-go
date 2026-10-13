package api

import (
	"net/http"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// ListSystemHistory handles GET /systems/{id}/history.
//
// @Summary     List system history
// @Description Returns historical revisions of a system
// @Tags        System History
// @Produce     json
// @Param       id        path   string   true   "System ID"
// @Param       limit     query  integer  false  "Maximum number of results"
// @Param       cursor    query  string   false  "Opaque pagination cursor"
// @Param       validTime  query  string   false  "Valid time filter (RFC 3339 date-time or interval)"
// @Param       keyword   query  string   false  "Comma-separated keywords to match against history fields"
// @Success     200  {object}  map[string]any
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/history [get]
func (h *SystemHandler) ListSystemHistory(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "id")

	if _, err := h.repo.GetByID(r.Context(), systemID); err != nil {
		writeRepositoryError(w, r, h.logger, err, "System", "Failed to load system")
		return
	}

	params, err := queryparams.SystemHistoryQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	revisions, total, err := h.historyRepo.List(r.Context(), systemID, params)
	if err != nil {
		h.logger.Error("Failed to list system history", zap.String("systemId", systemID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	systems := make([]*domains.System, 0, len(revisions))
	for _, rev := range revisions {
		system, err := h.historyRepo.DecodeRevisionSystem(rev)
		if err != nil {
			h.logger.Error("Failed to decode system history revision", zap.String("revId", rev.ID), zap.Error(err))
			continue
		}
		systems = append(systems, system)
	}

	writeCollection(w, r, h.logger, h.fc, h.cfg.API.BaseURL, systems, total, params.QueryParams)
}

// GetSystemHistoryRevision handles GET /systems/{id}/history/{revId}.
//
// @Summary     Get system history revision
// @Description Returns a specific historical revision of a system
// @Tags        System History
// @Produce     json
// @Param       id     path  string  true  "System ID"
// @Param       revId  path  string  true  "Revision ID"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/history/{revId} [get]
func (h *SystemHandler) GetSystemHistoryRevision(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "id")
	revID := chi.URLParam(r, "revId")

	revision, err := h.historyRepo.GetByID(r.Context(), systemID, revID)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "System history revision", "Failed to load system history revision")
		return
	}

	system, err := h.historyRepo.DecodeRevisionSystem(revision)
	if err != nil {
		h.logger.Error("Failed to decode system history revision", zap.String("revId", revID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Failed to decode system revision")
		return
	}

	writeResource(w, r, h.logger, h.fc, system)
}

// UpdateSystemHistoryRevision handles PUT /systems/{id}/history/{revId}.
//
// @Summary     Update system history revision
// @Description Replaces the snapshot for a specific system history revision (validTime cannot change)
// @Tags        System History
// @Accept      json
// @Param       id      path  string          true  "System ID"
// @Param       revId   path  string          true  "Revision ID"
// @Param       system  body  map[string]any  true  "System snapshot"
// @Success     204
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/history/{revId} [put]
func (h *SystemHandler) UpdateSystemHistoryRevision(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "id")
	revID := chi.URLParam(r, "revId")

	existingRevision, err := h.historyRepo.GetByID(r.Context(), systemID, revID)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "System history revision", "Failed to load system history revision")
		return
	}

	existingSystem, err := h.historyRepo.DecodeRevisionSystem(existingRevision)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "Failed to decode existing revision")
		return
	}

	contentType := r.Header.Get("Content-Type")
	updatedSystem, err := h.fc.DeserializeContext(r.Context(), contentType, r.Body)
	if err != nil {
		writeDeserializeError(w, r, err)
		return
	}

	// Requirement: validTime of the provided revision must not change.
	if !sameTimeRange(existingSystem.ValidTime, updatedSystem.ValidTime) {
		writeError(w, r, http.StatusBadRequest, "validTime cannot be changed for an existing history revision")
		return
	}

	updatedSystem.ID = existingSystem.ID
	if err := h.historyRepo.UpdateSnapshot(r.Context(), systemID, revID, updatedSystem); err != nil {
		h.logger.Error("Failed to update system history revision", zap.String("revId", revID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Failed to update system history revision")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteSystemHistoryRevision handles DELETE /systems/{id}/history/{revId}.
//
// @Summary     Delete system history revision
// @Description Deletes a specific system history revision
// @Tags        System History
// @Param       id     path  string  true  "System ID"
// @Param       revId  path  string  true  "Revision ID"
// @Success     204
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /systems/{id}/history/{revId} [delete]
func (h *SystemHandler) DeleteSystemHistoryRevision(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "id")
	revID := chi.URLParam(r, "revId")

	if _, err := h.historyRepo.GetByID(r.Context(), systemID, revID); err != nil {
		writeRepositoryError(w, r, h.logger, err, "System history revision", "Failed to load system history revision")
		return
	}

	if err := h.historyRepo.Delete(r.Context(), systemID, revID); err != nil {
		h.logger.Error("Failed to delete system history revision", zap.String("revId", revID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Failed to delete system history revision")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func sameTimeRange(a, b *common_shared.TimeRange) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	startEqual := (a.Start == nil && b.Start == nil) || (a.Start != nil && b.Start != nil && a.Start.Equal(*b.Start))
	endEqual := (a.End == nil && b.End == nil) || (a.End != nil && b.End != nil && a.End.Equal(*b.End))

	return startEqual && endEqual
}
