package main

import (
	"crypto/subtle"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

const authTimeout = 10 * time.Second

// Server wires configuration, the hub and the HTTP/WebSocket handlers together.
type Server struct {
	cfg      Config
	hub      *Hub
	signer   sessionSigner
	upgrader websocket.Upgrader
}

// NewServer creates a Server with an empty hub.
func NewServer(cfg Config) *Server {
	s := &Server{
		cfg:    cfg,
		hub:    NewHub(),
		signer: sessionSigner{secret: cfg.SessionSecret},
	}
	s.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     s.checkOrigin,
	}
	return s
}

// Handler returns the HTTP routes: /ws for WebSocket clients and /healthz for probes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.serveWs)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

func (s *Server) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // not a browser; authentication still applies
	}
	for _, allowed := range s.cfg.AllowedOrigins {
		if allowed == "*" || strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

// serveWs upgrades the request and starts the three goroutines that serve a
// connection: auth timeout, writer (the only goroutine that writes to the
// socket) and reader.
func (s *Server) serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	c := NewClient(conn)
	go s.authTimeout(c)
	go s.writePump(c)
	go s.readPump(c)
}

// authTimeout disconnects clients that do not authenticate in time.
func (s *Server) authTimeout(c *Client) {
	t := time.NewTimer(authTimeout)
	defer t.Stop()
	select {
	case <-t.C:
		if !c.isAuthenticated() {
			s.reject(c, "auth timeout")
		}
	case <-c.ctx.Done():
	}
}

// reject tells the client why authentication failed, then closes the
// connection. The write loop flushes the reason before closing.
func (s *Server) reject(c *Client, reason string) {
	c.trySend(Message{
		Type:      MessageTypeAuthResponse,
		Timestamp: getCurrentTimestamp(),
		Payload: map[string]interface{}{
			"success": false,
			"reason":  reason,
		},
	})
	c.disconnect()
}

// writePump is the only goroutine that writes to the connection.
func (s *Server) writePump(c *Client) {
	ping := time.NewTicker(pingPeriod)
	defer func() {
		ping.Stop()
		s.hub.unregister(c)
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}
		case <-ping.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.ctx.Done():
			c.flushAndClose()
			return
		}
	}
}

// readPump reads client messages until the connection ends.
func (s *Server) readPump(c *Client) {
	defer s.hub.unregister(c)

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				log.Printf("websocket read error: %v", err)
			}
			return
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))

		switch msg.Type {
		case MessageTypeAuth:
			s.handleAuth(c, msg)
		case MessageTypeMessage:
			if !c.isAuthenticated() {
				log.Printf("dropping message from unauthenticated client")
				continue
			}
			s.handleChat(c, msg)
		default:
			log.Printf("unknown message type: %q", msg.Type)
		}
	}
}

func (s *Server) handleAuth(c *Client, msg Message) {
	if c.isAuthenticated() {
		return
	}

	token, _ := msg.Payload["token"].(string)
	if token == "" {
		s.reject(c, "missing token")
		return
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.AuthToken)) != 1 {
		s.reject(c, "invalid token")
		return
	}

	// A client may reclaim a previous ID only with the signed resume token the
	// server issued for it. Otherwise it gets a fresh, server-generated ID.
	id := ""
	if want, _ := msg.Payload["clientId"].(string); want != "" {
		if resume, _ := msg.Payload["resumeToken"].(string); s.signer.verify(want, resume) {
			id = want
		}
	}
	if id == "" {
		id = newClientID()
	}
	clientType, _ := msg.Payload["clientType"].(string)
	c.authenticate(id, clientType)

	// Queue the response before registering so the client always sees it
	// before any join event.
	c.trySend(Message{
		Type:      MessageTypeAuthResponse,
		ClientID:  id,
		Timestamp: getCurrentTimestamp(),
		Payload: map[string]interface{}{
			"success":     true,
			"resumeToken": s.signer.sign(id),
		},
	})
	if !s.hub.register(c) {
		c.disconnect()
	}
}

func (s *Server) handleChat(c *Client, msg Message) {
	text, _ := msg.Payload["text"].(string)
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > maxTextRunes {
		return
	}
	recipient, _ := msg.Payload["recipientId"].(string)
	if recipient == "" {
		recipient = RecipientAll
	}

	// Rebuild the payload rather than relaying whatever the client sent.
	s.hub.routeChat(c, Message{
		Type:      MessageTypeMessage,
		ClientID:  c.getID(),
		Timestamp: getCurrentTimestamp(),
		Payload: map[string]interface{}{
			"text":        text,
			"recipientId": recipient,
		},
	})
}
