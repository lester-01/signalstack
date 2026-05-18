# Plan: WebSocket Real-Time Chat Demo (Next.js + Go) — With Direct Messaging

## TL;DR
Build full-stack WebSocket chat with direct messaging. Next.js frontend maintains list of connected clients, sends messages to specific user or everyone via `recipientId` field. Go backend routes private messages to recipient + sender only; broadcasts go to everyone. Frontend displays join/leave events with userCount, heartbeats, all messages (private only to participants), timestamps human-readable. Client selector dropdown lets users pick recipient. Auto-reconnect, localStorage persistence, error display.

---

## Updated Message Protocol

### Client → Server

**Send message** (public or private)
```
{
  "type": "message",
  "payload": {
    "text": "hello",
    "recipientId": "all"  // "all" for broadcast, or specific clientId for private
  }
}
```

### Server → Clients

**Broadcast message** (everyone sees it)
```
{
  "type": "message",
  "clientId": "sender-id",
  "timestamp": "2:45:30 PM",
  "payload": {
    "text": "hello",
    "recipientId": "all"
  }
}
```

**Private message** (only sender + recipient see it)
```
{
  "type": "message",
  "clientId": "sender-id",
  "timestamp": "2:45:30 PM",
  "payload": {
    "text": "private hello",
    "recipientId": "recipient-id"
  }
}
```

**Join event** (broadcast to all, includes updated client list + userCount)
```
{
  "type": "join",
  "clientId": "new-user-id",
  "timestamp": "2:45:30 PM",
  "payload": {
    "userCount": 3,
    "clientList": ["client-1", "client-2", "new-user-id"]
  }
}
```

**Leave event** (broadcast to remaining clients with updated list)
```
{
  "type": "leave",
  "clientId": "departed-user-id",
  "timestamp": "2:45:30 PM",
  "payload": {
    "userCount": 2,
    "clientList": ["client-1", "client-2"]
  }
}
```

**Heartbeat** (every 10s to all clients)
```
{
  "type": "heartbeat",
  "timestamp": "2:45:30 PM",
  "payload": {
    "userCount": 3,
    "clientList": ["client-1", "client-2", "client-3"]
  }
}
```

---

## Steps

### Phase 1: Backend Setup (Go)

1. **Create backend directory structure**
   - Create `backend/` folder at project root
   - Initialize `go.mod` with `go mod init backend`
   - Install Gorilla WebSocket: `go get github.com/gorilla/websocket`

2. **Implement backend/types.go**
   - Define `Message` struct with fields: Type, ClientID, Timestamp (string, formatted), Payload
   - Payload is `map[string]interface{}` to handle: text, recipientId, userCount, clientList
   - *No dependencies on other files yet*

3. **Implement backend/client.go**
   - Define `Client` struct with fields: ID (string), Conn (*websocket.Conn), Send (chan Message)
   - *Depends on types.go for Message type*

4. **Implement backend/hub.go**
   - Define `Hub` struct with: Clients (map[*Client]bool), Broadcast, Register, Unregister channels
   - Implement `NewHub()` constructor
   - Implement `Run()` method with select loop for concurrency
   - **Handle Register**: 
     * Add client to map
     * Build clientList (all IDs)
     * Create and broadcast join message with userCount + clientList to ALL clients
   - **Handle Unregister**: 
     * Remove client, close channel
     * Build updated clientList
     * Create and broadcast leave message with userCount + clientList to remaining clients
   - **Handle Broadcast**: 
     * Check message payload for recipientId
     * If recipientId == "all": send to all clients
     * If recipientId is specific ID: send only to that recipient + sender
     * Use buffered fallback (close+delete if full)
   - *Depends on client.go for Client type*

