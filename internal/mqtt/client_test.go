package mqtt

import (
	"testing"

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
