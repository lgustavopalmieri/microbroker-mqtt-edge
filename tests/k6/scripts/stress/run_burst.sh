#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────
# Burst 1K Test — Can the broker handle 1,000+ msg/sec?
#
# Pre-requisite:
#   Broker running on localhost:1883 (MQTT) and localhost:8080 (HTTP)
#
# Usage:
#   ./tests/k6/scripts/stress/run_burst.sh
# ──────────────────────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
K6_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
AUDIT_BASE="http://localhost:8080/audit-count"

TOPICS=(
  "machine/status"
  "machine/production"
  "machine/alarm"
  "machine/oee"
  "machine/counter"
)

echo "╔══════════════════════════════════════════════════════════╗"
echo "║     Burst 1K Test — can we hit 1,000 msg/sec?          ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

echo "==> Checking broker on localhost:1883..."
if ! nc -z localhost 1883 2>/dev/null; then
  echo "    ❌ Broker not reachable. Start it first."
  exit 1
fi

echo "==> Checking audit API on localhost:8080..."
if ! curl -sf "${AUDIT_BASE}/machine/status" > /dev/null 2>&1; then
  echo "    ❌ Audit API not reachable on :8080."
  exit 1
fi

# ── Baseline ─────────────────────────────────────────────────

echo "==> Capturing baseline counts..."
declare -A BASELINE
for topic in "${TOPICS[@]}"; do
  count=$(curl -sf "${AUDIT_BASE}/${topic}" | python3 -c "import sys,json; print(json.load(sys.stdin)['count'])")
  BASELINE["$topic"]=$count
done

# ── Run k6 ───────────────────────────────────────────────────

echo ""
echo "==> Running burst test (~90s)..."
echo ""

docker compose -f "$K6_DIR/docker-compose.k6.yml" run --rm \
  --entrypoint "k6" \
  k6 run /scripts/stress/burst_1k.js

# ── Wait for SQLite to drain ─────────────────────────────────

echo ""
echo "==> Waiting 2 minutes for SQLite to drain all queues..."
sleep 120

# ── Verify ───────────────────────────────────────────────────

echo "==> Verifying persistence per topic..."
echo ""

TOTAL_ACTUAL=0

printf "    %-25s %10s %8s\n" "TOPIC" "PERSISTED" "STATUS"
printf "    %-25s %10s %8s\n" "─────────────────────────" "──────────" "────────"

for topic in "${TOPICS[@]}"; do
  current=$(curl -sf "${AUDIT_BASE}/${topic}" | python3 -c "import sys,json; print(json.load(sys.stdin)['count'])")
  baseline=${BASELINE["$topic"]}
  actual=$((current - baseline))
  TOTAL_ACTUAL=$((TOTAL_ACTUAL + actual))

  if [ "$actual" -gt 0 ]; then
    printf "    %-25s %10d %8s\n" "$topic" "$actual" "✅"
  else
    printf "    %-25s %10d %8s\n" "$topic" "$actual" "⚠️"
  fi
done

echo ""
echo "    Total messages persisted: ${TOTAL_ACTUAL}"
echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║  Compare 'published_total' from k6 output with         ║"
echo "║  total persisted above. They should match.              ║"
echo "╚══════════════════════════════════════════════════════════╝"

echo ""
echo "==> Cleaning up..."
docker compose -f "$K6_DIR/docker-compose.k6.yml" down
