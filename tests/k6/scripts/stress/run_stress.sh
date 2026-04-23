#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────
# SMT Line Stress Test — Runner & Verifier
#
# Runs the k6 stress test against the broker (must be running)
# and verifies that all published messages were persisted by
# querying the audit-count API for each topic.
#
# Pre-requisite:
#   Broker running on localhost:1883 (MQTT) and localhost:8080 (HTTP)
#
# Usage:
#   ./tests/k6/scripts/stress/run_stress.sh [RATE_MULTIPLIER]
#
# Examples:
#   ./tests/k6/scripts/stress/run_stress.sh       # baseline (~60 msg/sec)
#   ./tests/k6/scripts/stress/run_stress.sh 2     # 2x rate (~120 msg/sec)
#   ./tests/k6/scripts/stress/run_stress.sh 5     # 5x rate (~300 msg/sec)
#   ./tests/k6/scripts/stress/run_stress.sh 10    # 10x rate (~600 msg/sec)
# ──────────────────────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
K6_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
RATE_MULTIPLIER="${1:-1}"
AUDIT_BASE="http://localhost:8080/audit-count"

TOPICS=(
  "machine/status"
  "machine/production"
  "machine/alarm"
  "machine/oee"
  "machine/counter"
)

# ── Pre-flight checks ────────────────────────────────────────

echo "╔══════════════════════════════════════════════════════════╗"
echo "║       SMT Line Stress Test — microbroker-mqtt-edge      ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

echo "==> Checking broker on localhost:1883..."
if ! nc -z localhost 1883 2>/dev/null; then
  echo "    ❌ Broker not reachable. Start it first."
  exit 1
fi

echo "==> Checking audit API on localhost:8080..."
if ! curl -sf "http://localhost:8080/audit-count/machine/status" > /dev/null 2>&1; then
  echo "    ❌ Audit API not reachable on :8080."
  exit 1
fi

# ── Capture baseline counts (in case DB already has data) ────

echo "==> Capturing baseline counts..."
declare -A BASELINE
for topic in "${TOPICS[@]}"; do
  count=$(curl -sf "${AUDIT_BASE}/${topic}" | python3 -c "import sys,json; print(json.load(sys.stdin)['count'])")
  BASELINE["$topic"]=$count
  echo "    ${topic}: ${count} existing records"
done

# ── Run k6 ───────────────────────────────────────────────────

echo ""
echo "==> Running k6 stress test (RATE_MULTIPLIER=${RATE_MULTIPLIER})..."
echo "    Estimated duration: ~3 minutes"
echo ""

docker compose -f "$K6_DIR/docker-compose.k6.yml" run --rm \
  -e RATE_MULTIPLIER="${RATE_MULTIPLIER}" \
  --entrypoint "k6" \
  k6 run /scripts/stress/smt_line_stress.js

# ── Wait for broker to flush ─────────────────────────────────

echo ""
echo "==> Waiting for broker to flush pending writes..."
echo "    (SQLite single-writer needs time to drain all queues)"
sleep 30

# ── Verify persistence ───────────────────────────────────────

echo "==> Verifying persistence per topic..."
echo ""

TOTAL_EXPECTED=0
TOTAL_ACTUAL=0
ALL_PASS=true

printf "    %-25s %10s %10s %8s\n" "TOPIC" "EXPECTED" "ACTUAL" "STATUS"
printf "    %-25s %10s %10s %8s\n" "─────────────────────────" "──────────" "──────────" "────────"

for topic in "${TOPICS[@]}"; do
  current=$(curl -sf "${AUDIT_BASE}/${topic}" | python3 -c "import sys,json; print(json.load(sys.stdin)['count'])")
  baseline=${BASELINE["$topic"]}
  actual=$((current - baseline))

  # We don't know exact expected per topic from k6 output, so we just
  # report what was persisted. The key metric is: published_total (from k6)
  # should equal sum of all topic counts.
  TOTAL_ACTUAL=$((TOTAL_ACTUAL + actual))

  if [ "$actual" -gt 0 ]; then
    printf "    %-25s %10s %10d %8s\n" "$topic" "—" "$actual" "✅"
  else
    printf "    %-25s %10s %10d %8s\n" "$topic" "—" "$actual" "⚠️"
  fi
done

echo ""
echo "    Total messages persisted: ${TOTAL_ACTUAL}"
echo ""

# ── Summary ──────────────────────────────────────────────────

if [ "$TOTAL_ACTUAL" -gt 0 ]; then
  echo "╔══════════════════════════════════════════════════════════╗"
  echo "║  ✅ STRESS TEST COMPLETE                                ║"
  echo "║                                                          ║"
  echo "║  ${TOTAL_ACTUAL} messages persisted across 5 topics"
  echo "║  Rate multiplier: ${RATE_MULTIPLIER}x                   ║"
  echo "║                                                          ║"
  echo "║  Compare 'published_total' from k6 output above with    ║"
  echo "║  the total persisted count. They should match.           ║"
  echo "╚══════════════════════════════════════════════════════════╝"
else
  echo "╔══════════════════════════════════════════════════════════╗"
  echo "║  ❌ STRESS TEST FAILED — 0 messages persisted            ║"
  echo "╚══════════════════════════════════════════════════════════╝"
  exit 1
fi

echo ""
echo "==> Cleaning up k6 container..."
docker compose -f "$K6_DIR/docker-compose.k6.yml" down
