# Benchmarking

`backend/cmd/loadtest` drives a running server with many WebSocket clients and reports delivery latency, throughput and server memory. This page explains what it measures and how to get numbers you can publish. There are no published results yet; the table at the bottom is for yours.

## What it measures

N clients connect and authenticate. A subset (the *senders*) broadcast at a fixed aggregate rate. Every client receives every broadcast, so one message is N deliveries.

- **Latency**: time from when a message was *scheduled* to be sent until a client receives it, over every delivery in the measurement window. Reported as p50, p90, p99, p99.9, max and mean.
- **Throughput**: deliveries per second, and the delivery ratio (deliveries received / deliveries expected).
- **Server cost** (needs `-metrics-url`): resident memory per connection, goroutine count, evictions and rate-limited drops.

## Run it

Start the server with metrics on and the per-client rate limit off (otherwise it throttles the load generator):

```bash
cd backend
AUTH_TOKEN=bench METRICS_ENABLED=true RATE_LIMIT=0 go run .
```

In another terminal:

```bash
go run ./cmd/loadtest -token bench -metrics-url http://localhost:8080/metrics \
  -clients 1000 -senders 20 -rate 200 -duration 30s -warmup 5s -out result.json
```

| Flag | Default | Meaning |
|---|---|---|
| `-clients` | 100 | Concurrent connections |
| `-senders` | 10 | How many of them send |
| `-rate` | 100 | Broadcast messages per second, total |
| `-payload` | 64 | Padding bytes per message (max 900) |
| `-duration` / `-warmup` | 30s / 5s | Measured window / unmeasured lead-in |
| `-connect-rate` | 100 | New connections per second while ramping |
| `-origin` | – | Origin header, if the server restricts origins |
| `-out` | – | Write full results as JSON |

`scripts/bench-sweep.sh` repeats the run across client counts and saves one JSON file per run.

## Methodology

- **No coordinated omission.** Each message is stamped with its *scheduled* send time. If the load generator falls behind, the delay shows up as latency instead of quietly lowering the offered load. The tool warns when a sender runs more than 100ms late.
- **One clock.** Senders and receivers are in the same process, so latency never depends on clock sync between machines.
- **Warmup excluded.** Messages are matched to the window by their scheduled time, so warmup and stragglers do not count.
- **Histogram accuracy.** Latencies are stored in log-linear buckets with at most 0.8% relative error and reported as the bucket's upper edge, so values are never understated.
- **Broadcast only.** Private (1:1) routing is not benchmarked.

## Getting numbers you can publish

1. **Separate machines.** On one machine the load generator competes with the server for CPU and the tail latency includes that. Run it from a second host, or at least pin them to different cores.
2. **File descriptors.** Each client needs a socket on both sides. Raise `ulimit -n` above the client count on both machines.
3. **Fresh server per run.** Memory per connection is the growth in the server's resident memory between before and after connecting, so a long-running server skews it. Restart the server between runs.
4. **Repeat.** Do at least three runs per configuration and report the median, with the spread.
5. **Record the environment.** CPU model and cores, RAM, OS and kernel, Go version, whether the machines were otherwise idle, and the exact command. The JSON captures the Go version, OS, architecture and CPU count of the load generator; note the server's hardware yourself.
6. **Heed the warnings.** A run with a delivery ratio below 99.9%, evictions, or a saturated load generator is reporting a limit of the setup. Say so instead of publishing it as a clean result.

Keep the JSON files with the results (for example in `docs/results/`) so they can be checked.

## Results

| Date | Server hardware | Load generator | Clients | Senders | Msgs/s | Deliveries/s | p50 | p99 | p99.9 | RSS/conn |
|---|---|---|---|---|---|---|---|---|---|---|
| – | – | – | – | – | – | – | – | – | – | – |
