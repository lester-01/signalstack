// Command loadtest drives a running SignalStack server with many WebSocket
// clients and reports delivery latency, throughput and server resource use.
//
// Every client receives every broadcast, so one message sent to N clients is N
// deliveries. A fixed number of "sender" clients broadcast at a fixed
// aggregate rate; each message carries the time it was *scheduled* to be sent,
// and every client measures the gap to the moment it receives it. Stamping the
// scheduled time (not the actual write time) means a load generator that falls
// behind shows up as latency instead of silently lowering the offered load
// (coordinated omission). Sender and receivers share one process and clock, so
// no clock synchronisation is needed even when the server is on another host.
//
// See docs/BENCHMARKING.md for methodology and caveats.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

const maxPayload = 900 // server caps chat text at 1000 characters; leave room for the stamp

type config struct {
	URL         string
	Token       string
	Origin      string
	MetricsURL  string
	Out         string
	Clients     int
	Senders     int
	Rate        float64
	Payload     int
	Duration    time.Duration
	Warmup      time.Duration
	ConnectRate float64
}

func parseFlags(args []string) (config, error) {
	var c config
	fs := flag.NewFlagSet("loadtest", flag.ContinueOnError)
	fs.StringVar(&c.URL, "url", "ws://localhost:8080/ws", "WebSocket URL of the server")
	fs.StringVar(&c.Token, "token", os.Getenv("AUTH_TOKEN"), "auth token (default: $AUTH_TOKEN)")
	fs.StringVar(&c.Origin, "origin", "", "Origin header to send (needed if the server restricts origins)")
	fs.StringVar(&c.MetricsURL, "metrics-url", "", "server /metrics URL; enables memory and server-side counters")
	fs.StringVar(&c.Out, "out", "", "write full results as JSON to this file")
	fs.IntVar(&c.Clients, "clients", 100, "number of concurrent clients")
	fs.IntVar(&c.Senders, "senders", 10, "how many of the clients send messages")
	fs.Float64Var(&c.Rate, "rate", 100, "aggregate broadcast messages per second across all senders")
	fs.IntVar(&c.Payload, "payload", 64, "padding bytes added to each message")
	fs.DurationVar(&c.Duration, "duration", 30*time.Second, "measurement window")
	fs.DurationVar(&c.Warmup, "warmup", 5*time.Second, "load applied before the window starts (not measured)")
	fs.Float64Var(&c.ConnectRate, "connect-rate", 100, "new connections per second while ramping up")
	if err := fs.Parse(args); err != nil {
		return c, err
	}
	switch {
	case c.Token == "":
		return c, errors.New("a token is required: pass -token or set AUTH_TOKEN")
	case c.Clients < 1:
		return c, errors.New("-clients must be at least 1")
	case c.Senders < 1 || c.Senders > c.Clients:
		return c, errors.New("-senders must be between 1 and -clients")
	case c.Rate <= 0:
		return c, errors.New("-rate must be positive")
	case c.Payload < 0 || c.Payload > maxPayload:
		return c, fmt.Errorf("-payload must be between 0 and %d", maxPayload)
	case c.Duration <= 0 || c.Warmup < 0:
		return c, errors.New("-duration must be positive and -warmup non-negative")
	case c.ConnectRate <= 0:
		return c, errors.New("-connect-rate must be positive")
	}
	return c, nil
}

// ---- message stamping ----

type stamp struct {
	sender, seq int
	intended    int64 // unix nanoseconds the message was scheduled to be sent
}

func buildText(sender, seq int, intended int64, padding int) string {
	return fmt.Sprintf("lt|%d|%d|%d|", sender, seq, intended) + strings.Repeat("x", padding)
}

func parseStamp(text string) (stamp, bool) {
	parts := strings.SplitN(text, "|", 5)
	if len(parts) != 5 || parts[0] != "lt" {
		return stamp{}, false
	}
	sender, e1 := strconv.Atoi(parts[1])
	seq, e2 := strconv.Atoi(parts[2])
	intended, e3 := strconv.ParseInt(parts[3], 10, 64)
	if e1 != nil || e2 != nil || e3 != nil {
		return stamp{}, false
	}
	return stamp{sender: sender, seq: seq, intended: intended}, true
}

