package main

// Message represents the JSON envelope for all WebSocket messages
type Message struct {
	Type      string                 `json:"type"`
	ClientID  string                 `json:"clientId,omitempty"`
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}
