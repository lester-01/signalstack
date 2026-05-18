# WebSocket Chat Demo - Test Checklist

## Overview
This checklist covers all verification tests for the WebSocket real-time chat demo. Use this for:
- Manual regression testing
- Debugging issues
- Future unit test creation
- CI/CD validation

---

## Prerequisites

**Terminal 1: Start Backend**
```bash
cd backend && go run .
# Expected output: "WebSocket server starting on :8080"
```

**Terminal 2: Start Frontend**
```bash
npm run dev
# Expected output: "Ready in X.Xs" on port 3000
```

**Browser: Open Two Tabs**
- Tab A: http://localhost:3000
- Tab B: http://localhost:3000

---

## Phase 1: Backend Verification

### ✅ 1.1 Backend Compiles
- [ ] Run: `cd backend && go build -o websocket-server .`
- [ ] Expected: No errors, binary created
- [ ] Verify: `ls -la websocket-server` shows file with size > 0

### ✅ 1.2 Backend Starts Successfully
- [ ] Run: `go run .` from backend directory
- [ ] Expected: Log output shows "WebSocket server starting on :8080"
- [ ] Verify: No panic or fatal errors

### ✅ 1.3 Server Listens on Port 8080
- [ ] Run: `netstat -tuln | grep 8080`
- [ ] Expected: Port 8080 shows LISTEN status
- [ ] Alternative: Try accessing from browser, should not show connection refused

---

## Phase 2: Frontend Initial Connection

### ✅ 2.1 Tab A Loads Successfully
- [ ] Open Tab A at http://localhost:3000
- [ ] Verify: Page renders without errors
- [ ] Check: "WebSocket Chat" header visible
- [ ] Check: Status badge visible
- [ ] Check: "Your ID: ..." displayed (shows loading ID)
- [ ] Check: Message input field disabled (status not yet "open")
- [ ] Check: Send button disabled

### ✅ 2.2 Tab A Connects to WebSocket
- [ ] Wait 2-3 seconds
- [ ] Verify: Status badge changes from "connecting" to "open"
- [ ] Verify: Client ID assigned (format: "c-XXXXXX")
- [ ] Verify: Input field becomes enabled
- [ ] Verify: Send button becomes enabled
- [ ] Verify: Join message appears in message history with timestamp
- [ ] Verify: User count shows "1 online"

### ✅ 2.3 Tab B Loads and Connects
- [ ] Open Tab B at http://localhost:3000
- [ ] Wait 2-3 seconds
- [ ] Verify: Status shows "open"
- [ ] Verify: Client ID assigned (DIFFERENT from Tab A)
- [ ] Verify: "2 users online" appears in header
- [ ] Verify: Join event from Tab B shown in both tabs
- [ ] Verify: Timestamps are human-readable (HH:MM:SS AM/PM format)

### ✅ 2.4 Client List Synchronized
- [ ] Tab A: Click recipient dropdown
- [ ] Verify: Shows "Everyone" option
- [ ] Verify: Shows Tab B's client ID as option
- [ ] Verify: Tab B's client ID shows in dropdown (not marked "you")
- [ ] Tab B: Click recipient dropdown
- [ ] Verify: Shows "Everyone" option
- [ ] Verify: Shows Tab A's client ID as option
- [ ] Verify: Tab A's client ID shows in dropdown (not marked "you")

---

## Phase 3: Broadcast Messaging

### ✅ 3.1 Send Broadcast from Tab A
- [ ] Tab A: Keep recipient as "Everyone"
- [ ] Tab A: Type "Hello from Tab A" in input
- [ ] Verify: Send button is enabled
- [ ] Tab A: Click Send button
- [ ] Verify: Input field clears
- [ ] Verify: Message appears in Tab A's message history
- [ ] Verify: Format is "[c]: Hello from Tab A" (c = first letter of client ID)
- [ ] Verify: Timestamp shows (HH:MM:SS AM/PM)