// ---- measurement window ----

// window holds the unix-nano interval whose messages are measured. Messages
// are matched by their scheduled time, so warmup traffic and stragglers are
// excluded without any shared flag.
type window struct{ start, end atomic.Int64 }

func newWindow() *window {
	w := &window{}
	w.start.Store(math.MaxInt64)
	return w
}

// ---- clients ----

type inbound struct {
	Type    string `json:"type"`
	Payload struct {
		Text    string `json:"text"`
		Success bool   `json:"success"`
		Reason  string `json:"reason"`
	} `json:"payload"`
}

type loadClient struct {
	conn     *websocket.Conn
	hist     *Histogram // latency of in-window deliveries; owned by the reader goroutine
	received uint64     // in-window deliveries; owned by the reader goroutine
	done     chan struct{}
}

func (lc *loadClient) readLoop(win *window, stopping *atomic.Bool, disconnects *atomic.Int64) {
	defer close(lc.done)
	for {
		_, data, err := lc.conn.ReadMessage()
		if err != nil {
			if !stopping.Load() {
				disconnects.Add(1)
			}
			return
		}
		recvAt := time.Now().UnixNano()
		var m inbound
		if json.Unmarshal(data, &m) != nil || m.Type != "message" {
			continue
		}
		st, ok := parseStamp(m.Payload.Text)
		if !ok || st.intended < win.start.Load() || st.intended >= win.end.Load() {
			continue
		}
		lc.hist.Record(time.Duration(recvAt - st.intended))
		lc.received++
	}
}

func dialAndAuth(ctx context.Context, cfg config) (*loadClient, time.Duration, error) {
	hdr := http.Header{}
	if cfg.Origin != "" {
		hdr.Set("Origin", cfg.Origin)
	}
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	start := time.Now()
	conn, _, err := dialer.DialContext(ctx, cfg.URL, hdr)
	if err != nil {
		return nil, 0, fmt.Errorf("dial: %w", err)
	}
	auth := map[string]interface{}{
		"type":    "auth",
		"payload": map[string]interface{}{"token": cfg.Token, "clientType": "loadtest"},
	}
	if err := conn.WriteJSON(auth); err != nil {
		conn.Close()
		return nil, 0, fmt.Errorf("auth write: %w", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			conn.Close()
			return nil, 0, fmt.Errorf("waiting for auth_response: %w", err)
		}
		var m inbound
		if json.Unmarshal(data, &m) != nil || m.Type != "auth_response" {
			continue
		}
		if !m.Payload.Success {
			conn.Close()
			return nil, 0, fmt.Errorf("auth rejected: %s", m.Payload.Reason)
		}
		break
	}
	_ = conn.SetReadDeadline(time.Time{})
	return &loadClient{conn: conn, hist: NewHistogram(), done: make(chan struct{})}, time.Since(start), nil
}

// ---- senders ----

type senderStats struct {
	inWindow int64
	errors   int64
	maxLag   time.Duration // how late the latest write was versus its schedule
}

func runSender(ctx context.Context, lc *loadClient, idx int, first time.Time, interval time.Duration, winStart, winEnd int64, padding int, out *senderStats) {
	next := first
	for seq := 0; next.UnixNano() < winEnd; seq++ {
		if wait := time.Until(next); wait > 0 {
			t := time.NewTimer(wait)
			select {
			case <-t.C:
			case <-ctx.Done():
				t.Stop()
				return
			}
		}
		if ctx.Err() != nil {
			return
		}
		if lag := time.Since(next); lag > out.maxLag {
			out.maxLag = lag
		}
		text := buildText(idx, seq, next.UnixNano(), padding)
		frame := `{"type":"message","payload":{"text":"` + text + `","recipientId":"all"}}`
		if err := lc.conn.WriteMessage(websocket.TextMessage, []byte(frame)); err != nil {
			out.errors++
			return
		}
		if next.UnixNano() >= winStart {
			out.inWindow++
		}
		next = next.Add(interval)
	}
}

