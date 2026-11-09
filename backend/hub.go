package main

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"
)

const heartbeatInterval = 10 * time.Second

// Hub tracks connected clients and routes messages between them.
//
// All state is guarded by mu. Sends never block: each client has a buffered
// queue, and a client whose queue is full is evicted.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	byID    map[string]*Client
	closed  bool
}

// NewHub returns an empty Hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		byID:    make(map[string]*Client),
	}
}

// register adds an authenticated client and announces it. If another
// connection already holds the same ID (a reconnect that raced the old
// connection's teardown), the old connection is replaced. It reports false if
// the hub is shutting down or the client has already been disconnected.
func (h *Hub) register(c *Client) bool {
	id := c.getID()

	h.mu.Lock()
	if h.closed || c.ctx.Err() != nil {
		h.mu.Unlock()
		return false
	}
	old := h.byID[id]
	if old != nil {
		delete(h.clients, old)
	}
	h.clients[c] = struct{}{}
	h.byID[id] = c
	h.mu.Unlock()

	if old != nil {
		old.disconnect()
	}
	h.broadcastJoin(id)
	return true
}

// unregister removes a client and tears down its connection. It is safe to
// call repeatedly and for clients that never registered.
func (h *Hub) unregister(c *Client) {
	id := c.getID()

	h.mu.Lock()
	_, ok := h.clients[c]
	if ok {
		delete(h.clients, c)
		if h.byID[id] == c {
			delete(h.byID, id)
		}
	}
	closed := h.closed
	h.mu.Unlock()

	// Always release the connection, even if the client never finished auth.
	c.disconnect()

	if ok && !closed {
		h.broadcastLeave(id)
	}
}

// Close disconnects every client gracefully and stops accepting new ones.
func (h *Hub) Close() {
	h.mu.Lock()
	h.closed = true
	clients := h.snapshotLocked()
	h.mu.Unlock()

	for _, c := range clients {
		c.disconnect()
	}
}

func (h *Hub) snapshotLocked() []*Client {
	clients := make([]*Client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	return clients
}

// snapshot returns a copy of the client set so callers can iterate without
// holding the lock.
func (h *Hub) snapshot() []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.snapshotLocked()
}

// clientIDs returns the IDs of all connected clients, sorted.
func (h *Hub) clientIDs() []string {
	h.mu.RLock()
	ids := make([]string, 0, len(h.byID))
	for id := range h.byID {
		ids = append(ids, id)
	}
	h.mu.RUnlock()
	sort.Strings(ids)
	return ids
}

// deliver queues msg for c, evicting c if its queue is full.
func (h *Hub) deliver(c *Client, msg Message) {
	if c.trySend(msg) {
		return
	}
	h.evict(c)
}

// evict drops a client that cannot keep up. Unregistering happens on its own
// goroutine so that a burst of evictions cannot recurse through broadcasts.
func (h *Hub) evict(c *Client) {
	if !c.evicted.CompareAndSwap(false, true) {
		return
	}
	log.Printf("evicting slow client %s", c.getID())
	c.forceClose()
	go h.unregister(c)
}

// broadcast sends msg to every connected client.
func (h *Hub) broadcast(msg Message) {
	for _, c := range h.snapshot() {
		h.deliver(c, msg)
	}
}

// routeChat delivers a chat message either to everyone or, for a private
// message, to the sender and the recipient only.
func (h *Hub) routeChat(sender *Client, msg Message) {
	recipient, _ := msg.Payload["recipientId"].(string)
	if recipient == "" || recipient == RecipientAll {
		h.broadcast(msg)
		return
	}

	h.deliver(sender, msg)
	if recipient == sender.getID() {
		return
	}
	h.mu.RLock()
	target := h.byID[recipient]
	h.mu.RUnlock()
	if target != nil {
		h.deliver(target, msg)
	}
}

func (h *Hub) broadcastJoin(id string) {
	h.broadcastPresence(MessageTypeJoin, id)
}

func (h *Hub) broadcastLeave(id string) {
	h.broadcastPresence(MessageTypeLeave, id)
}

func (h *Hub) broadcastPresence(msgType, id string) {
	ids := h.clientIDs()
	h.broadcast(Message{
		Type:      msgType,
		ClientID:  id,
		Timestamp: getCurrentTimestamp(),
		Payload: map[string]interface{}{
			"userCount":  len(ids),
			"clientList": ids,
		},
	})
}

// runHeartbeat broadcasts a heartbeat every interval until ctx is cancelled.
func (h *Hub) runHeartbeat(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			h.broadcast(Message{Type: MessageTypeHeartbeat, Timestamp: getCurrentTimestamp()})
		case <-ctx.Done():
			return
		}
	}
}
