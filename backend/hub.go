package main

import "sync"

// Hub maintains active client connections and routes messages
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]bool
	// broadcast  chan Message
	// register   chan *Client
	// unregister chan *Client
}

// NewHub creates and returns a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]bool),
		// broadcast:  make(chan Message),
		// register:   make(chan *Client),
		// unregister: make(chan *Client),
	}
}

// getAllClientIDs returns a slice of client IDs for all connected clients
func (h *Hub) getAllClientIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clientIDs := make([]string, 0, len(h.clients))
	for client := range h.clients {
		clientIDs = append(clientIDs, client.getID())
	}
	return clientIDs
}

/* // getAllClientInfo returns a slice of ClientInfo for all connected clients
func (h *Hub) getAllClientInfo() []ClientInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clientInfoList := make([]ClientInfo, 0, len(h.Clients))
	for client := range h.Clients {
		clientInfoList = append(clientInfoList, client.getClientInfo())
	}
	return clientInfoList
} */

// returns copy of all clients - used for iterating over clients without holding lock
func (h *Hub) getClientsSnapshot() []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*Client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	return clients
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()
	h.broadcastJoin(client)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	_, ok := h.clients[client]
	if ok {
		delete(h.clients, client)
	}
	h.mu.Unlock()

	if ok {
		client.disconnect()
		h.broadcastLeave(client)
	}
}

func (h *Hub) broadcastChatMessage(msg Message) {
	// Check for recipient ID in payload
	recipientID, hasRecipient := msg.Payload["recipientId"].(string)
	isPrivate := hasRecipient && recipientID != "all"

	if isPrivate {
		h.sendMessageToRecipient(msg, recipientID)
	} else {
		h.broadcastToAll(msg)
	}
}

func (h *Hub) sendMessageToClient(client *Client, msg Message) {
	// keep using channels for sending, so we can use the buffer as a queue
	select {
	case client.send <- msg:
	default:
		h.unregisterClient(client)
	}
}

func (h *Hub) sendMessageToRecipient(msg Message, recipientID string) {
	senderID := msg.ClientID
	clientsSnapshot := h.getClientsSnapshot()
	for _, tmpClient := range clientsSnapshot {
		if tmpClient.id == recipientID || tmpClient.id == senderID {
			h.sendMessageToClient(tmpClient, msg)
		}
	}
}

func (h *Hub) broadcastToAll(msg Message) {
	clientsSnapshot := h.getClientsSnapshot()
	for _, client := range clientsSnapshot {
		h.sendMessageToClient(client, msg)
	}
}

func (h *Hub) broadcastJoin(client *Client) {
	clientIDs := h.getAllClientIDs()

	// Broadcast join message with client list and user count
	joinMsg := Message{
		Type:      "join",
		ClientID:  client.getID(),
		Timestamp: getCurrentTimestamp(),
		Payload: map[string]interface{}{
			"userCount":  len(clientIDs),
			"clientList": clientIDs,
		},
	}
	h.broadcastToAll(joinMsg)
}

func (h *Hub) broadcastLeave(client *Client) {
	clientIDs := h.getAllClientIDs()

	// Broadcast leave message with updated client list
	leaveMsg := Message{
		Type:      "leave",
		ClientID:  client.getID(),
		Timestamp: getCurrentTimestamp(),
		Payload: map[string]interface{}{
			"userCount":  len(clientIDs),
			"clientList": clientIDs,
		},
	}
	h.broadcastToAll(leaveMsg)
}

/* func handleAuthMessage(h *Hub, client *Client, msg Message) {
	// For demo purposes, we'll just mark the client as authenticated without checking credentials
	client.setAuthenticated(true)

	response := Message{
		Type:      MessageTypeAuthResponse,
		ClientID:  client.getID(),
		Timestamp: getCurrentTimestamp(),
		Payload: map[string]interface{}{
			"success": true,
		},
	}
	h.sendMessageToClient(client, response)
} */
