package mqtt

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yourusername/connected-systems-go/internal/config"
	"github.com/yourusername/connected-systems-go/internal/model/domains"
	"github.com/yourusername/connected-systems-go/internal/pubsub"
	"github.com/yourusername/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

type fakeDatastreamStore struct {
	datastream *domains.Datastream
}

func (s *fakeDatastreamStore) GetByID(id string) (*domains.Datastream, error) {
	if s.datastream == nil || s.datastream.ID != id {
		return nil, repository.ErrNotFound
	}
	return s.datastream, nil
}

type fakeObservationStore struct {
	items   map[string]*domains.Observation
	creates int
	updates int
}

func (s *fakeObservationStore) GetByID(id string) (*domains.Observation, error) {
	observation, exists := s.items[id]
	if !exists {
		return nil, repository.ErrNotFound
	}
	return observation, nil
}

func (s *fakeObservationStore) Create(observation *domains.Observation) error {
	s.creates++
	copy := *observation
	s.items[observation.ID] = &copy
	return nil
}

func (s *fakeObservationStore) Update(observation *domains.Observation) error {
	s.updates++
	copy := *observation
	s.items[observation.ID] = &copy
	return nil
}

type fakeCommandStore struct {
	command *domains.Command
	status  map[string]*domains.CommandStatusReport
	creates int
	updates int
}

func (s *fakeCommandStore) GetByID(id string) (*domains.Command, error) {
	if s.command == nil || s.command.ID != id {
		return nil, repository.ErrNotFound
	}
	return s.command, nil
}

func (s *fakeCommandStore) GetStatusByID(commandID, statusID string) (*domains.CommandStatusReport, error) {
	status, exists := s.status[statusID]
	if !exists || status.CommandID != commandID {
		return nil, repository.ErrNotFound
	}
	return status, nil
}

func (s *fakeCommandStore) CreateStatus(status *domains.CommandStatusReport) error {
	s.creates++
	copy := *status
	s.status[status.ID] = &copy
	return nil
}

func (s *fakeCommandStore) UpdateStatus(status *domains.CommandStatusReport) error {
	s.updates++
	copy := *status
	s.status[status.ID] = &copy
	return nil
}

type capturedTransport struct {
	messages []capturedMessage
}

type capturedMessage struct {
	topic   string
	payload []byte
}

func (*capturedTransport) IsConnected() bool { return true }

func (t *capturedTransport) Publish(topic string, payload []byte) {
	t.messages = append(t.messages, capturedMessage{topic: topic, payload: payload})
}

func newIngestionTestPublisher(transport *capturedTransport) *pubsub.Publisher {
	return pubsub.NewPublisher("https://example.test/api", config.PubSubConfig{
		ResourceEvents:      config.PubSubFeatureConfig{Enabled: true},
		BatchResourceEvents: config.BatchResourceEventsConfig{Enabled: true, Window: time.Minute},
	}, transport, zap.NewNop())
}

func TestIngestObservationCreatesThenUpdatesCompleteResource(t *testing.T) {
	datastreams := &fakeDatastreamStore{datastream: &domains.Datastream{Base: domains.Base{ID: "ds-1"}}}
	observations := &fakeObservationStore{items: map[string]*domains.Observation{}}
	commands := &fakeCommandStore{status: map[string]*domains.CommandStatusReport{}}
	transport := &capturedTransport{}
	handler := newIngestionHandlers(zap.NewNop(), datastreams, observations, commands, newIngestionTestPublisher(transport))
	payload := []byte(`{"id":"obs-1","datastream@id":"ds-1","resultTime":"2026-08-22T12:00:00Z","result":12.5}`)

	require.NoError(t, handler.ingestObservation("datastreams/ds-1/observations:data", payload))
	require.NoError(t, handler.ingestObservation("datastreams/ds-1/observations:data", payload))
	handler.pubSubPublisher.Close()
	require.Equal(t, 1, observations.creates)
	require.Equal(t, 1, observations.updates)
	require.Len(t, transport.messages, 2)

	var createEvent, updateEvent pubsub.CloudEvent
	require.NoError(t, json.Unmarshal(transport.messages[0].payload, &createEvent))
	require.NoError(t, json.Unmarshal(transport.messages[1].payload, &updateEvent))
	require.Equal(t, "org.ogc.api.consys.observation.create", createEvent.Type)
	require.Equal(t, "org.ogc.api.consys.observation.update", updateEvent.Type)
	require.Equal(t, "datastreams/ds-1/observations:batch-events", transport.messages[0].topic)
}

