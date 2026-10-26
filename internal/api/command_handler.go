package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/contractvalidation"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters/json_formatters"
	queryparams "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/query_params"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
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
	fc                *formaters.MultiFormatFormatterCollection[*domains.Command]
	validator         *contractvalidation.Validator
	creates           *CreateOperations
}

func NewCommandHandler(
	cfg *config.Config,
	logger *zap.Logger,
	repo *repository.CommandRepository,
	controlStreamRepo *repository.ControlStreamRepository,
	fc *formaters.MultiFormatFormatterCollection[*domains.Command],
	validator *contractvalidation.Validator,
	creates *CreateOperations,
) *CommandHandler {
	return &CommandHandler{
		cfg:               cfg,
		logger:            logger,
		repo:              repo,
		controlStreamRepo: controlStreamRepo,
		fc:                fc,
		validator:         validator,
		creates:           creates,
	}
}

// ListCommands handles GET /commands
//
// @Summary     List commands
// @Description Returns a paginated collection of command resources
// @Tags        Commands
// @Produce     json
// @Param       limit          query  integer  false  "Maximum number of results"
// @Param       cursor         query  string   false  "Opaque pagination cursor"
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
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	commands, _, err := h.repo.List(r.Context(), params, nil)
	if err != nil {
		h.logger.Error("Failed to list commands", zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeCollection(w, r, h.logger, h.fc, h.cfg.API.BaseURL, commands, 0, params.QueryParams)
}

// ListControlStreamCommands handles GET /controlstreams/{id}/commands
//
// @Summary     List control stream commands
// @Description Returns commands associated with a given control stream
// @Tags        Commands
// @Produce     json
// @Param       controlStreamId  path   string   true   "Control Stream ID"
// @Param       limit            query  integer  false  "Maximum number of results"
// @Param       cursor           query  string   false  "Opaque pagination cursor"
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
	if _, err := h.controlStreamRepo.GetByID(r.Context(), controlStreamID); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Control stream", "Failed to load control stream")
		return
	}

	params, err := queryparams.CommandsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	commands, _, err := h.repo.ListByControlStream(r.Context(), controlStreamID, params)
	if err != nil {
		h.logger.Error("Failed to list commands", zap.String("controlStreamId", controlStreamID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeCollection(w, r, h.logger, h.fc, h.cfg.API.BaseURL, commands, 0, params.QueryParams)
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

	cmd, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command", "Failed to load command")
		return
	}

	writeResource(w, r, h.logger, h.fc, cmd)
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
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /controlstreams/{controlStreamId}/commands [post]
func (h *CommandHandler) CreateControlStreamCommand(w http.ResponseWriter, r *http.Request) {
	h.creates.HTTP(w, r, commandCreator, chi.URLParam(r, "controlStreamId"))
}

// commandFormatters builds the command representation HTTP serves, creates
// decode and MQTT publishes.
func commandFormatters(links *formaters.LinkResolver) *formaters.MultiFormatFormatterCollection[*domains.Command] {
	return formaters.NewFormatterCollection(formaters.Adapt(json_formatters.NewCommandJSONFormatter(links)))
}

// commandCreator creates a command in its control stream. The repository
// validates the parameters against the control stream schema.
var commandCreator = resourceCreator{
	Kind: "command", Name: "Command", Contract: contractvalidation.Command, Codec: formatterCodec[*domains.Command](commandFormatters), ParentField: "controlstream@id",
	Create: func(ctx context.Context, tx *repository.Repositories, in createInput) (string, error) {
		if _, err := tx.ControlStream.GetByID(ctx, in.Parent); err != nil {
			return "", parentError("controlstream", in.Parent, err)
		}
		command, err := commandFormatters(in.Links).DeserializeContext(ctx, in.Media, bytes.NewReader(in.Body))
		if err != nil {
			return "", decodeError(err)
		}
		command.ID = ""
		command.ControlStreamID = in.Parent
		if err := tx.Command.Create(ctx, command); err != nil {
			return "", err
		}
		return "/commands/" + url.PathEscape(command.ID), nil
	},
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
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /commands/{cmdId} [put]
func (h *CommandHandler) UpdateCommand(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "cmdId")

	existing, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command", "Failed to load command")
		return
	}

	cmd, err := h.fc.DeserializeContext(r.Context(), r.Header.Get("Content-Type"), r.Body)
	if err != nil {
		h.logger.Error("Failed to deserialize command", zap.Error(err))
		writeDeserializeError(w, r, err)
		return
	}

	cmd.ID = id
	cmd.ControlStreamID = existing.ControlStreamID
	// Read-only per spec: issueTime is fixed at creation, currentStatus and
	// executionTime are driven by status reports.
	cmd.IssueTime = existing.IssueTime
	cmd.CurrentStatus = existing.CurrentStatus
	cmd.ExecutionTime = existing.ExecutionTime
	if err := h.repo.Update(r.Context(), cmd); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command", "Failed to update command")
		return
	}

	w.WriteHeader(http.StatusNoContent)

	// A Resource Data Message carries the complete updated command.

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

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command", "Failed to delete command")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListCommandStatusReports handles GET /commands/{id}/status
//
// @Summary     List command status reports
// @Description Returns a paginated collection of status reports for a command
// @Tags        Commands
// @Produce     json
// @Param       cmdId   path   string   true   "Command ID"
// @Param       limit   query  integer  false  "Maximum number of results"
// @Param       cursor  query  string   false  "Opaque pagination cursor"
// @Success     200     {object}  CommandStatusCollectionResponse
// @Failure     400     {object}  map[string]string
// @Failure     404     {object}  map[string]string
// @Failure     500     {object}  map[string]string
// @Router      /commands/{cmdId}/status [get]
func (h *CommandHandler) ListCommandStatusReports(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	if _, err := h.repo.GetByID(r.Context(), commandID); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command", "Failed to load command")
		return
	}

	params, err := queryparams.CommandStatusQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	statuses, _, err := h.repo.ListStatuses(r.Context(), commandID, params)
	if err != nil {
		h.logger.Error("Failed to list command status reports", zap.String("cmdId", commandID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	items := make([]any, 0, len(statuses))
	for _, status := range statuses {
		items = append(items, commandStatusResponse(status))
	}

	links := params.BuildPaginationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query())

	w.Header().Set("Content-Type", mediatype.JSON)
	render.JSON(w, r, CommandStatusCollectionResponse{Items: items, Links: links})
}

// CreateCommandStatusReport handles POST /commands/{id}/status
//
// @Summary     Create command status report
// @Description Creates a status report for a command.
// @Tags        Commands
// @Accept      json
// @Param       cmdId   path  string          true  "Command ID"
// @Param       status  body  map[string]any  true  "Command status report"
// @Success     201
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /commands/{cmdId}/status [post]
func (h *CommandHandler) CreateCommandStatusReport(w http.ResponseWriter, r *http.Request) {
	h.creates.HTTP(w, r, commandStatusCreator, chi.URLParam(r, "cmdId"))
}

var commandStatusCreator = resourceCreator{
	Kind: "commandstatus", Name: "Command status report", Contract: contractvalidation.CommandStatus, ParentField: "command@id",
	// Encoded as GetCommandStatusReport serves it.
	Codec: jsonCodec[*domains.CommandStatusReport]{shape: commandStatusResponse},
	Create: func(ctx context.Context, tx *repository.Repositories, in createInput) (string, error) {
		if _, err := tx.Command.GetByID(ctx, in.Parent); err != nil {
			return "", parentError("command", in.Parent, err)
		}
		status, err := decodeCommandStatusPayload(bytes.NewReader(in.Body))
		if err != nil {
			return "", decodeError(err)
		}
		status.ID = ""
		status.CommandID = in.Parent
		if err := tx.Command.CreateStatus(ctx, status); err != nil {
			return "", err
		}
		return "/commands/" + url.PathEscape(in.Parent) + "/status/" + url.PathEscape(status.ID), nil
	},
}

// GetCommandStatusReport handles GET /commands/{id}/status/{statusId}
func (h *CommandHandler) GetCommandStatusReport(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	statusID := chi.URLParam(r, "statusId")

	status, err := h.repo.GetStatusByID(r.Context(), commandID, statusID)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command status report", "Failed to load command status report")
		return
	}

	w.Header().Set("Content-Type", mediatype.JSON)
	render.JSON(w, r, commandStatusResponse(status))
}

// UpdateCommandStatusReport handles PUT /commands/{id}/status/{statusId}
//
// @Summary     Update command status report
// @Description Replaces a command status report.
// @Tags        Commands
// @Accept      json
// @Param       cmdId     path  string                       true  "Command ID"
// @Param       statusId  path  string                       true  "Status report ID"
// @Param       status    body  map[string]any  true  "Command status report"
// @Success     204
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /commands/{cmdId}/status/{statusId} [put]
func (h *CommandHandler) UpdateCommandStatusReport(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	statusID := chi.URLParam(r, "statusId")

	existing, err := h.repo.GetStatusByID(r.Context(), commandID, statusID)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command status report", "Failed to load command status report")
		return
	}

	body, err := validateRawRequestBody(r, h.validator, contractvalidation.CommandStatus)
	if err != nil {
		writeValidationError(w, r, err)
		return
	}
	status, err := decodeCommandStatusPayload(bytes.NewReader(body))
	if err != nil {
		writeValidationError(w, r, err)
		return
	}

	status.ID = existing.ID
	status.CommandID = existing.CommandID
	status.ReportTime = existing.ReportTime
	if err := h.repo.UpdateStatusREST(r.Context(), status); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command status report", "Failed to update command status report")
		return
	}

	w.WriteHeader(http.StatusNoContent)

}

