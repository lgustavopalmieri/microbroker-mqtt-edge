# Documento de Fundação: Protocolo MQTT e Features do Micro Broker

## 1. Visão Geral do Projeto

Este projeto implementa um micro broker MQTT **do zero**, construído diretamente sobre TCP nativo em Go, projetado para edge computing na Indústria 4.0. Diferente de brokers genéricos como Mosquitto ou HiveMQ, este broker é propositalmente limitado e especializado:

- Aceita no máximo **5 clients simultâneos**
- Aceita no máximo **5 tópicos** (definidos por variável de ambiente)
- Clients podem **apenas publicar** (publish-only)
- Autenticação por **usuário e senha**
- Persistência em **SQLite** com filas FIFO nativas
- Pipeline de workers para **forwarding** de dados (Kafka, S3, REST, etc.)

## 2. Protocolo MQTT 3.1.1 — Análise Byte-a-Byte

Referência: [OASIS MQTT v3.1.1 Specification](https://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html)

### 2.1 Estrutura de um Control Packet

Todo pacote MQTT é composto por até 3 partes, sempre nesta ordem:

```
┌──────────────────────────┐
│  Fixed Header (obrigatório) │  ← Presente em TODOS os pacotes
├──────────────────────────┤
│  Variable Header (opcional) │  ← Presente em ALGUNS pacotes
├──────────────────────────┤
│  Payload (opcional)         │  ← Presente em ALGUNS pacotes
└──────────────────────────┘
```

### 2.2 Fixed Header

```
Byte 1:
  Bits [7-4]: Packet Type (4 bits)
  Bits [3-0]: Flags específicas do tipo

Byte 2+: Remaining Length (variable-length encoding)
  - Cada byte usa 7 bits para valor + 1 bit de continuação
  - Máximo de 4 bytes → até 268.435.455 bytes (256 MB)
```

Algoritmo de decodificação do Remaining Length em Go:

```go
func decodeRemainingLength(reader io.Reader) (int, error) {
    multiplier := 1
    value := 0
    buf := make([]byte, 1)

    for {
        if _, err := io.ReadFull(reader, buf); err != nil {
            return 0, err
        }
        encodedByte := buf[0]
        value += int(encodedByte&0x7F) * multiplier
        multiplier *= 128

        if multiplier > 128*128*128*128 {
            return 0, errors.New("malformed remaining length")
        }
        if encodedByte&0x80 == 0 {
            break
        }
    }
    return value, nil
}
```

Algoritmo de codificação:

```go
func encodeRemainingLength(length int) []byte {
    var encoded []byte
    for {
        encodedByte := byte(length % 128)
        length /= 128
        if length > 0 {
            encodedByte |= 0x80
        }
        encoded = append(encoded, encodedByte)
        if length == 0 {
            break
        }
    }
    return encoded
}
```

### 2.3 Tipos de Control Packets

Para nosso broker, precisamos implementar apenas um subconjunto do protocolo:

| Tipo | Valor | Direção | Necessário? | Descrição |
|------|-------|---------|-------------|-----------|
| CONNECT | 1 | Client → Server | ✅ SIM | Requisição de conexão |
| CONNACK | 2 | Server → Client | ✅ SIM | Resposta de conexão |
| PUBLISH | 3 | Client → Server | ✅ SIM | Publicação de mensagem |
| PUBACK | 4 | Server → Client | ✅ SIM (QoS 1) | Confirmação de publicação |
| SUBSCRIBE | 8 | Client → Server | ⚠️ PARCIAL | Nosso broker auto-subscreve internamente |
| SUBACK | 9 | Server → Client | ⚠️ PARCIAL | Resposta ao subscribe |
| PINGREQ | 12 | Client → Server | ✅ SIM | Keep-alive request |
| PINGRESP | 13 | Server → Client | ✅ SIM | Keep-alive response |
| DISCONNECT | 14 | Client → Server | ✅ SIM | Desconexão limpa |

Pacotes que NÃO precisamos implementar inicialmente:
- PUBREC/PUBREL/PUBCOMP (QoS 2 — não necessário para nosso caso)
- UNSUBSCRIBE/UNSUBACK (clients não subscrevem externamente)

### 2.4 CONNECT Packet — Detalhamento Completo

Este é o pacote mais complexo e o primeiro que o client envia.

```
Fixed Header:
  Byte 1: 0x10 (tipo=1, flags=0000)
  Byte 2+: Remaining Length

Variable Header (10 bytes fixos):
  Bytes 1-2: Protocol Name Length = 0x00, 0x04
  Bytes 3-6: "MQTT" (0x4D, 0x51, 0x54, 0x54)
  Byte 7:    Protocol Level = 0x04 (v3.1.1)
  Byte 8:    Connect Flags
             Bit 7: Username Flag
             Bit 6: Password Flag
             Bit 5: Will Retain
             Bits 4-3: Will QoS
             Bit 2: Will Flag
             Bit 1: Clean Session
             Bit 0: Reserved (deve ser 0)
  Bytes 9-10: Keep Alive (uint16, big-endian, em segundos)

Payload (na ordem):
  1. Client Identifier (UTF-8 string, obrigatório)
  2. Will Topic (se Will Flag = 1)
  3. Will Message (se Will Flag = 1)
  4. Username (se Username Flag = 1)
  5. Password (se Password Flag = 1)
```

Exemplo de parsing em Go:

```go
type ConnectPacket struct {
    ProtocolName  string
    ProtocolLevel byte
    ConnectFlags  byte
    KeepAlive     uint16
    ClientID      string
    WillTopic     string
    WillMessage   []byte
    Username      string
    Password      []byte
}

func (p *ConnectPacket) HasUsername() bool  { return p.ConnectFlags&0x80 != 0 }
func (p *ConnectPacket) HasPassword() bool  { return p.ConnectFlags&0x40 != 0 }
func (p *ConnectPacket) HasWill() bool      { return p.ConnectFlags&0x04 != 0 }
func (p *ConnectPacket) CleanSession() bool { return p.ConnectFlags&0x02 != 0 }
func (p *ConnectPacket) WillQoS() byte      { return (p.ConnectFlags >> 3) & 0x03 }
func (p *ConnectPacket) WillRetain() bool   { return p.ConnectFlags&0x20 != 0 }
```

### 2.5 CONNACK Packet

```
Fixed Header:
  Byte 1: 0x20 (tipo=2, flags=0000)
  Byte 2: 0x02 (remaining length = 2)

Variable Header:
  Byte 1: Connect Acknowledge Flags
           Bits 7-1: Reserved (0)
           Bit 0: Session Present
  Byte 2: Return Code
           0x00 = Connection Accepted
           0x01 = Unacceptable protocol version
           0x02 = Identifier rejected
           0x03 = Server unavailable
           0x04 = Bad username or password
           0x05 = Not authorized
```

```go
func buildConnack(sessionPresent bool, returnCode byte) []byte {
    packet := []byte{0x20, 0x02}
    if sessionPresent {
        packet = append(packet, 0x01)
    } else {
        packet = append(packet, 0x00)
    }
    packet = append(packet, returnCode)
    return packet
}
```

### 2.6 PUBLISH Packet

```
Fixed Header:
  Byte 1: 0x3X
    Bits [7-4]: 0011 (tipo=3)
    Bit 3: DUP flag
    Bits 2-1: QoS level (00=QoS0, 01=QoS1, 10=QoS2)
    Bit 0: RETAIN flag
  Byte 2+: Remaining Length

Variable Header:
  Topic Name (UTF-8 string: 2 bytes length + string)
  Packet Identifier (2 bytes, apenas se QoS > 0)

Payload:
  Application Message (bytes restantes)
```

```go
type PublishPacket struct {
    DUP       bool
    QoS       byte
    Retain    bool
    TopicName string
    PacketID  uint16 // apenas se QoS > 0
    Payload   []byte
}

func parsePublishFlags(firstByte byte) (dup bool, qos byte, retain bool) {
    dup = firstByte&0x08 != 0
    qos = (firstByte >> 1) & 0x03
    retain = firstByte&0x01 != 0
    return
}
```

### 2.7 PUBACK Packet (QoS 1)

```
Fixed Header:
  Byte 1: 0x40 (tipo=4, flags=0000)
  Byte 2: 0x02 (remaining length = 2)

Variable Header:
  Bytes 1-2: Packet Identifier (mesmo do PUBLISH)
```

```go
func buildPuback(packetID uint16) []byte {
    return []byte{
        0x40, 0x02,
        byte(packetID >> 8),
        byte(packetID & 0xFF),
    }
}
```

### 2.8 PINGREQ / PINGRESP

```
PINGREQ:  [0xC0, 0x00]  (2 bytes, sem variable header, sem payload)
PINGRESP: [0xD0, 0x00]  (2 bytes, sem variable header, sem payload)
```

### 2.9 DISCONNECT

```
DISCONNECT: [0xE0, 0x00]  (2 bytes, sem variable header, sem payload)
```

### 2.10 Leitura de UTF-8 Strings (padrão MQTT)

Todas as strings no MQTT são prefixadas com 2 bytes de comprimento (big-endian):

```go
func readUTF8String(reader io.Reader) (string, error) {
    lenBuf := make([]byte, 2)
    if _, err := io.ReadFull(reader, lenBuf); err != nil {
        return "", err
    }
    length := int(binary.BigEndian.Uint16(lenBuf))
    strBuf := make([]byte, length)
    if _, err := io.ReadFull(reader, strBuf); err != nil {
        return "", err
    }
    return string(strBuf), nil
}

func writeUTF8String(s string) []byte {
    length := len(s)
    buf := make([]byte, 2+length)
    binary.BigEndian.PutUint16(buf, uint16(length))
    copy(buf[2:], s)
    return buf
}
```


## 3. Features — Análise Detalhada e Decisões de Design

### 3.1 Feature: Conexão TCP Nativa com Autenticação

**Requisitos:**
- Listener TCP na porta configurável (padrão 1883)
- Máximo de 5 clients simultâneos
- Autenticação por username/password via CONNECT packet
- Credenciais definidas por variável de ambiente
- Client deve manter conexão estável (keep-alive)
- 6º client é rejeitado silenciosamente (conexão fechada)

**Fluxo de Conexão:**

```
Client                          Broker
  │                               │
  │──── TCP Connect ─────────────>│  1. Aceita conexão TCP
  │                               │  2. Verifica limite de clients (≤5)
  │                               │     Se cheio → fecha TCP imediatamente
  │──── CONNECT Packet ──────────>│  3. Parse do CONNECT
  │                               │  4. Valida protocol name/level
  │                               │  5. Valida username/password
  │<──── CONNACK (0x00) ─────────│  6. Aceita → CONNACK success
  │                               │     Rejeita → CONNACK error + fecha
  │                               │
  │──── PINGREQ ─────────────────>│  7. Keep-alive periódico
  │<──── PINGRESP ───────────────│
  │                               │
  │──── DISCONNECT ──────────────>│  8. Desconexão limpa
  │                               │     Libera slot de client
```

**Decisões de implementação:**

1. O controle de conexões usa um `sync.Mutex` + contador atômico para garantir thread-safety
2. Cada conexão aceita gera uma goroutine dedicada para leitura
3. Keep-alive timeout: se o client não enviar nada em 1.5x o keep-alive, desconectamos
4. Credenciais: `BROKER_USERNAME` e `BROKER_PASSWORD` como env vars

```go
// Exemplo de controle de conexões
type ConnectionManager struct {
    mu         sync.Mutex
    clients    map[string]*Client
    maxClients int
}

func (cm *ConnectionManager) CanAccept() bool {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    return len(cm.clients) < cm.maxClients
}

func (cm *ConnectionManager) Add(client *Client) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    if len(cm.clients) >= cm.maxClients {
        return ErrMaxClientsReached
    }
    cm.clients[client.ID] = client
    return nil
}

func (cm *ConnectionManager) Remove(clientID string) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    delete(cm.clients, clientID)
}
```

### 3.2 Feature: Tópicos Configuráveis

**Requisitos:**
- Tópicos definidos por variável de ambiente
- Máximo de 5 tópicos
- Clients podem APENAS publicar nos tópicos definidos
- Publicação em tópico não permitido → rejeitar silenciosamente (ou fechar conexão)

**Configuração via env:**

```bash
# Formato: lista separada por vírgula
BROKER_TOPICS="machine/status,machine/production,machine/alarm,machine/oee,machine/counter"
```

**Validação de tópicos:**

```go
type TopicRegistry struct {
    allowedTopics map[string]struct{}
}

func NewTopicRegistry(topics []string) (*TopicRegistry, error) {
    if len(topics) == 0 || len(topics) > 5 {
        return nil, fmt.Errorf("topic count must be between 1 and 5, got %d", len(topics))
    }
    allowed := make(map[string]struct{}, len(topics))
    for _, t := range topics {
        t = strings.TrimSpace(t)
        if t == "" {
            return nil, fmt.Errorf("empty topic name not allowed")
        }
        allowed[t] = struct{}{}
    }
    return &TopicRegistry{allowedTopics: allowed}, nil
}

func (tr *TopicRegistry) IsAllowed(topic string) bool {
    _, ok := tr.allowedTopics[topic]
    return ok
}

func (tr *TopicRegistry) Topics() []string {
    topics := make([]string, 0, len(tr.allowedTopics))
    for t := range tr.allowedTopics {
        topics = append(topics, t)
    }
    return topics
}
```

### 3.3 Feature: Ingestão de Dados com Filas FIFO

**Requisitos:**
- 1 fila FIFO por tópico (máximo 5 filas)
- Cada dado recebido via PUBLISH entra na fila do respectivo tópico
- Processamento sequencial por fila (garante ordem)
- Após salvar no SQLite, dado vai para canal de workers

**Arquitetura das filas:**

```
PUBLISH (topic=machine/status)  ──→  Fila FIFO [topic: machine/status]
PUBLISH (topic=machine/alarm)   ──→  Fila FIFO [topic: machine/alarm]
PUBLISH (topic=machine/oee)     ──→  Fila FIFO [topic: machine/oee]
...

Cada fila tem 1 consumer goroutine:
  Fila → Consumer → SQLite (com lock) → Canal de Workers
```

**Implementação da fila FIFO nativa:**

```go
// Message representa um dado recebido do broker
type Message struct {
    ClientID  string
    Topic     string
    Payload   []byte
    Timezone  string
    Timestamp time.Time
}

// TopicQueue é uma fila FIFO thread-safe para um tópico
type TopicQueue struct {
    topic    string
    messages chan Message
    done     chan struct{}
}

func NewTopicQueue(topic string, bufferSize int) *TopicQueue {
    return &TopicQueue{
        topic:    topic,
        messages: make(chan Message, bufferSize),
        done:     make(chan struct{}),
    }
}

func (q *TopicQueue) Enqueue(msg Message) {
    q.messages <- msg // bloqueia se buffer cheio (backpressure)
}

// StartConsumer processa mensagens sequencialmente
func (q *TopicQueue) StartConsumer(ctx context.Context, store Store, workerChan chan<- Message) {
    go func() {
        defer close(q.done)
        for {
            select {
            case <-ctx.Done():
                return
            case msg, ok := <-q.messages:
                if !ok {
                    return
                }
                // 1. Salva no SQLite com lock
                if err := store.SaveRawData(ctx, msg); err != nil {
                    // log error, mas não perde a mensagem
                    // retry ou dead-letter queue
                    continue
                }
                // 2. Repassa para workers
                workerChan <- msg
            }
        }
    }()
}
```

### 3.4 Feature: Persistência SQLite (raw_data)

**Requisitos:**
- Tabela `raw_data` com colunas: client, topic, timezone, timestamp, payload (JSON)
- Lock na tabela durante transação (SQLite já faz isso nativamente com WAL mode)
- Garantia de que o dado foi salvo antes de consumir o próximo da fila
- 5 filas concorrentes escrevendo → serialização via mutex

**Schema:**

```sql
CREATE TABLE IF NOT EXISTS raw_data (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    client    TEXT    NOT NULL,
    topic     TEXT    NOT NULL,
    timezone  TEXT    NOT NULL,
    timestamp TEXT    NOT NULL,  -- ISO 8601
    payload   TEXT    NOT NULL,  -- JSON
    created_at TEXT   NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_raw_data_topic ON raw_data(topic);
CREATE INDEX idx_raw_data_timestamp ON raw_data(timestamp);
```

**Implementação do Store com lock:**

```go
type SQLiteStore struct {
    db *sql.DB
    mu sync.Mutex // garante serialização entre as 5 filas
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
    db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
    if err != nil {
        return nil, fmt.Errorf("opening sqlite: %w", err)
    }
    // SQLite com WAL mode permite leituras concorrentes
    // mas escritas são serializadas — nosso mutex garante isso explicitamente
    db.SetMaxOpenConns(1) // SQLite não suporta múltiplas escritas simultâneas
    return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) SaveRawData(ctx context.Context, msg Message) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback()

    _, err = tx.ExecContext(ctx,
        `INSERT INTO raw_data (client, topic, timezone, timestamp, payload)
         VALUES (?, ?, ?, ?, ?)`,
        msg.ClientID,
        msg.Topic,
        msg.Timezone,
        msg.Timestamp.Format(time.RFC3339Nano),
        string(msg.Payload),
    )
    if err != nil {
        return fmt.Errorf("insert raw_data: %w", err)
    }

    return tx.Commit()
}
```

**Nota sobre CGO e SQLite:**
- `mattn/go-sqlite3` requer CGO — aceitável para edge computing
- Alternativa: `modernc.org/sqlite` (pure Go, sem CGO) — melhor para cross-compilation
- Decisão: usar `modernc.org/sqlite` para manter zero dependências de C

### 3.5 Feature: Workers (Pipeline de Forwarding)

**Requisitos:**
- Após salvar no SQLite, dado vai para um canal compartilhado
- Múltiplos workers consomem deste canal simultaneamente
- Workers são plugins: S3, Kafka, REST, WebSocket, cálculos (OEE, contagem)
- Arquitetura fan-out: cada dado é enviado para TODOS os workers registrados

**Padrão Fan-Out:**

```
                    ┌──→ Worker S3
                    │
SQLite → Canal ─────┼──→ Worker Kafka
                    │
                    ┼──→ Worker REST
                    │
                    └──→ Worker OEE Calculator
```

**Interface do Worker:**

```go
// Worker define o contrato para qualquer plugin de forwarding
type Worker interface {
    // Name retorna o nome identificador do worker
    Name() string
    // Process recebe uma mensagem já persistida e a processa
    Process(ctx context.Context, msg Message) error
    // Close libera recursos do worker
    Close() error
}

// WorkerDispatcher distribui mensagens para todos os workers registrados
type WorkerDispatcher struct {
    workers []Worker
    input   <-chan Message
    logger  Logger
}

func NewWorkerDispatcher(input <-chan Message, workers []Worker, logger Logger) *WorkerDispatcher {
    return &WorkerDispatcher{
        workers: workers,
        input:   input,
        logger:  logger,
    }
}

func (d *WorkerDispatcher) Start(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case msg, ok := <-d.input:
                if !ok {
                    return
                }
                // Fan-out: envia para todos os workers
                var wg sync.WaitGroup
                for _, w := range d.workers {
                    wg.Add(1)
                    go func(worker Worker) {
                        defer wg.Done()
                        if err := worker.Process(ctx, msg); err != nil {
                            d.logger.Error("worker failed",
                                "worker", worker.Name(),
                                "topic", msg.Topic,
                                "error", err,
                            )
                        }
                    }(w)
                }
                wg.Wait() // espera todos processarem antes do próximo
            }
        }
    }()
}
```

## 4. Variáveis de Ambiente

```bash
# Servidor TCP
BROKER_HOST=0.0.0.0
BROKER_PORT=1883

# Autenticação
BROKER_USERNAME=machine01
BROKER_PASSWORD=secret123

# Tópicos (máximo 5, separados por vírgula)
BROKER_TOPICS=machine/status,machine/production,machine/alarm,machine/oee,machine/counter

# Limites
BROKER_MAX_CLIENTS=5
BROKER_QUEUE_BUFFER_SIZE=10000

# SQLite
BROKER_DB_PATH=./data/broker.db

# Timezone padrão
BROKER_TIMEZONE=America/Sao_Paulo
```

## 5. Fluxo Completo de Dados

```
┌─────────────┐     TCP      ┌──────────────────┐
│  Device/     │─────────────>│  TCP Listener    │
│  Node-RED/   │   CONNECT    │  (net.Listen)    │
│  MQTT Client │              └────────┬─────────┘
└─────────────┘                        │
                                       ▼
                              ┌──────────────────┐
                              │  Connection      │
                              │  Manager         │
                              │  (max 5 clients) │
                              └────────┬─────────┘
                                       │ PUBLISH
                                       ▼
                              ┌──────────────────┐
                              │  Topic Registry  │
                              │  (valida tópico) │
                              └────────┬─────────┘
                                       │
                                       ▼
                              ┌──────────────────┐
                              │  Topic Queue     │
                              │  (FIFO por       │
                              │   tópico, max 5) │
                              └────────┬─────────┘
                                       │ sequencial
                                       ▼
                              ┌──────────────────┐
                              │  SQLite Store    │
                              │  (raw_data)      │
                              │  com lock/tx     │
                              └────────┬─────────┘
                                       │
                                       ▼
                              ┌──────────────────┐
                              │  Worker          │
                              │  Dispatcher      │
                              │  (fan-out)       │
                              └────────┬─────────┘
                                       │
                          ┌────────────┼────────────┐
                          ▼            ▼            ▼
                     ┌────────┐  ┌────────┐  ┌────────┐
                     │ Worker │  │ Worker │  │ Worker │
                     │ S3     │  │ Kafka  │  │ OEE    │
                     └────────┘  └────────┘  └────────┘
```

## 6. Decisões Técnicas Importantes

### 6.1 QoS Level

Para nosso caso de uso (edge computing industrial), implementaremos:
- **QoS 0** (at most once): para dados de alta frequência onde perda eventual é aceitável
- **QoS 1** (at least once): para dados críticos como alarmes e contagem de peças

NÃO implementaremos QoS 2 (exactly once) — a complexidade do handshake de 4 pacotes não justifica o benefício no nosso cenário, pois a persistência em SQLite já garante a durabilidade.

### 6.2 Por que não usar SUBSCRIBE externamente?

No nosso modelo, o broker é o próprio consumidor. Os clients apenas publicam dados. O broker internamente "subscreve" a todos os tópicos configurados e processa os dados. Isso simplifica enormemente a implementação e elimina a necessidade de gerenciar árvores de subscriptions.

Porém, devemos responder a pacotes SUBSCRIBE que clients enviem (alguns clients MQTT enviam SUBSCRIBE automaticamente após CONNECT). A resposta será um SUBACK com o QoS solicitado, mas internamente não precisamos manter estado de subscriptions.

### 6.3 Backpressure

Se o SQLite não conseguir acompanhar a taxa de ingestão:
1. O canal da fila FIFO enche (buffer configurável)
2. O `Enqueue` bloqueia
3. O handler do PUBLISH bloqueia
4. O TCP buffer do client enche
5. O client percebe a lentidão

Isso é backpressure natural e desejável. O buffer da fila (`BROKER_QUEUE_BUFFER_SIZE`) deve ser dimensionado para absorver picos.

### 6.4 Graceful Shutdown

O broker deve fazer shutdown gracioso:
1. Parar de aceitar novas conexões TCP
2. Enviar DISCONNECT para todos os clients conectados
3. Drenar todas as filas FIFO (processar mensagens pendentes)
4. Aguardar workers finalizarem
5. Fechar conexão SQLite
6. Encerrar

```go
func gracefulShutdown(ctx context.Context, cancel context.CancelFunc, server *Server) {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    <-sigCh
    log.Println("Shutting down gracefully...")
    cancel() // sinaliza todas as goroutines via context

    // Aguarda com timeout
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()

    if err := server.Shutdown(shutdownCtx); err != nil {
        log.Printf("Forced shutdown: %v", err)
    }
}
```

## 7. Referências

- [MQTT v3.1.1 OASIS Standard](https://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html) — Especificação oficial
- [Eclipse Mosquitto](https://github.com/eclipse-mosquitto/mosquitto) — Broker de referência em C
- [Mochi MQTT](https://github.com/mochi-mqtt/server) — Broker embeddable em Go (referência de arquitetura)
- [Sol MQTT Broker](https://github.com/codepr/sol) — Broker minimalista em C (referência de baixo nível)
- [Solace MQTT 3.1.1 Conformance Spec](https://docs.solace.com/API/MQTT-311-Prtl-Conformance-Spec/Table%20of%20Contents.htm) — Detalhamento byte-a-byte

Content was rephrased for compliance with licensing restrictions.
