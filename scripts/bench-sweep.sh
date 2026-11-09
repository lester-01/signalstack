#!/usr/bin/env bash
# Runs the load test at several client counts against an already running server
# and saves one JSON result per run. See docs/BENCHMARKING.md.
#
# Start the server first, ideally restarting it between runs:
#   cd backend && AUTH_TOKEN=bench METRICS_ENABLED=true RATE_LIMIT=0 go run .
#
# Usage: AUTH_TOKEN=bench scripts/bench-sweep.sh
# Overrides: URL, METRICS_URL, CLIENTS, SENDERS, RATE, DURATION, WARMUP, OUT
set -euo pipefail

URL=${URL:-ws://localhost:8080/ws}
METRICS_URL=${METRICS_URL:-http://localhost:8080/metrics}
CLIENTS=${CLIENTS:-"100 500 1000 2000"}
SENDERS=${SENDERS:-20}
RATE=${RATE:-200}
DURATION=${DURATION:-30s}
WARMUP=${WARMUP:-5s}
OUT=${OUT:-bench-results}

: "${AUTH_TOKEN:?set AUTH_TOKEN to the token the server uses}"

cd "$(dirname "$0")/../backend"
mkdir -p "$OUT"

for n in $CLIENTS; do
  echo "=== $n clients ==="
  go run ./cmd/loadtest -url "$URL" -token "$AUTH_TOKEN" -metrics-url "$METRICS_URL" \
    -clients "$n" -senders "$SENDERS" -rate "$RATE" -duration "$DURATION" -warmup "$WARMUP" \
    -out "$OUT/clients-$n.json"
done