### ✅ 3.2 Broadcast Received in Tab B
- [ ] Verify: Same message appears in Tab B
- [ ] Verify: Message content identical
- [ ] Verify: Timestamp identical
- [ ] Verify: Message format identical

### ✅ 3.3 Send Broadcast from Tab B
- [ ] Tab B: Keep recipient as "Everyone"
- [ ] Tab B: Type "Response from Tab B" in input
- [ ] Tab B: Click Send button
- [ ] Verify: Message appears in Tab B's history
- [ ] Verify: Message appears in Tab A's history immediately
- [ ] Verify: Format includes correct client ID initial

### ✅ 3.4 Message Format Validation
- [ ] All broadcast messages format: "[X]: message text"
- [ ] All have valid timestamps (HH:MM:SS AM/PM)
- [ ] No duplicate messages
- [ ] Messages in correct chronological order

---

## Phase 4: Private Messaging

### ✅ 4.1 Send Private Message from Tab B to Tab A
- [ ] Tab B: Click recipient dropdown
- [ ] Tab B: Select Tab A's client ID
- [ ] Verify: Dropdown now shows Tab A's ID (selected)
- [ ] Tab B: Type "Private message to Tab A" in input
- [ ] Tab B: Click Send button
- [ ] Verify: Message appears in Tab B's history
- [ ] Verify: Message format includes "(private)" tag
- [ ] Verify: Message shown in purple or italic text
- [ ] Verify: Format: "[c] (private): Private message to Tab A"

### ✅ 4.2 Private Message NOT Visible to Recipient (Tab A)
- [ ] Tab A: Check message history
- [ ] Verify: Private message from Tab B NOT visible
- [ ] Verify: No new messages since last broadcast
- [ ] Verify: Only heartbeats and previous broadcasts visible

### ✅ 4.3 Private Message Visible ONLY to Sender
- [ ] Tab B: Message visible in Tab B's history ✅
- [ ] Tab A: Message NOT visible in Tab A's history ✅
- [ ] Privacy constraint enforced at server level ✅

### ✅ 4.4 Send Private Message from Tab A to Tab B
- [ ] Tab A: Click recipient dropdown
- [ ] Tab A: Select Tab B's client ID
- [ ] Tab A: Type "Private to Tab B" in input
- [ ] Tab A: Click Send button
- [ ] Verify: Message appears in Tab A with (private) tag
- [ ] Verify: Message appears in Tab B
- [ ] Verify: NOT visible to other clients (if any)

### ✅ 4.5 Switch Back to Broadcast
- [ ] Tab A: Click recipient dropdown
- [ ] Tab A: Select "Everyone"
- [ ] Tab A: Type "Public broadcast" in input
- [ ] Tab A: Click Send button
- [ ] Verify: Message appears in both Tab A and Tab B
- [ ] Verify: Format is "[c]: Public broadcast" (NO private tag)
- [ ] Verify: No purple/italic formatting

---

## Phase 5: Heartbeat & Auto-Updates

### ✅ 5.1 Heartbeat Message Appears
- [ ] Wait for at least 10 seconds
- [ ] Verify: Message appears in both tabs
- [ ] Verify: Format is "💓 [Heartbeat] X users online"
- [ ] Verify: User count accurate (should be 2)
- [ ] Verify: Timestamp present (HH:MM:SS AM/PM)

### ✅ 5.2 Heartbeat Regular Interval
- [ ] Note timestamp of first heartbeat
- [ ] Wait for second heartbeat
- [ ] Verify: Approximately 10 seconds apart
- [ ] Verify: Third heartbeat arrives ~10s after second
- [ ] Acceptable variance: ±1 second

### ✅ 5.3 User Count in Heartbeat Updates
- [ ] With 2 tabs: Heartbeat shows "2 users online"
- [ ] Close one tab (simulate user leaving)
- [ ] Wait for next heartbeat
- [ ] Verify: Shows "1 users online"
- [ ] Note: Grammar could be improved but functionality correct

---

## Phase 6: Join & Leave Events

