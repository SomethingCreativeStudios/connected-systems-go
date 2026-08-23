package mqtt

import "fmt"

// Topic helpers for OGC Connected Systems Pub/Sub (AsyncAPI) MQTT topics.

// SystemEventTopic returns the Resource Data topic for a specific system's events.
// Pattern: systems/{systemId}/events:data
func SystemEventTopic(systemID string) string {
	return fmt.Sprintf("systems/%s/events:data", systemID)
}

// SystemEventsTopic returns the Resource Data topic for the canonical top-level
// system event collection.
// Pattern: systemEvents:data
func SystemEventsTopic() string {
	return "systemEvents:data"
}

// ObservationTopic returns the Resource Data topic for observations on a specific datastream.
// Pattern: datastreams/{dataStreamId}/observations:data
func ObservationTopic(datastreamID string) string {
	return fmt.Sprintf("datastreams/%s/observations:data", datastreamID)
}

// ObservationsWildcardTopic returns the wildcard subscription topic for all datastream observations.
// Pattern: datastreams/+/observations:data
func ObservationsWildcardTopic() string {
	return "datastreams/+/observations:data"
}

// CommandTopic returns the Resource Data topic for commands on a specific control stream.
// Pattern: controlstreams/{controlStreamId}/commands:data
func CommandTopic(controlStreamID string) string {
	return fmt.Sprintf("controlstreams/%s/commands:data", controlStreamID)
}

// CommandStatusTopic returns the topic for command status updates.
// Pattern: commands/{cmdId}/status:data
func CommandStatusTopic(cmdID string) string {
	return fmt.Sprintf("commands/%s/status:data", cmdID)
}

// CommandStatusWildcardTopic returns the wildcard subscription topic for all command status updates.
// Pattern: commands/+/status:data
func CommandStatusWildcardTopic() string {
	return "commands/+/status:data"
}
