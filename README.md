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
```

- **Connection** (`internal/modules/connection`) — accepts TCP connections, authenticates via username/password, filters allowed topics, and forwards PUBLISH packets to the ingestion layer.
- **Ingestion** (`internal/modules/ingestion`) — one FIFO queue per topic with sequential consumer. Each message is persisted to the `raw_data` table (SQLite) before proceeding.
- **Processing** (`internal/modules/processing`) — receives already-persisted messages and distributes to all registered workers in parallel.
- **Audit** (`internal/modules/audit`) — REST API to query and count persisted messages by topic.
- **Auth** (`internal/modules/auth`) — authenticator implementation based on environment variables.

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `BROKER_HOST` | No | `0.0.0.0` | TCP server bind address |
| `BROKER_PORT` | No | `1883` | TCP broker port |
| `BROKER_HTTP_PORT` | No | `8080` | HTTP API (audit) port |
| `BROKER_USERNAME` | **Yes** | — | MQTT authentication username |
| `BROKER_PASSWORD` | **Yes** | — | MQTT authentication password |
| `BROKER_TOPICS` | **Yes** | — | Allowed topics (1–5, comma-separated) |
| `BROKER_MAX_CLIENTS` | No | `5` | Maximum concurrent connections (1–5) |
| `BROKER_QUEUE_BUFFER_SIZE` | No | `10000` | Internal queue buffer size |
| `BROKER_DB_PATH` | No | `./data/broker.db` | SQLite database file path |
| `BROKER_TIMEZONE` | No | `UTC` | Timezone recorded with each message |

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