### ✅ 6.1 Join Event Format
- [ ] When Tab B connected, check message history
- [ ] Verify: Message format "→ User c joined (2 online)"
- [ ] Verify: Shows user count at time of join
- [ ] Verify: Displayed in green or distinct color
- [ ] Verify: Has valid timestamp

### ✅ 6.2 Leave Event Detection
- [ ] Close Tab B
- [ ] Tab A: Check message history
- [ ] Verify: Leave message appears "← User c left (1 online)"
- [ ] Verify: User count decrements correctly
- [ ] Verify: Displayed in orange or distinct color
- [ ] Verify: Timestamp present

### ✅ 6.3 Client List Updates on Leave
- [ ] Tab A: Click recipient dropdown
- [ ] Verify: Tab B's client ID no longer in list
- [ ] Verify: Only "Everyone" option available
- [ ] Verify: User count in header updated to "1 online"

### ✅ 6.4 Reconnect Shows New Join
- [ ] Close Tab B completely
- [ ] Open new Tab B at http://localhost:3000
- [ ] Tab B: Connects with NEW client ID (different from before)
- [ ] Verify: Tab A sees new join message
- [ ] Verify: New client ID in dropdown
- [ ] Verify: "2 users online" restored

---

## Phase 7: Input Validation & Button States

### ✅ 7.1 Send Button Disabled - Empty Input
- [ ] Tab A: Click input field
- [ ] Verify: Input is empty
- [ ] Verify: Send button is DISABLED
- [ ] Verify: Button appears greyed out

### ✅ 7.2 Send Button Enabled - With Text
- [ ] Tab A: Type any message (e.g., "test")
- [ ] Verify: Send button becomes ENABLED
- [ ] Verify: Button is clickable (not greyed out)

### ✅ 7.3 Whitespace Validation
- [ ] Tab A: Type only spaces "     " in input
- [ ] Verify: Send button is DISABLED
- [ ] Verify: Button does NOT enable for whitespace-only input
- [ ] Tab A: Add non-space character "    x"
- [ ] Verify: Send button becomes ENABLED
- [ ] Tab A: Remove non-space character back to spaces
- [ ] Verify: Send button becomes DISABLED again

### ✅ 7.4 Input Field Disabled When Offline
- [ ] Verify: Input field enabled when status="open" ✅
- [ ] Stop backend server (Ctrl+C in backend terminal)
- [ ] Verify: Status changes to "closed"
- [ ] Verify: Input field becomes DISABLED
- [ ] Verify: Send button becomes DISABLED
- [ ] Restart backend: `go run .`
- [ ] Verify: After reconnect, status="open"
- [ ] Verify: Input field becomes ENABLED again

### ✅ 7.5 Input Clears After Send
- [ ] Tab A: Type "test message" in input
- [ ] Tab A: Click Send button
- [ ] Verify: Input field is now EMPTY
- [ ] Verify: Send button is DISABLED (because input is empty)

---

## Phase 8: Error Handling

### ✅ 8.1 Error Display - Backend Down
- [ ] Backend running ✅
- [ ] Frontend connected ✅
- [ ] Stop backend: Ctrl+C in backend terminal
- [ ] Wait 1-2 seconds
- [ ] Tab A: Check for error message display
- [ ] Verify: Error message visible (red background card)
- [ ] Verify: Text contains error info (e.g., "WebSocket connection error")
- [ ] Verify: Status shows "closed" then "reconnecting"

### ✅ 8.2 Error Clears on Reconnect
- [ ] Backend still stopped ✅
- [ ] Restart backend: `go run .`
- [ ] Tab A: Wait for reconnection (exponential backoff)
- [ ] Verify: Status changes to "open"
- [ ] Verify: Error message disappears
- [ ] Verify: Input field becomes enabled again

### ✅ 8.3 No Duplicate Messages on Reconnect
- [ ] Note the message count in Tab A before stopping backend
- [ ] Stop backend
- [ ] Wait 2-3 seconds
- [ ] Restart backend
- [ ] Wait for reconnect
- [ ] Verify: No duplicate messages appear
- [ ] Verify: Messages in same order as before

