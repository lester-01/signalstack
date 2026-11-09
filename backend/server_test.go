package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

const testToken = "test-token"

func testConfig() Config {
	return Config{
		Addr:           ":0",
		AuthToken:      testToken,
		AllowedOrigins: []string{"http://localhost:3000"},
		SessionSecret:  []byte("test-secret"),
	}
}

func startServer(t *testing.T, cfg Config) (*Server, string) {
	t.Helper()
	s := NewServer(cfg)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(s.hub.Close)
	return s, "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
}

// has reports whether a client with this ID is registered.
func (h *Hub) has(id string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.byID[id]
	return ok
}

// testClient is a WebSocket peer whose reader never blocks: messages go to
// inbox and anything beyond its capacity is dropped.
type testClient struct {
	conn   *websocket.Conn
	id     string
	resume string
	inbox  chan Message
	done   chan struct{}
	err    error // valid once done is closed
}

func dialClient(url string) (*testClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	c := &testClient{conn: conn, inbox: make(chan Message, 8192), done: make(chan struct{})}
	go func() {
		defer close(c.done)
		for {
			var m Message
			if err := conn.ReadJSON(&m); err != nil {
				c.err = err
				return
			}
			select {
			case c.inbox <- m:
			default:
			}
		}
	}()
	return c, nil
}

func connect(t *testing.T, url string) *testClient {
	t.Helper()
	c, err := dialClient(url)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { c.conn.Close() })
	return c
}

func ofType(typ string) func(Message) bool {
	return func(m Message) bool { return m.Type == typ }
}

func (c *testClient) send(m Message) error { return c.conn.WriteJSON(m) }

func (c *testClient) auth(token, id, resume string) error {
	p := map[string]interface{}{"token": token, "clientType": "test"}
	if id != "" {
		p["clientId"] = id
	}
	if resume != "" {
		p["resumeToken"] = resume
	}
	return c.send(Message{Type: MessageTypeAuth, Payload: p})
}

func (c *testClient) chat(recipient, text string) error {
	return c.send(Message{Type: MessageTypeMessage, Payload: map[string]interface{}{"text": text, "recipientId": recipient}})
}

// expect waits for a message matching match.
func (c *testClient) expect(match func(Message) bool, d time.Duration) (Message, bool) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		select {
		case m := <-c.inbox:
			if match(m) {
				return m, true
			}
		case <-timer.C:
			return Message{}, false
		case <-c.done:
			for {
				select {
				case m := <-c.inbox:
					if match(m) {
						return m, true
					}
				default:
					return Message{}, false
				}
			}
		}
	}
}

func (c *testClient) waitClosed(d time.Duration) bool {
	select {
	case <-c.done:
		return true
	case <-time.After(d):
		return false
	}
}

// finishAuth waits for a successful auth_response and records the identity.
func (c *testClient) finishAuth(t *testing.T) {
	t.Helper()
	m, ok := c.expect(ofType(MessageTypeAuthResponse), 2*time.Second)
	if !ok {
		t.Fatal("no auth_response received")
	}
	if success, _ := m.Payload["success"].(bool); !success {
		t.Fatalf("auth rejected: %v", m.Payload["reason"])
	}
	c.id = m.ClientID
	c.resume, _ = m.Payload["resumeToken"].(string)
}

func login(t *testing.T, url string) *testClient {
	t.Helper()
	c := connect(t, url)
	if err := c.auth(testToken, "", ""); err != nil {
		t.Fatalf("auth write: %v", err)
	}
	c.finishAuth(t)
	return c
}

func eventually(t *testing.T, d time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", what)
}

func TestRejectedAuthIsDelivered(t *testing.T) {
	_, url := startServer(t, testConfig())
	cases := []struct{ token, reason string }{{"wrong", "invalid token"}, {"", "missing token"}}
	for _, tc := range cases {
		for i := 0; i < 25; i++ {
			c := connect(t, url)
			if err := c.auth(tc.token, "", ""); err != nil {
				t.Fatal(err)
			}
			m, ok := c.expect(ofType(MessageTypeAuthResponse), 2*time.Second)
			if !ok {
				t.Fatalf("attempt %d (%s): rejection reason never arrived", i, tc.reason)
			}
			if success, _ := m.Payload["success"].(bool); success || m.Payload["reason"] != tc.reason {
				t.Fatalf("unexpected payload: %v", m.Payload)
			}
			if !c.waitClosed(2 * time.Second) {
				t.Fatal("connection was not closed after rejection")
			}
		}
	}
}

