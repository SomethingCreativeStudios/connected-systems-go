package mqtt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResourceDataTopicsFollowCanonicalRESTPaths(t *testing.T) {
	require.Equal(t, "systems/sys-1/events:data", SystemEventTopic("sys-1"))
	require.Equal(t, "systemEvents:data", SystemEventsTopic())
	require.Equal(t, "datastreams/ds-1/observations:data", ObservationTopic("ds-1"))
	require.Equal(t, "datastreams/+/observations:data", ObservationsWildcardTopic())
	require.Equal(t, "controlstreams/cs-1/commands:data", CommandTopic("cs-1"))
	require.Equal(t, "commands/cmd-1/status:data", CommandStatusTopic("cmd-1"))
	require.Equal(t, "commands/+/status:data", CommandStatusWildcardTopic())
}

func TestResourceDataTopicParsingRejectsLegacyTopics(t *testing.T) {
	require.Equal(t, "ds-1", extractDatastreamID("datastreams/ds-1/observations:data"))
	require.Empty(t, extractDatastreamID("datastreams/ds-1/observations"))
	require.Equal(t, "cmd-1", extractCommandStatusID("commands/cmd-1/status:data"))
	require.Empty(t, extractCommandStatusID("controls/cs-1/commands/cmd-1/status"))
}
