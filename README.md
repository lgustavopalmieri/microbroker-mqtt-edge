# microbroker-mqtt-edge

Broker MQTT leve e embarcável para ambientes edge/industrial. Recebe mensagens via protocolo MQTT 3.1.1 sobre TCP, persiste em SQLite local e despacha para workers configuráveis. Zero dependências externas em runtime — basta o binário e um volume para o banco.

## Arquitetura

```
Cliente MQTT ──TCP:1883──▶ Session (auth + topic filter)
                              │
                              ▼
                          Ingestion (fila FIFO por tópico → SQLite)
                              │
                              ▼
                          Dispatch (fan-out para workers)
```

- **Session** — aceita conexões TCP, autentica via usuário/senha, filtra tópicos permitidos e encaminha PUBLISH para a camada de ingestão.
- **Ingestion** — uma fila FIFO por tópico com consumer sequencial. Cada mensagem é persistida na tabela `raw_data` (SQLite) antes de seguir adiante.
- **Dispatch** — recebe mensagens já persistidas e distribui para todos os workers registrados em paralelo.

## Variáveis de Ambiente

| Variável | Obrigatória | Default | Descrição |
|---|---|---|---|
| `BROKER_HOST` | Não | `0.0.0.0` | Endereço de bind do servidor TCP |
| `BROKER_PORT` | Não | `1883` | Porta TCP do broker |
| `BROKER_HTTP_PORT` | Não | `8080` | Porta HTTP da API de audit |
| `BROKER_USERNAME` | **Sim** | — | Usuário para autenticação MQTT |
| `BROKER_PASSWORD` | **Sim** | — | Senha para autenticação MQTT |
| `BROKER_TOPICS` | **Sim** | — | Tópicos permitidos (1–5, separados por vírgula) |
| `BROKER_MAX_CLIENTS` | Não | `5` | Máximo de conexões simultâneas (1–5) |
| `BROKER_QUEUE_BUFFER_SIZE` | Não | `10000` | Tamanho do buffer de cada fila interna |
| `BROKER_DB_PATH` | Não | `./data/broker.db` | Caminho do arquivo SQLite |
| `BROKER_TIMEZONE` | Não | `UTC` | Timezone gravado junto com cada mensagem |

## Como Rodar

### Binário local

```bash
# copie e ajuste as variáveis
cp .env.example .env

# build
go build -o microbroker ./cmd/main.go

# execute
./microbroker
```

### Docker Compose

```bash
cp .env.example .env
docker compose up -d
```

O volume `broker-data` persiste o SQLite em `/data/broker.db` dentro do container.

## Conectando um Cliente MQTT

Qualquer client MQTT 3.1.1 funciona (mosquitto_pub, MQTTX, paho-mqtt, etc.).

### Parâmetros de conexão

| Parâmetro | Valor |
|---|---|
| Host | IP/hostname da máquina |
| Porta | `1883` (ou o valor de `BROKER_PORT`) |
| Usuário | valor de `BROKER_USERNAME` |
| Senha | valor de `BROKER_PASSWORD` |
| QoS | `0` ou `1` |

### Exemplo com mosquitto_pub

```bash
mosquitto_pub \
  -h 127.0.0.1 \
  -p 1883 \
  -u machine01 \
  -P secret123 \
  -t "machine/status" \
  -m '{"machine_id":"CNC-01","status":"running","temperature":72.5,"rpm":1200}'
```

### Exemplo com Python (paho-mqtt)

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

## Payload JSON

O broker aceita qualquer payload em bytes, mas o uso esperado é JSON. O conteúdo é gravado como texto na coluna `payload` da tabela `raw_data`.

### Exemplos por tópico

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

## Estrutura do Banco (SQLite)

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

Índices em `topic`, `timestamp` e `client`.

## API REST — Audit

Endpoint para consultar mensagens persistidas por tópico.

```
GET /audit/{topic}
```

### Exemplo

```bash
curl http://localhost:8080/audit/machine/status
```

### Resposta

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

Retorna `[]` quando não há registros para o tópico.

## Estrutura do Projeto

```
cmd/main.go                          → Entrypoint e wiring
internal/
  config/                            → Carregamento de variáveis de ambiente
  common/                            → Logger compartilhado
  modules/
    protocol/                        → Codec MQTT 3.1.1 (encoder/decoder)
    session/                         → Servidor TCP, auth, connection manager
    ingestion/                       → Pipeline FIFO + persistência SQLite
    dispatch/                        → Fan-out para workers
  platform/
    database/                        → Conexão SQLite + migrations
```

## Licença

MIT