func TestServerIssuesIDsAndRejectsSpoofing(t *testing.T) {
	s, url := startServer(t, testConfig())
	victim := login(t, url)

	attacker := connect(t, url)
	if err := attacker.auth(testToken, victim.id, "not-a-valid-token"); err != nil {
		t.Fatal(err)
	}
	attacker.finishAuth(t)

	if attacker.id == victim.id {
		t.Fatal("attacker claimed the victim's ID without a valid resume token")
	}
	if !strings.HasPrefix(attacker.id, "c-") {
		t.Fatalf("unexpected ID format %q", attacker.id)
	}
	eventually(t, 2*time.Second, "both clients registered under distinct IDs", func() bool {
		return len(s.hub.clientIDs()) == 2
	})
}

func TestResumeTokenReclaimsIDAndReplacesOldConnection(t *testing.T) {
	s, url := startServer(t, testConfig())
	first := login(t, url)

	second := connect(t, url)
	if err := second.auth(testToken, first.id, first.resume); err != nil {
		t.Fatal(err)
	}
	second.finishAuth(t)

	if second.id != first.id {
		t.Fatalf("resume gave %q, want %q", second.id, first.id)
	}
	if !first.waitClosed(2 * time.Second) {
		t.Fatal("old connection with the same ID was not closed")
	}
	eventually(t, 2*time.Second, "exactly one client registered", func() bool {
		return len(s.hub.clientIDs()) == 1 && s.hub.has(first.id)
	})
}

func TestMessageRouting(t *testing.T) {
	_, url := startServer(t, testConfig())
	a, b, c := login(t, url), login(t, url), login(t, url)
	isMsg := ofType(MessageTypeMessage)

	if err := a.chat(b.id, "secret"); err != nil {
		t.Fatal(err)
	}
	for name, cl := range map[string]*testClient{"sender": a, "recipient": b} {
		m, ok := cl.expect(isMsg, 2*time.Second)
		if !ok || m.Payload["text"] != "secret" || m.ClientID != a.id {
			t.Fatalf("%s did not get the private message (got %+v, ok=%v)", name, m, ok)
		}
	}
	if m, ok := c.expect(isMsg, 300*time.Millisecond); ok {
		t.Fatalf("bystander received a private message: %+v", m)
	}

	if err := a.chat(RecipientAll, "hello"); err != nil {
		t.Fatal(err)
	}
	for name, cl := range map[string]*testClient{"a": a, "b": b, "c": c} {
		if m, ok := cl.expect(isMsg, 2*time.Second); !ok || m.Payload["text"] != "hello" {
			t.Fatalf("%s missed the broadcast", name)
		}
	}
}

func TestPresenceEvents(t *testing.T) {
	_, url := startServer(t, testConfig())
	a := login(t, url)
	b := login(t, url)

	if _, ok := a.expect(func(m Message) bool { return m.Type == MessageTypeJoin && m.ClientID == b.id }, 2*time.Second); !ok {
		t.Fatal("a never saw b join")
	}
	b.conn.Close()
	m, ok := a.expect(func(m Message) bool { return m.Type == MessageTypeLeave && m.ClientID == b.id }, 2*time.Second)
	if !ok {
		t.Fatal("a never saw b leave")
	}
	if n, _ := m.Payload["userCount"].(float64); n != 1 {
		t.Fatalf("userCount after leave = %v, want 1", m.Payload["userCount"])
	}
}

func TestInvalidChatIsDropped(t *testing.T) {
	_, url := startServer(t, testConfig())
	a := login(t, url)
	for _, text := range []string{"", "   ", strings.Repeat("x", maxTextRunes+1)} {
		if err := a.chat(RecipientAll, text); err != nil {
			t.Fatal(err)
		}
	}
	if m, ok := a.expect(ofType(MessageTypeMessage), 300*time.Millisecond); ok {
		t.Fatalf("invalid message was delivered: %+v", m)
	}

	// A client that never authenticated cannot chat.
	anon := connect(t, url)
	if err := anon.chat(RecipientAll, "hi"); err != nil {
		t.Fatal(err)
	}
	if m, ok := a.expect(ofType(MessageTypeMessage), 300*time.Millisecond); ok {
		t.Fatalf("unauthenticated message was delivered: %+v", m)
	}
}

func TestOversizedFrameClosesConnection(t *testing.T) {
	_, url := startServer(t, testConfig())
	a := login(t, url)
	_ = a.chat(RecipientAll, strings.Repeat("x", 4*maxMessageSize))
	if !a.waitClosed(2 * time.Second) {
		t.Fatal("connection stayed open after an oversized frame")
	}
}

func TestOriginCheck(t *testing.T) {
	_, url := startServer(t, testConfig())

	_, resp, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": {"http://evil.example"}})
	if err == nil {
		t.Fatal("connection from a disallowed origin was accepted")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 for a disallowed origin, got %v", resp)
	}

	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": {"http://localhost:3000"}})
	if err != nil {
		t.Fatalf("allowed origin rejected: %v", err)
	}
	conn.Close()
}