func TestIngestObservationRejectsIncompleteOrWrongParent(t *testing.T) {
	datastreams := &fakeDatastreamStore{datastream: &domains.Datastream{Base: domains.Base{ID: "ds-1"}}}
	observations := &fakeObservationStore{items: map[string]*domains.Observation{}}
	handler := newIngestionHandlers(zap.NewNop(), datastreams, observations, &fakeCommandStore{}, nil)

	err := handler.ingestObservation("datastreams/ds-1/observations:data", []byte(`{"resultTime":"2026-08-22T12:00:00Z","result":1}`))
	require.ErrorContains(t, err, "id is required")

	err = handler.ingestObservation("datastreams/ds-1/observations:data", []byte(`{"id":"obs-1","datastream@id":"ds-2","resultTime":"2026-08-22T12:00:00Z","result":1}`))
	require.ErrorContains(t, err, "does not match topic datastream")
	require.Zero(t, observations.creates)
}

func TestIngestObservationValidatesDatastreamSchema(t *testing.T) {
	datastreams := &fakeDatastreamStore{datastream: &domains.Datastream{
		Base: domains.Base{ID: "ds-1"},
		Schema: &domains.DatastreamSchema{
			ObsFormat:    "application/json",
			ResultSchema: &domains.DatastreamDataComponent{Type: "Quantity"},
		},
	}}
	observations := &fakeObservationStore{items: map[string]*domains.Observation{}}
	handler := newIngestionHandlers(zap.NewNop(), datastreams, observations, &fakeCommandStore{}, nil)

	err := handler.ingestObservation("datastreams/ds-1/observations:data", []byte(`{"id":"obs-1","datastream@id":"ds-1","resultTime":"2026-08-22T12:00:00Z","result":"not-a-number"}`))
	require.ErrorContains(t, err, "must be a number")
	require.Zero(t, observations.creates)
}

func TestIngestCommandStatusPersistsFullResourceAndPublishesEvent(t *testing.T) {
	commands := &fakeCommandStore{
		command: &domains.Command{Base: domains.Base{ID: "cmd-1"}, ControlStreamID: "cs-1"},
		status:  map[string]*domains.CommandStatusReport{},
	}
	transport := &capturedTransport{}
	handler := newIngestionHandlers(zap.NewNop(), &fakeDatastreamStore{}, &fakeObservationStore{}, commands, newIngestionTestPublisher(transport))
	payload := []byte(`{"id":"status-1","command@id":"cmd-1","reportTime":"2026-08-22T12:00:00Z","statusCode":"EXECUTING"}`)

	require.NoError(t, handler.ingestCommandStatus("commands/cmd-1/status:data", payload))
	require.Equal(t, 1, commands.creates)
	require.Len(t, transport.messages, 2)

	var event pubsub.CloudEvent
	require.NoError(t, json.Unmarshal(transport.messages[0].payload, &event))
	require.Equal(t, "org.ogc.api.consys.commandstatus.create", event.Type)
}

func TestIngestCommandStatusRejectsLegacyPartialPayload(t *testing.T) {
	commands := &fakeCommandStore{
		command: &domains.Command{Base: domains.Base{ID: "cmd-1"}, ControlStreamID: "cs-1"},
		status:  map[string]*domains.CommandStatusReport{},
	}
	handler := newIngestionHandlers(zap.NewNop(), &fakeDatastreamStore{}, &fakeObservationStore{}, commands, nil)

	err := handler.ingestCommandStatus("commands/cmd-1/status:data", []byte(`{"currentStatus":"COMPLETED"}`))
	require.ErrorContains(t, err, "statusCode is required")
	require.Zero(t, commands.creates)
}
