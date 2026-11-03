# SignalStack

Realtime chat demo built to showcase WebSocket infrastructure patterns in Go (backend) and Next.js (frontend).

Not a consumer chat product—a systems playground for concurrent connections, routing, graceful shutdown, and reconnect.

## What it demonstrates

- Hub-client routing for many concurrent clients without blocking
- Dual goroutines per WebSocket connection (safe read/write)
- Graceful shutdown and context cancellation
- Exponential-backoff reconnect on the client
- Public broadcast and private 1:1 message routing

Full write-up: [ARCHITECTURE.md](ARCHITECTURE.md).

## Demo

Video walkthrough:

https://github.com/user-attachments/assets/f102ed96-9df4-4a40-8cb3-fa7ed6bd5c59

Open multiple browser tabs to see realtime sync. Screenshots are under [`demo/`](demo/).

## Quick start

Prerequisites: **Go 1.21+**, **Node.js 18+**, two terminals.

```bash
# Terminal 1: backend
cd backend && go run .

# Terminal 2: frontend
npm install && npm run dev
```

Open http://localhost:3000 in multiple tabs.

## Features

| Feature | Why it matters |
|---------|----------------|
| Public & private messaging | One envelope supports broadcast and unicast |
| Realtime connection sync | User list state rides with messages |
| Graceful disconnect | Cleanup that scales past toy demos |
| Auto-reconnect | Backoff without thundering herd |
| Server-side timestamps | Consistent ordering across clients |
| Auth gate with timeout | Simple permission / timing pattern |
