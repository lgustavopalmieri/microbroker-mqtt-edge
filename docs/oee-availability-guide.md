# OEE Availability — End-to-End Guide

This guide walks you through running the broker, sending machine state events via MQTT, and watching the availability metric update in real time — either via WebSocket or a REST query.

---

## How it works (big picture)

```
Your machine / PLC
        │
        │  MQTT PUBLISH  (topic: machine/state)
        ▼
  microbroker-mqtt-edge  (TCP :1883)
        │
        ├── persists state_change → SQLite (state_intervals table)
        │
        └── feeds Live Engine (ticks every 1 s)
                │
                ├── WebSocket push  → ws://localhost:8080/availability/{machine}/ws
                └── REST query      → http://localhost:8080/availability/{machine}?from=...&to=...
```

Every time a machine transitions between states (`running`, `stopped`, `setup`, etc.), you publish one MQTT message. The broker records the pair of intervals, calculates **Availability = Run Time / Planned Production Time**, and broadcasts the result to any connected WebSocket client.

---

## Prerequisites

| Tool | Purpose | Install |
|---|---|---|
| Go 1.21+ **or** Docker | Run the broker | [go.dev](https://go.dev) / [docker.com](https://docker.com) |
| `mosquitto_pub` | Send MQTT messages from the terminal | `brew install mosquitto` / `apt install mosquitto-clients` |
| `wscat` | Test the WebSocket stream in the terminal | `npm install -g wscat` |
| `curl` | Query the REST endpoint | pre-installed on most systems |

---

## Step 1 — Configure

```bash
cp .env.example .env
```

Open `.env` and set at minimum:

```bash
BROKER_USERNAME=admin
BROKER_PASSWORD=secret

# machine/state must be in this list
BROKER_TOPICS=machine/state,machine/production,machine/cycle,machine/quality,machine/alarm

# Enable the OEE module
BROKER_OEE_ENABLED=true
BROKER_OEE_STATE_TOPIC=machine/state
BROKER_OEE_TICK_INTERVAL=1s
BROKER_OEE_WS_ENABLED=true
```

---

## Step 2 — (Optional) Define your shift schedule

Availability = Run Time / **Planned Production Time**. Without a shift config the broker uses the full query window as planned time, which still works but gives you less accurate numbers.

Create a file `shifts.json`:

```json
[
  {
    "id": "morning-cnc01",
    "name": "Morning Shift",
    "machine_id": "CNC-01",
    "start_minute": 480,
    "end_minute": 960,
    "weekdays": [1, 2, 3, 4, 5],
    "timezone": "America/Sao_Paulo",
    "breaks": [
      { "start_minute": 600, "end_minute": 630, "type": "rest" },
      { "start_minute": 720, "end_minute": 750, "type": "meal"  }
    ]
  }
]
```

> **`start_minute` / `end_minute`**: minutes from midnight. 480 = 08:00, 960 = 16:00.
> **`weekdays`**: 0 = Sunday … 6 = Saturday. `[1,2,3,4,5]` = Monday–Friday.
> **`machine_id`**: use `"*"` to apply the shift to all machines that have no specific config.

Add the path to `.env`:

```bash
BROKER_OEE_SHIFTS_PATH=/absolute/path/to/shifts.json
```

---

## Step 3 — Start the broker

**Binary:**

```bash
go build -o microbroker ./cmd/broker
./microbroker
```

**Docker Compose:**

```bash
docker compose up -d
docker compose logs -f broker
```

You should see:

```
INFO  database ready       path=/data/broker.db
INFO  audit API started    address=0.0.0.0:8080
INFO  broker ready
```

---

## Step 4 — Open a real-time WebSocket stream

In a new terminal, connect before sending any events so you see the first snapshot arrive:

```bash
wscat -c "ws://localhost:8080/availability/CNC-01/ws"
```

The connection is open. The broker will push a JSON snapshot every second once the machine starts sending state changes.

---

## Step 5 — Send state_change events

Open another terminal. Use `mosquitto_pub` to simulate a machine transitioning between states.

### Machine starts running

```bash
mosquitto_pub -h localhost -p 1883 \
  -u admin -P secret \
  -t machine/state \
  -m '{"machine_id":"CNC-01","state":"running","timestamp":"'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"}'
```

### Machine stops (unplanned downtime)

```bash
mosquitto_pub -h localhost -p 1883 \
  -u admin -P secret \
  -t machine/state \
  -m '{"machine_id":"CNC-01","state":"stopped","reason":"jam","timestamp":"'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"}'
```

### Machine resumes

```bash
mosquitto_pub -h localhost -p 1883 \
  -u admin -P secret \
  -t machine/state \
  -m '{"machine_id":"CNC-01","state":"running","timestamp":"'"$(date -u +%Y-%m-%dT%H:%M:%SZ)"'"}'
```

### Full state list

| State | `IsDowntime` | `IsPlannedStop` | Typical cause |
|---|---|---|---|
| `running` | no | no | Normal production |
| `stopped` | **yes** | no | Unplanned breakdown / jam |
| `setup` | **yes** | **yes** | Changeover / tooling |
| `maintenance` | **yes** | **yes** | Scheduled maintenance |
| `idle` | no | no | Waiting for material / operator |
| `off` | no | no | Shift end / power-down |

---

## Step 6 — Watch the WebSocket terminal

After the first state transition the engine starts tracking `CNC-01`. Every second you should see a JSON snapshot like this appear in the `wscat` terminal:

```json
{
  "machine_id": "CNC-01",
  "window_from": "2026-06-04T08:00:00Z",
  "window_to":   "2026-06-04T09:15:00Z",
  "availability": 0.9333,
  "planned_time_ns": 4500000000000,
  "run_time_ns":     4200000000000,
  "planned_downtime_ns": 0,
  "unplanned_downtime_ns": 300000000000,
  "interval_count": 3,
  "has_data": true,
  "computed_at": "2026-06-04T09:15:42Z",
  "flags": {}
}
```

> `availability: 0.9333` means **93.33%** — the machine ran for 70 out of 75 minutes.
> Each time you publish a `stopped` event the number drops; `running` makes it recover.

---

## Step 7 — Query availability on demand (REST)

Use any time window you want. Timestamps must be RFC 3339.

```bash
curl -s "http://localhost:8080/availability/CNC-01?from=2026-06-04T06:00:00Z&to=2026-06-04T14:00:00Z" | jq .
```

Response:

```json
{
  "machine_id": "CNC-01",
  "window_from": "2026-06-04T06:00:00Z",
  "window_to":   "2026-06-04T14:00:00Z",
  "availability": 0.8881,
  "has_data": true,
  "flags": {}
}
```

### Response flags

| Flag | Meaning |
|---|---|
| `shift_config_missing` | No shift defined for this machine; full window used as planned time |
| `open_interval_clipped` | Machine is still running; the open interval was clipped to `to` |
| `no_data` | No state transitions recorded in the window |

---

## Full flow recap

```
1. Start broker          →  go build && ./microbroker
2. Open WS terminal      →  wscat -c ws://localhost:8080/availability/CNC-01/ws
3. Publish "running"     →  mosquitto_pub ... state=running
                             WebSocket shows first snapshot (availability ≈ 1.0)
4. Publish "stopped"     →  mosquitto_pub ... state=stopped
                             WebSocket shows availability dropping each second
5. Publish "running"     →  mosquitto_pub ... state=running
                             WebSocket shows availability recovering
6. REST query any time   →  curl http://localhost:8080/availability/CNC-01?from=...&to=...
```

---

## Troubleshooting

**No snapshot on WebSocket**
- Check `BROKER_OEE_ENABLED=true` in `.env`
- Confirm `machine/state` is in `BROKER_TOPICS`
- The engine only tracks machines after they send their first event — connect WS *after* the first MQTT publish, or wait for the next tick

**`availability: 0` with `shift_config_missing: true`**
- Normal when no `shifts.json` is configured; the full window is used as planned time
- Add a `shifts.json` and set `BROKER_OEE_SHIFTS_PATH` for accurate planned time

**MQTT publish rejected**
- Verify username/password match `BROKER_USERNAME` / `BROKER_PASSWORD`
- Confirm `machine/state` is listed in `BROKER_TOPICS` (exact match, no wildcards)

**`400` from REST endpoint**
- `from` and `to` must be valid RFC 3339 timestamps (e.g. `2026-06-04T08:00:00Z`)
- `from` must be strictly before `to`