// DeleteCommandStatusReport handles DELETE /commands/{id}/status/{statusId}
func (h *CommandHandler) DeleteCommandStatusReport(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	statusID := chi.URLParam(r, "statusId")

	if err := h.repo.DeleteStatus(r.Context(), commandID, statusID); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command status report", "Failed to delete command status report")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListCommandResults handles GET /commands/{id}/result
//
// @Summary     List command results
// @Description Returns a paginated collection of results for a command
// @Tags        Commands
// @Produce     json
// @Param       cmdId   path   string   true   "Command ID"
// @Param       limit   query  integer  false  "Maximum number of results"
// @Param       cursor  query  string   false  "Opaque pagination cursor"
// @Success     200     {object}  CommandResultCollectionResponse
// @Failure     400     {object}  map[string]string
// @Failure     404     {object}  map[string]string
// @Failure     500     {object}  map[string]string
// @Router      /commands/{cmdId}/result [get]
func (h *CommandHandler) ListCommandResults(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	if _, err := h.repo.GetByID(r.Context(), commandID); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command", "Failed to load command")
		return
	}

	params, err := queryparams.CommandResultQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	results, _, err := h.repo.ListResults(r.Context(), commandID, params)
	if err != nil {
		h.logger.Error("Failed to list command results", zap.String("cmdId", commandID), zap.Error(err))
		writeError(w, r, http.StatusInternalServerError, "Internal server error")
		return
	}

	items := make([]any, 0, len(results))
	for _, result := range results {
		items = append(items, result)
	}

	links := params.BuildPaginationLinks(h.cfg.API.BaseURL+r.URL.Path, r.URL.Query())

	w.Header().Set("Content-Type", mediatype.JSON)
	render.JSON(w, r, CommandResultCollectionResponse{Items: items, Links: links})
}

