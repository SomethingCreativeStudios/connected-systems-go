package mqtt

import "strings"

// extractDatastreamID parses the datastream ID from a topic like
// "datastreams/{id}/observations:data".
func extractDatastreamID(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) == 3 && parts[0] == "datastreams" && parts[1] != "" && parts[2] == "observations:data" {
		return parts[1]
	}
	return ""
}

// extractCommandStatusID parses the command ID from a topic like
// "commands/{cmdId}/status:data".
func extractCommandStatusID(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) == 3 && parts[0] == "commands" && parts[1] != "" && parts[2] == "status:data" {
		return parts[1]
	}
	return ""
}
