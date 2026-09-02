package main

import (
	"encoding/json"
)

// Report is intentionally both human-friendly in logs and machine-readable at
// the end of a run. It contains counts only, never endpoint credentials.
type Report struct {
	Mode      string         `json:"mode"`
	Namespace string         `json:"namespace"`
	Created   map[string]int `json:"created,omitempty"`
	Sent      int            `json:"sent,omitempty"`
	Failed    int            `json:"failed,omitempty"`
	Skipped   int            `json:"skipped,omitempty"`
}

func NewReport(mode, namespace string) Report {
	return Report{Mode: mode, Namespace: namespace, Created: make(map[string]int)}
}

func (r *Report) AddCreated(resource string) {
	r.Created[resource]++
}

func (r *Report) AddSent() {
	r.Sent++
}

func (r *Report) AddFailed() {
	r.Failed++
}

func (r *Report) AddSkipped() {
	r.Skipped++
}

func (r *Report) JSON() string {
	encoded, err := json.Marshal(r)
	if err != nil {
		return `{"error":"could not encode report"}`
	}
	return string(encoded)
}
