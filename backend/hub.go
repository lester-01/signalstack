package main

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"
)

const (
	heartbeatInterval = 10 * time.Second
	// presenceInterval is how long join/leave changes are collected before one
	// batched presence message is sent. Without batching, N clients leaving at
	// once would send N events to each of N clients.
	presenceInterval = 100 * time.Millisecond
)

// Hub tracks connected clients and routes messages between them.
//
// All state is guarded by mu. Sends never block: each client has a buffered
// queue, and a client whose queue is full is evicted.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	byID    map[string]*Client
	closed  bool

	// Clients whose online state changed since the last presence flush.
	pmu            sync.Mutex
	dirty          map[string]struct{}
	flushScheduled bool

	metrics *Metrics
}

// NewHub returns an empty Hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		byID:    make(map[string]*Client),
		dirty:   make(map[string]struct{}),
		metrics: &Metrics{},
	}
}

// register adds an authenticated client and announces it. welcome builds the
// first message the client receives; it is given the sorted IDs of everyone
// online (including the client). If another connection already holds the same
// ID (a reconnect that raced the old connection's teardown), the old
// connection is replaced. It reports false if the hub is shutting down or the
// client has already been disconnected.
func (h *Hub) register(c *Client, welcome func(clientList []string) Message) bool {
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
	// The welcome message carries a snapshot of who is online and is queued
	// while the lock is held. Broadcasts can only see c after we unlock, so
	// every presence update c receives afterwards is a delta against this list.
	c.trySend(welcome(h.idsLocked()))
	h.mu.Unlock()

	h.metrics.connectionsTotal.Add(1)
	if old != nil {
		old.disconnect()
	}
	h.notePresence(id)
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
		h.notePresence(id)
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

func (h *Hub) idsLocked() []string {
	ids := make([]string, 0, len(h.byID))
	for id := range h.byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// clientIDs returns the IDs of all connected clients, sorted.
func (h *Hub) clientIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.idsLocked()
}

func (h *Hub) clientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.byID)
}

// deliver queues msg for c, evicting c if its queue is full.
func (h *Hub) deliver(c *Client, msg Message) {
	if c.trySend(msg) {
		if msg.Type == MessageTypeMessage {
			h.metrics.messagesDelivered.Add(1)
		}
		return
	}
	if c.ctx.Err() != nil {
		return // already disconnecting; it is not "slow", there is just nobody to deliver to
	}
	h.evict(c)
}

// evict drops a client that cannot keep up. Unregistering happens on its own
// goroutine so that a burst of evictions cannot recurse through broadcasts.
func (h *Hub) evict(c *Client) {
	if !c.evicted.CompareAndSwap(false, true) {
		return
	}
	h.metrics.evictions.Add(1)
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

// notePresence records that id joined or left and schedules a batched
// presence message. Only the fact that the state changed is recorded; the
// message reports each client's actual state at flush time, so the order in
// which changes were noted cannot matter.
func (h *Hub) notePresence(id string) {
	h.pmu.Lock()
	defer h.pmu.Unlock()
	h.dirty[id] = struct{}{}
	if !h.flushScheduled {
		h.flushScheduled = true
		time.AfterFunc(presenceInterval, h.flushPresence)
	}
}

// flushPresence broadcasts one presence message covering every change since
// the last flush. A client's welcome message carries the full list; this is
// only the delta, and applying it is idempotent.
func (h *Hub) flushPresence() {
	h.pmu.Lock()
	changed := h.dirty
	h.dirty = make(map[string]struct{})
	h.flushScheduled = false
	h.pmu.Unlock()

	joined, left := []string{}, []string{}
	h.mu.RLock()
	for id := range changed {
		if _, online := h.byID[id]; online {
			joined = append(joined, id)
		} else {
			left = append(left, id)
		}
	}
	count := len(h.byID)
	h.mu.RUnlock()
	if len(joined)+len(left) == 0 {
		return
	}
	sort.Strings(joined)
	sort.Strings(left)

	h.broadcast(Message{
		Type:      MessageTypePresence,
		Timestamp: getCurrentTimestamp(),
		Payload: map[string]interface{}{
			"joined":    joined,
			"left":      left,
			"userCount": count,
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
