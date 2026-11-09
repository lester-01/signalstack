package main

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	l := newRateLimiter(10) // burst 20
	t0 := time.Unix(1000, 0)

	for i := 0; i < 20; i++ {
		if !l.allow(t0) {
			t.Fatalf("message %d within the burst was rejected", i+1)
		}
	}
	if l.allow(t0) {
		t.Fatal("message beyond the burst was allowed")
	}

	// 100ms at 10/s refills exactly one token.
	if !l.allow(t0.Add(100 * time.Millisecond)) {
		t.Fatal("token was not refilled")
	}
	if l.allow(t0.Add(100 * time.Millisecond)) {
		t.Fatal("more than one token was refilled")
	}

	// The bucket never exceeds its burst, however long the idle time.
	later := t0.Add(time.Hour)
	for i := 0; i < 20; i++ {
		if !l.allow(later) {
			t.Fatalf("message %d after idling was rejected", i+1)
		}
	}
	if l.allow(later) {
		t.Fatal("bucket grew past its burst while idle")
	}
}

func TestRateLimiterDisabled(t *testing.T) {
	if newRateLimiter(0) != nil || newRateLimiter(-1) != nil {
		t.Fatal("a non-positive rate should disable limiting")
	}
	var l *rateLimiter
	for i := 0; i < 1000; i++ {
		if !l.allow(time.Now()) {
			t.Fatal("a nil limiter must allow everything")
		}
	}
}
