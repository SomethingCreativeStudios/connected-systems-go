package mqtt

import "strings"

// extractDatastreamID parses the datastream ID from a topic like "datastreams/{id}/observations".
func extractDatastreamID(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) >= 2 && parts[0] == "datastreams" {
		return parts[1]
	}
	return ""
}

// extractCommandStatusIDs parses control stream ID and command ID from a topic
// like "controls/{controlStreamId}/commands/{cmdId}/status".
func extractCommandStatusIDs(topic string) (controlStreamID, cmdID string) {
	parts := strings.Split(topic, "/")
	if len(parts) >= 5 && parts[0] == "controls" && parts[2] == "commands" && parts[4] == "status" {
		return parts[1], parts[3]
	}
	return "", ""
}
