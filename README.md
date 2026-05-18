# 📡 WebSocket Demo — Full Stack Spec (Next.js + Go)

## 🎯 Goal

Build a minimal real-time WebSocket demo:

* Next.js frontend (shadcn already installed)
* Go backend WebSocket server (single binary)
* JSON messaging protocol
* Broadcast messaging between clients
* Server heartbeat broadcast (no ACK)
* Auto-reconnect client logic

---

# 🧱 Repo Structure

```
root/
  app/                      # Next.js app (existing)
  components/
  hooks/
  lib/

  backend/
    go.mod
    main.go
    hub.go
    client.go
    types.go
```

---

# 🚀 System Overview

```
Browser (Next.js)
   ws://localhost:8080/ws
           |
           v
Go WebSocket Server (:8080)
```

No proxies. No Next.js server involvement.

---

# 📦 Message Protocol (STRICT)

All messages MUST follow this envelope:

```json
{
  "type": "message | heartbeat | join",
  "clientId": "string",
  "timestamp": 1234567890,
  "payload": {}
}
```

---

## Client → Server

### Send chat message

```json
{
  "type": "message",
  "payload": {
    "text": "hello"
  }
}
```

---

## Server → Clients

### Broadcast message

```json
{
  "type": "message",
  "clientId": "abc123",
  "timestamp": 1234567890,
  "payload": {
    "text": "hello"
  }
}
```

---

### Join event

```json
{
  "type": "join",
  "clientId": "abc123",
  "timestamp": 1234567890,
  "payload": {}
}
```

---

### Heartbeat (server → all clients every 10s)

```json
{
  "type": "heartbeat",
  "timestamp": 1234567890,
  "payload": {}
}
```

---

# 🧠 Backend (Go) — IMPLEMENTATION

## Dependencies

Use:

* Go standard library
* Gorilla WebSocket:

  * Gorilla WebSocket

---

## backend/go.mod

Initialize:

```bash
go mod init backend
go get github.com/gorilla/websocket
```

---

## backend/types.go

Define core message structure:

```go
package main

type Message struct {
	Type      string                 `json:"type"`
	ClientID  string                 `json:"clientId,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
}
```

---

## backend/client.go

```go
package main

import "github.com/gorilla/websocket"

type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan Message
}
```

---

## backend/hub.go

Core concurrency hub.

### Responsibilities:

* register clients
* unregister clients
* broadcast messages

```go
package main

type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan Message),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {

		case client := <-h.Register:
			h.Clients[client] = true

		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}

		case msg := <-h.Broadcast:
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
```

---

## backend/main.go

### WebSocket upgrade + server entrypoint

```go
package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func generateID() string {
	return fmt.Sprintf("c-%d", rand.Intn(1000000))
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)

	client := &Client{
		ID:   generateID(),
		Conn: conn,
		Send: make(chan Message, 10),
	}

	hub.Register <- client

	joinMsg := Message{
		Type:      "join",
		ClientID:  client.ID,
		Timestamp: time.Now().Unix(),
		Payload:   map[string]interface{}{},
	}
	hub.Broadcast <- joinMsg

	// writer
	go func() {
		for msg := range client.Send {
			_ = conn.WriteJSON(msg)
		}
	}()

	// reader
	go func() {
		defer func() {
			hub.Unregister <- client
			conn.Close()
		}()

		for {
			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				break
			}

			msg.ClientID = client.ID
			msg.Timestamp = time.Now().Unix()

			hub.Broadcast <- msg
		}
	}()
}

func startHeartbeat(hub *Hub) {
	ticker := time.NewTicker(10 * time.Second)

	for range ticker.C {
		hub.Broadcast <- Message{
			Type:      "heartbeat",
			Timestamp: time.Now().Unix(),
			Payload:   map[string]interface{}{},
		}
	}
}

func main() {
	hub := NewHub()
	go hub.Run()
	go startHeartbeat(hub)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	http.ListenAndServe(":8080", nil)
}
```

---

# ⚛️ Frontend (Next.js)

## Install note

No new dependencies required.

Use native:

```ts
WebSocket
```

---

# 📁 hooks/useWebSocket.ts

```ts
"use client";

import { useEffect, useRef, useState } from "react";

type Status = "connecting" | "open" | "closed" | "reconnecting";

export function useWebSocket(url: string) {
  const ws = useRef<WebSocket | null>(null);
  const retry = useRef(0);
  const [status, setStatus] = useState<Status>("connecting");
  const [messages, setMessages] = useState<any[]>([]);

  const connect = () => {
    setStatus(retry.current ? "reconnecting" : "connecting");

    const socket = new WebSocket(url);
    ws.current = socket;

    socket.onopen = () => {
      setStatus("open");
      retry.current = 0;
    };

    socket.onmessage = (event) => {
      const data = JSON.parse(event.data);
      setMessages((prev) => [...prev, data]);
    };

    socket.onclose = () => {
      setStatus("closed");

      const delay = Math.min(1000 * 2 ** retry.current, 10000);
      retry.current += 1;

      setTimeout(() => connect(), delay);
    };
  };

  useEffect(() => {
    connect();
    return () => ws.current?.close();
  }, []);

  const sendMessage = (text: string) => {
    ws.current?.send(
      JSON.stringify({
        type: "message",
        payload: { text },
      })
    );
  };

  return { status, messages, sendMessage };
}
```

---

# 📄 app/page.tsx

```tsx
"use client";

import { useState } from "react";
import { useWebSocket } from "@/hooks/useWebSocket";

export default function Page() {
  const { status, messages, sendMessage } = useWebSocket(
    "ws://localhost:8080/ws"
  );

  const [input, setInput] = useState("");

  return (
    <div className="p-6 space-y-4">
      <div className="text-sm">
        Status: <b>{status}</b>
      </div>

      <div className="border p-3 h-64 overflow-auto">
        {messages.map((m, i) => (
          <div key={i} className="text-xs">
            [{m.type}] {m.payload?.text ?? ""}
          </div>
        ))}
      </div>

      <div className="flex gap-2">
        <input
          className="border p-2 flex-1"
          value={input}
          onChange={(e) => setInput(e.target.value)}
        />
        <button
          className="border px-3"
          onClick={() => {
            sendMessage(input);
            setInput("");
          }}
        >
          Send
        </button>
      </div>
    </div>
  );
}
```

---

# 🔁 Behavior Summary

## Server

* accepts WS connections
* assigns client IDs
* broadcasts messages to all clients
* sends heartbeat every 10s
* cleans up on connection error

---

## Client

* connects on load
* auto-reconnects with backoff
* sends JSON messages
* renders message stream
* displays connection status

---

# 🚫 Explicit Constraints (DO NOT VIOLATE)

Agent must NOT add:

* Socket.IO
* authentication
* database
* message persistence
* rooms/channels
* Next.js API websocket proxy
* ACK-based heartbeat
* production scaling patterns
* docker/kubernetes
* complex state managers

---

# ✅ Acceptance Criteria

Demo is successful if:

1. Two browser tabs can connect
2. Messages sent in one tab appear in the other
3. Server heartbeat appears every 10s
4. Closing/reopening tab reconnects automatically
5. Server does not crash on disconnect
6. No external dependencies beyond Gorilla WS

