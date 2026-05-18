package main

// Hub maintains active client connections and routes messages
type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
}

// NewHub creates and returns a new Hub instance
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan Message),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop handling client registration and message routing
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true

			// Build client list
			clientList := make([]string, 0, len(h.Clients))
			for c := range h.Clients {
				clientList = append(clientList, c.ID)
			}

			// Broadcast join message with client list and user count
			joinMsg := Message{
				Type:      "join",
				ClientID:  client.ID,
				Timestamp: getCurrentTimestamp(),
				Payload: map[string]interface{}{
					"userCount":  len(h.Clients),
					"clientList": clientList,
				},
			}
			// Send join to all clients (including the new one)
			for c := range h.Clients {
				select {
				case c.Send <- joinMsg:
				default:
					close(c.Send)
					delete(h.Clients, c)
				}
			}

		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)

				// Build updated client list
				clientList := make([]string, 0, len(h.Clients))
				for c := range h.Clients {
					clientList = append(clientList, c.ID)
				}

				// Broadcast leave message with updated client list
				leaveMsg := Message{
					Type:      "leave",
					ClientID:  client.ID,
					Timestamp: getCurrentTimestamp(),
					Payload: map[string]interface{}{
						"userCount":  len(h.Clients),
						"clientList": clientList,
					},
				}
				// Send leave to all remaining clients
				for c := range h.Clients {
					select {
					case c.Send <- leaveMsg:
					default:
						close(c.Send)
						delete(h.Clients, c)
					}
				}
			}

		case msg := <-h.Broadcast:
			// Check for recipient ID in payload
			recipientID, hasRecipient := msg.Payload["recipientId"].(string)
			isPrivate := hasRecipient && recipientID != "all"

			if isPrivate {
				// Private message: send to recipient + sender only
				senderID := msg.ClientID
				for client := range h.Clients {
					if client.ID == recipientID || client.ID == senderID {
						select {
						case client.Send <- msg:
						default:
							close(client.Send)
							delete(h.Clients, client)
						}
					}
				}
			} else {
				// Broadcast message: send to all clients
				for client := range h.Clients {
					select {
					case client.Send <- msg:
					default:
						close(client.Send)
						delete(h.Clients, client)
					}
				}
			}
		}
	}
}
