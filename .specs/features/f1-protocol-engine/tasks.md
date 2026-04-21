# F1: MQTT Protocol Engine — Tasks

**Design**: Inline (módulo puro sem arquitetura complexa)
**Spec**: `.specs/features/f1-protocol-engine/spec.md`
**Status**: Draft

---

## Execution Plan

### Phase 1: Foundation (Sequential)

Tipos, constantes e codecs base que tudo depende.

```
T1 → T2 → T3
```

### Phase 2: Decoders (Parallel OK)

Decoders independentes que dependem apenas da foundation.

```
       ┌→ T4 [P] ─┐
T3 ────┼→ T5 [P] ─┼──→ T7
       └→ T6 [P] ─┘
```

### Phase 3: Encoders + Integration (Sequential)

Encoder e reader que completam o módulo.

```
T7 → T8
```

---

## Task Breakdown

### T1: Tipos, Constantes e Erros do Protocolo

**What**: Criar o pacote `internal/protocol` com todos os tipos, constantes e erros do MQTT 3.1.1.
**Where**: `internal/protocol/packet.go`, `internal/protocol/errors.go`
**Depends on**: None
**Reuses**: Nenhum código existente
**Requirement**: PROTO-01

**Done when**:

- [ ] `PacketType` definido como `byte` com constantes para todos os 15 tipos
- [ ] `ConnackReturnCode` definido com todas as 6 constantes
- [ ] `FixedHeader` struct com PacketType, Flags, RemainingLength
- [ ] `ConnectPacket` struct com todos os campos (ProtocolName, ProtocolLevel, flags, KeepAlive, ClientID, WillTopic, WillMessage, Username, Password)
- [ ] `PublishPacket` struct com DUP, QoS, Retain, TopicName, PacketID, Payload
- [ ] `SubscribePacket` struct com PacketID e slice de Subscription
- [ ] Erros sentinela: `ErrMalformedRemainingLength`, `ErrInvalidProtocol`, `ErrInvalidPacketType`, `ErrInvalidQoS`, `ErrEmptyPayload`, `ErrTruncatedData`
- [ ] Métodos helper em ConnectPacket: `HasUsername()`, `HasPassword()`, `HasWill()`, `IsCleanSession()`, `WillQoS()`, `WillRetain()`
- [ ] Gate check passes: `go build ./internal/protocol/...`

**Tests**: unit
**Gate**: build

---

### T2: Remaining Length Codec

**What**: Implementar encode e decode do campo Remaining Length com variable-length encoding.
**Where**: `internal/protocol/codec.go`, `internal/protocol/codec_test.go`
**Depends on**: T1
**Reuses**: Nenhum

**Requirement**: PROTO-02

**Done when**:

- [ ] `DecodeRemainingLength(reader io.Reader) (int, error)` implementado
- [ ] `EncodeRemainingLength(length int) []byte` implementado
- [ ] Testes cobrindo: 1 byte (0, 64, 127), 2 bytes (128, 321, 16383), 3 bytes (16384, 2097151), 4 bytes (2097152, 268435455)
- [ ] Teste de erro: mais de 4 bytes → `ErrMalformedRemainingLength`
- [ ] Teste de roundtrip: encode → decode para todos os valores de boundary
- [ ] Gate check passes: `go test -race ./internal/protocol/...`
- [ ] Test count: ≥10 tests pass

**Tests**: unit
**Gate**: quick

---

### T3: UTF-8 String Codec

**What**: Implementar leitura e escrita de strings UTF-8 no formato MQTT (2 bytes length prefix + conteúdo).
**Where**: `internal/protocol/codec.go` (append), `internal/protocol/codec_test.go` (append)
**Depends on**: T2
**Reuses**: Nenhum

**Requirement**: PROTO-03

**Done when**:

- [ ] `ReadUTF8String(reader io.Reader) (string, error)` implementado
- [ ] `WriteUTF8String(s string) []byte` implementado
- [ ] `ReadBinaryData(reader io.Reader) ([]byte, error)` implementado (para Password e WillMessage)
- [ ] Testes cobrindo: string vazia, string curta, string com caracteres UTF-8 multibyte, string longa
- [ ] Teste de erro: reader com bytes insuficientes
- [ ] Teste de roundtrip: write → read
- [ ] Gate check passes: `go test -race ./internal/protocol/...`
- [ ] Test count: ≥6 tests pass (novos)

**Tests**: unit
**Gate**: quick

---

### T4: CONNECT Packet Decoder [P]

