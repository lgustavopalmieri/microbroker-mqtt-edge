# microbroker-mqtt-edge

Open-source, lightweight and embeddable MQTT broker for edge and industrial (Industry 4.0) environments. Receives messages via MQTT 3.1.1 over TCP, persists to local SQLite, and dispatches to configurable workers. Zero external runtime dependencies — just the binary and a volume for the database.

## Architecture

```
MQTT Client ──TCP:1883──▶ Connection (auth + topic filter)
                              │
                              ▼
                          Ingestion (FIFO queue per topic → SQLite)
                              │
                              ▼
                          Processing (fan-out to workers)
                              │
                    ┌─────────┴──────────┐
                    ▼                    ▼
              Logger worker       OEE state worker
                               (self-filters machine/state)
                                         │
                              ┌──────────┴───────────┐
                              ▼                      ▼
                        state_intervals DB     Live Engine (~1s tick)
                                                     │
                                              CompositeSink
                                            ┌────────┴────────┐
                                            ▼                 ▼
                                         LogSink       WebsocketSink
                                                     /availability/{m}/ws
```

- **Connection** — accepts TCP connections, authenticates via username/password, filters allowed topics, and forwards PUBLISH packets to the ingestion layer.
- **Ingestion** — one FIFO queue per topic with sequential consumer. Each message is persisted to the `raw_data` table (SQLite) before proceeding.
- **Processing** — receives already-persisted messages and distributes to all registered workers in parallel.
- **Audit** — REST API to query and count persisted messages by topic.
- **OEE Availability** — when `BROKER_OEE_ENABLED=true`, a state-change worker ingests `machine/state` payloads, persists paired `state_intervals`, and drives a live engine that recomputes availability every tick. Results are pushed to subscribed WebSocket clients and queryable via REST.

## Environment Variables

### Broker

| Variable | Required | Default | Description |
|---|---|---|---|
| `BROKER_HOST` | No | `0.0.0.0` | TCP server bind address |
| `BROKER_PORT` | No | `1883` | TCP broker port |
| `BROKER_HTTP_PORT` | No | `8080` | HTTP API port |
| `BROKER_USERNAME` | **Yes** | — | MQTT authentication username |
| `BROKER_PASSWORD` | **Yes** | — | MQTT authentication password |
| `BROKER_TOPICS` | **Yes** | — | Allowed topics (1–5, comma-separated, by data type) |
| `BROKER_MAX_CLIENTS` | No | `5` | Maximum concurrent connections (1–5) |
| `BROKER_QUEUE_BUFFER_SIZE` | No | `10000` | Internal queue buffer size |
| `BROKER_DB_PATH` | No | `./data/broker.db` | SQLite database file path |
| `BROKER_TIMEZONE` | No | `UTC` | Timezone recorded with each message |

### OEE / Availability (optional)

| Variable | Required | Default | Description |
|---|---|---|---|
| `BROKER_OEE_ENABLED` | No | `false` | Master switch; set `true` to activate the OEE module |
| `BROKER_OEE_STATE_TOPIC` | No | `machine/state` | Topic carrying `state_change` events — **must** be in `BROKER_TOPICS` |
| `BROKER_OEE_TICK_INTERVAL` | No | `1s` | Live availability recompute cadence (Go duration, e.g. `500ms`, `2s`) |
| `BROKER_OEE_WS_ENABLED` | No | `true` | Register the WebSocket sink and endpoint |
| `BROKER_OEE_SHIFTS_PATH` | No | `""` | Path to a JSON shifts file seeded at startup; empty = no seeding |

## How to Run

### Local Binary

```bash
cp .env.example .env
go build -o microbroker ./cmd/broker
./microbroker
```

### Docker Compose

```bash
cp .env.example .env
docker compose up -d
```

The `broker-data` volume persists SQLite to `/data/broker.db` inside the container.

## Connecting an MQTT Client

Any MQTT 3.1.1 client works (mosquitto_pub, MQTTX, paho-mqtt, etc.).

| Parameter | Value |
|---|---|
| Host | IP/hostname of the machine |
| Port | `1883` (or `BROKER_PORT`) |
| Username | value of `BROKER_USERNAME` |
| Password | value of `BROKER_PASSWORD` |
| QoS | `0` or `1` |

## Payload Format

The broker accepts any payload in bytes, but the expected usage is JSON. Each message is stored with the following structure:

| Column | Type | Description |
|---|---|---|
| `client` | TEXT | Client identifier |
| `topic` | TEXT | MQTT topic |
| `timezone` | TEXT | Configured timezone |
| `timestamp` | TEXT | Message timestamp |
| `payload` | TEXT | Raw payload content |

### Example — machine/status

```json
{
  "machine_id": "CNC-01",
  "status": "running",
  "temperature": 72.5,
  "rpm": 1200
}
```

## OEE — machine/state payload

When `BROKER_OEE_ENABLED=true`, publish `state_change` events on the configured state topic (default `machine/state`):

```json
{
  "machine_id": "CNC-01",
  "state": "stopped",
  "previous_state": "running",
  "reason": "planned maintenance",
  "timestamp": "2026-06-01T08:47:00Z"
}
```

Valid states: `running` · `stopped` · `setup` · `idle` · `maintenance` · `off`

The broker persists each state transition as a `state_interval` row, drives the live availability engine, and feeds the query endpoint.

## REST API — OEE Availability

### On-demand availability query

```
GET /availability/{machine}?from=<rfc3339>&to=<rfc3339>
```

| Parameter | In | Description |
|---|---|---|
| `machine` | path | Machine ID (matches `machine_id` in the payload) |
| `from` | query | Window start (RFC 3339, e.g. `2026-06-01T06:00:00Z`) |
| `to` | query | Window end (RFC 3339, must be after `from`) |

**Response 200**

```json
{
  "machine_id": "CNC-01",
  "window_from": "2026-06-01T06:00:00Z",
  "window_to":   "2026-06-01T14:00:00Z",
  "availability": 0.8881,
  "planned_time_ns": 25200000000000,
  "run_time_ns":     22382100000000,
  "planned_downtime_ns": 0,
  "unplanned_downtime_ns": 2817900000000,
  "interval_count": 3,
  "has_data": true,
  "flags": {}
}
```

Flags that may appear in the response body:

| Flag | Meaning |
|---|---|
| `shift_config_missing` | No shift config found; wall-clock window used as planned time |
| `open_interval_clipped` | An open (in-progress) interval was clipped to `to` |
| `no_data` | No intervals in the window; availability is 0 |

### Live availability stream (WebSocket)

```
GET /availability/{machine}/ws
```

Upgrades to a WebSocket connection. The server pushes a JSON `AvailabilitySnapshot` on every engine tick (default every `1s`). Slow or disconnected clients are dropped without blocking the engine.

## REST API — Audit

### Get messages by topic

```
GET /audit/{topic}
```

| Parameter | In | Description |
|---|---|---|
| `topic` | path | MQTT topic (e.g. `machine/status`) |

### Count messages by topic

```
GET /audit-count/{topic}
```

| Parameter | In | Description |
|---|---|---|
| `topic` | path | MQTT topic (e.g. `machine/status`) |

## License

MIT
