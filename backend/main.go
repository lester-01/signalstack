package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	MessageTypeMessage      = "message"
	MessageTypeAuth         = "auth"
	MessageTypeAuthResponse = "auth_response"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// generateID creates a unique client ID
func generateID() string {
	return fmt.Sprintf("c-%d", rand.Intn(1000000))
}

// getCurrentTimestamp returns current time in human-readable format HH:MM:SS AM/PM
func getCurrentTimestamp() string {
	return time.Now().Format("03:04:05 PM")
}

// serveWs handles WebSocket connections
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	//TODO: if clientId is provided in Message payload, use that instead of generating a new one.
	//TODO: also, use uuid instead of random int for better uniqueness
	client := &Client{
		ID:   generateID(),
		Conn: conn,
		Send: make(chan Message, 10),
	}

	hub.Register <- client

	// Start auth timeout timer
	go func(client *Client) {
		time.Sleep(10 * time.Second)

		if !client.Authenticated {
			log.Printf("Auth timeout")

			client.Conn.WriteJSON(Message{
				Type: "auth_response",
				Payload: map[string]interface{}{
					"success": false,
					"reason":  "auth timeout",
				},
			})

			client.Conn.Close()
		}
	}(client)

	// Writer goroutine: reads from client.Send channel and writes to WebSocket
	go func(client *Client) {
		defer func() {
			conn.Close()
		}()
		for msg := range client.Send {
			err := conn.WriteJSON(msg)
			if err != nil {
				log.Println("Write error:", err)
				return
			}
		}
	}(client)

	// Reader goroutine: reads from WebSocket and writes to hub.Broadcast
	go func(client *Client) {
		defer func() {
			hub.Unregister <- client
			conn.Close()
		}()

		for {
			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				break
			}

			//
			switch msg.Type {
			case "auth":
				handleAuth(client, msg)
			case "message":
				if !client.Authenticated {
					log.Printf("Unauthenticated message attempt")
					continue
				}

				handleChatMessage(hub, client, msg)
			default:
				log.Printf("Unknown message type: %s", msg.Type)
			}
			//

		}
	}(client)
}

func handleAuth(client *Client, msg Message) {
	token, _ := msg.Payload["token"].(string)
	clientID, _ := msg.Payload["clientId"].(string)
	clientType, _ := msg.Payload["clientType"].(string)

	// Validate token
	if token == "" {
		response := Message{
			Type: "auth_response",
			Payload: map[string]interface{}{
				"success": false,
				"reason":  "missing token",
			},
		}

		client.Conn.WriteJSON(response)
		client.Conn.Close()
		return
	}
	// hardcode token for demo purposes
	if token != "secret-token" {
		response := Message{
			Type: "auth_response",
			Payload: map[string]interface{}{
				"success": false,
				"reason":  "invalid token",
			},
		}

		client.Conn.WriteJSON(response)
		client.Conn.Close()
		return
	}

	// Session restore logic
	if clientID != "" {
		client.ID = clientID
	}

	client.Authenticated = true
	client.Token = token
	client.ClientType = clientType

	response := Message{
		Type:      "auth_response",
		ClientID:  client.ID,
		Timestamp: getCurrentTimestamp(),
		Payload: map[string]interface{}{
			"success": true,
		},
	}

	client.Conn.WriteJSON(response)
}

func handleChatMessage(hub *Hub, client *Client, msg Message) {
	// Add clientId and formatted timestamp
	msg.ClientID = client.ID
	msg.Timestamp = getCurrentTimestamp()

	// Ensure recipientId defaults to "all"
	if msg.Payload == nil {
		msg.Payload = make(map[string]interface{})
	}
	if _, ok := msg.Payload["recipientId"]; !ok {
		msg.Payload["recipientId"] = "all"
	}

	hub.Broadcast <- msg
}

// startHeartbeat sends heartbeat messages to all clients every 10 seconds
func startHeartbeat(hub *Hub) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Build current client list
		clientList := make([]string, 0, len(hub.Clients))
		for client := range hub.Clients {
			clientList = append(clientList, client.ID)
		}

		heartbeatMsg := Message{
			Type:      "heartbeat",
			Timestamp: getCurrentTimestamp(),
			Payload: map[string]interface{}{
				"userCount":  len(hub.Clients),
				"clientList": clientList,
			},
		}

		hub.Broadcast <- heartbeatMsg
	}
}

func main() {
	hub := NewHub()
	go hub.Run()
	go startHeartbeat(hub)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Println("WebSocket server starting on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe error:", err)
	}
}
