package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Event is one SSE data frame sent to the client.
type Event struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	Model   string `json:"model,omitempty"`
	Message string `json:"message,omitempty"`
	Task    string `json:"task,omitempty"`
	Agent   string `json:"agent,omitempty"`
	Action  string `json:"action,omitempty"`
	Step    int    `json:"step,omitempty"`
}

// InitHeaders sets response headers for server-sent events.
func InitHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") //禁用 nginx proxy_buffering
}

// Write sends a single SSE data frame and flushes it.
func Write(w http.ResponseWriter, flusher http.Flusher, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
