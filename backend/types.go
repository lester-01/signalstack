package main

import "time"

// Message types exchanged over the WebSocket.
const (
	MessageTypeMessage      = "message"
	MessageTypeAuth         = "auth"
	MessageTypeAuthResponse = "auth_response"
	MessageTypeHeartbeat    = "heartbeat"
	MessageTypeJoin         = "join"
	MessageTypeLeave        = "leave"
)

// RecipientAll addresses a chat message to every connected client.
const RecipientAll = "all"

// Message is the JSON envelope for every WebSocket frame.
type Message struct {
	Type      string                 `json:"type"`
	ClientID  string                 `json:"clientId,omitempty"`
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}

// getCurrentTimestamp returns the server time as "03:04:05 PM".
func getCurrentTimestamp() string {
	return time.Now().Format("03:04:05 PM")
}
