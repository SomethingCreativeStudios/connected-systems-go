package mqtt

import (
	"encoding/json"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// IngestionHandlers bridges MQTT messages to database writes.
type IngestionHandlers struct {
	logger      *zap.Logger
	observation *repository.ObservationRepository
	command     *repository.CommandRepository
}

// NewIngestionHandlers creates ingestion handlers backed by the given repositories.
func NewIngestionHandlers(logger *zap.Logger, repos *repository.Repositories) *IngestionHandlers {
	return &IngestionHandlers{
		logger:      logger,
		observation: repos.Observation,
		command:     repos.Command,
	}
}

// HandleObservation is an MQTT message handler for datastreams/+/observations.
// It parses the datastream ID from the topic, unmarshals the payload as an Observation,
// and persists it to the database.
func (h *IngestionHandlers) HandleObservation(topic string, payload []byte) {
	datastreamID := extractDatastreamID(topic)
	if datastreamID == "" {
		h.logger.Warn("Could not extract datastream ID from MQTT topic", zap.String("topic", topic))
		return
	}

	var obs domains.Observation
	if err := json.Unmarshal(payload, &obs); err != nil {
		h.logger.Error("Failed to unmarshal observation from MQTT",
			zap.String("topic", topic),
			zap.Error(err),
		)
		return
	}

	obs.DatastreamID = datastreamID
	if err := h.observation.Create(&obs); err != nil {
		h.logger.Error("Failed to persist observation from MQTT",
			zap.String("datastreamId", datastreamID),
			zap.Error(err),
		)
		return
	}

	h.logger.Debug("Observation ingested via MQTT",
		zap.String("datastreamId", datastreamID),
		zap.String("observationId", obs.ID),
	)
}

// HandleCommandStatus is an MQTT message handler for controls/+/commands/+/status.
// It parses the control stream ID and command ID from the topic, unmarshals the
// payload as a command status update, and updates the command in the database.
func (h *IngestionHandlers) HandleCommandStatus(topic string, payload []byte) {
	controlStreamID, cmdID := extractCommandStatusIDs(topic)
	if controlStreamID == "" || cmdID == "" {
		h.logger.Warn("Could not extract IDs from MQTT command status topic", zap.String("topic", topic))
		return
	}

	var statusUpdate struct {
		CurrentStatus domains.CommandStatus `json:"currentStatus"`
	}
	if err := json.Unmarshal(payload, &statusUpdate); err != nil {
		h.logger.Error("Failed to unmarshal command status from MQTT",
			zap.String("topic", topic),
			zap.Error(err),
		)
		return
	}

	if statusUpdate.CurrentStatus == "" {
		h.logger.Warn("Command status update missing currentStatus", zap.String("topic", topic))
		return
	}

	cmd, err := h.command.GetByID(cmdID)
	if err != nil {
		h.logger.Error("Command not found for MQTT status update",
			zap.String("cmdId", cmdID),
			zap.Error(err),
		)
		return
	}

	cmd.CurrentStatus = statusUpdate.CurrentStatus
	if err := h.command.Update(cmd); err != nil {
		h.logger.Error("Failed to update command status from MQTT",
			zap.String("cmdId", cmdID),
			zap.Error(err),
		)
		return
	}

	h.logger.Debug("Command status updated via MQTT",
		zap.String("cmdId", cmdID),
		zap.String("status", string(statusUpdate.CurrentStatus)),
	)
}
