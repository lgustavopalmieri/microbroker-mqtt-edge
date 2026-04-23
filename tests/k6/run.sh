#!/usr/bin/env bash
# ──────────────────────────────────────────────────────
# Runs the k6 MQTT smoke test and validates persistence
# via the audit REST API.
#
# Pre-requisite: the broker must already be running on
# the host (localhost:1883 for MQTT, localhost:8080 for
# the audit API).
#
# Usage: ./tests/k6/run.sh [MESSAGE_COUNT]
# ──────────────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MESSAGE_COUNT="${1:-50}"
TOPIC="machine/status"
AUDIT_URL="http://localhost:8080/audit/${TOPIC}"

# Verify broker is reachable
echo "==> Checking broker is running on localhost:1883..."
if ! nc -z localhost 1883 2>/dev/null; then
  echo "    ❌ Broker not reachable on localhost:1883."
  echo "    Start it first: docker compose up -d  (or go run ./cmd/main.go)"
  exit 1
fi

echo "==> Building k6 image..."
docker compose -f "$SCRIPT_DIR/docker-compose.k6.yml" build

echo "==> Running k6 (MESSAGE_COUNT=${MESSAGE_COUNT})..."
docker compose -f "$SCRIPT_DIR/docker-compose.k6.yml" run --rm \
  -e MESSAGE_COUNT="${MESSAGE_COUNT}" \
  k6

# Give the broker a moment to flush writes
sleep 2

echo "==> Verifying persistence via audit API..."
RESPONSE=$(curl -sf "${AUDIT_URL}")
COUNT=$(echo "${RESPONSE}" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))")

echo "    Expected: ${MESSAGE_COUNT}"
echo "    Got:      ${COUNT}"

if [ "${COUNT}" -ge "${MESSAGE_COUNT}" ]; then
  echo "==> ✅ PASS — all ${MESSAGE_COUNT} messages persisted correctly."
  EXIT_CODE=0
else
  echo "==> ❌ FAIL — expected ${MESSAGE_COUNT}, got ${COUNT}."
  EXIT_CODE=1
fi

echo "==> Cleaning up k6 container..."
docker compose -f "$SCRIPT_DIR/docker-compose.k6.yml" down

exit ${EXIT_CODE}
