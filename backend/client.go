package main

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// sendBufferSize is the per-client outbound queue. A client whose queue is
	// full is considered too slow and is disconnected rather than blocking the hub.
	sendBufferSize = 256

	writeWait      = 10 * time.Second // deadline for a single write
	flushTimeout   = 2 * time.Second  // time allowed to flush queued messages on close
	pongWait       = 60 * time.Second // connection is dropped if no pong/message arrives in this time
	pingPeriod     = pongWait * 9 / 10
	maxMessageSize = 8 * 1024 // largest inbound frame in bytes
	maxTextRunes   = 1000     // largest chat message in characters
)

// Client is one connected WebSocket peer.
type Client struct {
	mu            sync.RWMutex
	id            string
	authenticated bool
	clientType    string

	ctx     context.Context
	cancel  context.CancelFunc
	evicted atomic.Bool

	conn *websocket.Conn
	send chan Message
}

// NewClient wraps conn. The client has no ID until it authenticates.
func NewClient(conn *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		ctx:    ctx,
		cancel: cancel,
		conn:   conn,
		send:   make(chan Message, sendBufferSize),
	}
}

func (c *Client) getID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.id
}

func (c *Client) isAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.authenticated
}

// authenticate marks the client as authenticated and sets its identity.
func (c *Client) authenticate(id, clientType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.authenticated = true
	c.id = id
	c.clientType = clientType
}

// trySend queues msg without blocking. It reports false if the queue is full.
func (c *Client) trySend(msg Message) bool {
	select {
	case c.send <- msg:
		return true
	default:
		return false
	}
}

// disconnect asks the write loop to flush what is queued, send a close frame
// and close the connection. It is safe to call more than once.
func (c *Client) disconnect() {
	c.cancel()
}

// forceClose drops the connection immediately without flushing. Used for
// clients that are not keeping up, where flushing would just block.
func (c *Client) forceClose() {
	c.cancel()
	_ = c.conn.Close()
}

// flushAndClose writes any queued messages, then a normal-closure frame.
// It must only be called from the write loop.
func (c *Client) flushAndClose() {
	_ = c.conn.SetWriteDeadline(time.Now().Add(flushTimeout))
	for {
		select {
		case msg := <-c.send:
			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}
		default:
			_ = c.conn.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			)
			return
		}
	}
}
