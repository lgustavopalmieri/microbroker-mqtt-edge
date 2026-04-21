# F1: MQTT Protocol Engine — Specification

## Problem Statement

Precisamos de um engine de parsing e encoding de pacotes MQTT 3.1.1 que opere sobre bytes puros, sem nenhuma dependência de I/O ou infraestrutura. Este é o alicerce de todo o broker — se o parsing estiver errado, nada funciona.

## Goals

- [ ] Parser completo e correto de todos os packet types necessários (CONNECT, PUBLISH, SUBSCRIBE, PINGREQ, DISCONNECT)
- [ ] Encoder completo para respostas do servidor (CONNACK, PUBACK, SUBACK, PINGRESP)
- [ ] 100% de cobertura dos edge cases do protocolo (malformed packets, limites de tamanho)
- [ ] Zero dependências externas — apenas stdlib Go

## Out of Scope

| Feature | Reason |
|---------|--------|
| QoS 2 packets (PUBREC, PUBREL, PUBCOMP) | Não necessário para v1 |
| MQTT 5.0 properties | Fora do escopo do projeto |
| Streaming decoder (partial reads) | Complexidade desnecessária; lemos pacotes completos |

---

## User Stories

### P1: Tipos e Constantes do Protocolo ⭐ MVP

**User Story**: Como desenvolvedor do broker, preciso de tipos Go bem definidos para todos os packet types MQTT 3.1.1 para que o código seja type-safe e auto-documentado.

**Why P1**: Fundação para todo o parsing e encoding.

**Acceptance Criteria**:

1. WHEN importo o pacote `protocol` THEN devo ter acesso a constantes para todos os 15 packet types MQTT
2. WHEN uso PacketType THEN o compilador deve impedir valores inválidos via tipo customizado
3. WHEN preciso dos return codes do CONNACK THEN devo ter constantes nomeadas (ConnAccepted, ConnRefusedBadAuth, etc.)
4. WHEN preciso representar um FixedHeader THEN devo ter uma struct com PacketType, Flags e RemainingLength

**Independent Test**: Compilação do pacote + verificação de que constantes existem e têm valores corretos.

---

### P1: Remaining Length Codec ⭐ MVP

**User Story**: Como desenvolvedor do broker, preciso codificar e decodificar o campo Remaining Length do MQTT para ler e escrever pacotes de qualquer tamanho.

**Why P1**: Sem isso, não conseguimos ler nenhum pacote MQTT.

**Acceptance Criteria**:

1. WHEN decodifico remaining length de 1 byte (0-127) THEN devo obter o valor correto
2. WHEN decodifico remaining length de 2 bytes (128-16383) THEN devo obter o valor correto
3. WHEN decodifico remaining length de 3 bytes (16384-2097151) THEN devo obter o valor correto
4. WHEN decodifico remaining length de 4 bytes (2097152-268435455) THEN devo obter o valor correto
5. WHEN decodifico remaining length com mais de 4 bytes THEN devo retornar erro "malformed remaining length"
6. WHEN codifico um valor THEN o resultado deve ser decodificável de volta ao mesmo valor (roundtrip)
7. WHEN codifico 0 THEN devo obter [0x00]
8. WHEN codifico 127 THEN devo obter [0x7F]
9. WHEN codifico 128 THEN devo obter [0x80, 0x01]
10. WHEN codifico 268435455 THEN devo obter [0xFF, 0xFF, 0xFF, 0x7F]

**Independent Test**: Testes unitários com table-driven tests cobrindo todos os ranges.

---

### P1: UTF-8 String Codec ⭐ MVP

**User Story**: Como desenvolvedor do broker, preciso ler e escrever strings UTF-8 no formato MQTT (2 bytes de length prefix + string) para parsear campos como ClientID, TopicName, Username.

**Why P1**: Todas as strings no MQTT usam este formato.

**Acceptance Criteria**:

1. WHEN leio uma UTF-8 string válida THEN devo obter a string correta
2. WHEN leio uma string vazia (length=0) THEN devo obter string vazia sem erro
3. WHEN o reader não tem bytes suficientes THEN devo retornar erro de I/O
4. WHEN escrevo uma string THEN o resultado deve ter 2 bytes de length + conteúdo
5. WHEN faço roundtrip (write → read) THEN devo obter a string original

**Independent Test**: Testes unitários com strings de vários tamanhos.

---

### P1: CONNECT Packet Decoder ⭐ MVP

**User Story**: Como broker, preciso parsear o pacote CONNECT enviado pelo client para extrair ClientID, Username, Password, KeepAlive e flags de conexão.

**Why P1**: Primeiro pacote que todo client envia. Sem isso, nenhuma conexão funciona.

**Acceptance Criteria**:

1. WHEN recebo um CONNECT válido com username e password THEN devo extrair todos os campos corretamente
2. WHEN recebo um CONNECT sem username/password THEN devo parsear sem erro e flags devem ser false
3. WHEN o protocol name não é "MQTT" THEN devo retornar erro
4. WHEN o protocol level não é 4 THEN devo retornar erro
5. WHEN o reserved bit (bit 0 do connect flags) não é 0 THEN devo retornar erro
6. WHEN o CONNECT tem Will Flag THEN devo extrair WillTopic e WillMessage
7. WHEN o CONNECT tem CleanSession=true THEN a flag deve ser true na struct
8. WHEN os dados estão truncados THEN devo retornar erro de I/O