// ---- server metrics ----

func parseMetrics(body string) (map[string]float64, error) {
	out := map[string]float64{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.LastIndex(line, " ")
		if i < 0 {
			return nil, fmt.Errorf("malformed metrics line %q", line)
		}
		v, err := strconv.ParseFloat(line[i+1:], 64)
		if err != nil {
			return nil, fmt.Errorf("malformed metrics line %q", line)
		}
		out[line[:i]] = v
	}
	return out, nil
}

func scrape(ctx context.Context, url string) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %s (start the server with METRICS_ENABLED=true)", url, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseMetrics(string(body))
}

// ---- results ----

type latencyMS struct {
	Mean, P50, P90, P99, P999, Max float64
}

type result struct {
	Config      map[string]interface{} `json:"config"`
	Environment map[string]interface{} `json:"environment"`
	Connect     struct {
		Requested  int       `json:"requested"`
		Connected  int       `json:"connected"`
		Failed     int       `json:"failed"`
		RampSecs   float64   `json:"ramp_seconds"`
		SetupMS    latencyMS `json:"dial_and_auth_ms"`
		FirstError string    `json:"first_error,omitempty"`
	} `json:"connect"`
	Run struct {
		WindowSecs         float64   `json:"window_seconds"`
		Senders            int       `json:"senders"`
		MessagesSent       int64     `json:"messages_sent"`
		ExpectedDeliveries int64     `json:"expected_deliveries"`
		Deliveries         int64     `json:"deliveries"`
		DeliveryRatio      float64   `json:"delivery_ratio"`
		DeliveriesPerSec   float64   `json:"deliveries_per_second"`
		Disconnects        int64     `json:"disconnects"`
		SendErrors         int64     `json:"send_errors"`
		MaxSenderLagMS     float64   `json:"max_sender_lag_ms"`
		LatencyMS          latencyMS `json:"latency_ms"`
	} `json:"run"`
	Server   map[string]float64 `json:"server,omitempty"`
	Warnings []string           `json:"warnings,omitempty"`
}

func ms(d time.Duration) float64 { return math.Round(float64(d)/float64(time.Microsecond)) / 1000 }

func summarize(h *Histogram) latencyMS {
	return latencyMS{ms(h.Mean()), ms(h.Percentile(50)), ms(h.Percentile(90)), ms(h.Percentile(99)), ms(h.Percentile(99.9)), ms(h.Max())}
}

