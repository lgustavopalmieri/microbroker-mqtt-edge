# Load Tests — k6 + MQTT

Resilience tests for the broker using [k6](https://k6.io) with the [xk6-mqtt](https://github.com/pmalhaire/xk6-mqtt) extension.

## Prerequisites

- Docker and Docker Compose
- Broker already running on the host (via `docker compose up -d` or `go run ./cmd/main.go`)

The k6 container connects to the broker via `host.docker.internal:1883`.

## How to Run

```bash
# 50 messages (default)
./tests/k6/run.sh

# custom quantity
./tests/k6/run.sh 200
```

## What It Does

1. Builds a k6 image with MQTT support
2. k6 connects via MQTT (with auth), publishes N messages QoS 1 to the `machine/status` topic
3. Queries the `GET /audit/machine/status` API and validates that all messages were persisted to SQLite
4. Removes the k6 container

## Structure

```
tests/k6/
├── Dockerfile.k6              # k6 v1.7.1 + xk6-mqtt v0.40.3
├── docker-compose.k6.yml      # k6 container only
├── run.sh                     # test orchestrator
└── scripts/
    └── mqtt_publish.js        # k6 script that publishes via MQTT
```

## Environment Variables (docker-compose)

| Variable | Default | Description |
|---|---|---|
| `BROKER_ADDR` | `host.docker.internal:1883` | Broker address |
| `BROKER_USER` | `machine01` | MQTT username |
| `BROKER_PASS` | `secret123` | MQTT password |
| `MQTT_TOPIC` | `machine/status` | Publication topic |
| `MESSAGE_COUNT` | `50` | Messages per run |

## Expected Output

```
==> Running k6 (MESSAGE_COUNT=50)...
     ✓ publisher connected
     ✓ publish ok
     checks...: 100.00% ✓ 51 ✗ 0
==> Verifying persistence via audit API...
    Expected: 50
    Got:      50
==> ✅ PASS — all 50 messages persisted correctly.
```
