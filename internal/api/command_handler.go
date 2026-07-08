package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters"
	queryparams "github.com/yourusername/connected-systems-go/internal/model/query_params"
	"github.com/yourusername/connected-systems-go/internal/mqtt"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// CommandCollectionResponse follows the collection shape used by other dynamic-data resources.
type CommandCollectionResponse struct {
	Items []any               `json:"items"`
	Links common_shared.Links `json:"links,omitempty"`
}

// CommandStatusCollectionResponse follows commandStatusCollection.json.
type CommandStatusCollectionResponse struct {
	Items []any               `json:"items"`
	Links common_shared.Links `json:"links,omitempty"`
}

// CommandResultCollectionResponse follows commandResultCollection.json.
type CommandResultCollectionResponse struct {
	Items []any               `json:"items"`
	Links common_shared.Links `json:"links,omitempty"`
}

// CommandHandler handles command endpoints.
type CommandHandler struct {
	cfg               *config.Config
	logger            *zap.Logger
	repo              *repository.CommandRepository
	controlStreamRepo *repository.ControlStreamRepository
	mqttManager       *mqtt.Manager
	fc                *formaters.MultiFormatFormatterCollection[*domains.Command]
}

func NewCommandHandler(
	cfg *config.Config,
	logger *zap.Logger,
	repo *repository.CommandRepository,
	controlStreamRepo *repository.ControlStreamRepository,
	mqttManager *mqtt.Manager,
	fc *formaters.MultiFormatFormatterCollection[*domains.Command],
) *CommandHandler {
	return &CommandHandler{
		cfg:               cfg,
		logger:            logger,
		repo:              repo,
		controlStreamRepo: controlStreamRepo,
		mqttManager:       mqttManager,
		fc:                fc,
	}
}

