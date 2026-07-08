package mqtt

import "fmt"

// Topic helpers for OGC Connected Systems Part 3 (AsyncAPI) MQTT topics.

// SystemEventTopic returns the topic for a specific system's events.
// Pattern: systems/{systemId}/events
func SystemEventTopic(systemID string) string {
	return fmt.Sprintf("systems/%s/events", systemID)
}

// SystemEventsTopic returns the wildcard topic for all system events.
// Pattern: systems/events
func SystemEventsTopic() string {
	return "systems/events"
}

// ObservationTopic returns the topic for observations on a specific datastream.
// Pattern: datastreams/{dataStreamId}/observations
func ObservationTopic(datastreamID string) string {
	return fmt.Sprintf("datastreams/%s/observations", datastreamID)
}

// ObservationsWildcardTopic returns the wildcard subscription topic for all datastream observations.
// Pattern: datastreams/+/observations
func ObservationsWildcardTopic() string {
	return "datastreams/+/observations"
}

// CommandTopic returns the topic for commands on a specific control stream.
// Pattern: controls/{controlStreamId}/commands
func CommandTopic(controlStreamID string) string {
	return fmt.Sprintf("controls/%s/commands", controlStreamID)
}

// CommandStatusTopic returns the topic for command status updates.
// Pattern: controls/{controlStreamId}/commands/{cmdId}/status
func CommandStatusTopic(controlStreamID, cmdID string) string {
	return fmt.Sprintf("controls/%s/commands/%s/status", controlStreamID, cmdID)
}

// CommandStatusWildcardTopic returns the wildcard subscription topic for all command status updates.
// Pattern: controls/+/commands/+/status
func CommandStatusWildcardTopic() string {
	return "controls/+/commands/+/status"
}
