package main

import (
	"context"
	"sync"

	//"github.com/go-gost/x/ctx"
	"github.com/gorilla/websocket"
)

// Client represents a connected WebSocket client
type Client struct {
	mu     sync.RWMutex
	disconnectOnce sync.Once
	ctx    context.Context
	cancel context.CancelFunc
	id     string
	conn   *websocket.Conn
	send   chan Message

	authenticated bool
	token         string
	clientType    string
}

/* // ClientInfo returns info about a client, minus the mutex, channel and connection
type ClientInfo struct {
	id            string
	authenticated bool
	token         string
	clientType    string
} */

func NewClient(conn *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		//ID:   generateID(), //we will set the ID after authentication(create if client doesnt send one)
		ctx:    ctx,
		cancel: cancel,
		conn:   conn,
		send:   make(chan Message, 10),
	}
}

/* func (c *Client) getClientInfo() ClientInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return ClientInfo{
		id:            c.id,
		authenticated: c.authenticated,
		token:         c.token,
		clientType:    c.clientType,
	}
} */

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

/* func (c *Client) setAuthenticated(authenticated bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.authenticated = authenticated
} */

/* func (c *Client) closeWsConnection() {
	c.conn.Close()
} */

// a method that takes in a function to update the client struct in a thread-safe way
// useful for updating multiple fields at once without needing to acquire and release the lock multiple times
func (c *Client) updateClient(updateFunc func(*Client)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	updateFunc(c)
}

func (c *Client) disconnect() {
    c.disconnectOnce.Do(func() {
        c.cancel()
        c.conn.Close()
    })
}