**What**: Implementar o decoder do pacote CONNECT que extrai todos os campos do variable header e payload.
**Where**: `internal/protocol/decoder.go`, `internal/protocol/decoder_test.go`
**Depends on**: T3
**Reuses**: `ReadUTF8String`, `ReadBinaryData` de T3

**Requirement**: PROTO-04

**Done when**:

- [ ] `DecodeConnect(data []byte) (*ConnectPacket, error)` implementado
- [ ] Valida protocol name = "MQTT"
- [ ] Valida protocol level = 4
- [ ] Valida reserved bit = 0
- [ ] Extrai todas as flags corretamente (Username, Password, Will, CleanSession, WillQoS, WillRetain)
- [ ] Extrai KeepAlive (uint16 big-endian)
- [ ] Extrai ClientID (obrigatório)
- [ ] Extrai WillTopic + WillMessage (se Will Flag)
- [ ] Extrai Username (se Username Flag)
- [ ] Extrai Password (se Password Flag)
- [ ] Testes: connect válido com auth, connect sem auth, connect com will, protocol name inválido, protocol level inválido, reserved bit inválido, dados truncados
- [ ] Gate check passes: `go test -race ./internal/protocol/...`
- [ ] Test count: ≥8 tests pass (novos)

**Tests**: unit
**Gate**: quick

---

### T5: PUBLISH Packet Decoder [P]

**What**: Implementar o decoder do pacote PUBLISH que extrai tópico, payload, QoS e PacketID.
**Where**: `internal/protocol/decoder.go` (append), `internal/protocol/decoder_test.go` (append)
**Depends on**: T3
**Reuses**: `ReadUTF8String` de T3

**Requirement**: PROTO-05

**Done when**:

- [ ] `DecodePublish(header FixedHeader, data []byte) (*PublishPacket, error)` implementado
- [ ] Extrai flags do FixedHeader: DUP, QoS, Retain
- [ ] Extrai TopicName via UTF-8 string
- [ ] Extrai PacketID apenas se QoS > 0
- [ ] Extrai Payload (bytes restantes)
- [ ] Valida QoS != 3
- [ ] Valida TopicName não vazio
- [ ] Testes: QoS 0 sem PacketID, QoS 1 com PacketID, QoS inválido (3), tópico vazio, payload vazio (válido), dados truncados
- [ ] Gate check passes: `go test -race ./internal/protocol/...`
- [ ] Test count: ≥6 tests pass (novos)

**Tests**: unit
**Gate**: quick

---

### T6: SUBSCRIBE Packet Decoder [P]

**What**: Implementar o decoder do pacote SUBSCRIBE que extrai PacketID e lista de subscriptions.
**Where**: `internal/protocol/decoder.go` (append), `internal/protocol/decoder_test.go` (append)
**Depends on**: T3
**Reuses**: `ReadUTF8String` de T3

**Requirement**: PROTO-06

**Done when**:

- [ ] `DecodeSubscribe(data []byte) (*SubscribePacket, error)` implementado
- [ ] Extrai PacketID (uint16 big-endian)
- [ ] Extrai lista de TopicFilter + QoS pairs
- [ ] Valida payload não vazio
- [ ] Valida QoS de cada subscription (0, 1 ou 2)
- [ ] Testes: 1 subscription, múltiplas subscriptions, payload vazio, QoS inválido
- [ ] Gate check passes: `go test -race ./internal/protocol/...`
- [ ] Test count: ≥4 tests pass (novos)

**Tests**: unit
**Gate**: quick

---

### T7: Encoder de Respostas do Servidor

**What**: Implementar encoders para CONNACK, PUBACK, SUBACK e PINGRESP.
**Where**: `internal/protocol/encoder.go`, `internal/protocol/encoder_test.go`
**Depends on**: T4, T5, T6 (para validar que os tipos estão corretos)
**Reuses**: `EncodeRemainingLength` de T2

**Requirement**: PROTO-07

**Done when**:

- [ ] `EncodeConnack(sessionPresent bool, returnCode ConnackReturnCode) []byte` implementado
- [ ] `EncodePuback(packetID uint16) []byte` implementado
- [ ] `EncodeSuback(packetID uint16, returnCodes []byte) []byte` implementado
- [ ] `EncodePingresp() []byte` implementado
- [ ] Testes byte-a-byte: CONNACK accepted [0x20,0x02,0x00,0x00], CONNACK bad auth [0x20,0x02,0x00,0x04], CONNACK session present [0x20,0x02,0x01,0x00]
- [ ] Testes byte-a-byte: PUBACK com PacketID=10 [0x40,0x02,0x00,0x0A]
- [ ] Testes byte-a-byte: SUBACK com 1 e 2 return codes
- [ ] Testes byte-a-byte: PINGRESP [0xD0,0x00]
- [ ] Gate check passes: `go test -race ./internal/protocol/...`
- [ ] Test count: ≥8 tests pass (novos)

