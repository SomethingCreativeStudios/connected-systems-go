package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// CommandCollectionResponse follows the collection shape used by other dynamic-data resources.
type CommandCollectionResponse struct {
	Items []any               `json:"items"`
	Links common_shared.Links `json:"links,omitempty"`
}

// CommandHandler handles command endpoints.
type CommandHandler struct {
	cfg               *config.Config
	logger            *zap.Logger
	repo              *repository.CommandRepository
	controlStreamRepo *repository.ControlStreamRepository
}

func NewCommandHandler(
	cfg *config.Config,
	logger *zap.Logger,
	repo *repository.CommandRepository,
	controlStreamRepo *repository.ControlStreamRepository,
) *CommandHandler {
	return &CommandHandler{
		cfg:               cfg,
		logger:            logger,
		repo:              repo,
		controlStreamRepo: controlStreamRepo,
	}
}

// ListCommands handles GET /commands
func (h *CommandHandler) ListCommands(w http.ResponseWriter, r *http.Request) {
	params := queryparams.CommandsQueryParams{}.BuildFromRequest(r)

	commands, total, err := h.repo.List(params, nil)
	if err != nil {
		h.logger.Error("Failed to list commands", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	items := make([]any, 0, len(commands))
	for _, cmd := range commands {
		items = append(items, cmd)
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(commands))

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, CommandCollectionResponse{Items: items, Links: links})
}

// ListControlStreamCommands handles GET /controlstreams/{id}/commands
func (h *CommandHandler) ListControlStreamCommands(w http.ResponseWriter, r *http.Request) {
	controlStreamID := chi.URLParam(r, "controlStreamId")
	if _, err := h.controlStreamRepo.GetByID(controlStreamID); err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Control stream not found"})
		return
	}

	params := queryparams.CommandsQueryParams{}.BuildFromRequest(r)

	commands, total, err := h.repo.ListByControlStream(controlStreamID, params)
	if err != nil {
		h.logger.Error("Failed to list commands", zap.String("controlStreamId", controlStreamID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	items := make([]any, 0, len(commands))
	for _, cmd := range commands {
		items = append(items, cmd)
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(commands))

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, CommandCollectionResponse{Items: items, Links: links})
}

// GetCommand handles GET /commands/{id}
func (h *CommandHandler) GetCommand(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "cmdId")

	cmd, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get command", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Command not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, cmd)
}

// CreateControlStreamCommand handles POST /controlstreams/{id}/commands
func (h *CommandHandler) CreateControlStreamCommand(w http.ResponseWriter, r *http.Request) {
	controlStreamID := chi.URLParam(r, "controlStreamId")
	if _, err := h.controlStreamRepo.GetByID(controlStreamID); err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Control stream not found"})
		return
	}

	cmd, err := decodeCommandPayload(r)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	cmd.ControlStreamID = controlStreamID
	if err := h.repo.Create(cmd); err != nil {
		h.logger.Error("Failed to create command", zap.String("controlStreamId", controlStreamID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create command"})
		return
	}

	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/commands/" + cmd.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

// UpdateCommand handles PUT /commands/{id}
func (h *CommandHandler) UpdateCommand(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "cmdId")

	existing, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Command not found", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Command not found"})
		return
	}

	cmd, err := decodeCommandPayload(r)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	cmd.ID = id
	cmd.ControlStreamID = existing.ControlStreamID
	if err := h.repo.Update(cmd); err != nil {
		h.logger.Error("Failed to update command", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update command"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteCommand handles DELETE /commands/{id}
func (h *CommandHandler) DeleteCommand(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "cmdId")

	if _, err := h.repo.GetByID(id); err != nil {
		h.logger.Error("Command not found", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Command not found"})
		return
	}

	if err := h.repo.Delete(id); err != nil {
		h.logger.Error("Failed to delete command", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to delete command"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func decodeCommandPayload(r *http.Request) (*domains.Command, error) {
	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}

	cmd := &domains.Command{}

	if sfID, ok := raw["samplingFeature@id"].(string); ok && sfID != "" {
		cmd.SamplingFeatureID = &sfID
	}

	if sender, ok := raw["sender"].(string); ok {
		cmd.Sender = sender
	}

	if status, ok := raw["currentStatus"].(string); ok && status != "" {
		cmd.CurrentStatus = domains.CommandStatus(status)
	}

	if issueTimeStr, ok := raw["issueTime"].(string); ok && issueTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, issueTimeStr); err == nil {
			cmd.IssueTime = &t
		}
	}

	if execTimeArr, ok := raw["executionTime"].([]interface{}); ok && len(execTimeArr) == 2 {
		tr := &common_shared.TimeRange{}
		if s, ok := execTimeArr[0].(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				tr.Start = &t
			}
		}
		if e, ok := execTimeArr[1].(string); ok {
			if t, err := time.Parse(time.RFC3339, e); err == nil {
				tr.End = &t
			}
		}
		cmd.ExecutionTime = tr
	}

	if procLink, ok := raw["procedure@link"].(map[string]any); ok {
		link := &common_shared.Link{}
		if href, ok := procLink["href"].(string); ok {
			link.Href = href
		}
		if rel, ok := procLink["rel"].(string); ok {
			link.Rel = rel
		}
		cmd.ProcedureLink = link
	}

	if params, ok := raw["parameters"]; ok {
		paramBytes, err := json.Marshal(params)
		if err == nil {
			cmd.Parameters = paramBytes
		}
	}

	return cmd, nil
}