---

## Phase 9: Reconnection & Backoff

### ✅ 9.1 Exponential Backoff Times
- [ ] Note current time
- [ ] Stop backend
- [ ] Tab A: Watch status change to "closed" then "reconnecting"
- [ ] Stop Tab A from reconnecting: Keep backend off
- [ ] Backend stops: Should retry at 1s delay
- [ ] Restart backend at T+0s
- [ ] Wait to see connection retry at approximately:
  - [ ] 1st attempt: ~0s (immediate after backend available)
  - [ ] Status returns to "open" within 1-2 seconds
- [ ] Test sequence:
  1. Stop backend at time T0
  2. Note attempt times in console logs if available
  3. Should follow pattern: 1s, 2s, 4s, 8s, 10s (max)

### ✅ 9.2 Client ID Persistence
- [ ] Tab A: Note current client ID (e.g., "c-372956")
- [ ] Tab A: Refresh page (F5 or Cmd+R)
- [ ] Wait for reconnection (~2-3 seconds)
- [ ] Verify: SAME client ID appears in "Your ID" field
- [ ] Verify: localStorage preserved the ID
- [ ] Verify: Backend does NOT see duplicate connection (still "2 users online" with Tab B)

### ✅ 9.3 Reconnect After Page Refresh
- [ ] Tab A: Page is refreshed
- [ ] Verify: Status shows "connecting" initially
- [ ] Verify: Status changes to "open" within 2-3 seconds
- [ ] Verify: Tab B still sees "2 users online"
- [ ] Verify: No join duplicates in message history

---

## Phase 10: UI/UX Features

### ✅ 10.1 Auto-Scroll to Latest Message
- [ ] Tab A: Send multiple messages ("msg1", "msg2", "msg3")
- [ ] Verify: Message list auto-scrolls to show newest message
- [ ] Verify: Latest message visible without manual scroll
- [ ] Scroll up in message history manually
- [ ] Tab B: Send a message
- [ ] Verify: Tab A auto-scrolls back to bottom to show new message
- [ ] Verify: Smooth scroll animation (not jarring jump)

### ✅ 10.2 Recipient Dropdown Styling
- [ ] Tab A: Click recipient dropdown
- [ ] Verify: Dropdown expands showing options
- [ ] Verify: Current selection highlighted
- [ ] Verify: Other clients listed below "Everyone"
- [ ] Tab A: Select different recipient
- [ ] Verify: Dropdown shows new selection
- [ ] Verify: Next message sent to new recipient

### ✅ 10.3 Message History Display
- [ ] Tab A: Scroll through message history
- [ ] Verify: Messages properly formatted
- [ ] Verify: Timestamps readable
- [ ] Verify: Color coding works:
  - [ ] Broadcast: Dark color (default)
  - [ ] Private: Purple/Italic
  - [ ] Join: Green
  - [ ] Leave: Orange
  - [ ] Heartbeat: Gray

### ✅ 10.4 Status Badge Colors
- [ ] Connecting: Secondary color (gray)
- [ ] Open: Default color (blue/green)
- [ ] Closed: Destructive color (red)
- [ ] Reconnecting: Secondary color (gray)

### ✅ 10.5 Responsive Layout
- [ ] Tab A: View on desktop width
- [ ] Verify: Layout looks good, no overflow
- [ ] Verify: All controls visible and usable
- [ ] Resize browser to smaller width
- [ ] Verify: Layout still functional (may stack but not broken)

---

## Phase 11: Message Format Validation

### ✅ 11.1 Broadcast Message Format
- [ ] Send any broadcast message
- [ ] Verify format: `[X]: message text`
- [ ] Where X = first letter of client ID
- [ ] Verify: Timestamp in [HH:MM:SS AM/PM]
- [ ] Verify: No extra brackets or formatting

