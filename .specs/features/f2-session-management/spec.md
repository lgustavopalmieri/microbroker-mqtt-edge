# F2: Session Management — Specification

## Problem Statement

O broker precisa aceitar conexões TCP de clients MQTT, autenticá-los, gerenciar o ciclo de vida da conexão (keep-alive, disconnect) e limitar o número de clients simultâneos. Este módulo é a "porta de entrada" do broker.

## Goals

- [ ] TCP listener funcional que aceita conexões na porta configurável
- [ ] Autenticação por username/password via variáveis de ambiente
- [ ] Controle de máximo 5 clients simultâneos com rejeição do 6º
- [ ] Keep-alive funcional com timeout automático
- [ ] Packet handler que orquestra o fluxo CONNECT → PUBLISH → DISCONNECT

## Out of Scope

| Feature | Reason |
|---------|--------|
| TLS/SSL | v2 |
| Múltiplos listeners (WebSocket) | v2 |
| Session persistence (CleanSession=0) | Complexidade desnecessária para edge |
| Will Message publishing | Parsing aceito, mas sem ação |

---

## User Stories

### P1: Entidade Client e Erros de Domínio ⭐ MVP

**User Story**: Como desenvolvedor do broker, preciso de uma entidade Client que represente uma conexão ativa com seus metadados e estado.

**Why P1**: Fundação para todo o gerenciamento de sessão.

**Acceptance Criteria**:

1. WHEN crio um Client THEN devo ter ID, conn, keepAlive e timestamps
2. WHEN chamo `ResetDeadline()` THEN o deadline da conexão TCP deve ser atualizado para `now + 1.5 * keepAlive`
3. WHEN chamo `Write(data)` THEN os bytes devem ser escritos na conexão TCP
4. WHEN chamo `Close()` THEN a conexão TCP deve ser fechada

**Independent Test**: Testes unitários com `net.Pipe()`.

---

### P1: Autenticação ⭐ MVP

**User Story**: Como operador do broker, quero que apenas clients com username/password corretos possam se conectar, para garantir segurança no edge.

**Why P1**: Segurança básica é obrigatória.

**Acceptance Criteria**:

1. WHEN client envia username e password corretos THEN autenticação deve retornar true
2. WHEN client envia username incorreto THEN autenticação deve retornar false
3. WHEN client envia password incorreto THEN autenticação deve retornar false
4. WHEN client não envia username/password THEN autenticação deve retornar false
5. WHEN as credenciais são carregadas de env vars THEN devem ser usadas para validação

**Independent Test**: Table-driven tests com credenciais válidas e inválidas.

---

### P1: Connection Manager (Max Clients) ⭐ MVP

**User Story**: Como operador do broker, quero limitar a 5 clients simultâneos para garantir performance previsível no edge.

**Why P1**: Constraint fundamental do projeto.

**Acceptance Criteria**:

1. WHEN menos de 5 clients estão conectados THEN `CanAccept()` retorna true
2. WHEN 5 clients estão conectados THEN `CanAccept()` retorna false
3. WHEN `Add(client)` é chamado com limite atingido THEN retorna `ErrMaxClientsReached`
4. WHEN `Remove(clientID)` é chamado THEN o slot é liberado
5. WHEN 2 goroutines chamam `Add` simultaneamente THEN apenas uma deve ter sucesso se restar 1 slot
6. WHEN um client com mesmo ID já existe THEN o antigo deve ser desconectado (replace)

**Independent Test**: Testes unitários + teste de concorrência.

---

### P1: TCP Server e Accept Loop ⭐ MVP

**User Story**: Como broker, preciso escutar na porta TCP configurada e aceitar conexões de clients MQTT.

**Why P1**: Sem TCP listener, nada funciona.

**Acceptance Criteria**:

1. WHEN o server inicia THEN deve escutar na porta configurada
2. WHEN um client TCP conecta THEN deve aceitar a conexão e iniciar handler
3. WHEN o limite de clients é atingido THEN deve fechar a conexão TCP imediatamente
4. WHEN o context é cancelado THEN o listener deve parar de aceitar conexões
5. WHEN ocorre erro no accept THEN deve logar e continuar aceitando

**Independent Test**: Teste de integração com `net.Dial` local.

---

### P1: Packet Handler (Orquestração) ⭐ MVP

**User Story**: Como broker, preciso processar o fluxo completo de pacotes: CONNECT → autenticação → loop de PUBLISH/PINGREQ/DISCONNECT.

**Why P1**: Integra protocol + auth + connection manager.

**Acceptance Criteria**:

1. WHEN client envia CONNECT válido com auth correta THEN deve receber CONNACK accepted
2. WHEN client envia CONNECT com auth incorreta THEN deve receber CONNACK bad auth e ser desconectado
3. WHEN client envia PUBLISH em tópico permitido THEN mensagem deve ser enviada ao canal de ingestão
4. WHEN client envia PUBLISH em tópico não permitido THEN deve ser ignorado silenciosamente
5. WHEN client envia PINGREQ THEN deve receber PINGRESP
6. WHEN client envia DISCONNECT THEN deve ser removido do connection manager
7. WHEN client envia SUBSCRIBE THEN deve receber SUBACK com QoS solicitado
8. WHEN keep-alive expira THEN client deve ser desconectado
9. WHEN a conexão TCP cai inesperadamente THEN client deve ser removido do connection manager

**Independent Test**: Teste de integração com `net.Pipe()` simulando client MQTT.

---

## Edge Cases

- WHEN o primeiro pacote não é CONNECT THEN system SHALL fechar a conexão
- WHEN client envia segundo CONNECT THEN system SHALL fechar a conexão (protocol violation)
- WHEN keep-alive é 0 THEN system SHALL não aplicar timeout
- WHEN client desconecta sem DISCONNECT THEN system SHALL liberar o slot após timeout
- WHEN múltiplos clients tentam conectar simultaneamente THEN system SHALL serializar via mutex

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
|---------------|-------|-------|--------|
| SESS-01 | P1: Entidade Client | Tasks | Pending |
| SESS-02 | P1: Autenticação | Tasks | Pending |
| SESS-03 | P1: Connection Manager | Tasks | Pending |
| SESS-04 | P1: TCP Server | Tasks | Pending |
| SESS-05 | P1: Packet Handler | Tasks | Pending |

**Coverage:** 5 total, 5 mapped to tasks, 0 unmapped ✅

---

## Success Criteria

- [ ] Client MQTT (mosquitto_pub) consegue conectar, publicar e desconectar
- [ ] 6º client é rejeitado quando 5 estão conectados
- [ ] Credenciais incorretas resultam em CONNACK com return code 0x04
- [ ] Keep-alive funciona (client desconectado após timeout)
- [ ] Todos os testes passam com `go test -race ./internal/session/...`
