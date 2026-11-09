package main

import (
	"math/bits"
	"time"
)

// Histogram records durations with a fixed relative error and no per-sample
// memory, so a long run at high message rates stays cheap. Values are kept in
// microseconds in log-linear buckets: exact below 256µs, then 128 buckets per
// power of two (each bucket spans at most 1/128, about 0.8%, of its value).
//
// A Histogram is not safe for concurrent use. Give each goroutine its own and
// Merge them afterwards.
type Histogram struct {
	counts []uint64
	total  uint64
	sum    uint64 // microseconds
	max    uint64 // microseconds
}

const (
	subBits  = 7
	sub      = 1 << subBits // buckets per power of two
	maxValue = 1 << 40      // microseconds (about 12 days); larger values are clamped
)

// NewHistogram returns an empty histogram.
func NewHistogram() *Histogram {
	return &Histogram{counts: make([]uint64, bucketOf(maxValue)+1)}
}

func bucketOf(v uint64) int {
	if v < 2*sub {
		return int(v)
	}
	s := bits.Len64(v) - (subBits + 1)
	return s*sub + int(v>>uint(s))
}

// bucketBounds returns the smallest and largest value (microseconds) that map to bucket b.
func bucketBounds(b int) (lo, hi uint64) {
	if b < 2*sub {
		return uint64(b), uint64(b)
	}
	s := uint(b/sub - 1)
	m := uint64(b) - uint64(s)*sub
	lo = m << s
	return lo, lo + (1 << s) - 1
}

// Record adds one observation. Negative durations count as zero.
func (h *Histogram) Record(d time.Duration) {
	var v uint64
	if d > 0 {
		v = uint64(d / time.Microsecond)
	}
	if v > maxValue {
		v = maxValue
	}
	h.counts[bucketOf(v)]++
	h.total++
	h.sum += v
	if v > h.max {
		h.max = v
	}
}

// Merge adds all observations from o.
func (h *Histogram) Merge(o *Histogram) {
	for i, c := range o.counts {
		h.counts[i] += c
	}
	h.total += o.total
	h.sum += o.sum
	if o.max > h.max {
		h.max = o.max
	}
}

// Count returns the number of observations.
func (h *Histogram) Count() uint64 { return h.total }

// Max returns the largest observation.
func (h *Histogram) Max() time.Duration { return time.Duration(h.max) * time.Microsecond }

// Mean returns the average observation.
func (h *Histogram) Mean() time.Duration {
	if h.total == 0 {
		return 0
	}
	return time.Duration(h.sum/h.total) * time.Microsecond
}

// Percentile returns the value at or below which p percent (0-100) of
// observations fall. It reports the upper edge of the bucket, so it never
// understates, and never exceeds the largest observation.
func (h *Histogram) Percentile(p float64) time.Duration {
	if h.total == 0 {
		return 0
	}
	rank := uint64(p / 100 * float64(h.total))
	if float64(rank) < p/100*float64(h.total) {
		rank++ // ceiling
	}
	if rank < 1 {
		rank = 1
	}
	var seen uint64
	for b, c := range h.counts {
		seen += c
		if seen >= rank {
			_, hi := bucketBounds(b)
			if hi > h.max {
				hi = h.max
			}
			return time.Duration(hi) * time.Microsecond
		}
	}
	return h.Max()
}