func TestUnauthenticatedCloseDoesNotLeakGoroutines(t *testing.T) {
	_, url := startServer(t, testConfig())
	time.Sleep(100 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	for i := 0; i < 100; i++ {
		c, err := dialClient(url)
		if err != nil {
			t.Fatal(err)
		}
		c.conn.Close()
	}
	eventually(t, 3*time.Second, "server goroutines for closed, unauthenticated connections to exit", func() bool {
		return runtime.NumGoroutine() <= baseline+5
	})
}

func TestSlowConsumerIsEvictedWithoutBlockingOthers(t *testing.T) {
	// Shrink the server-side socket buffer of the one connection we mark, so a
	// stalled reader backs up quickly and deterministically.
	var shrinkNext atomic.Bool
	s := NewServer(testConfig())
	ts := httptest.NewUnstartedServer(s.Handler())
	ts.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		if state == http.StateNew && shrinkNext.CompareAndSwap(true, false) {
			if tcp, ok := conn.(*net.TCPConn); ok {
				_ = tcp.SetWriteBuffer(2048)
			}
		}
	}
	ts.Start()
	t.Cleanup(ts.Close)
	t.Cleanup(s.hub.Close)
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	// The slow client authenticates, then never reads again.
	shrinkNext.Store(true)
	slowConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { slowConn.Close() })
	if tcp, ok := slowConn.UnderlyingConn().(*net.TCPConn); ok {
		_ = tcp.SetReadBuffer(1024)
	}
	if err := slowConn.WriteJSON(Message{Type: MessageTypeAuth, Payload: map[string]interface{}{"token": testToken}}); err != nil {
		t.Fatal(err)
	}
	slowID := ""
	_ = slowConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for slowID == "" {
		var m Message
		if err := slowConn.ReadJSON(&m); err != nil {
			t.Fatalf("slow client auth: %v", err)
		}
		if m.Type == MessageTypeAuthResponse {
			slowID = m.ClientID
		}
	}

	sender := login(t, url)
	text := strings.Repeat("x", 1000)
	evicted := false
	for i := 0; i < 20000 && !evicted; i++ {
		if err := sender.chat(RecipientAll, text); err != nil {
			t.Fatalf("sender write failed: %v", err)
		}
		if i%10 == 0 {
			time.Sleep(time.Millisecond)
			evicted = !s.hub.has(slowID)
		}
	}
	if !evicted {
		t.Fatal("slow client was never evicted")
	}
	if !s.hub.has(sender.id) {
		t.Fatal("healthy sender was evicted along with the slow client")
	}

	// The hub must still serve new clients.
	late := login(t, url)
	if err := late.chat(RecipientAll, "still alive"); err != nil {
		t.Fatal(err)
	}
	if _, ok := late.expect(func(m Message) bool { return m.Type == MessageTypeMessage && m.Payload["text"] == "still alive" }, 2*time.Second); !ok {
		t.Fatal("hub stopped delivering after evicting a slow client")
	}
}

func TestShutdownSendsCloseFrame(t *testing.T) {
	s, url := startServer(t, testConfig())
	a := login(t, url)

	s.hub.Close()

	if !a.waitClosed(2 * time.Second) {
		t.Fatal("client was not disconnected on shutdown")
	}
	if !websocket.IsCloseError(a.err, websocket.CloseNormalClosure) {
		t.Fatalf("want a normal-closure frame, got %v", a.err)
	}
}

// TestChurn is mainly useful under `go test -race`: many clients connect,
// chat publicly and privately, and disconnect concurrently.
func TestChurn(t *testing.T) {
	s, url := startServer(t, testConfig())
	var (
		wg   sync.WaitGroup
		ids  sync.Map
		stop = time.Now().Add(2 * time.Second)
	)
	for w := 0; w < 20; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(stop) {
				c, err := dialClient(url)
				if err != nil {
					return
				}
				if c.auth(testToken, "", "") != nil {
					c.conn.Close()
					continue
				}
				m, ok := c.expect(ofType(MessageTypeAuthResponse), 2*time.Second)
				if !ok {
					c.conn.Close()
					continue
				}
				ids.Store(m.ClientID, struct{}{})
				for i := 0; i < 10; i++ {
					target := RecipientAll
					if i%3 == 0 {
						ids.Range(func(k, _ interface{}) bool { target = k.(string); return false })
					}
					_ = c.chat(target, "hi")
					time.Sleep(2 * time.Millisecond)
				}
				c.conn.Close()
				<-c.done
			}
		}()
	}
	wg.Wait()

	eventually(t, 3*time.Second, "all clients to unregister", func() bool {
		return len(s.hub.clientIDs()) == 0 && len(s.hub.snapshot()) == 0
	})
}
