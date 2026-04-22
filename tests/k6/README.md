# Testes de Carga — k6 + MQTT

Testes de resiliência do broker usando [k6](https://k6.io) com a extensão [xk6-mqtt](https://github.com/pmalhaire/xk6-mqtt).

## Pré-requisitos

- Docker e Docker Compose
- Broker já rodando no host (via `docker compose up -d` ou `go run ./cmd/main.go`)

O k6 roda em container e se conecta ao broker via `host.docker.internal:1883`.

## Como rodar

```bash
# 50 mensagens (default)
./tests/k6/run.sh

# quantidade customizada
./tests/k6/run.sh 200
```

## O que faz

1. Builda um k6 customizado com suporte a MQTT
2. k6 conecta ao broker do host via `host.docker.internal:1883` (com auth), publica N mensagens QoS 1 no tópico `machine/status`
3. Consulta a API `GET /audit/machine/status` e valida se todas as mensagens foram persistidas no SQLite
4. Remove o container do k6

## Estrutura

```
tests/k6/
├── Dockerfile.k6              # k6 v1.7.1 + xk6-mqtt v0.40.3
├── docker-compose.k6.yml      # broker + k6 isolados
├── run.sh                     # orquestrador do teste
└── scripts/
    └── mqtt_publish.js        # script k6 que publica via MQTT
```

## Variáveis de ambiente (docker-compose)

| Variável | Default | Descrição |
|---|---|---|
| `BROKER_ADDR` | `host.docker.internal:1883` | Endereço do broker |
| `BROKER_USER` | `machine01` | Usuário MQTT |
| `BROKER_PASS` | `secret123` | Senha MQTT |
| `MQTT_TOPIC` | `machine/status` | Tópico de publicação |
| `MESSAGE_COUNT` | `50` | Mensagens por execução |

## Saída esperada

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
