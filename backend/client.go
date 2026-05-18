package main

import "github.com/gorilla/websocket"

// Client represents a connected WebSocket client
type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan Message
}