**Tests**: unit
**Gate**: quick

---

### T8: Fixed Header Reader (Packet Reader)

**What**: Implementar o reader de alto nível que lê o fixed header + remaining bytes de qualquer pacote MQTT de um io.Reader.
**Where**: `internal/protocol/reader.go`, `internal/protocol/reader_test.go`
**Depends on**: T7
**Reuses**: `DecodeRemainingLength` de T2

**Requirement**: PROTO-08

**Done when**:

- [ ] `ReadPacket(reader io.Reader) (FixedHeader, []byte, error)` implementado
- [ ] Lê byte 1 → extrai PacketType (bits 7-4) e Flags (bits 3-0)
- [ ] Lê remaining length via `DecodeRemainingLength`
- [ ] Lê exatamente `remainingLength` bytes do reader
- [ ] Valida PacketType != 0 e != 15 (reserved)
- [ ] Testes: CONNECT completo, PUBLISH completo, PINGREQ (0 bytes remaining), DISCONNECT, packet type inválido (0, 15), EOF no primeiro byte, EOF no remaining length, EOF no body
- [ ] Gate check passes: `go test -race ./internal/protocol/...`
- [ ] Test count: ≥8 tests pass (novos)

**Tests**: unit
**Gate**: quick

**Commit**: `feat(protocol): complete MQTT 3.1.1 protocol engine`

---

## Parallel Execution Map

```
Phase 1 (Sequential):
  T1 ──→ T2 ──→ T3

Phase 2 (Parallel):
  T3 complete, then:
    ├── T4 [P]  CONNECT decoder
    ├── T5 [P]  PUBLISH decoder    } Can run simultaneously
    └── T6 [P]  SUBSCRIBE decoder

Phase 3 (Sequential):
  T4, T5, T6 complete, then:
    T7 ──→ T8
```

---

## Task Granularity Check

| Task | Scope | Status |
|------|-------|--------|
| T1: Tipos e Constantes | 2 files (types + errors) | ✅ Granular |
| T2: Remaining Length Codec | 1 function pair + tests | ✅ Granular |
| T3: UTF-8 String Codec | 1 function pair + tests | ✅ Granular |
| T4: CONNECT Decoder | 1 function + tests | ✅ Granular |
| T5: PUBLISH Decoder | 1 function + tests | ✅ Granular |
| T6: SUBSCRIBE Decoder | 1 function + tests | ✅ Granular |
| T7: Encoders | 4 functions + tests (coeso) | ⚠️ OK — 4 funções simples no mesmo arquivo |
| T8: Fixed Header Reader | 1 function + tests | ✅ Granular |

---

## Diagram-Definition Cross-Check

| Task | Depends On (body) | Diagram Shows | Status |
|------|-------------------|---------------|--------|
| T1 | None | Start | ✅ Match |
| T2 | T1 | T1 → T2 | ✅ Match |
| T3 | T2 | T2 → T3 | ✅ Match |
| T4 | T3 | T3 → T4 | ✅ Match |
| T5 | T3 | T3 → T5 | ✅ Match |
| T6 | T3 | T3 → T6 | ✅ Match |
| T7 | T4, T5, T6 | T4,T5,T6 → T7 | ✅ Match |
| T8 | T7 | T7 → T8 | ✅ Match |

---

## Test Co-location Validation

| Task | Code Layer | Matrix Requires | Task Says | Status |
|------|-----------|----------------|-----------|--------|
| T1 | Protocol types/constants | unit | unit (build gate) | ✅ OK |
| T2 | Protocol decoder (codec) | unit | unit | ✅ OK |
| T3 | Protocol decoder (codec) | unit | unit | ✅ OK |
| T4 | Protocol decoder | unit | unit | ✅ OK |
| T5 | Protocol decoder | unit | unit | ✅ OK |
| T6 | Protocol decoder | unit | unit | ✅ OK |
| T7 | Protocol encoder | unit | unit | ✅ OK |
| T8 | Protocol decoder (reader) | unit | unit | ✅ OK |
