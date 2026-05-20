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
	MessageTypeHeartbeat    = "heartbeat"
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

	client := NewClient(conn)

	// Start auth timeout timer
	go func(hub *Hub, client *Client) {
		select {
		case <-time.After(10 * time.Second):
			if !client.isAuthenticated() {
				log.Printf("Auth timeout")

				msg := Message{
					Type: MessageTypeAuthResponse,
					Payload: map[string]interface{}{
						"success": false,
						"reason":  "auth timeout",
					},
				}

				hub.sendMessageToClient(client, msg)
				client.disconnect()
			}
		case <-client.ctx.Done():
			return
		}
	}(hub, client)

	// Writer goroutine: reads from client.Send channel and writes to WebSocket
	go func(hub *Hub, client *Client) {
		defer func() {
			hub.unregisterClient(client)
		}()

		for {
			select {
			case msg := <-client.send:
				err := conn.WriteJSON(msg)
				if err != nil {
					log.Println("Write error:", err)
					return
				}

			case <-client.ctx.Done():
				return
			}
		}
	}(hub, client)

	// Reader goroutine: reads from WebSocket and writes to hub.Broadcast
	go func(hub *Hub, client *Client) {
		defer func() {
			hub.unregisterClient(client)
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

			switch msg.Type {
			case MessageTypeAuth:
				handleAuth(hub, client, msg)
			case MessageTypeMessage:
				if !client.isAuthenticated() {
					log.Printf("Unauthenticated message attempt")
					continue
				}
				handleChatMessage(hub, client, msg)
			default:
				log.Printf("Unknown message type: %s", msg.Type)
			}
		}
	}(hub, client)
}

func handleAuth(hub *Hub, client *Client, msg Message) {
	// if already authenticated, ignore
	if client.isAuthenticated() {
		log.Printf("Client %s attempted to authenticate again", client.getID())
		return
	}
	
	token, _ := msg.Payload["token"].(string)
	clientID, _ := msg.Payload["clientId"].(string)
	clientType, _ := msg.Payload["clientType"].(string)

	// Validate token
	if token == "" {
		response := Message{
			Type: MessageTypeAuthResponse,
			Payload: map[string]interface{}{
				"success": false,
				"reason":  "missing token",
			},
		}
		// client not yet registered, send back response then close connection
		hub.sendMessageToClient(client, response)
		client.disconnect()
		return
	}
	// hardcode token for demo purposes
	if token != "secret-token" {
		response := Message{
			Type: MessageTypeAuthResponse,
			Payload: map[string]interface{}{
				"success": false,
				"reason":  "invalid token",
			},
		}

		// client not yet registered, send back response then close connection
		hub.sendMessageToClient(client, response)
		client.disconnect()
		return
	}

	// Session restore logic
	//generate new client ID if not provided (for new sessions)
	//TODO: also, use uuid instead of random int for better uniqueness
	if clientID == "" {
		clientID = generateID()
	}

	// use the function method to update several fields ofthe client struct  at once in a thread-safe way
	client.updateClient(
		func(c *Client) {
			c.authenticated = true
			c.token = token
			c.clientType = clientType
			c.id = clientID // ensure client ID is set (either from session restore or new)
		},
	)

	// register client with the hub after successful authentication
	hub.registerClient(client)

	response := Message{
		Type:      MessageTypeAuthResponse,
		ClientID:  client.getID(),
		Timestamp: getCurrentTimestamp(),
		Payload: map[string]interface{}{
			"success": true,
		},
	}

	hub.sendMessageToClient(client, response)
}

func handleChatMessage(hub *Hub, client *Client, msg Message) {
	// Add clientId and formatted timestamp
	msg.ClientID = client.getID()
	msg.Timestamp = getCurrentTimestamp()

	// Ensure recipientId defaults to "all"
	if msg.Payload == nil {
		msg.Payload = make(map[string]interface{})
	}
	if _, ok := msg.Payload["recipientId"]; !ok {
		msg.Payload["recipientId"] = "all"
	}

	hub.broadcastChatMessage(msg)
}

/* // sendMessage sends a message to the specified client
// send channel has queue buffer - if it is full, we can just drop the message and close the connection to avoid blocking the hub
func (c *Client) sendMessage(msg Message) {
	select {
	case c.Send <- msg:
	default:
		close(c.Send)
	}
} */

// startHeartbeat sends heartbeat messages to all clients every 10 seconds
func startHeartbeat(hub *Hub) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		heartbeatMsg := Message{
			Type:      MessageTypeHeartbeat,
			Timestamp: getCurrentTimestamp(),
		}

		hub.broadcastToAll(heartbeatMsg)
	}
}

func main() {
	hub := NewHub()
	//go hub.Run()
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
