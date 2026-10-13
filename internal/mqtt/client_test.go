package mqtt

import (
	"testing"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPublishedEchoTrackingConsumesOnlyMatchingMessages(t *testing.T) {
	manager := NewManager(Config{}, zap.NewNop())
	manager.rememberPublishedMessage("topic/a", []byte(`{"id":"one"}`))

	require.False(t, manager.consumePublishedEcho("topic/a", []byte(`{"id":"two"}`)))
	require.False(t, manager.consumePublishedEcho("topic/b", []byte(`{"id":"one"}`)))
	require.True(t, manager.consumePublishedEcho("topic/a", []byte(`{"id":"one"}`)))
	require.False(t, manager.consumePublishedEcho("topic/a", []byte(`{"id":"one"}`)))
}

func TestPublishedEchoTrackingCountsRepeatedPublications(t *testing.T) {
	manager := NewManager(Config{}, zap.NewNop())
	manager.rememberPublishedMessage("topic/a", []byte("same"))
	manager.rememberPublishedMessage("topic/a", []byte("same"))

	require.True(t, manager.consumePublishedEcho("topic/a", []byte("same")))
	require.True(t, manager.consumePublishedEcho("topic/a", []byte("same")))
	require.False(t, manager.consumePublishedEcho("topic/a", []byte("same")))
}

func TestTopicMatchesSubscriptionFilter(t *testing.T) {
	require.True(t, topicMatchesFilter("datastreams/ds-1/observations:data", "datastreams/+/observations:data"))
	require.True(t, topicMatchesFilter("a/b/c", "a/#"))
	require.False(t, topicMatchesFilter("datastreams/ds-1/observations:data/extra", "datastreams/+/observations:data"))
	require.False(t, topicMatchesFilter("controlstreams/cs-1/commands:data", "commands/+/status:data"))
}

// fakeToken is an already-completed paho token.
type fakeToken struct{ err error }

func (t fakeToken) Wait() bool                     { return true }
func (t fakeToken) WaitTimeout(time.Duration) bool { return true }
func (t fakeToken) Done() <-chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}
func (t fakeToken) Error() error { return t.err }

// fakeClient records subscriptions; every other paho method is unused here.
type fakeClient struct {
	paho.Client
	subscribed []string
}

func (c *fakeClient) IsConnected() bool { return true }
func (c *fakeClient) Subscribe(topic string, _ byte, _ paho.MessageHandler) paho.Token {
	c.subscribed = append(c.subscribed, topic)
	return fakeToken{}
}

// With a clean session the broker forgets subscriptions on reconnect, so the
// connect handler has to replay them or ingestion silently stops.
func TestReconnectRestoresSubscriptions(t *testing.T) {
	client := &fakeClient{}
	manager := NewManager(Config{}, zap.NewNop())
	manager.client = client

	noop := func(paho.Client, paho.Message) {}
	require.NoError(t, manager.Subscribe("datastreams/+/observations:data", noop))
	require.NoError(t, manager.Subscribe("commands/+/status:data", noop))
	client.subscribed = nil

	manager.onConnect(client)

	require.ElementsMatch(t, []string{"datastreams/+/observations:data", "commands/+/status:data"}, client.subscribed)
}