func run(ctx context.Context, cfg config) (*result, error) {
	res := &result{
		Config: map[string]interface{}{
			"url": cfg.URL, "clients": cfg.Clients, "senders": cfg.Senders, "rate_msgs_per_sec": cfg.Rate,
			"payload_bytes": cfg.Payload, "duration_seconds": cfg.Duration.Seconds(),
			"warmup_seconds": cfg.Warmup.Seconds(), "connect_rate_per_sec": cfg.ConnectRate,
		},
		Environment: map[string]interface{}{
			"go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "num_cpu": runtime.NumCPU(),
			"started_at": time.Now().UTC().Format(time.RFC3339),
		},
	}
	warn := func(format string, a ...interface{}) { res.Warnings = append(res.Warnings, fmt.Sprintf(format, a...)) }

	var baseline map[string]float64
	if cfg.MetricsURL != "" {
		var err error
		if baseline, err = scrape(ctx, cfg.MetricsURL); err != nil {
			return nil, fmt.Errorf("metrics: %w", err)
		}
	}

	// ---- connect ----
	win := newWindow()
	var stopping atomic.Bool
	var disconnects atomic.Int64

	type dialResult struct {
		lc   *loadClient
		took time.Duration
		err  error
	}
	results := make(chan dialResult, cfg.Clients)
	ticker := time.NewTicker(time.Duration(float64(time.Second) / cfg.ConnectRate))
	rampStart := time.Now()
	launched := 0
ramp:
	for ; launched < cfg.Clients; launched++ {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			break ramp
		}
		go func() {
			lc, took, err := dialAndAuth(ctx, cfg)
			if err == nil {
				go lc.readLoop(win, &stopping, &disconnects)
			}
			results <- dialResult{lc, took, err}
		}()
	}
	ticker.Stop()

	setup := NewHistogram()
	var clients []*loadClient
	for i := 0; i < launched; i++ {
		r := <-results
		if r.err != nil {
			res.Connect.Failed++
			if res.Connect.FirstError == "" {
				res.Connect.FirstError = r.err.Error()
			}
			continue
		}
		setup.Record(r.took)
		clients = append(clients, r.lc)
	}
	res.Connect.Requested = cfg.Clients
	res.Connect.Connected = len(clients)
	res.Connect.RampSecs = time.Since(rampStart).Seconds()
	res.Connect.SetupMS = summarize(setup)

	stop := func() {
		stopping.Store(true)
		for _, lc := range clients {
			lc.conn.Close()
		}
		for _, lc := range clients {
			select {
			case <-lc.done:
			case <-time.After(5 * time.Second):
			}
		}
	}

	if ctx.Err() != nil {
		stop()
		return nil, ctx.Err()
	}
	if len(clients) < cfg.Senders {
		stop()
		return nil, fmt.Errorf("only %d clients connected (first error: %s); need at least %d senders", len(clients), res.Connect.FirstError, cfg.Senders)
	}
	if res.Connect.Failed > 0 {
		warn("%d of %d clients failed to connect (first error: %s)", res.Connect.Failed, cfg.Clients, res.Connect.FirstError)
	}

	var connected map[string]float64
	if cfg.MetricsURL != "" {
		time.Sleep(time.Second) // let the server settle before sampling memory
		var err error
		if connected, err = scrape(ctx, cfg.MetricsURL); err != nil {
			stop()
			return nil, fmt.Errorf("metrics: %w", err)
		}
	}

	// ---- run ----
	t0 := time.Now().Add(500 * time.Millisecond)
	winStart := t0.Add(cfg.Warmup)
	winEnd := winStart.Add(cfg.Duration)
	win.start.Store(winStart.UnixNano())
	win.end.Store(winEnd.UnixNano())

	interval := time.Duration(float64(time.Second) * float64(cfg.Senders) / cfg.Rate)
	stats := make([]senderStats, cfg.Senders)
	var wg sync.WaitGroup
	for i := 0; i < cfg.Senders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			first := t0.Add(interval * time.Duration(i) / time.Duration(cfg.Senders)) // stagger senders
			runSender(ctx, clients[i], i, first, interval, winStart.UnixNano(), winEnd.UnixNano(), cfg.Payload, &stats[i])
		}(i)
	}
	wg.Wait()
	select { // let in-flight messages arrive
	case <-time.After(2 * time.Second):
	case <-ctx.Done():
	}

	var final map[string]float64
	if cfg.MetricsURL != "" {
		var err error
		if final, err = scrape(ctx, cfg.MetricsURL); err != nil {
			warn("final metrics scrape failed: %v", err)
		}
	}
	stop()

	// ---- aggregate ----
	merged := NewHistogram()
	var delivered int64
	for _, lc := range clients {
		merged.Merge(lc.hist)
		delivered += int64(lc.received)
	}
	var sent, sendErrs int64
	var maxLag time.Duration
	for _, s := range stats {
		sent += s.inWindow
		sendErrs += s.errors
		if s.maxLag > maxLag {
			maxLag = s.maxLag
		}
	}
	expected := sent * int64(len(clients))

	r := &res.Run
	r.WindowSecs = cfg.Duration.Seconds()
	r.Senders = cfg.Senders
	r.MessagesSent = sent
	r.ExpectedDeliveries = expected
	r.Deliveries = delivered
	if expected > 0 {
		r.DeliveryRatio = float64(delivered) / float64(expected)
	}
	r.DeliveriesPerSec = float64(delivered) / cfg.Duration.Seconds()
	r.Disconnects = disconnects.Load()
	r.SendErrors = sendErrs
	r.MaxSenderLagMS = ms(maxLag)
	r.LatencyMS = summarize(merged)

	if r.DeliveryRatio < 0.999 {
		warn("only %.2f%% of expected deliveries arrived; the server dropped or evicted clients, or the load generator could not keep up", r.DeliveryRatio*100)
	}
	if r.Disconnects > 0 {
		warn("%d clients were disconnected during the run", r.Disconnects)
	}
	if maxLag > 100*time.Millisecond {
		warn("a sender wrote up to %.0fms late; the load generator is saturated, so latency includes its own delay (run it on a separate machine or lower the load)", ms(maxLag))
	}

	if baseline != nil && connected != nil {
		res.Server = map[string]float64{}
		if rss0, ok := baseline["process_resident_memory_bytes"]; ok {
			if rss1, ok := connected["process_resident_memory_bytes"]; ok && len(clients) > 0 {
				res.Server["rss_bytes_per_connection"] = math.Round((rss1 - rss0) / float64(len(clients)))
				res.Server["rss_bytes_baseline"] = rss0
				res.Server["rss_bytes_connected"] = rss1
			}
		}
		res.Server["goroutines_connected"] = connected["go_goroutines"]
		if final != nil {
			res.Server["goroutines_end"] = final["go_goroutines"]
			res.Server["rss_bytes_end"] = final["process_resident_memory_bytes"]
			res.Server["evictions"] = final["signalstack_evictions_total"] - baseline["signalstack_evictions_total"]
			limited := `signalstack_messages_dropped_total{reason="rate_limited"}`
			res.Server["rate_limited_drops"] = final[limited] - baseline[limited]
			if res.Server["rate_limited_drops"] > 0 {
				warn("the server rate-limited %.0f messages; start it with RATE_LIMIT=0 for benchmarking", res.Server["rate_limited_drops"])
			}
			if res.Server["evictions"] > 0 {
				warn("the server evicted %.0f slow clients", res.Server["evictions"])
			}
		}
	}
	return res, nil
}

