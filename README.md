# microbroker-mqtt-edge

Lightweight and embeddable MQTT broker for edge/industrial environments. Receives messages via MQTT 3.1.1 protocol over TCP, persists to local SQLite, and dispatches to configurable workers. Zero external runtime dependencies — just the binary and a volume for the database.

## Architecture

```
MQTT Client ──TCP:1883──▶ Session (auth + topic filter)
                              │
                              ▼
                          Ingestion (FIFO queue per topic → SQLite)
                              │
                              ▼
                          Dispatch (fan-out to workers)
```

- **Session** — accepts TCP connections, authenticates via username/password, filters allowed topics, and forwards PUBLISH to the ingestion layer.
- **Ingestion** — one FIFO queue per topic with sequential consumer. Each message is persisted to the `raw_data` table (SQLite) before proceeding.
- **Dispatch** — receives already-persisted messages and distributes to all registered workers in parallel.

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
# copy and adjust variables
cp .env.example .env

# build
go build -o microbroker ./cmd/main.go

# run
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

### Connection Parameters

| Parameter | Value |
|---|---|
| Host | IP/hostname of the machine |
| Port | `1883` (or the value of `BROKER_PORT`) |
| Username | value of `BROKER_USERNAME` |
| Password | value of `BROKER_PASSWORD` |
| QoS | `0` or `1` |

### Example with mosquitto_pub

```bash
mosquitto_pub \
  -h 127.0.0.1 \
  -p 1883 \
  -u machine01 \
  -P secret123 \
  -t "machine/status" \
  -m '{"machine_id":"CNC-01","status":"running","temperature":72.5,"rpm":1200}'
```

### Example with Python (paho-mqtt)

```python
import paho.mqtt.client as mqtt
import json

client = mqtt.Client(client_id="sensor-01")
client.username_pw_set("machine01", "secret123")
client.connect("127.0.0.1", 1883)

payload = {
    "machine_id": "CNC-01",
    "status": "running",
    "temperature": 72.5,
    "rpm": 1200
}

client.publish("machine/status", json.dumps(payload), qos=1)
client.disconnect()
```

## JSON Payload

The broker accepts any payload in bytes, but the expected usage is JSON. The content is stored as text in the `payload` column of the `raw_data` table.

### Examples by Topic

**machine/status**
```json
{
  "machine_id": "CNC-01",
  "status": "running",
  "temperature": 72.5,
  "rpm": 1200
}
```

**machine/production**
```json
{
  "machine_id": "CNC-01",
  "order_id": "OP-2026-0042",
  "parts_produced": 150,
  "parts_target": 500
}
```

**machine/alarm**
```json
{
  "machine_id": "CNC-01",
  "alarm_code": "E-102",
  "severity": "critical",
  "message": "Overheating detected"
}
```

**machine/oee**
```json
{
  "machine_id": "CNC-01",
  "availability": 0.92,
  "performance": 0.87,
  "quality": 0.99,
  "oee": 0.79
}
```

**machine/counter**
```json
{
  "machine_id": "CNC-01",
  "counter_name": "cycle_count",
  "value": 48230
}
```

## Database Schema (SQLite)

```sql
CREATE TABLE raw_data (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    client     TEXT NOT NULL,
    topic      TEXT NOT NULL,
    timezone   TEXT NOT NULL,
    timestamp  TEXT NOT NULL,
    payload    TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

Indexes on `topic`, `timestamp`, and `client`.

## REST API — Audit

Endpoint to query persisted messages by topic.

```
GET /audit/{topic}
```

### Example

```bash
curl http://localhost:8080/audit/machine/status
```

### Response

```json
[
  {
    "client_id": "sensor-01",
    "topic": "machine/status",
    "timezone": "America/Sao_Paulo",
    "timestamp": "2026-04-22T10:30:00.000000000-03:00",
    "payload": "{\"machine_id\":\"CNC-01\",\"status\":\"running\",\"temperature\":72.5,\"rpm\":1200}"
  }
]
```

Returns `[]` when there are no records for the topic.

## Project Structure

```
cmd/main.go                          → Entrypoint and wiring
internal/
  config/                            → Environment variable loading
  common/                            → Shared logger
  modules/
    protocol/                        → MQTT 3.1.1 codec (encoder/decoder)
    session/                         → TCP server, auth, connection manager
    ingestion/                       → FIFO pipeline + SQLite persistence
    dispatch/                        → Fan-out to workers
  platform/
    database/                        → SQLite connection + migrations
```

## License

MIT
