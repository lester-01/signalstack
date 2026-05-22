# signalstack

> A real-time chat platform showcasing production-grade WebSocket infrastructure patterns in Go

A full-stack demonstration of concurrent connection handling, graceful shutdown, message routing, and resilience patterns. Built with **Go** (backend) and **Next.js** (frontend), signalstack isn't just another chat app—it's proof you can architect systems that scale.

---

## See It In Action

**Video Demo** (2 min walkthrough):  
<!-- ![](./demo/demo.mp4)
<video src="./demo/demo.mp4" controls controlsList="nodownload" preload></video> -->

https://github.com/lester-01/signalstack/blob/main/demo/demo.mp4

<!-- <video autoplay muted loop playsinline controls width="100%">
  <source src="./demo/demo.mp4" type="video/mp4">
</video> -->
Open multiple browser tabs to see real-time sync. Send public messages (everyone sees), private messages (1-to-1 only), watch users connect/disconnect, and trigger auto-reconnect by killing your network. It just works.

---

## Quick Start

Get it running in 5 minutes:

### Prerequisites
- **Go 1.21+** | **Node.js 18+** | Two terminal windows

### Setup

```bash
# Terminal 1: Backend
cd backend && go run .

# Terminal 2: Frontend
npm install && npm run dev
```

Open http://localhost:3000 in multiple tabs. That's it.

---

## What You're Looking At

This isn't a polished chat app showcase. It's a **systems engineering playground**.

Every line of code solves a real problem:
- **Hub-Client routing** → How do you deliver a message to 1000 concurrent clients without blocking?
- **Dual goroutines per connection** → Why can't a single goroutine safely read & write a WebSocket?
- **Graceful shutdown + context cancellation** → How do you clean up 1000 goroutines without dangling resources?
- **Exponential backoff reconnection** → How do you handle network failures without overwhelming the server?
- **Message routing logic** → How do you enforce that private messages only reach 2 clients?

**Curious?** → [See the full architecture breakdown](ARCHITECTURE.md)

---

## Key Features

| Feature | Why It Matters |
|---------|----------------|
| **Public & Private Messaging** | Single message envelope supports both broadcast + unicast routing logic |
| **Real-Time Connection Sync** | State embedded in messages (no separate user list endpoint); all clients see consistent user list |
| **Graceful Connection Handling** | Safe resource cleanup on disconnect; prevents leaks at scale |
| **Auto-Reconnect (Exponential Backoff)** | Resilience pattern for unreliable networks; prevents thundering herd after outages |
| **Server-Side Timestamping** | Consistent message ordering across all clients (ignores client clock skew) |
| **Authentication Gate with Timeout** | Prevents bot spam; demonstrates permission/timing patterns |

---

## Technical Stack

| Layer | Tech |
|-------|------|
| **Backend** | Go + Gorilla WebSocket |
| **Frontend** | Next.js + React + TypeScript |
| **Styling** | Tailwind CSS + shadcn/ui |
| **Transport** | JSON over WebSocket (custom protocol) |
| **State** | Context API (frontend), In-memory Hub (backend) |

---

## Want More?

Curious about the deep technical details? Everything you need is here:

- **[Full Architecture Breakdown →](ARCHITECTURE.md)** — All 11 design patterns with detailed explanations
- **[Message Protocol & API →](ARCHITECTURE.md#message-protocol--api)** — Complete JSON schemas
- **[Deployment Guide →](ARCHITECTURE.md#deployment-guide)** — Docker, cloud setup, production checklist
- **[Performance Analysis →](ARCHITECTURE.md#performance-considerations)** — Concurrency limits, latency, scaling math

---

## Roadmap (Selected Highlights)

See [full roadmap](ARCHITECTURE.md#full-roadmap) for all phases.

- [ ] **Message Persistence** — PostgreSQL/MongoDB for history
- [ ] **Real User Auth** — OAuth2/JWT (not demo token)
- [ ] **Group Chats** — Channel-based routing
- [ ] **Typing Indicators** — Real-time "X is typing..."
- [ ] **Horizontal Scaling** — Redis pub/sub for multi-server deployments

---

## For Recruiters

**Why This Project Matters:**

If you're building real-time systems (trading platforms, collaborative tools, live dashboards, notifications), you need engineers who understand the plumbing. signalstack demonstrates:

1. **Concurrent System Design** — Go concurrency primitives (goroutines, channels, mutexes, context) applied to prevent race conditions, deadlocks, and resource leaks
2. **Production Patterns** — Graceful shutdown, buffered fallbacks, exponential backoff, authentication gates
3. **Scalable Architecture** — Hub-Client routing, snapshot isolation, non-blocking dispatch designed to scale from 100 to 100,000+ concurrent connections
4. **Real-Time State Sync** — Frontend-backend synchronization strategy that minimizes lock contention and handles reconnections safely
5. **Systems Thinking** — Every pattern exists to solve a real problem; no over-engineering

**The bottom line:** Chat apps are everywhere. The infrastructure patterns inside scale to ANY real-time system. Hire me for YOUR infrastructure challenges.

---

**Questions?** Open an [issue on GitHub](https://github.com/lester-01/signalstack) or explore [ARCHITECTURE.md](ARCHITECTURE.md) for deep dives.

---

## Screenshots

<details>
<summary>Click to view UI screenshots</summary>

![Screenshot 1](demo/Screenshot%202026-05-20%20204448.png)
![Screenshot 2](demo/Screenshot%202026-05-20%20204650.png)
![Screenshot 3](demo/Screenshot%202026-05-20%20204728.png)
![Screenshot 4](demo/Screenshot%202026-05-20%20204855.png)
![Screenshot 5](demo/Screenshot%202026-05-20%20204911.png)

</details>

---

Happy coding!
