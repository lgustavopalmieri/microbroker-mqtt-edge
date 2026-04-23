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
├── run.sh                     # smoke test orchestrator
└── scripts/
    ├── mqtt_publish.js        # smoke test — single client, N messages
    └── stress/
        ├── config.js          # shared broker config
        ├── payloads.js        # realistic payload generators per machine type
        ├── smt_line_stress.js # stress test — 5 clients, 5 topics, ramping
        ├── run_stress.sh      # stress test orchestrator + verification
        ├── burst_1k.js        # burst test — 5 clients, max throughput, no sleep
        └── run_burst.sh       # burst test orchestrator (2 min drain wait)
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

## Stress Test — SMT Line Simulation

Simulates a complete SMT (Surface Mount Technology) production line with 5 machines publishing simultaneously to 5 different topics.

### Machine Profiles

| VU | Machine | Topic | Base rate |
|---|---|---|---|
| 1 | Solder Paste Printer | `machine/status` | 10 msg/sec |
| 2 | Pick-and-Place (chipshooter) | `machine/production` | 20 msg/sec |
| 3 | Reflow Oven | `machine/alarm` | 5 msg/sec |
| 4 | AOI Inspection | `machine/oee` | 10 msg/sec |
| 5 | Line Controller / MES | `machine/counter` | 15 msg/sec |

Total baseline: ~60 msg/sec. Rates scale with `RATE_MULTIPLIER`.

### Test Phases (~3 min)

1. Warm-up (30s) — ramp 0 → 5 VUs
2. Sustained (60s) — steady at baseline rate
3. Spike (30s) — all VUs active
4. Recovery (30s) — back to baseline
5. Cool-down (30s) — ramp 5 → 0 VUs

### How to Run

```bash
# baseline (~60 msg/sec)
./tests/k6/scripts/stress/run_stress.sh

# 2x rate (~120 msg/sec)
./tests/k6/scripts/stress/run_stress.sh 2

# 5x rate (~300 msg/sec)
./tests/k6/scripts/stress/run_stress.sh 5

# 10x rate (~600 msg/sec) — push the limits
./tests/k6/scripts/stress/run_stress.sh 10
```

### Verification

After k6 finishes, the script queries `GET /audit-count/{topic}` for each topic and reports how many messages were persisted. Compare the `published_total` counter from k6 output with the total persisted count — they should match.

## Burst Test — 1,000 msg/sec Target

Pure throughput test. 5 clients publishing as fast as possible with zero sleep — the only throttle is the QoS 1 PUBACK round-trip. Tests whether the broker can sustain 1,000+ msg/sec.

### Test Phases (~90s)

1. Connect (10s) — ramp 0 → 5 VUs
2. Full blast (60s) — all 5 VUs, batches of 10, no sleep
3. Cool-down (20s) — ramp 5 → 0 VUs

After the test, waits 2 minutes for SQLite to drain all queued messages before verifying.

### How to Run

```bash
./tests/k6/scripts/stress/run_burst.sh
```
