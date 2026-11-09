package main

import (
	"math"
	"time"
)

// rateLimiter is a token bucket. It is not safe for concurrent use: each
// client's limiter is only touched by that client's reader goroutine.
// A nil *rateLimiter allows everything.
type rateLimiter struct {
	rate   float64 // tokens added per second
	burst  float64 // bucket capacity
	tokens float64
	last   time.Time
}

// newRateLimiter allows ratePerSec messages per second with a burst of twice
// that. It returns nil (unlimited) when ratePerSec <= 0.
func newRateLimiter(ratePerSec float64) *rateLimiter {
	if ratePerSec <= 0 {
		return nil
	}
	burst := math.Max(1, math.Ceil(2*ratePerSec))
	return &rateLimiter{rate: ratePerSec, burst: burst, tokens: burst}
}

func (l *rateLimiter) allow(now time.Time) bool {
	if l == nil {
		return true
	}
	if l.last.IsZero() {
		l.last = now
	}
	l.tokens = math.Min(l.burst, l.tokens+now.Sub(l.last).Seconds()*l.rate)
	l.last = now
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}
