package mqtt

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/model/formaters/json_formatters"
	"github.com/yourusername/connected-systems-go/internal/pubsub"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"github.com/yourusername/connected-systems-go/internal/resourcevalidation"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type datastreamStore interface {
	GetByID(id string) (*domains.Datastream, error)
}

type observationStore interface {
	Create(observation *domains.Observation) error
	GetByID(id string) (*domains.Observation, error)
	Update(observation *domains.Observation) error
}

type commandStore interface {
	GetByID(id string) (*domains.Command, error)
	CreateStatus(status *domains.CommandStatusReport) error
	GetStatusByID(commandID, statusID string) (*domains.CommandStatusReport, error)
	UpdateStatus(status *domains.CommandStatusReport) error
}

// IngestionHandlers bridges complete Pub/Sub Resource Data Messages to
// validated database writes and Resource Event notifications.
type IngestionHandlers struct {
	logger               *zap.Logger
	datastream           datastreamStore
	observation          observationStore
	command              commandStore
	pubSubPublisher      *pubsub.Publisher
	observationFormatter *json_formatters.ObservationJSONFormatter
}

// NewIngestionHandlers creates ingestion handlers backed by the given repositories.
func NewIngestionHandlers(logger *zap.Logger, repos *repository.Repositories, publisher *pubsub.Publisher) *IngestionHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	return newIngestionHandlers(logger, repos.Datastream, repos.Observation, repos.Command, publisher)
}

func newIngestionHandlers(
	logger *zap.Logger,
	datastream datastreamStore,
	observation observationStore,
	command commandStore,
	publisher *pubsub.Publisher,
) *IngestionHandlers {
	return &IngestionHandlers{
		logger:               logger,
		datastream:           datastream,
		observation:          observation,
		command:              command,
		pubSubPublisher:      publisher,
		observationFormatter: json_formatters.NewObservationJSONFormatter(nil),
	}
}

// HandleObservation handles a complete Observation Resource Data Message from
// datastreams/{id}/observations:data. A new ID creates the resource; an existing ID
// updates it. Partial create payloads are intentionally rejected.
func (h *IngestionHandlers) HandleObservation(topic string, payload []byte) {
	if err := h.ingestObservation(topic, payload); err != nil {
		h.logger.Warn("Rejected observation Resource Data Message",
			zap.String("topic", topic),
			zap.Error(err),
		)
	}
}

func (h *IngestionHandlers) ingestObservation(topic string, payload []byte) error {
	datastreamID := extractDatastreamID(topic)
	if datastreamID == "" {
		return fmt.Errorf("topic must match datastreams/{datastreamId}/observations:data")
	}

	datastream, err := h.datastream.GetByID(datastreamID)
	if err != nil {
		return fmt.Errorf("load datastream %q: %w", datastreamID, err)
	}

	observation, err := h.observationFormatter.Deserialize(nil, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("decode complete observation: %w", err)
	}
	if observation.ID == "" {
		return fmt.Errorf("id is required for a complete Resource Data Message")
	}
	if observation.DatastreamID == "" {
		return fmt.Errorf("datastream@id is required for a complete Resource Data Message")
	}
	if observation.DatastreamID != datastreamID {
		return fmt.Errorf("datastream@id %q does not match topic datastream %q", observation.DatastreamID, datastreamID)
	}
	if err := resourcevalidation.ValidateObservationAgainstDatastreamSchema(observation, datastream, json_formatters.JSONContentType); err != nil {
		return fmt.Errorf("observation result does not match datastream schema: %w", err)
	}

	operation := pubsub.OperationCreate
	existing, err := h.observation.GetByID(observation.ID)
	switch {
	case err == nil:
		if existing.DatastreamID != datastreamID {
			return fmt.Errorf("observation %q belongs to datastream %q", observation.ID, existing.DatastreamID)
		}
		if err := h.observation.Update(observation); err != nil {
			return fmt.Errorf("update observation %q: %w", observation.ID, err)
		}
		operation = pubsub.OperationUpdate
	case isNotFound(err):
		if err := h.observation.Create(observation); err != nil {
			return fmt.Errorf("create observation %q: %w", observation.ID, err)
		}
	default:
		return fmt.Errorf("look up observation %q: %w", observation.ID, err)
	}

	if h.pubSubPublisher != nil {
		h.pubSubPublisher.PublishChange(pubsub.Change{
			ResourceType:   "observation",
			ResourceID:     observation.ID,
			Operation:      operation,
			SubjectPath:    "/observations/" + observation.ID,
			ParentPath:     "/datastreams/" + datastreamID,
			CollectionPath: "/datastreams/" + datastreamID + "/observations",
		})
	}
	h.logger.Debug("Observation Resource Data Message ingested",
		zap.String("datastreamId", datastreamID),
		zap.String("observationId", observation.ID),
		zap.String("operation", string(operation)),
	)
	return nil
}

// HandleCommandStatus handles a complete CommandStatus Resource Data Message
// from commands/{cmdId}/status:data.
func (h *IngestionHandlers) HandleCommandStatus(topic string, payload []byte) {
	if err := h.ingestCommandStatus(topic, payload); err != nil {
		h.logger.Warn("Rejected command status Resource Data Message",
			zap.String("topic", topic),
			zap.Error(err),
		)
	}
}

func (h *IngestionHandlers) ingestCommandStatus(topic string, payload []byte) error {
	commandID := extractCommandStatusID(topic)
	if commandID == "" {
		return fmt.Errorf("topic must match commands/{cmdId}/status:data")
	}

	command, err := h.command.GetByID(commandID)
	if err != nil {
		return fmt.Errorf("load command %q: %w", commandID, err)
	}
	status, err := json_formatters.DecodeCommandStatusReport(bytes.NewReader(payload), true)
	if err != nil {
		return fmt.Errorf("decode complete command status: %w", err)
	}
	if status.CommandID != commandID {
		return fmt.Errorf("command@id %q does not match topic command %q", status.CommandID, commandID)
	}

	operation := pubsub.OperationCreate
	_, err = h.command.GetStatusByID(commandID, status.ID)
	switch {
	case err == nil:
		if err := h.command.UpdateStatus(status); err != nil {
			return fmt.Errorf("update command status %q: %w", status.ID, err)
		}
		operation = pubsub.OperationUpdate
	case isNotFound(err):
		if err := h.command.CreateStatus(status); err != nil {
			return fmt.Errorf("create command status %q: %w", status.ID, err)
		}
	default:
		return fmt.Errorf("look up command status %q: %w", status.ID, err)
	}

	if h.pubSubPublisher != nil {
		h.pubSubPublisher.PublishChange(pubsub.Change{
			ResourceType:   "commandstatus",
			ResourceID:     status.ID,
			Operation:      operation,
			SubjectPath:    "/commands/" + commandID + "/status/" + status.ID,
			ParentPath:     "/commands/" + commandID,
			CollectionPath: "/commands/" + commandID + "/status",
		})
	}
	h.logger.Debug("Command status Resource Data Message ingested",
		zap.String("controlStreamId", command.ControlStreamID),
		zap.String("cmdId", commandID),
		zap.String("statusId", status.ID),
		zap.String("operation", string(operation)),
	)
	return nil
}

func isNotFound(err error) bool {
	return errors.Is(err, repository.ErrNotFound) || errors.Is(err, gorm.ErrRecordNotFound)
}
