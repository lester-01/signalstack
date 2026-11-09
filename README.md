# SignalStack

A realtime chat server in Go with a Next.js client, built to explore how a WebSocket server handles many concurrent connections: routing, backpressure, shutdown, and reconnects. It is a systems exercise, not a chat product.

Design notes and the wire protocol: [ARCHITECTURE.md](ARCHITECTURE.md).

## What it demonstrates

- **Routing without blocking.** A mutex-guarded hub with a per-client send queue. A client that falls behind is evicted instead of stalling everyone else.
- **One writer per socket.** Each connection runs a reader and a writer goroutine, so the socket is never written concurrently.
- **Clean teardown.** Queued messages are flushed and a close frame is sent on disconnect; `SIGINT`/`SIGTERM` shuts the whole server down the same way.
- **Authenticated sessions.** Constant-time token check, auth timeout, Origin allowlist, and server-issued client IDs. A client reclaims its ID after a reconnect only with an HMAC-signed resume token.
- **Batched presence.** Joins and leaves are collected for 100 ms and sent as one message, so a mass disconnect costs one message per client instead of one per departure. A new client gets the full list of who is online in its welcome message.
- **Per-client rate limiting** (token bucket) and an opt-in Prometheus-style `/metrics` endpoint.
- **Public and private messages** over one envelope.
- **Client reconnect** with exponential backoff and jitter.

## Demo

https://github.com/user-attachments/assets/f102ed96-9df4-4a40-8cb3-fa7ed6bd5c59

Open several browser tabs to see realtime sync. To re-record the video and screenshots, see [docs/DEMO.md](docs/DEMO.md). More screenshots are in [`demo/`](demo/).

## Quick start

Requires Go 1.22+ and Node.js 20.9+.

```bash
# terminal 1: backend (AUTH_TOKEN is required)
cd backend
AUTH_TOKEN=change-me go run .

# terminal 2: frontend
cp .env.example .env.local        # NEXT_PUBLIC_AUTH_TOKEN must match AUTH_TOKEN
npm install
npm run dev
```

Open http://localhost:3000 in a few tabs.

## Configuration

Backend (environment variables):

| Variable | Default | Purpose |
|---|---|---|
| `AUTH_TOKEN` | required | Shared token clients must send when authenticating |
| `ADDR` / `PORT` | `:8080` | Listen address |
| `ALLOWED_ORIGINS` | `http://localhost:3000` | Comma-separated browser origins allowed to connect; `*` allows any |
| `SESSION_SECRET` | random per process | Signs resume tokens; set it to keep sessions across restarts |
| `RATE_LIMIT` | `10` | Chat messages per second allowed per client (burst is twice that); `0` disables |
| `METRICS_ENABLED` | `false` | Serve `/metrics` (connections, messages, drops, evictions, memory, goroutines) |

Frontend (`.env.local`): `NEXT_PUBLIC_WS_URL` (default `ws://localhost:8080/ws`) and `NEXT_PUBLIC_AUTH_TOKEN`. `NEXT_PUBLIC_*` values are bundled into the browser build, so the token is a shared demo credential, not a secret.

## Tests

```bash
cd backend && go test -race ./...
```

The suite covers auth rejection delivery, ID spoofing and resume, presence batching, rate limiting, metrics, private routing, slow-client eviction, origin checks, goroutine leaks, shutdown, and a concurrent connect/chat/disconnect churn run for the race detector. CI runs it on every push (`.github/workflows/ci.yml`).

## Benchmarking

`backend/cmd/loadtest` is a load generator that reports delivery latency percentiles, throughput and memory per connection. No results are published yet; see [docs/BENCHMARKING.md](docs/BENCHMARKING.md) for the method and how to produce them.

## Layout

```
backend/    Go server (hub, client, server, config, session, metrics, rate limit) and tests
backend/cmd/loadtest/  Load generator and latency histogram
docs/       Benchmarking guide
scripts/    Benchmark sweep (bench-sweep.sh) and demo capture (capture.mjs)
app/        Next.js pages
providers/  WebSocket provider and useChat() hook
components/chat/  Chat UI: presence list, message log, composer, connection status
lib/        Client ID colors and helpers
demo/       Screenshots and walkthrough video
```

## Limitations

- Auth is a single shared token. It gates access but does not identify users.
- Everything is in memory: no message history, and one process only.
- Rate limiting is per connection, not per user or IP.
- No benchmarks have been run on dedicated hardware, so there are no published throughput or latency figures.
- No frontend tests.
- Timestamps are server-local `hh:mm:ss AM/PM` strings with no date.