// ListCommands handles GET /commands
//
// @Summary     List commands
// @Description Returns a paginated collection of command resources
// @Tags        Commands
// @Produce     json
// @Param       limit          query  integer  false  "Maximum number of results"
// @Param       offset         query  integer  false  "Result offset"
// @Param       id             query  string   false  "Comma-separated resource IDs"
// @Param       q              query  string   false  "Comma-separated keywords for full-text search"
// @Param       controlStream  query  string   false  "Comma-separated control stream IDs"
// @Param       system         query  string   false  "Comma-separated system IDs"
// @Param       foi            query  string   false  "Comma-separated feature of interest IDs"
// @Param       currentStatus  query  string   false  "Comma-separated status values"
// @Param       issueTime      query  string   false  "Issue time filter (RFC 3339 date-time or interval)"
// @Param       executionTime  query  string   false  "Execution time filter (RFC 3339 date-time or interval)"
// @Success     200  {object}  CommandCollectionResponse
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /commands [get]
func (h *CommandHandler) ListCommands(w http.ResponseWriter, r *http.Request) {
	params, err := queryparams.CommandsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	commands, total, err := h.repo.List(params, nil)
	if err != nil {
		h.logger.Error("Failed to list commands", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	items, err := h.fc.SerializeAll(acceptHeader, commands)
	if err != nil {
		h.logger.Error("Failed to serialize commands", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize commands"})
		return
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(commands))

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, CommandCollectionResponse{Items: items, Links: links})
}

// ListControlStreamCommands handles GET /controlstreams/{id}/commands
//
// @Summary     List control stream commands
// @Description Returns commands associated with a given control stream
// @Tags        Commands
// @Produce     json
// @Param       controlStreamId  path   string   true   "Control Stream ID"
// @Param       limit            query  integer  false  "Maximum number of results"
// @Param       offset           query  integer  false  "Result offset"
// @Param       q                query  string   false  "Comma-separated keywords for full-text search"
// @Param       system           query  string   false  "Comma-separated system IDs"
// @Param       foi              query  string   false  "Comma-separated feature of interest IDs"
// @Param       currentStatus    query  string   false  "Comma-separated status values"
// @Param       issueTime        query  string   false  "Issue time filter (RFC 3339 date-time or interval)"
// @Param       executionTime    query  string   false  "Execution time filter (RFC 3339 date-time or interval)"
// @Success     200  {object}  CommandCollectionResponse
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /controlstreams/{controlStreamId}/commands [get]
func (h *CommandHandler) ListControlStreamCommands(w http.ResponseWriter, r *http.Request) {
	controlStreamID := chi.URLParam(r, "controlStreamId")
	if _, err := h.controlStreamRepo.GetByID(controlStreamID); err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Control stream not found"})
		return
	}

	params, err := queryparams.CommandsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	commands, total, err := h.repo.ListByControlStream(controlStreamID, params)
	if err != nil {
		h.logger.Error("Failed to list commands", zap.String("controlStreamId", controlStreamID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	items, err := h.fc.SerializeAll(acceptHeader, commands)
	if err != nil {
		h.logger.Error("Failed to serialize commands", zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize commands"})
		return
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(commands))

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, CommandCollectionResponse{Items: items, Links: links})
}

// GetCommand handles GET /commands/{id}
//
// @Summary     Get command
// @Description Returns a single command resource by ID
// @Tags        Commands
// @Produce     json
// @Param       cmdId  path  string  true  "Command ID"
// @Success     200  {object}  map[string]any
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /commands/{cmdId} [get]
func (h *CommandHandler) GetCommand(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "cmdId")

	cmd, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Failed to get command", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Command not found"})
		return
	}

	acceptHeader := r.Header.Get("Accept")
	serialized, err := h.fc.Serialize(acceptHeader, cmd)
	if err != nil {
		h.logger.Error("Failed to serialize command", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to serialize command"})
		return
	}

	w.Header().Set("Content-Type", h.fc.GetResponseContentType(acceptHeader))
	render.JSON(w, r, serialized)
}

// CreateControlStreamCommand handles POST /controlstreams/{id}/commands
//
// @Summary     Create command
// @Description Creates a new command in the given control stream
// @Tags        Commands
// @Accept      json
// @Param       controlStreamId  path  string          true  "Control Stream ID"
// @Param       command          body  map[string]any  true  "Command resource"
// @Success     201
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /controlstreams/{controlStreamId}/commands [post]
func (h *CommandHandler) CreateControlStreamCommand(w http.ResponseWriter, r *http.Request) {
	controlStreamID := chi.URLParam(r, "controlStreamId")
	if _, err := h.controlStreamRepo.GetByID(controlStreamID); err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Control stream not found"})
		return
	}

	cmd, err := h.fc.Deserialize(r.Header.Get("Content-Type"), r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize command", zap.Error(err))
		writeDeserializeError(w, r, err)
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

	// Publish to MQTT (fire-and-forget)
	h.publishCommandToMQTT(controlStreamID, cmd)
}

// UpdateCommand handles PUT /commands/{id}
//
// @Summary     Update command
// @Description Replaces a command resource by ID
// @Tags        Commands
// @Accept      json
// @Param       cmdId    path  string          true  "Command ID"
// @Param       command  body  map[string]any  true  "Command resource"
// @Success     204
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /commands/{cmdId} [put]
func (h *CommandHandler) UpdateCommand(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "cmdId")

	existing, err := h.repo.GetByID(id)
	if err != nil {
		h.logger.Error("Command not found", zap.String("id", id), zap.Error(err))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Command not found"})
		return
	}

	cmd, err := h.fc.Deserialize(r.Header.Get("Content-Type"), r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize command", zap.Error(err))
		writeDeserializeError(w, r, err)
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
//
// @Summary     Delete command
// @Description Deletes a command resource by ID
// @Tags        Commands
// @Param       cmdId  path  string  true  "Command ID"
// @Success     204
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /commands/{cmdId} [delete]
func (h *CommandHandler) DeleteCommand(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "cmdId")

	if err := h.repo.Delete(id); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "Command not found"})
		default:
			h.logger.Error("Failed to delete command", zap.String("id", id), zap.Error(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to delete command"})
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListCommandStatusReports handles GET /commands/{id}/status
func (h *CommandHandler) ListCommandStatusReports(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	if _, err := h.repo.GetByID(commandID); err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Command not found"})
		return
	}

	params, err := queryparams.CommandStatusQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	statuses, total, err := h.repo.ListStatuses(commandID, params)
	if err != nil {
		h.logger.Error("Failed to list command status reports", zap.String("cmdId", commandID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	items := make([]any, 0, len(statuses))
	for _, status := range statuses {
		items = append(items, commandStatusResponse(status))
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(statuses))

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, CommandStatusCollectionResponse{Items: items, Links: links})
}

// CreateCommandStatusReport handles POST /commands/{id}/status
func (h *CommandHandler) CreateCommandStatusReport(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	if _, err := h.repo.GetByID(commandID); err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Command not found"})
		return
	}

	status, err := decodeCommandStatusPayload(r)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	status.CommandID = commandID
	if err := h.repo.CreateStatus(status); err != nil {
		h.logger.Error("Failed to create command status report", zap.String("cmdId", commandID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create command status report"})
		return
	}

	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/commands/" + commandID + "/status/" + status.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

// GetCommandStatusReport handles GET /commands/{id}/status/{statusId}
func (h *CommandHandler) GetCommandStatusReport(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	statusID := chi.URLParam(r, "statusId")

	status, err := h.repo.GetStatusByID(commandID, statusID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Command status report not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, commandStatusResponse(status))
}

// UpdateCommandStatusReport handles PUT /commands/{id}/status/{statusId}
func (h *CommandHandler) UpdateCommandStatusReport(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	statusID := chi.URLParam(r, "statusId")

	existing, err := h.repo.GetStatusByID(commandID, statusID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Command status report not found"})
		return
	}

	status, err := decodeCommandStatusPayload(r)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	status.ID = existing.ID
	status.CommandID = existing.CommandID
	if status.ReportTime.IsZero() {
		status.ReportTime = existing.ReportTime
	}
	if err := h.repo.UpdateStatus(status); err != nil {
		h.logger.Error("Failed to update command status report", zap.String("statusId", statusID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update command status report"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteCommandStatusReport handles DELETE /commands/{id}/status/{statusId}
func (h *CommandHandler) DeleteCommandStatusReport(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	statusID := chi.URLParam(r, "statusId")

	if err := h.repo.DeleteStatus(commandID, statusID); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "Command status report not found"})
		default:
			h.logger.Error("Failed to delete command status report", zap.String("statusId", statusID), zap.Error(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to delete command status report"})
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListCommandResults handles GET /commands/{id}/result
func (h *CommandHandler) ListCommandResults(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	if _, err := h.repo.GetByID(commandID); err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Command not found"})
		return
	}

	params := queryparams.CommandResultQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	results, total, err := h.repo.ListResults(commandID, params)
	if err != nil {
		h.logger.Error("Failed to list command results", zap.String("cmdId", commandID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Internal server error"})
		return
	}

	items := make([]any, 0, len(results))
	for _, result := range results {
		items = append(items, result)
	}

	totalInt := int(total)
	links := params.QueryParams.BuildPagintationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query(), &totalInt, len(results))

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, CommandResultCollectionResponse{Items: items, Links: links})
}

// CreateCommandResult handles POST /commands/{id}/result
func (h *CommandHandler) CreateCommandResult(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	if _, err := h.repo.GetByID(commandID); err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Command not found"})
		return
	}

	result, err := decodeCommandResultPayload(r)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	result.CommandID = commandID
	if err := h.repo.CreateResult(result); err != nil {
		h.logger.Error("Failed to create command result", zap.String("cmdId", commandID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to create command result"})
		return
	}

	location := strings.TrimRight(h.cfg.API.BaseURL, "/") + "/commands/" + commandID + "/result/" + result.ID
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

// GetCommandResult handles GET /commands/{id}/result/{resultId}
func (h *CommandHandler) GetCommandResult(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	resultID := chi.URLParam(r, "resultId")

	result, err := h.repo.GetResultByID(commandID, resultID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Command result not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	render.JSON(w, r, result)
}

// UpdateCommandResult handles PUT /commands/{id}/result/{resultId}
func (h *CommandHandler) UpdateCommandResult(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	resultID := chi.URLParam(r, "resultId")

	existing, err := h.repo.GetResultByID(commandID, resultID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "Command result not found"})
		return
	}

	result, err := decodeCommandResultPayload(r)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	result.ID = existing.ID
	result.CommandID = existing.CommandID
	if err := h.repo.UpdateResult(result); err != nil {
		h.logger.Error("Failed to update command result", zap.String("resultId", resultID), zap.Error(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "Failed to update command result"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteCommandResult handles DELETE /commands/{id}/result/{resultId}
func (h *CommandHandler) DeleteCommandResult(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	resultID := chi.URLParam(r, "resultId")

	if err := h.repo.DeleteResult(commandID, resultID); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": "Command result not found"})
		default:
			h.logger.Error("Failed to delete command result", zap.String("resultId", resultID), zap.Error(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "Failed to delete command result"})
		}
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
		cmd.ExecutionTime = common_shared.NonEmptyTimeRange(tr)
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

func decodeCommandStatusPayload(r *http.Request) (*domains.CommandStatusReport, error) {
	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}

	statusRaw, ok := raw["statusCode"].(string)
	if !ok || statusRaw == "" {
		return nil, errors.New("statusCode is required")
	}
	statusCode := domains.CommandStatus(statusRaw)
	if !isValidCommandStatus(statusCode) {
		return nil, errors.New("statusCode is invalid")
	}

	status := &domains.CommandStatusReport{
		StatusCode: statusCode,
	}

	if reportTimeRaw, exists := raw["reportTime"]; exists {
		reportTimeStr, ok := reportTimeRaw.(string)
		if !ok {
			return nil, errors.New("reportTime must be an ISO 8601 string")
		}
		if reportTimeStr != "" {
			reportTime, err := time.Parse(time.RFC3339, reportTimeStr)
			if err != nil {
				return nil, errors.New("reportTime must be an ISO 8601 string")
			}
			status.ReportTime = reportTime
		}
	}

	if percentRaw, exists := raw["percentCompletion"]; exists {
		percent, ok := percentRaw.(float64)
		if !ok {
			return nil, errors.New("percentCompletion must be a number")
		}
		if percent < 0 || percent > 100 {
			return nil, errors.New("percentCompletion must be between 0 and 100")
		}
		status.PercentCompletion = &percent
	}

	if executionRaw, exists := raw["executionTime"]; exists {
		executionBytes, err := json.Marshal(executionRaw)
		if err != nil {
			return nil, errors.New("Invalid executionTime payload")
		}
		var executionTime common_shared.TimeRange
		if err := json.Unmarshal(executionBytes, &executionTime); err != nil {
			return nil, errors.New("executionTime must be a valid time period")
		}
		status.ExecutionTime = common_shared.NonEmptyTimeRange(&executionTime)
	}

	if message, ok := raw["message"].(string); ok {
		status.Message = message
	}

	if resultsRaw, exists := raw["results"]; exists {
		if _, ok := resultsRaw.([]any); !ok {
			return nil, errors.New("results must be an array")
		}
		resultsBytes, err := json.Marshal(resultsRaw)
		if err != nil {
			return nil, errors.New("Invalid results payload")
		}
		status.Results = resultsBytes
	}

	return status, nil
}

func decodeCommandResultPayload(r *http.Request) (*domains.CommandResult, error) {
	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}

	result := &domains.CommandResult{}
	variants := []struct {
		key         string
		target      *json.RawMessage
		requireHref bool
	}{
		{key: "data", target: &result.Data},
		{key: "observation@link", target: &result.ObservationLink, requireHref: true},
		{key: "observationSet@link", target: &result.ObservationSetLink, requireHref: true},
		{key: "datastream@link", target: &result.DatastreamLink, requireHref: true},
		{key: "external@link", target: &result.ExternalLink, requireHref: true},
	}

	var present int
	for _, variant := range variants {
		value, exists := raw[variant.key]
		if !exists {
			continue
		}
		if variant.requireHref {
			if err := validateLinkLikePayload(variant.key, value); err != nil {
				return nil, err
			}
		}
		valueBytes, err := json.Marshal(value)
		if err != nil {
			return nil, errors.New("Invalid command result payload")
		}
		*variant.target = valueBytes
		present++
	}

	if present == 0 {
		return nil, errors.New("command result must include one result variant")
	}
	if present > 1 {
		return nil, errors.New("command result must include exactly one result variant")
	}

	return result, nil
}

func validateLinkLikePayload(field string, value any) error {
	obj, ok := value.(map[string]any)
	if !ok {
		return errors.New(field + " must be an object")
	}
	href, ok := obj["href"].(string)
	if !ok || href == "" {
		return errors.New(field + " must include href")
	}
	return nil
}

func isValidCommandStatus(status domains.CommandStatus) bool {
	switch status {
	case domains.CommandStatusPending,
		domains.CommandStatusAccepted,
		domains.CommandStatusRejected,
		domains.CommandStatusScheduled,
		domains.CommandStatusUpdated,
		domains.CommandStatusCanceled,
		domains.CommandStatusExecuting,
		domains.CommandStatusFailed,
		domains.CommandStatusCompleted:
		return true
	default:
		return false
	}
}

func commandResponse(cmd *domains.Command) *domains.Command {
	if cmd == nil {
		return nil
	}
	out := *cmd
	out.ExecutionTime = common_shared.NonEmptyTimeRange(out.ExecutionTime)
	return &out
}

func commandStatusResponse(status *domains.CommandStatusReport) *domains.CommandStatusReport {
	if status == nil {
		return nil
	}
	out := *status
	out.ExecutionTime = common_shared.NonEmptyTimeRange(out.ExecutionTime)
	return &out
}

// publishCommandToMQTT publishes a command to the MQTT topic for its control stream.
// Non-blocking — errors are logged.
func (h *CommandHandler) publishCommandToMQTT(controlStreamID string, cmd *domains.Command) {
	if h.mqttManager == nil || !h.mqttManager.IsConnected() {
		return
	}

	payload, err := json.Marshal(cmd)
	if err != nil {
		h.logger.Error("Failed to marshal command for MQTT", zap.Error(err))
		return
	}

	h.mqttManager.Publish(mqtt.CommandTopic(controlStreamID), payload)
}