func printSummary(res *result, cfg config) {
	c, r := res.Connect, res.Run
	fmt.Printf("\nSignalStack load test  (%s, %s/%s, %d CPUs)\n", res.Environment["go"], res.Environment["os"], res.Environment["arch"], res.Environment["num_cpu"])
	fmt.Printf("  target       %s\n", cfg.URL)
	fmt.Printf("  clients      %d connected, %d failed; ramp %.1fs; dial+auth p50 %.1fms p99 %.1fms\n", c.Connected, c.Failed, c.RampSecs, c.SetupMS.P50, c.SetupMS.P99)
	fmt.Printf("  load         %d senders, %.0f msg/s offered, %d B padding; %.0fs window after %.0fs warmup\n", r.Senders, cfg.Rate, cfg.Payload, r.WindowSecs, cfg.Warmup.Seconds())
	fmt.Printf("  deliveries   %d of %d expected (%.3f%%), %.0f/s\n", r.Deliveries, r.ExpectedDeliveries, r.DeliveryRatio*100, r.DeliveriesPerSec)
	l := r.LatencyMS
	fmt.Printf("  latency      p50 %.2fms  p90 %.2fms  p99 %.2fms  p99.9 %.2fms  max %.2fms  mean %.2fms\n", l.P50, l.P90, l.P99, l.P999, l.Max, l.Mean)
	if res.Server != nil {
		if v, ok := res.Server["rss_bytes_per_connection"]; ok {
			fmt.Printf("  server       %.1f KiB RSS per connection; %.0f goroutines; %.0f evictions\n", v/1024, res.Server["goroutines_connected"], res.Server["evictions"])
		}
	}
	for _, w := range res.Warnings {
		fmt.Printf("  WARNING      %s\n", w)
	}
	fmt.Println()
}

func main() {
	cfg, err := parseFlags(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "loadtest:", err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	res, err := run(ctx, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "loadtest:", err)
		os.Exit(1)
	}
	printSummary(res, cfg)

	if cfg.Out != "" {
		b, _ := json.MarshalIndent(res, "", "  ")
		if err := os.WriteFile(cfg.Out, append(b, '\n'), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "loadtest: write results:", err)
			os.Exit(1)
		}
		fmt.Println("results written to", cfg.Out)
	}
}
