package pubsub

import (
	"testing"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/stretchr/testify/require"
)

func TestTopicCodecAndOwnershipTemplates(t *testing.T) {
	for _, id := range []string{"abc-_.~09", "a/b", "%2F", "a+b# c", "é漢字"} {
		escaped := EncodeID(id)
		decoded, err := DecodeID(escaped)
		require.NoError(t, err)
		require.Equal(t, id, decoded)
	}
	for _, id := range []string{"", "%61", "%2f", "%", "a+b", "a b", "%FF", "%00"} {
		_, err := DecodeID(id)
		require.Error(t, err, id)
	}
	target, err := ParseDataTopic("node1", "node1/systems/root/subsystems/a%2Fb/subsystems/%252F/datastreams/d%23/observations:data")
	require.NoError(t, err)
	require.Equal(t, "d#", target.ParentID)
	require.Equal(t, "/datastreams/d%23/observations", target.CanonicalCollection)
	require.Equal(t, "observation", target.Capability.ResourceType)
	target, err = ParseDataTopic("node1", "node1/systems:data/sml-json")
	require.NoError(t, err)
	require.Equal(t, mediatype.SensorML, target.MediaType)
	for _, topic := range []string{"systems:data", "node2/systems:data", "node1/systems:events", "node1/observations:data", "node1/systems:data/json", "node1/systems:data/geo-json", "node1/systems/%61/events:data", "node1/systems/x/events:data/foo/bar"} {
		_, err = ParseDataTopic("node1", topic)
		require.Error(t, err, topic)
	}
}
func TestPermissionsIncludeFutureResourcesAndRejectProhibitedTopics(t *testing.T) {
	cfg := config.PubSubConfig{ResourceData: config.PubSubFeatureConfig{Enabled: true}, ResourceEvents: config.PubSubFeatureConfig{Enabled: true}, BatchResourceEvents: config.BatchResourceEventsConfig{Enabled: true}}
	for _, filter := range []string{"node1/#", "node1/systems/+/subsystems/+/events:data", "node1/+/x", "#", "+/systems:data", "node1/systems:data/#"} {
		require.True(t, FilterOffered("node1", filter, cfg), filter)
	}
	for _, filter := range []string{"node2/#", "$share/group/node1/#", "node1/systems/#/x", "node1/observations:data", "node1/systems:events", "node1/systems:data/xml", "node1/systems/x/events+"} {
		require.False(t, FilterOffered("node1", filter, cfg), filter)
	}
	require.True(t, TopicOffered("node1", "node1/deployments/a/subdeployments/b/subdeployments/c", cfg))
	require.False(t, TopicOffered("node1", "node1/systems/x/datastreams/y/observations/z", cfg))
}
