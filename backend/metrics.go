package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
)

// Metrics are cheap atomic counters updated on the hot path and rendered in
// the Prometheus text exposition format by the opt-in /metrics endpoint.
type Metrics struct {
	connectionsTotal   atomic.Uint64 // sessions registered with the hub
	authSuccess        atomic.Uint64
	authFailures       atomic.Uint64
	messagesReceived   atomic.Uint64 // chat messages accepted from clients
	messagesDelivered  atomic.Uint64 // chat messages queued to a recipient
	evictions          atomic.Uint64 // slow clients dropped
	droppedInvalid     atomic.Uint64
	droppedUnauth      atomic.Uint64
	droppedRateLimited atomic.Uint64
}

// write renders all metrics. connected is the current number of registered clients.
func (m *Metrics) write(w io.Writer, connected int) {
	gauge := func(name, help string, v interface{}) {
		fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n%s %v\n", name, help, name, name, v)
	}
	counter := func(name, help string, v uint64) {
		fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n%s %d\n", name, help, name, name, v)
	}

	gauge("signalstack_connected_clients", "Clients currently registered with the hub.", connected)
	counter("signalstack_connections_total", "Sessions registered since start.", m.connectionsTotal.Load())

	fmt.Fprint(w, "# HELP signalstack_auth_total Authentication attempts by result.\n# TYPE signalstack_auth_total counter\n")
	fmt.Fprintf(w, "signalstack_auth_total{result=\"success\"} %d\n", m.authSuccess.Load())
	fmt.Fprintf(w, "signalstack_auth_total{result=\"failure\"} %d\n", m.authFailures.Load())

	counter("signalstack_messages_received_total", "Chat messages accepted from clients.", m.messagesReceived.Load())
	counter("signalstack_messages_delivered_total", "Chat messages queued for delivery to a client (one per recipient).", m.messagesDelivered.Load())
	counter("signalstack_evictions_total", "Clients disconnected because their send queue was full.", m.evictions.Load())

	fmt.Fprint(w, "# HELP signalstack_messages_dropped_total Inbound messages rejected, by reason.\n# TYPE signalstack_messages_dropped_total counter\n")
	fmt.Fprintf(w, "signalstack_messages_dropped_total{reason=\"invalid\"} %d\n", m.droppedInvalid.Load())
	fmt.Fprintf(w, "signalstack_messages_dropped_total{reason=\"unauthenticated\"} %d\n", m.droppedUnauth.Load())
	fmt.Fprintf(w, "signalstack_messages_dropped_total{reason=\"rate_limited\"} %d\n", m.droppedRateLimited.Load())

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	gauge("go_goroutines", "Number of goroutines.", runtime.NumGoroutine())
	gauge("go_memstats_heap_alloc_bytes", "Bytes of allocated heap objects.", ms.HeapAlloc)
	gauge("go_memstats_sys_bytes", "Bytes obtained from the OS.", ms.Sys)
	counter("go_gc_cycles_total", "Completed GC cycles.", uint64(ms.NumGC))
	if rss, ok := processRSSBytes(); ok {
		gauge("process_resident_memory_bytes", "Resident memory of the process (Linux only).", rss)
	}
}

// processRSSBytes returns the resident set size from /proc (Linux only).
func processRSSBytes() (uint64, bool) {
	b, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0, false
	}
	f := strings.Fields(string(b))
	if len(f) < 2 {
		return 0, false
	}
	pages, err := strconv.ParseUint(f[1], 10, 64)
	if err != nil {
		return 0, false
	}
	return pages * uint64(os.Getpagesize()), true
}
