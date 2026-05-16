package api

import (
	"errors"
	"net/http"
	"strings"

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
	fc                *formaters.MultiFormatFormatterCollection[*domains.Command]
}

func NewCommandHandler(
	cfg *config.Config,
	logger *zap.Logger,
	repo *repository.CommandRepository,
	controlStreamRepo *repository.ControlStreamRepository,
	fc *formaters.MultiFormatFormatterCollection[*domains.Command],
) *CommandHandler {
	return &CommandHandler{
		cfg:               cfg,
		logger:            logger,
		repo:              repo,
		controlStreamRepo: controlStreamRepo,
		fc:                fc,
	}
}

// ListCommands handles GET /commands
//
// @Summary     List commands
// @Description Returns a paginated collection of command resources
// @Tags        Commands
// @Produce     json
// @Param       limit   query  int  false  "Maximum number of results"
// @Param       offset  query  int  false  "Result offset"
// @Success     200  {object}  CommandCollectionResponse
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /commands [get]
func (h *CommandHandler) ListCommands(w http.ResponseWriter, r *http.Request) {
	params , err := queryparams.CommandsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
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
// @Param       controlStreamId  path   string  true   "Control Stream ID"
// @Param       limit            query  int     false  "Maximum number of results"
// @Param       offset           query  int     false  "Result offset"
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

	params , err := queryparams.CommandsQueryParams{}.BuildFromRequest(r, h.cfg.API.DefaultLimit)
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