**Independent Test**: Table-driven tests com bytes literais construídos manualmente.

---

### P1: PUBLISH Packet Decoder ⭐ MVP

**User Story**: Como broker, preciso parsear o pacote PUBLISH para extrair o tópico, payload e QoS para processar a mensagem.

**Why P1**: Core do broker — sem isso não recebemos dados.

**Acceptance Criteria**:

1. WHEN recebo PUBLISH QoS 0 THEN devo extrair topic e payload, sem PacketID
2. WHEN recebo PUBLISH QoS 1 THEN devo extrair topic, payload e PacketID
3. WHEN o tópico é vazio THEN devo retornar erro
4. WHEN o QoS é 3 (inválido) THEN devo retornar erro
5. WHEN extraio as flags do primeiro byte THEN DUP, QoS e Retain devem estar corretos

**Independent Test**: Table-driven tests com bytes literais.

---

### P1: SUBSCRIBE Packet Decoder ⭐ MVP

**User Story**: Como broker, preciso parsear o pacote SUBSCRIBE para responder com SUBACK, mesmo que internamente não mantenhamos estado de subscriptions.

**Why P1**: Muitos clients MQTT enviam SUBSCRIBE automaticamente após CONNECT.

**Acceptance Criteria**:

1. WHEN recebo SUBSCRIBE com 1 tópico THEN devo extrair PacketID, TopicFilter e QoS
2. WHEN recebo SUBSCRIBE com múltiplos tópicos THEN devo extrair todos
3. WHEN o payload está vazio THEN devo retornar erro (protocol violation)

**Independent Test**: Table-driven tests.

---

### P1: Encoder de Respostas do Servidor ⭐ MVP

**User Story**: Como broker, preciso construir pacotes de resposta (CONNACK, PUBACK, SUBACK, PINGRESP) para enviar ao client.

**Why P1**: Sem respostas, o client não sabe se a conexão foi aceita ou se o PUBLISH foi recebido.

**Acceptance Criteria**:

1. WHEN construo CONNACK com return code 0x00 THEN os bytes devem ser [0x20, 0x02, 0x00, 0x00]
2. WHEN construo CONNACK com session present THEN byte 3 deve ser 0x01
3. WHEN construo CONNACK com bad auth THEN return code deve ser 0x04
4. WHEN construo PUBACK com PacketID=10 THEN os bytes devem ser [0x40, 0x02, 0x00, 0x0A]
5. WHEN construo SUBACK com 2 return codes THEN devo ter PacketID + 2 bytes de return codes
6. WHEN construo PINGRESP THEN os bytes devem ser [0xD0, 0x00]

**Independent Test**: Verificação byte-a-byte dos pacotes construídos.

---

### P1: Fixed Header Reader ⭐ MVP

**User Story**: Como broker, preciso ler o fixed header de qualquer pacote MQTT para determinar o tipo e tamanho antes de ler o restante.

**Why P1**: Entry point de todo o parsing — lê tipo + remaining length.

**Acceptance Criteria**:

1. WHEN leio o primeiro byte THEN devo extrair PacketType (bits 7-4) e Flags (bits 3-0) corretamente
2. WHEN leio remaining length THEN devo usar o codec de variable-length
3. WHEN o reader retorna EOF no primeiro byte THEN devo retornar erro
4. WHEN o packet type é 0 ou 15 (reserved) THEN devo retornar erro

**Independent Test**: Table-driven tests com io.Reader mockado.

---

## Edge Cases

- WHEN remaining length encoding usa mais de 4 bytes THEN system SHALL retornar erro
- WHEN UTF-8 string tem length maior que bytes disponíveis THEN system SHALL retornar erro de I/O
- WHEN CONNECT packet tem dados truncados em qualquer campo THEN system SHALL retornar erro
- WHEN PUBLISH tem QoS=3 (bits 1 e 2 ambos setados) THEN system SHALL retornar erro
- WHEN packet type é 0 ou 15 THEN system SHALL retornar erro (reserved)
- WHEN SUBSCRIBE payload está vazio THEN system SHALL retornar erro (protocol violation)

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
|---------------|-------|-------|--------|
| PROTO-01 | P1: Tipos e Constantes | Done | ✅ Verified |
| PROTO-02 | P1: Remaining Length Codec | Done | ✅ Verified |
| PROTO-03 | P1: UTF-8 String Codec | Done | ✅ Verified |
| PROTO-04 | P1: CONNECT Decoder | Done | ✅ Verified |
| PROTO-05 | P1: PUBLISH Decoder | Done | ✅ Verified |
| PROTO-06 | P1: SUBSCRIBE Decoder | Done | ✅ Verified |
| PROTO-07 | P1: Encoder de Respostas | Done | ✅ Verified |
| PROTO-08 | P1: Fixed Header Reader | Done | ✅ Verified |

**Coverage:** 8 total, 8 implemented, 8 verified ✅

---

## Success Criteria

- [x] Qualquer client MQTT 3.1.1 (mosquitto_pub, Node-RED, paho) consegue completar handshake CONNECT/CONNACK
- [x] Pacotes PUBLISH são parseados corretamente com payload intacto
- [x] Todos os testes unitários passam com `go test -race ./internal/modules/protocol/...` (68 testes)
- [x] Zero dependências externas no pacote protocol
