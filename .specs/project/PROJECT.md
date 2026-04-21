# Micro Broker MQTT Edge

**Vision:** Um micro broker MQTT construído do zero sobre TCP nativo em Go, projetado para edge computing na Indústria 4.0 — aceita dados de máquinas via MQTT, persiste em SQLite e distribui para workers extensíveis.

**For:** Engenheiros de automação industrial e equipes de IoT que precisam de um broker MQTT leve, seguro e controlado no edge, sem dependência de brokers comerciais.

**Solves:** Brokers open-source não oferecem ACL fino, filas robustas e controle de dados sem versões pagas. Este projeto elimina essa dependência com um broker propositalmente limitado e especializado.

## Goals

- Implementar protocolo MQTT 3.1.1 do zero via TCP com suporte a QoS 0 e QoS 1
- Aceitar até 5 clients simultâneos com autenticação por username/password
- Garantir persistência ordenada de todos os dados em SQLite (zero data loss)
- Distribuir dados persistidos para workers extensíveis (Kafka, S3, REST, cálculos)
- Manter zero dependências externas no core (apenas stdlib Go + SQLite driver)

## Tech Stack

**Core:**

- Language: Go 1.22+
- Protocol: MQTT 3.1.1 (implementação própria sobre TCP)
- Database: SQLite (via `modernc.org/sqlite` — pure Go)

**Key dependencies:**

- `modernc.org/sqlite` — SQLite driver sem CGO
- `github.com/stretchr/testify` — assertions em testes
- `go.uber.org/mock/gomock` — mocking em testes
- `log/slog` — logging (stdlib)

## Scope

**v1 includes:**

- Parser/encoder completo de pacotes MQTT 3.1.1 (CONNECT, CONNACK, PUBLISH, PUBACK, SUBSCRIBE, SUBACK, PINGREQ, PINGRESP, DISCONNECT)
- TCP listener com controle de conexões (max 5 clients)
- Autenticação por username/password via env vars
- Tópicos configuráveis via env vars (max 5)
- Filas FIFO nativas por tópico com processamento sequencial
- Persistência em SQLite (tabela raw_data) com lock transacional
- Worker dispatcher com fan-out para workers registrados
- Worker de log (debug/dev) como implementação de referência
- Graceful shutdown completo
- Suite de testes unitários e de integração

**Explicitly out of scope:**

- QoS 2 (exactly once delivery)
- Retained messages
- Will messages (parsing aceito, mas sem ação)
- Wildcards em tópicos (+, #)
- TLS/SSL (será adicionado em v2)
- Workers concretos (Kafka, S3, REST) — apenas interface e logger worker
- WebSocket listener
- Dashboard ou API de monitoramento

## Constraints

- Zero dependências externas no core (exceto SQLite driver e libs de teste)
- Máximo 5 clients simultâneos (design constraint, não limitação técnica)
- Máximo 5 tópicos por broker
- SQLite como único storage (sem PostgreSQL, sem Redis)