5. **Implement backend/main.go**
   - Import: encoding/json, fmt, math/rand, net/http, time, gorilla/websocket
   - Define `upgrader` with CheckOrigin allowing all origins
   - Implement `generateID()` function (format: "c-{random}")
   - Implement `formatTimestamp()` function: convert Unix time to human-readable "HH:MM:SS AM/PM" format
   - Implement `serveWs()` handler:
     * Upgrade connection
     * Create client with generated ID
     * Send to hub.Register (which broadcasts join to all)
     * Spawn writer goroutine (read from client.Send, write to conn)
     * Spawn reader goroutine:
       - Read incoming message
       - Add clientId, formatted timestamp
       - Extract recipientId from payload (default to "all")
       - Broadcast via hub
     * Handle disconnect cleanup (hub.Unregister)
   - Implement `startHeartbeat()` function:
     * Ticker every 10s
     * Get current client count + list from hub
     * Build heartbeat with userCount + clientList
     * Broadcast to hub
   - Implement `main()`:
     * Create hub, run it in goroutine
     * Start heartbeat in goroutine
     * Register /ws handler
     * Start server on :8080
   - *Depends on types.go, client.go, hub.go*

6. **Test backend startup**
   - Run `go run ./backend` from project root
   - Verify no compile errors
   - Server should listen on :8080

---

### Phase 2: Frontend Hook Implementation

7. **Create hooks/useWebSocket.ts** 
   - Define Message type matching backend protocol (with recipientId field)
   - Define Status type: "connecting" | "open" | "closed" | "reconnecting"
   - Implement custom hook:
     * useRef for ws connection, retry count
     * useState for: status, messages, clientId, error, connectedClients (list of all client IDs)
     * Implement `connect()` function:
       - Create WebSocket to `ws://localhost:8080/ws`
       - onopen: set status=open, reset retry
       - onmessage: parse JSON
         * If type == "join" or "leave": extract clientList from payload, update connectedClients state, add to messages
         * If type == "message" or "heartbeat": add to messages
         * If first message and clientId not set: extract from join payload
       - onerror: set error state
       - onclose: set status=closed, schedule reconnect with exponential backoff (min 1s, max 10s)
     * Load clientId from localStorage on mount
     * useEffect: call connect() on mount, cleanup by closing socket on unmount
     * Export hook with: status, messages, clientId, connectedClients, error, sendMessage function
   - sendMessage function takes (text, recipientId) as params, default recipientId="all"
   - *Depends on types from backend but defined locally*

8. **Create lib/utils/timestamp.ts (utility)**
   - Export `isPrivateMessage(msg, currentClientId)` — returns true if recipientId is set and not "all" and includes current client
   - Export `formatMessageDisplay(msg)` — returns readable string based on type
     * "message": `"${senderInitial}: text"` or `"${senderInitial} (private): text"`
     * "join": `"→ User ${senderInitial} joined (${userCount} online)"`
     * "leave": `"← User ${senderInitial} left (${userCount} online)"`
     * "heartbeat": `"[Heartbeat] ${userCount} users online"`

---

### Phase 3: Frontend UI Implementation

9. **Add shadcn components via CLI**
   - Use shadcn CLI to add: Select, Card, Badge, ScrollArea, Separator
   - Ensure components installed to components/ui/

10. **Update app/page.tsx**
   - Use useWebSocket hook to get status, messages, clientId, connectedClients, error, sendMessage
   - Use useState for input field, selectedRecipient (default "all")
   - Layout structure (use Card + ScrollArea for polish):
     * **Header section**:
       - Status badge (connecting/open/closed/reconnecting) using Badge
       - Your client ID display
       - User count from latest heartbeat
     * **Error display section** (red bg Card, show if error state)
     * **Connected clients dropdown** (Select component)
       - Options: "Everyone" (value="all"), then all other client IDs
       - Show which one is current user with label suffix
       - onChange updates selectedRecipient state
     * **Message list** (ScrollArea with Card border)
       - Scrollable container
       - Each message rendered with formatMessageDisplay utility
       - Private messages get visual indicator (color/italics)
       - Auto-scroll to bottom on new message
     * **Input + Send section** (bottom)
       - Input field for text (disabled if status !== "open")
       - Select dropdown for recipient (shows selected option)
       - Send button (disabled if input is empty OR status !== "open")
       - onClick: sendMessage(input, selectedRecipient), clear input
   - Use shadcn components:
     * Button, Input, Badge, Card, ScrollArea, Select, Separator
   - Styling: Tailwind + shadcn defaults
   - *Depends on useWebSocket hook, lib/utils/timestamp, all shadcn components*

