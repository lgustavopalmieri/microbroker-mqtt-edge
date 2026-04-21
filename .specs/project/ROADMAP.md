# Roadmap

**Current Milestone:** v1 — Core Broker
**Status:** In Progress

---

## v1 — Core Broker

**Goal:** Broker funcional que aceita conexões MQTT, persiste dados em SQLite e distribui para workers.
**Target:** Todas as features P1 implementadas e testadas.

### Features

**F1: MQTT Protocol Engine** - COMPLETE ✅

- Parsing e encoding de todos os packet types necessários (CONNECT, CONNACK, PUBLISH, PUBACK, SUBSCRIBE, SUBACK, PINGREQ, PINGRESP, DISCONNECT)
- Remaining Length encoding/decoding
- UTF-8 string handling conforme spec MQTT
- Constantes e tipos do protocolo
- 68 testes unitários passando com race detector

**F2: Session Management** - COMPLETE ✅

- TCP listener com accept loop
- Controle de conexões (max 5 clients)
- Autenticação por username/password
- Keep-alive com timeout
- Packet handler (orquestra protocol + session)
- Graceful disconnect
- 31 testes passando com race detector

**F3: Topic Registry** - COMPLETE ✅ (implementada dentro de F2)

- Registro de tópicos via env vars
- Validação de tópicos no PUBLISH
- Máximo 5 tópicos
- 8 testes unitários + 2 testes de integração

**F4: Data Ingestion Pipeline** - COMPLETE ✅

- Filas FIFO nativas por tópico (Go channels)
- Consumer sequencial por fila
- Persistência em SQLite (tabela raw_data) com modernc.org/sqlite (pure Go)
- Lock transacional para consistência (mutex + WAL mode)
- Backpressure natural via channels
- 21 testes (6 unitários queue + 5 unitários pipeline + 10 integração SQLite)

**F5: Worker Dispatch** - PLANNED

- Interface Worker extensível
- Fan-out dispatcher (cada mensagem → todos os workers)
- Logger worker como implementação de referência
- Graceful shutdown de workers

**F6: Configuration & Bootstrap** - PLANNED

- Carregamento de config via env vars
- Wiring de todos os módulos no main.go
- Graceful shutdown com signal handling
- Dockerfile e docker-compose atualizados

---

## v2 — Security & Plugins (Future)

**Goal:** TLS, workers concretos, métricas.

### Features

**TLS/SSL Support** - PLANNED
**Kafka Worker** - PLANNED
**S3 Worker** - PLANNED
**REST Worker** - PLANNED
**Prometheus Metrics** - PLANNED

---

## v3 — Intelligence (Future)

**Goal:** Cálculos industriais em tempo real.

### Features

**OEE Calculator Worker** - PLANNED
**Piece Counter Worker** - PLANNED
**Production Order Tracker** - PLANNED
**WebSocket Output** - PLANNED

---

## Future Considerations

- Suporte a MQTT 5.0
- Cluster mode (múltiplos brokers)
- Dashboard web de monitoramento
- Replay de dados do SQLite
