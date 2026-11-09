package main

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func httpBase(wsURL string) string {
	return "http" + strings.TrimPrefix(strings.TrimSuffix(wsURL, "/ws"), "ws")
}

// scrape fetches /metrics and returns each series (name plus labels) with its value.
func scrape(t *testing.T, wsURL string) map[string]float64 {
	t.Helper()
	resp, err := http.Get(httpBase(wsURL) + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/metrics status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	out := map[string]float64{}
	for _, line := range strings.Split(string(body), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.LastIndex(line, " ")
		v, err := strconv.ParseFloat(line[i+1:], 64)
		if err != nil {
			t.Fatalf("bad metrics line %q", line)
		}
		out[line[:i]] = v
	}
	return out
}

func TestMetricsEndpoint(t *testing.T) {
	cfg := testConfig()
	cfg.MetricsEnabled = true
	_, url := startServer(t, cfg)

	a := login(t, url)
	bad := connect(t, url)
	if err := bad.auth("wrong", "", ""); err != nil {
		t.Fatal(err)
	}
	bad.waitClosed(2 * time.Second)

	if err := a.chat(RecipientAll, "hi"); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.expect(ofType(MessageTypeMessage), 2*time.Second); !ok {
		t.Fatal("message was not delivered")
	}

	checks := map[string]float64{
		"signalstack_connected_clients":                        1,
		"signalstack_connections_total":                        1,
		`signalstack_auth_total{result="success"}`:             1,
		`signalstack_auth_total{result="failure"}`:             1,
		"signalstack_messages_received_total":                  1,
		"signalstack_messages_delivered_total":                 1,
		"signalstack_evictions_total":                          0,
		`signalstack_messages_dropped_total{reason="invalid"}`: 0,
	}
	eventually(t, 2*time.Second, "metrics to reflect the activity", func() bool {
		m := scrape(t, url)
		for name, want := range checks {
			if m[name] != want {
				return false
			}
		}
		return m["go_goroutines"] > 0
	})
}

func TestMetricsDisabledByDefault(t *testing.T) {
	_, url := startServer(t, testConfig())
	resp, err := http.Get(httpBase(url) + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("/metrics status = %d, want 404 when disabled", resp.StatusCode)
	}
}

func TestRateLimitDropsExcessMessages(t *testing.T) {
	cfg := testConfig()
	cfg.RateLimit = 5 // burst of 10
	s, url := startServer(t, cfg)
	a := login(t, url)

	const sent = 40
	for i := 0; i < sent; i++ {
		if err := a.chat(RecipientAll, "spam"); err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(300 * time.Millisecond)

	delivered, errs := 0, 0
	for drained := false; !drained; {
		select {
		case m := <-a.inbox:
			switch m.Type {
			case MessageTypeMessage:
				delivered++
			case MessageTypeError:
				if m.Payload["reason"] != "rate limited" {
					t.Fatalf("unexpected error payload: %v", m.Payload)
				}
				errs++
			}
		default:
			drained = true
		}
	}
	if errs == 0 {
		t.Fatal("no rate-limit error was sent to the client")
	}
	if delivered < 10 || delivered > 14 {
		t.Fatalf("delivered %d of %d, want roughly the burst of 10", delivered, sent)
	}
	eventually(t, 2*time.Second, "dropped counter to account for the rest", func() bool {
		return int(s.hub.metrics.droppedRateLimited.Load())+delivered == sent
	})
}