// CreateCommandResult handles POST /commands/{id}/result
//
// @Summary     Create command result
// @Description Creates one result resource for a command.
// @Tags        Commands
// @Accept      json
// @Param       cmdId   path  string          true  "Command ID"
// @Param       result  body  map[string]any  true  "Command result"
// @Success     201
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /commands/{cmdId}/result [post]
func (h *CommandHandler) CreateCommandResult(w http.ResponseWriter, r *http.Request) {
	h.creates.HTTP(w, r, commandResultCreator, chi.URLParam(r, "cmdId"))
}

var commandResultCreator = resourceCreator{
	Kind: "commandresult", Name: "Command result", Contract: contractvalidation.CommandResult, Codec: jsonCodec[*domains.CommandResult]{}, ParentField: "command@id",
	Create: func(ctx context.Context, tx *repository.Repositories, in createInput) (string, error) {
		if _, err := tx.Command.GetByID(ctx, in.Parent); err != nil {
			return "", parentError("command", in.Parent, err)
		}
		result, err := decodeCommandResultPayload(bytes.NewReader(in.Body))
		if err != nil {
			return "", decodeError(err)
		}
		result.ID = ""
		result.CommandID = in.Parent
		if err := tx.Command.CreateResult(ctx, result); err != nil {
			return "", err
		}
		return "/commands/" + url.PathEscape(in.Parent) + "/result/" + url.PathEscape(result.ID), nil
	},
}

