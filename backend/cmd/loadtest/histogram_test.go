package main

import (
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"
)

func TestBucketsAreContiguousAndConsistent(t *testing.T) {
	prevHi := uint64(0)
	for b := 0; b <= bucketOf(maxValue); b++ {
		lo, hi := bucketBounds(b)
		if b > 0 && lo != prevHi+1 {
			t.Fatalf("bucket %d starts at %d, previous ended at %d", b, lo, prevHi)
		}
		if bucketOf(lo) != b || bucketOf(hi) != b {
			t.Fatalf("bucket %d bounds [%d,%d] map to %d and %d", b, lo, hi, bucketOf(lo), bucketOf(hi))
		}
		if lo >= 2*sub && float64(hi-lo+1)/float64(lo) > 1.0/float64(sub)+1e-9 {
			t.Fatalf("bucket %d is wider than %.2f%% of its value", b, 100.0/float64(sub))
		}
		prevHi = hi
	}
}

func TestPercentilesMatchExactWithinBucketError(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	h := NewHistogram()
	samples := make([]uint64, 200000)
	for i := range samples {
		// log-normal around 2ms with a long tail, in microseconds
		v := uint64(math.Exp(rng.NormFloat64()*1.2+math.Log(2000))) + 1
		samples[i] = v
		h.Record(time.Duration(v) * time.Microsecond)
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })

	for _, p := range []float64{50, 90, 99, 99.9} {
		rank := int(math.Ceil(p / 100 * float64(len(samples))))
		exact := float64(samples[rank-1])
		got := float64(h.Percentile(p) / time.Microsecond)
		if got < exact || got > exact*(1+1.0/float64(sub))+1 {
			t.Errorf("p%v = %.0fµs, exact %.0fµs (want within +%.2f%%)", p, got, exact, 100.0/float64(sub))
		}
	}
	if h.Count() != uint64(len(samples)) {
		t.Fatalf("count = %d", h.Count())
	}
	if want := time.Duration(samples[len(samples)-1]) * time.Microsecond; h.Max() != want || h.Percentile(100) != want {
		t.Fatalf("max = %v / p100 = %v, want %v", h.Max(), h.Percentile(100), want)
	}
}

func TestSmallValuesAreExact(t *testing.T) {
	h := NewHistogram()
	for v := 1; v <= 100; v++ {
		h.Record(time.Duration(v) * time.Microsecond)
	}
	if got := h.Percentile(50); got != 50*time.Microsecond {
		t.Fatalf("p50 = %v, want 50µs", got)
	}
	if got := h.Percentile(99); got != 99*time.Microsecond {
		t.Fatalf("p99 = %v, want 99µs", got)
	}
	if got := h.Mean(); got != 50*time.Microsecond { // (5050/100)=50 after integer division
		t.Fatalf("mean = %v, want 50µs", got)
	}
}

func TestMergeEqualsRecordingTogether(t *testing.T) {
	a, b, both := NewHistogram(), NewHistogram(), NewHistogram()
	rng := rand.New(rand.NewSource(2))
	for i := 0; i < 50000; i++ {
		d := time.Duration(rng.Intn(5_000_000)) * time.Microsecond
		both.Record(d)
		if i%2 == 0 {
			a.Record(d)
		} else {
			b.Record(d)
		}
	}
	a.Merge(b)
	for _, p := range []float64{50, 99, 99.9, 100} {
		if a.Percentile(p) != both.Percentile(p) {
			t.Fatalf("p%v differs after merge: %v vs %v", p, a.Percentile(p), both.Percentile(p))
		}
	}
	if a.Count() != both.Count() || a.Mean() != both.Mean() {
		t.Fatal("count or mean differs after merge")
	}
}

func TestEmptyNegativeAndHugeValues(t *testing.T) {
	h := NewHistogram()
	if h.Percentile(99) != 0 || h.Mean() != 0 || h.Max() != 0 {
		t.Fatal("empty histogram should report zeros")
	}
	h.Record(-5 * time.Second)
	h.Record(time.Duration(math.MaxInt64)) // clamped, must not panic
	if h.Count() != 2 {
		t.Fatalf("count = %d, want 2", h.Count())
	}
	if h.Percentile(1) != 0 {
		t.Fatal("negative durations should count as zero")
	}
}

func TestParseStamp(t *testing.T) {
	text := buildText(7, 42, 1234567890, 10)
	s, ok := parseStamp(text)
	if !ok || s.sender != 7 || s.seq != 42 || s.intended != 1234567890 {
		t.Fatalf("parseStamp(%q) = %+v, %v", text, s, ok)
	}
	if len(text) < 10 {
		t.Fatalf("payload padding missing: %q", text)
	}
	for _, bad := range []string{"", "hello", "lt|1|2", "lt|a|2|3|x", "xx|1|2|3|"} {
		if _, ok := parseStamp(bad); ok {
			t.Errorf("parseStamp(%q) should fail", bad)
		}
	}
}

func TestParseMetrics(t *testing.T) {
	body := "# HELP x y\n# TYPE x gauge\nsignalstack_connected_clients 12\nsignalstack_auth_total{result=\"failure\"} 3\nprocess_resident_memory_bytes 1.5e+07\n\n"
	m, err := parseMetrics(body)
	if err != nil {
		t.Fatal(err)
	}
	if m["signalstack_connected_clients"] != 12 || m[`signalstack_auth_total{result="failure"}`] != 3 || m["process_resident_memory_bytes"] != 1.5e7 {
		t.Fatalf("unexpected parse result: %v", m)
	}
	if _, err := parseMetrics("broken line without number\n"); err == nil {
		t.Fatal("expected an error for a malformed line")
	}
}