### ✅ 11.2 Private Message Format
- [ ] Send any private message
- [ ] Verify format: `[X] (private): message text`
- [ ] Verify: "(private)" tag present
- [ ] Verify: Displayed in purple/italic
- [ ] Verify: Timestamp correct

### ✅ 11.3 Join Message Format
- [ ] Close and reopen tab to trigger join
- [ ] Verify format: `→ User X joined (N online)`
- [ ] Where X = first letter of client ID
- [ ] Where N = total users including new one
- [ ] Verify: Green or distinct color
- [ ] Verify: Arrow symbol visible

### ✅ 11.4 Leave Message Format
- [ ] Close a tab to trigger leave
- [ ] Verify format: `← User X left (N online)`
- [ ] Where N = total users after departure
- [ ] Verify: Orange or distinct color
- [ ] Verify: Arrow symbol visible

### ✅ 11.5 Heartbeat Format
- [ ] Wait for heartbeat
- [ ] Verify format: `💓 [Heartbeat] N users online`
- [ ] Verify: Heart emoji present
- [ ] Verify: User count accurate
- [ ] Verify: Gray color or distinct styling

---

## Phase 12: Performance & Stress Tests

### ✅ 12.1 Rapid Message Sending
- [ ] Tab A: Send 5 messages in quick succession
- [ ] Verify: All messages received
- [ ] Verify: No messages lost
- [ ] Verify: Correct order maintained
- [ ] Verify: Timestamps accurate and increasing

### ✅ 12.2 Multiple Connections
- [ ] Open Tab C at http://localhost:3000
- [ ] Verify: Status shows "3 users online"
- [ ] Tab A: Send broadcast
- [ ] Verify: Appears in Tab B and Tab C
- [ ] Tab B: Send private to Tab C
- [ ] Verify: NOT in Tab A, visible in Tab B and Tab C
- [ ] Close Tab C
- [ ] Verify: Leave message, count back to 2 users

### ✅ 12.3 No Memory Leaks / Crashes
- [ ] Let system run with 2-3 tabs for 2+ minutes
- [ ] Send messages occasionally
- [ ] Backend console: No error output
- [ ] Frontend console (F12): No error messages
- [ ] Browser: Page remains responsive
- [ ] No visible slowdown or memory spike

### ✅ 12.4 Large Message Text
- [ ] Tab A: Type a long message (500+ characters)
- [ ] Verify: Sends without truncation
- [ ] Verify: Received fully in other tabs
- [ ] Verify: Formatting preserved
- [ ] Verify: Timestamps correct

---

## Phase 13: Cross-Browser Compatibility (Optional)

### ✅ 13.1 Chrome/Chromium
- [ ] Open both tabs in Chrome
- [ ] Run all messaging tests
- [ ] Verify: All functionality works

### ✅ 13.2 Firefox (if available)
- [ ] Open both tabs in Firefox
- [ ] Run messaging tests
- [ ] Verify: All functionality works

### ✅ 13.3 Safari (if on macOS)
- [ ] Open both tabs in Safari
- [ ] Run messaging tests
- [ ] Verify: All functionality works

---

## Phase 14: TypeScript/Build Validation

### ✅ 14.1 TypeScript Compilation
- [ ] Run: `npm run typecheck`
- [ ] Expected: No errors, clean output
- [ ] Verify: All types correctly defined
- [ ] Verify: No implicit any types

### ✅ 14.2 Build Production
- [ ] Run: `npm run build`
- [ ] Expected: No errors
- [ ] Verify: .next/ directory created
- [ ] Verify: No type errors in build output

### ✅ 14.3 Go Build
- [ ] Run: `cd backend && go build -o websocket-server .`
- [ ] Expected: No errors
- [ ] Verify: Binary created with correct permissions
- [ ] Verify: Binary size reasonable (5-20 MB typical)

---

## Phase 15: Cleanup & Reset

### ✅ 15.1 Close All Connections
- [ ] Close all browser tabs
- [ ] Verify: Backend logs show disconnect messages
- [ ] Verify: No errors in backend console