---

### Phase 4: Verification & Testing

11. **Local testing — Basic connectivity**
    - Terminal 1: `cd backend && go run ./`
    - Terminal 2: `npm run dev` (Next.js frontend)
    - Open browser tab 1: http://localhost:3000
    - Open browser tab 2: http://localhost:3000
    - Verify both tabs show status "open"
    - Verify each tab shows different clientId
    - Verify both tabs show same "Connected Clients" list with both IDs
    - Verify join messages appear in both tabs with updated user count

12. **Test broadcast messages**
    - In tab 1: leave recipient as "Everyone", send "Hello everyone"
    - Verify message appears in both tabs with tab 1's clientId
    - Verify format is readable (clientId: text)
    - Verify timestamp is human-readable

13. **Test private messages**
    - In tab 1: select tab 2's clientId from dropdown, send "Private hello"
    - Verify message appears in both tab 1 and tab 2
    - Verify message appears with private indicator
    - In tab 2 (NEW tab 3): open third client, send broadcast message
    - Verify tab 3 does NOT see the private message from tab 1 → tab 2
    - Verify tab 3 sees the new broadcast

14. **Test heartbeats & user count**
    - Wait 10s
    - Verify "[Heartbeat]" message appears in all tabs
    - Verify user count stays accurate (3 with 3 tabs open)

15. **Test leave event**
    - Close tab 2
    - Verify tab 1 and 3 see leave message with updated user count (now 2)
    - Verify tab 2's clientId removed from dropdown in remaining tabs

16. **Test error handling**
    - Stop backend server
    - Verify tabs show status "closed" → "reconnecting"
    - Verify error message displays
    - Restart backend
    - Verify tabs auto-reconnect, status returns to "open"
    - Verify no duplicate messages on reconnect

17. **Test send button disabled states**
    - With empty input: button should be disabled
    - With text: button should be enabled (if status="open")
    - With status="closed": button should be disabled

18. **Test client ID persistence**
    - Refresh tab 1
    - Verify same clientId appears (from localStorage)
    - Verify reconnects without creating duplicate client entry

19. **Test input validation**
    - Try sending message with only spaces
    - Should not send (treat as empty)
    - Try sending normal message
    - Should work

20. **Test dropdown recipient selection**
    - Verify "Everyone" is default
    - Switch to specific user, send message
    - Verify it's private (other users don't see it)
    - Switch back to "Everyone", send message
    - Verify it's broadcast (all users see it)

---

## Relevant Files

**Frontend:**
- [app/page.tsx](app/page.tsx) — Chat UI with recipient selector, message list, status, error display
- [hooks/useWebSocket.ts](hooks/useWebSocket.ts) — New file, WebSocket connection logic with client list sync + auto-reconnect
- [lib/utils/timestamp.ts](lib/utils/timestamp.ts) — New file, message formatting + privacy detection utilities
- [components/ui/button.tsx](components/ui/button.tsx) — Existing shadcn component
- [components/ui/input.tsx](components/ui/input.tsx) — Existing shadcn component (or add if missing)
- [components/ui/select.tsx](components/ui/select.tsx) — New shadcn component to add
- [components/ui/card.tsx](components/ui/card.tsx) — New shadcn component to add
- [components/ui/badge.tsx](components/ui/badge.tsx) — New shadcn component to add
- [components/ui/scroll-area.tsx](components/ui/scroll-area.tsx) — New shadcn component to add
- [components/ui/separator.tsx](components/ui/separator.tsx) — New shadcn component to add

**Backend:**
- `backend/go.mod` — New file, Go module definition
- `backend/types.go` — New file, Message struct with recipientId in payload
- `backend/client.go` — New file, Client struct
- `backend/hub.go` — New file, Hub with private message routing logic
- `backend/main.go` — New file, HTTP server + WebSocket handler + message routing

---

## Verification Checklist