// GetCommandResult handles GET /commands/{id}/result/{resultId}
func (h *CommandHandler) GetCommandResult(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	resultID := chi.URLParam(r, "resultId")

	result, err := h.repo.GetResultByID(r.Context(), commandID, resultID)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command result", "Failed to load command result")
		return
	}

	w.Header().Set("Content-Type", mediatype.JSON)
	render.JSON(w, r, result)
}

// UpdateCommandResult handles PUT /commands/{id}/result/{resultId}
//
// @Summary     Update command result
// @Description Replaces a command result resource.
// @Tags        Commands
// @Accept      json
// @Param       cmdId     path  string                 true  "Command ID"
// @Param       resultId  path  string                 true  "Result ID"
// @Param       result    body  map[string]any  true  "Command result"
// @Success     204
// @Failure     400  {object}  ValidationErrorResponse
// @Failure     404  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /commands/{cmdId}/result/{resultId} [put]
func (h *CommandHandler) UpdateCommandResult(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	resultID := chi.URLParam(r, "resultId")

	existing, err := h.repo.GetResultByID(r.Context(), commandID, resultID)
	if err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command result", "Failed to load command result")
		return
	}

	body, err := validateRawRequestBody(r, h.validator, contractvalidation.CommandResult)
	if err != nil {
		writeValidationError(w, r, err)
		return
	}
	result, err := decodeCommandResultPayload(bytes.NewReader(body))
	if err != nil {
		writeValidationError(w, r, err)
		return
	}

	result.ID = existing.ID
	result.CommandID = existing.CommandID
	if err := h.repo.UpdateResult(r.Context(), result); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command result", "Failed to update command result")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteCommandResult handles DELETE /commands/{id}/result/{resultId}
func (h *CommandHandler) DeleteCommandResult(w http.ResponseWriter, r *http.Request) {
	commandID := chi.URLParam(r, "cmdId")
	resultID := chi.URLParam(r, "resultId")

	if err := h.repo.DeleteResult(r.Context(), commandID, resultID); err != nil {
		writeRepositoryError(w, r, h.logger, err, "Command result", "Failed to delete command result")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func decodeCommandStatusPayload(reader io.Reader) (*domains.CommandStatusReport, error) {
	return json_formatters.DecodeCommandStatusReport(reader, false)
}

func decodeCommandResultPayload(reader io.Reader) (*domains.CommandResult, error) {
	var raw map[string]any
	if err := json.NewDecoder(reader).Decode(&raw); err != nil {
		return nil, err
	}
	for key := range raw {
		switch key {
		case "data", "observation@link", "observationSet@link", "datastream@link", "external@link":
		default:
			return nil, &common_shared.UnknownFieldError{Field: key}
		}
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
			return nil, errors.New("invalid command result payload")
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