### ✅ 15.2 Restart Fresh
- [ ] Kill backend (Ctrl+C)
- [ ] Kill frontend (Ctrl+C)
- [ ] Clear browser cache (optional)
- [ ] Start backend: `cd backend && go run .`
- [ ] Start frontend: `npm run dev`
- [ ] Open new tab at localhost:3000
- [ ] Verify: Clean connection, no errors

---

## Quick Test Sequence (5 minutes)

Use this for quick smoke tests:

1. **Start servers** (1 min)
   - [ ] Backend: `cd backend && go run .`
   - [ ] Frontend: `npm run dev`

2. **Open tabs** (1 min)
   - [ ] Tab A: localhost:3000 - verify "open" status
   - [ ] Tab B: localhost:3000 - verify "2 users online"

3. **Test messaging** (2 min)
   - [ ] Tab A broadcast: appears in Tab B ✅
   - [ ] Tab B private to Tab A: NOT in Tab A ✅
   - [ ] Tab A broadcast: appears in Tab B ✅

4. **Verify features** (1 min)
   - [ ] Recipient dropdown works ✅
   - [ ] Send button disabled when empty ✅
   - [ ] Heartbeat appears ✅
   - [ ] Timestamps human-readable ✅

---

## Debugging Guide

### Backend Issues

**Problem: "Port 8080 already in use"**
- Solution: `lsof -i :8080` and kill the process
- Alternative: Change port in main.go and recompile

**Problem: "WebSocket connection error" in frontend**
- Solution: Verify backend is running on :8080
- Check firewall: `sudo ufw allow 8080`

**Problem: Messages not reaching other tabs**
- Solution: Check hub.Broadcast channel in hub.go
- Verify no panics in backend logs

### Frontend Issues

**Problem: "Input field stays disabled after backend restart"**
- Solution: Manually refresh page (F5)
- Or wait for next page interaction

**Problem: "Messages show wrong timestamps"**
- Solution: Check getCurrentTimestamp() in backend main.go
- Verify system clock is correct

**Problem: "Private messages visible to wrong clients"**
- Solution: Verify hub.go routing logic
- Check payload["recipientId"] extraction

### Connection Issues

**Problem: "Auto-reconnect not working"**
- Solution: Check exponential backoff in useWebSocket.ts
- Verify setTimeout is not being cleared prematurely

**Problem: "Client ID not persisting after refresh"**
- Solution: Check localStorage.setItem("websocket-clientid", ...)
- Verify localStorage is not disabled in browser

---

## Notes

- Timestamps should match system time
- Client IDs are random 6-digit numbers (c-XXXXXX format)
- Message delay should be <100ms in local development
- Heartbeat is broadcast every ~10 seconds (±1s variance acceptable)
- Private messages are end-to-end at server level (no third-party visibility)

---

## Test Results Tracking

Date: _____________
Tester: _____________
Build: _____________

| Test Phase | Status | Notes |
|------------|--------|-------|
| 1. Backend Verification | ☐ Pass ☐ Fail | |
| 2. Frontend Connection | ☐ Pass ☐ Fail | |
| 3. Broadcast Messaging | ☐ Pass ☐ Fail | |
| 4. Private Messaging | ☐ Pass ☐ Fail | |
| 5. Heartbeat & Updates | ☐ Pass ☐ Fail | |
| 6. Join & Leave Events | ☐ Pass ☐ Fail | |
| 7. Input Validation | ☐ Pass ☐ Fail | |
| 8. Error Handling | ☐ Pass ☐ Fail | |
| 9. Reconnection | ☐ Pass ☐ Fail | |
| 10. UI/UX Features | ☐ Pass ☐ Fail | |
| 11. Message Formats | ☐ Pass ☐ Fail | |
| 12. Performance | ☐ Pass ☐ Fail | |

**Overall Result**: ☐ Pass ☐ Fail

**Issues Found**:
- 
- 
- 

**Signed Off By**: _________________ Date: _____________