1. Backend compiles: `go run ./backend` starts without errors on :8080
2. Frontend connects: Page loads, status shows "open" after <1s
3. Client list syncs: Joining tab shows in all other tabs' dropdown
4. Broadcast messages: Send to "Everyone", all tabs see it
5. Private messages: Send to specific user, only sender + recipient see it
6. Join event: Shows readable format with user count
7. Leave event: Shows when tab closes, user count decrements, clientId removed from dropdowns
8. Heartbeat: Appears every ~10s with current user count
9. Timestamps: All messages show human-readable time format
10. Send button disabled: When input empty or connection closed
11. Input validation: Whitespace-only messages don't send
12. Error display: Shows when backend unavailable, clears on reconnect
13. Auto-reconnect: Exponential backoff works (1s, 2s, 4s, 8s, 10s max)
14. Client ID persistence: Refreshing page keeps same clientId
15. No crashes: Backend handles multiple clients + rapid connects/disconnects
16. Third-tab privacy: New client doesn't see prior private messages

---

## Decisions

- **Message routing**: `recipientId` field in payload (set to "all" for broadcast, specific clientId for private)
- **Client list sync**: Full list sent on join/leave/heartbeat events, frontend maintains in state
- **Private message privacy**: Only sender + recipient see private messages; third parties don't see them at all
- **Timestamp format**: Human-readable "HH:MM:SS AM/PM" format for all messages
- **Message types**: Four types in protocol: message, join, leave, heartbeat
- **Client list UI**: Select/Dropdown with "Everyone" + individual client IDs
- **UI components**: shadcn Select, Card, Badge, ScrollArea, Separator for polished appearance
- **Button disable logic**: Disabled when (1) input is empty/whitespace-only OR (2) status !== "open"
- **Client persistence**: localStorage key "websocket-clientid"
- **Exponential backoff**: 1s → 2s → 4s → 8s → 10s max
- **No ACK pattern**: Heartbeat and all messages are one-way (fire-and-forget)

---

## Scope Boundaries

**Included:**
- WebSocket connection management with auto-reconnect (exponential backoff)
- JSON message protocol with heartbeat, join, leave events
- Private & broadcast messaging (recipientId routing)
- Multi-client broadcast (public messages)
- Server-side message filtering (private messages only to participants)
- Client list synchronization on join/leave/heartbeat
- User count tracking
- Error display + status indicators
- Message history display in current session (only received messages)
- Client ID persistence via localStorage
- Human-readable timestamps (server-side formatted)
- Select dropdown for recipient picking
- Empty message validation (button disabled)

**Explicitly NOT included (per spec):**
- Socket.IO or any abstraction layer
- Authentication or authorization
- Database or message persistence beyond current session
- Rooms/channels (private messaging is 1:1 or broadcast, not rooms)
- Next.js API proxy (direct WS connection to Go backend)
- Docker/Kubernetes
- Production scaling
- Message delivery receipts/ACKs
- Typing indicators
- Message editing/deletion
- User profiles or online status (just IDs)
- Tests/automation

---

## Implementation Notes

1. **Backend message routing logic (hub.go)**:
   - When broadcasting a message, check `payload["recipientId"]`
   - If "all": send to all clients in hub.Clients
   - If specific ID: find that client in hub.Clients, send only to that client AND the sender
   - This prevents third parties from seeing private conversations

2. **Frontend private message detection (timestamp.ts)**:
   - A message is private if: `msg.payload.recipientId` is set AND not equal to "all" AND current clientId is sender OR in recipient list
   - Format private messages with visual indicator (e.g., "[PRIVATE from X]" or italicized)

3. **Auto-scroll behavior**:
   - ScrollArea should auto-scroll to bottom when new message arrives
   - Implement via useEffect watching messages array, scroll message list container to bottom

4. **Select dropdown behavior**:
   - If "Everyone" selected: dropdown shows "Send to Everyone"
   - If specific user selected: dropdown shows their clientId
   - On join/leave, update dropdown options dynamically
   - Keep selection if recipient user still connected; reset to "Everyone" if they leave

5. **Whitespace validation**:
   - In sendMessage handler: trim input, check if length > 0 before sending
   - Button disabled state: `disabled={!input.trim() || status !== "open"}`
