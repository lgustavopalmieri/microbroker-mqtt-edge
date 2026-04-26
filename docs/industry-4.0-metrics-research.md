# Métricas Fundamentais da Indústria 4.0 — Pesquisa e Modelagem de Dados

## 1. Visão Geral

Este documento detalha as métricas fundamentais para gerenciamento de manufatura em tempo real,
os dados necessários para calculá-las, o fluxo de dados desde a máquina até o cálculo,
e exemplos de implementação em Go.

Fontes principais:
- [OEE.com — Calculating OEE](https://www.oee.com/calculating-oee/)
- [OEE.com — Six Big Losses](https://www.oee.com/oee-six-big-losses/)
- [OEE Academy — OEE with Mixed Output](https://oee.academy/oee-academy/oee-calculation-faq/oee-with-mixed-output/)
- [Symestic — OEE Definition](https://www.symestic.com/en-us/blog/oee-definition-factors-calculation-with-examples)
- [Symestic — Manufacturing Order Management](https://www.symestic.com/en-us/what-is/manufacturing-order-management)
- [Microsoft — Production Order Lifecycle](https://docs.microsoft.com/en-us/dynamics365/supply-chain/production-control/create-production-orders)

---

## 2. As Seis Grandes Perdas (Six Big Losses)

Antes de entrar nas fórmulas, é essencial entender as categorias de perda que o OEE mede.
O framework das Six Big Losses, originado do TPM (Total Productive Maintenance) de Seiichi Nakajima,
classifica todas as perdas de produtividade em seis tipos, agrupados nos três fatores do OEE:

### Perdas de Disponibilidade (Availability Losses)
1. **Falha de Equipamento (Equipment Failure)** — Paradas não planejadas: quebras, falha de ferramental, manutenção não programada, falta de operador/material
2. **Setup e Ajustes (Setup & Adjustments)** — Paradas planejadas: changeover, limpeza, aquecimento, manutenção preventiva, inspeções de qualidade

### Perdas de Performance (Performance Losses)
3. **Micro-paradas (Idling & Minor Stops)** — Paradas curtas (< 5 min): atolamentos, desalinhamento de sensores, obstrução de fluxo
4. **Velocidade Reduzida (Reduced Speed)** — Ciclos mais lentos que o ideal: desgaste, lubrificação inadequada, material de baixa qualidade

### Perdas de Qualidade (Quality Losses)
5. **Defeitos de Processo (Process Defects)** — Peças defeituosas durante produção estável: configuração incorreta, erro de operador
6. **Perda de Rendimento (Reduced Yield)** — Peças defeituosas durante startup até estabilização: após changeover, aquecimento

---

## 3. Fórmulas das Métricas

### 3.1 OEE (Overall Equipment Effectiveness)

O OEE é o produto de três fatores independentes, cada um expresso como percentual:

```
OEE = Disponibilidade × Performance × Qualidade
```

Um OEE de 100% significa: zero paradas, velocidade máxima, zero defeitos.
O benchmark world-class é 85% (Disponibilidade 90%, Performance 95%, Qualidade 99.9%).

#### 3.1.1 Disponibilidade (Availability)

Mede quanto do tempo planejado a máquina realmente operou.

```
Disponibilidade = Tempo de Operação / Tempo de Produção Planejado

Onde:
  Tempo de Produção Planejado = Duração do Turno − Paradas Programadas (intervalos, refeições)
  Tempo de Operação = Tempo de Produção Planejado − Tempo de Parada (planejadas + não planejadas)
```

#### 3.1.2 Performance

Mede se a máquina operou na velocidade máxima teórica durante o tempo que esteve rodando.

```
Performance = (Tempo de Ciclo Ideal × Total de Peças) / Tempo de Operação
```

Forma alternativa usando taxa:
```
Performance = (Total de Peças / Tempo de Operação) / Taxa Ideal de Produção
```

Performance nunca deve exceder 100%. Se exceder, o Tempo de Ciclo Ideal está configurado incorretamente.

#### 3.1.3 Qualidade (Quality)

Mede a proporção de peças boas na primeira passagem (First Pass Yield).

```
Qualidade = Peças Boas / Total de Peças

Onde:
  Peças Boas = Total de Peças − Peças Rejeitadas (scrap + retrabalho)
```

#### 3.1.4 Fórmula Simplificada do OEE

```
OEE = (Peças Boas × Tempo de Ciclo Ideal) / Tempo de Produção Planejado
```

### 3.2 Performance Ponderada (Weighted Performance)

Quando uma máquina produz múltiplos produtos com tempos de ciclo diferentes no mesmo turno,
a performance simples não funciona. Usa-se a performance ponderada:

```
Performance Ponderada = Σ(Peças_i × TempoCicloIdeal_i) / Tempo de Operação
```

Para cada "run" (período produzindo um produto específico):
```
Target Output_i = Tempo da Run_i / TempoCicloIdeal_i
Performance_i = Peças Reais_i / Target Output_i
```

A performance do turno é a média ponderada pelo tempo de cada run.

### 3.3 Planejamento Diário / Por Turno

```
Produção Esperada por Turno = Tempo de Produção Planejado / Tempo de Ciclo Ideal
```

Com múltiplos produtos:
```
Produção Esperada = Σ (Tempo Alocado_i / TempoCicloIdeal_i)
```

### 3.4 Ordens de Produção

Uma ordem de produção contém:
- Identificador único
- Produto a ser fabricado
- Quantidade planejada
- Data/hora de início planejado e real
- Data/hora de término planejado e real
- Quantidade produzida (boas + rejeitadas)
- Status do ciclo de vida: Created → Released → Started → InProgress → Completed → Closed

Cada peça produzida deve ser vinculada à ordem de produção ativa no momento da produção.

---

## 4. Dados Fundamentais (Payloads)

Para calcular todas as métricas acima, os seguintes dados devem ser coletados da máquina:

### 4.1 Dados em Tempo Real (via MQTT)


#### 4.1.1 Contagem de Peças (Piece Count)

Cada sinal de peça produzida pela máquina. É o dado mais fundamental.

```json
{
  "type": "piece_count",
  "machine_id": "CNC-01",
  "order_id": "OP-2026-001",
  "product_id": "PART-A",
  "quantity": 1,
  "is_good": true,
  "timestamp": "2026-04-25T08:30:15.123Z"
}
```

Tópico MQTT sugerido: `machine/{machine_id}/production`

#### 4.1.2 Estado da Máquina (Machine State)

Mudanças de estado da máquina. Essencial para calcular disponibilidade.

```json
{
  "type": "state_change",
  "machine_id": "CNC-01",
  "state": "running",
  "previous_state": "stopped",
  "reason": "",
  "timestamp": "2026-04-25T08:00:00.000Z"
}
```

Estados possíveis:
- `running` — Máquina produzindo
- `stopped` — Parada não planejada (breakdown, falta de material)
- `setup` — Changeover / setup (parada planejada)
- `idle` — Ociosa (sem demanda, aguardando)
- `maintenance` — Manutenção preventiva (parada planejada)
- `off` — Desligada (fora do turno)

Tópico MQTT sugerido: `machine/{machine_id}/state`

#### 4.1.3 Tempo de Ciclo (Cycle Time)

Tempo real de cada ciclo de produção. Essencial para calcular performance.

```json
{
  "type": "cycle_time",
  "machine_id": "CNC-01",
  "duration_ms": 1050,
  "ideal_cycle_time_ms": 1000,
  "timestamp": "2026-04-25T08:30:15.123Z"
}
```

Tópico MQTT sugerido: `machine/{machine_id}/cycle`

#### 4.1.4 Rejeição / Defeito (Reject)

Quando uma peça é identificada como defeituosa.

```json
{
  "type": "reject",
  "machine_id": "CNC-01",
  "order_id": "OP-2026-001",
  "product_id": "PART-A",
  "reason": "dimensional_out_of_spec",
  "quantity": 1,
  "timestamp": "2026-04-25T08:31:00.000Z"
}
```

Tópico MQTT sugerido: `machine/{machine_id}/quality`

### 4.2 Dados de Configuração (cadastrados no sistema, não via MQTT)

#### 4.2.1 Turno (Shift)

```
Turno:
  - ID
  - Nome (Manhã, Tarde, Noite)
  - Hora de início
  - Hora de término
  - Pausas programadas [{início, fim, tipo}]
  - Dias da semana ativos
```

#### 4.2.2 Produto (Product)

```
Produto:
  - ID
  - Nome / Código
  - Tempo de Ciclo Ideal (ms)
  - Taxa Ideal de Produção (peças/min)
```

#### 4.2.3 Ordem de Produção (Production Order)

```
Ordem de Produção:
  - ID
  - Produto ID
  - Quantidade Planejada
  - Máquina ID
  - Turno ID (opcional)
  - Data/Hora Início Planejado
  - Data/Hora Fim Planejado
  - Data/Hora Início Real
  - Data/Hora Fim Real
  - Quantidade Produzida (boas)
  - Quantidade Rejeitada
  - Status (created, released, started, in_progress, completed, closed)
```

---

## 5. Fluxo de Dados

```
┌─────────────┐     MQTT      ┌──────────────┐     Pipeline     ┌──────────────┐
│   Máquina   │──────────────▶│  Micro Broker │────────────────▶│   Storage    │
│  (PLC/IoT)  │  piece_count  │   (TCP/MQTT)  │   Ingestão +    │  (SQLite/DB) │
│             │  state_change │               │   Persistência  │              │
│             │  cycle_time   │               │                 │              │
│             │  reject       │               │                 │              │
└─────────────┘               └──────────────┘                 └──────┬───────┘
                                                                       │
                                                                       ▼
                                                               ┌──────────────┐
                                                               │  Processing  │
                                                               │  (Workers)   │
                                                               │              │
                                                               │ • OEE Calc   │
                                                               │ • Forwarding │
                                                               │ • Alertas    │
                                                               └──────────────┘
```

### Fluxo detalhado:

1. **Máquina publica** dados via MQTT em tópicos dedicados por tipo de dado
2. **Broker ingere** e persiste cada mensagem na fila FIFO por tópico (garantia de ordem)
3. **Pipeline** roteia para a queue correta e persiste no banco
4. **Workers** processam: calculam métricas, encaminham para sistemas externos
5. **Cálculos** são feitos agregando dados por turno/ordem/período

---

## 6. Modelagem em Go

### 6.1 Tipos de Domínio

```go
package domain

import "time"

// MachineState representa os estados possíveis de uma máquina.
type MachineState string

const (
	StateRunning     MachineState = "running"
	StateStopped     MachineState = "stopped"
	StateSetup       MachineState = "setup"
	StateIdle        MachineState = "idle"
	StateMaintenance MachineState = "maintenance"
	StateOff         MachineState = "off"
)

// IsDowntime retorna true se o estado representa uma parada
// que conta contra a disponibilidade.
func (s MachineState) IsDowntime() bool {
	return s == StateStopped || s == StateSetup || s == StateMaintenance
}

// IsPlannedStop retorna true se a parada é planejada.
func (s MachineState) IsPlannedStop() bool {
	return s == StateSetup || s == StateMaintenance
}

// OrderStatus representa o ciclo de vida de uma ordem de produção.
type OrderStatus string

const (
	OrderCreated    OrderStatus = "created"
	OrderReleased   OrderStatus = "released"
	OrderStarted    OrderStatus = "started"
	OrderInProgress OrderStatus = "in_progress"
	OrderCompleted  OrderStatus = "completed"
	OrderClosed     OrderStatus = "closed"
)

// Product define um produto com seu tempo de ciclo ideal.
type Product struct {
	ID             string
	Name           string
	IdealCycleTime time.Duration // tempo de ciclo ideal para 1 peça
}

// IdealRate retorna a taxa ideal de produção em peças por minuto.
func (p Product) IdealRate() float64 {
	if p.IdealCycleTime == 0 {
		return 0
	}
	return float64(time.Minute) / float64(p.IdealCycleTime)
}

// Shift define um turno de trabalho.
type Shift struct {
	ID        string
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Breaks    []Break
}

// PlannedProductionTime retorna o tempo de produção planejado
// (duração do turno menos pausas programadas).
func (s Shift) PlannedProductionTime() time.Duration {
	total := s.EndTime.Sub(s.StartTime)
	for _, b := range s.Breaks {
		total -= b.Duration()
	}
	return total
}

// Break representa uma pausa programada dentro de um turno.
type Break struct {
	StartTime time.Time
	EndTime   time.Time
	Type      string // "rest", "meal", "cleaning"
}

// Duration retorna a duração da pausa.
func (b Break) Duration() time.Duration {
	return b.EndTime.Sub(b.StartTime)
}

// ProductionOrder representa uma ordem de produção.
type ProductionOrder struct {
	ID              string
	ProductID       string
	MachineID       string
	PlannedQuantity int
	PlannedStart    time.Time
	PlannedEnd      time.Time
	ActualStart     *time.Time
	ActualEnd       *time.Time
	GoodCount       int
	RejectCount     int
	Status          OrderStatus
}

// TotalCount retorna o total de peças produzidas (boas + rejeitadas).
func (o ProductionOrder) TotalCount() int {
	return o.GoodCount + o.RejectCount
}

// Completion retorna o percentual de conclusão da ordem.
func (o ProductionOrder) Completion() float64 {
	if o.PlannedQuantity == 0 {
		return 0
	}
	return float64(o.GoodCount) / float64(o.PlannedQuantity) * 100
}

// Duration retorna a duração real da ordem (se iniciada).
func (o ProductionOrder) Duration() time.Duration {
	if o.ActualStart == nil {
		return 0
	}
	end := time.Now()
	if o.ActualEnd != nil {
		end = *o.ActualEnd
	}
	return end.Sub(*o.ActualStart)
}
```

### 6.2 Payloads MQTT (structs para deserialização)

```go
package payload

import "time"

// PieceCount é o payload recebido quando a máquina produz uma peça.
type PieceCount struct {
	Type      string    `json:"type"`       // "piece_count"
	MachineID string    `json:"machine_id"`
	OrderID   string    `json:"order_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	IsGood    bool      `json:"is_good"`
	Timestamp time.Time `json:"timestamp"`
}

// StateChange é o payload recebido quando o estado da máquina muda.
type StateChange struct {
	Type          string    `json:"type"`           // "state_change"
	MachineID     string    `json:"machine_id"`
	State         string    `json:"state"`
	PreviousState string    `json:"previous_state"`
	Reason        string    `json:"reason"`
	Timestamp     time.Time `json:"timestamp"`
}

// CycleTime é o payload recebido a cada ciclo de produção.
type CycleTime struct {
	Type             string    `json:"type"`               // "cycle_time"
	MachineID        string    `json:"machine_id"`
	DurationMs       int64     `json:"duration_ms"`
	IdealCycleTimeMs int64     `json:"ideal_cycle_time_ms"`
	Timestamp        time.Time `json:"timestamp"`
}

// Reject é o payload recebido quando uma peça é rejeitada.
type Reject struct {
	Type      string    `json:"type"`       // "reject"
	MachineID string    `json:"machine_id"`
	OrderID   string    `json:"order_id"`
	ProductID string    `json:"product_id"`
	Reason    string    `json:"reason"`
	Quantity  int       `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
}
```

### 6.3 Cálculo do OEE

```go
package oee

import "time"

// Input contém todos os dados necessários para calcular o OEE de um período.
type Input struct {
	// Tempo de produção planejado (turno - pausas)
	PlannedProductionTime time.Duration

	// Tempo total de parada (planejadas + não planejadas)
	// dentro do tempo de produção planejado
	StopTime time.Duration

	// Tempo de ciclo ideal para o produto sendo fabricado
	IdealCycleTime time.Duration

	// Total de peças produzidas (boas + rejeitadas)
	TotalCount int

	// Total de peças boas (sem defeito na primeira passagem)
	GoodCount int
}

// Result contém o OEE e seus três fatores componentes.
type Result struct {
	Availability float64 // 0.0 a 1.0
	Performance  float64 // 0.0 a 1.0
	Quality      float64 // 0.0 a 1.0
	OEE          float64 // 0.0 a 1.0
}

// Calculate calcula o OEE a partir dos dados de entrada.
// Retorna todos os fatores como valores entre 0.0 e 1.0.
func Calculate(in Input) Result {
	if in.PlannedProductionTime == 0 {
		return Result{}
	}

	// Disponibilidade = Tempo de Operação / Tempo de Produção Planejado
	runTime := in.PlannedProductionTime - in.StopTime
	if runTime < 0 {
		runTime = 0
	}
	availability := float64(runTime) / float64(in.PlannedProductionTime)

	// Performance = (Tempo de Ciclo Ideal × Total de Peças) / Tempo de Operação
	var performance float64
	if runTime > 0 && in.IdealCycleTime > 0 {
		netRunTime := in.IdealCycleTime * time.Duration(in.TotalCount)
		performance = float64(netRunTime) / float64(runTime)
	}
	// Performance não pode exceder 1.0 (indicaria ciclo ideal incorreto)
	if performance > 1.0 {
		performance = 1.0
	}

	// Qualidade = Peças Boas / Total de Peças
	var quality float64
	if in.TotalCount > 0 {
		quality = float64(in.GoodCount) / float64(in.TotalCount)
	}

	return Result{
		Availability: availability,
		Performance:  performance,
		Quality:      quality,
		OEE:          availability * performance * quality,
	}
}
```

### 6.4 Performance Ponderada (múltiplos produtos)

```go
package oee

import "time"

// ProductRun representa um período de produção de um produto específico.
type ProductRun struct {
	ProductID      string
	IdealCycleTime time.Duration
	RunDuration    time.Duration // tempo que a máquina operou neste produto
	PiecesProduced int
}

// CalculateWeightedPerformance calcula a performance ponderada
// quando múltiplos produtos com tempos de ciclo diferentes
// são produzidos no mesmo período.
//
// Fórmula: Σ(Peças_i × TempoCicloIdeal_i) / TempoOperaçãoTotal
func CalculateWeightedPerformance(runs []ProductRun) float64 {
	var totalNetRunTime time.Duration
	var totalRunTime time.Duration

	for _, run := range runs {
		// Tempo teórico para produzir as peças na velocidade ideal
		netRunTime := run.IdealCycleTime * time.Duration(run.PiecesProduced)
		totalNetRunTime += netRunTime
		totalRunTime += run.RunDuration
	}

	if totalRunTime == 0 {
		return 0
	}

	perf := float64(totalNetRunTime) / float64(totalRunTime)
	if perf > 1.0 {
		perf = 1.0
	}
	return perf
}
```

### 6.5 Planejamento de Produção por Turno

```go
package planning

import "time"

// ShiftTarget calcula a produção esperada para um turno.
type ShiftTarget struct {
	ShiftDuration         time.Duration // duração total do turno
	PlannedBreaks         time.Duration // total de pausas programadas
	IdealCycleTime        time.Duration // tempo de ciclo ideal do produto
	TargetOEE             float64       // OEE alvo (ex: 0.85)
}

// ExpectedOutput retorna a quantidade de peças boas esperadas no turno,
// considerando o OEE alvo.
//
// Fórmula: (Tempo Planejado / Tempo Ciclo Ideal) × OEE Alvo
func (s ShiftTarget) ExpectedOutput() int {
	plannedTime := s.ShiftDuration - s.PlannedBreaks
	if plannedTime <= 0 || s.IdealCycleTime <= 0 {
		return 0
	}

	theoreticalMax := float64(plannedTime) / float64(s.IdealCycleTime)
	return int(theoreticalMax * s.TargetOEE)
}

// TheoreticalMax retorna o máximo teórico de peças (100% OEE).
func (s ShiftTarget) TheoreticalMax() int {
	plannedTime := s.ShiftDuration - s.PlannedBreaks
	if plannedTime <= 0 || s.IdealCycleTime <= 0 {
		return 0
	}
	return int(float64(plannedTime) / float64(s.IdealCycleTime))
}

// MultiProductTarget calcula a produção esperada quando múltiplos
// produtos são planejados para o mesmo turno.
type MultiProductTarget struct {
	ProductID      string
	IdealCycleTime time.Duration
	AllocatedTime  time.Duration // tempo alocado para este produto
}

// ExpectedOutputMulti retorna a produção esperada por produto.
func ExpectedOutputMulti(targets []MultiProductTarget, targetOEE float64) map[string]int {
	result := make(map[string]int, len(targets))
	for _, t := range targets {
		if t.IdealCycleTime <= 0 || t.AllocatedTime <= 0 {
			continue
		}
		theoretical := float64(t.AllocatedTime) / float64(t.IdealCycleTime)
		result[t.ProductID] = int(theoretical * targetOEE)
	}
	return result
}
```

### 6.6 Rastreamento de Ordem de Produção

```go
package order

import (
	"fmt"
	"time"
)

// Status representa o ciclo de vida da ordem.
type Status string

const (
	Created    Status = "created"
	Released   Status = "released"
	Started    Status = "started"
	InProgress Status = "in_progress"
	Completed  Status = "completed"
	Closed     Status = "closed"
)

// Order representa uma ordem de produção com rastreamento completo.
type Order struct {
	ID              string
	ProductID       string
	MachineID       string
	PlannedQuantity int
	PlannedStart    time.Time
	PlannedEnd      time.Time
	ActualStart     *time.Time
	ActualEnd       *time.Time
	GoodCount       int
	RejectCount     int
	Status          Status
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Start marca o início real da produção.
func (o *Order) Start(t time.Time) error {
	if o.Status != Released {
		return fmt.Errorf("cannot start order in status %s", o.Status)
	}
	o.ActualStart = &t
	o.Status = Started
	o.UpdatedAt = t
	return nil
}

// RecordPiece registra uma peça produzida vinculada a esta ordem.
func (o *Order) RecordPiece(good bool, t time.Time) error {
	if o.Status != Started && o.Status != InProgress {
		return fmt.Errorf("cannot record piece for order in status %s", o.Status)
	}
	if good {
		o.GoodCount++
	} else {
		o.RejectCount++
	}
	o.Status = InProgress
	o.UpdatedAt = t
	return nil
}

// Complete marca o término da ordem.
func (o *Order) Complete(t time.Time) error {
	if o.Status != InProgress && o.Status != Started {
		return fmt.Errorf("cannot complete order in status %s", o.Status)
	}
	o.ActualEnd = &t
	o.Status = Completed
	o.UpdatedAt = t
	return nil
}

// TotalCount retorna o total de peças produzidas.
func (o *Order) TotalCount() int {
	return o.GoodCount + o.RejectCount
}

// CompletionRate retorna o percentual de conclusão (peças boas / planejado).
func (o *Order) CompletionRate() float64 {
	if o.PlannedQuantity == 0 {
		return 0
	}
	return float64(o.GoodCount) / float64(o.PlannedQuantity) * 100
}

// Duration retorna a duração real da ordem.
func (o *Order) Duration() time.Duration {
	if o.ActualStart == nil {
		return 0
	}
	end := time.Now()
	if o.ActualEnd != nil {
		end = *o.ActualEnd
	}
	return end.Sub(*o.ActualStart)
}

// QualityRate retorna a taxa de qualidade da ordem.
func (o *Order) QualityRate() float64 {
	total := o.TotalCount()
	if total == 0 {
		return 0
	}
	return float64(o.GoodCount) / float64(total)
}
```

---

## 7. Exemplo Completo: Cálculo de OEE de um Turno

Dados do turno:
- Duração: 8 horas (480 min)
- Pausas: 2×15 min + 1×30 min = 60 min
- Paradas: 47 min (breakdown)
- Tempo de ciclo ideal: 1.0 segundo
- Total produzido: 19.271 peças
- Rejeitadas: 423 peças

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	// Dados do turno
	shiftDuration := 480 * time.Minute
	breaks := 60 * time.Minute
	stopTime := 47 * time.Minute
	idealCycleTime := 1 * time.Second
	totalCount := 19271
	rejectCount := 423
	goodCount := totalCount - rejectCount

	// Tempo de Produção Planejado
	plannedProductionTime := shiftDuration - breaks // 420 min

	// Tempo de Operação
	runTime := plannedProductionTime - stopTime // 373 min

	// Disponibilidade
	availability := float64(runTime) / float64(plannedProductionTime)
	// 373/420 = 88.81%

	// Performance
	netRunTime := idealCycleTime * time.Duration(totalCount)
	performance := float64(netRunTime) / float64(runTime)
	// (1s × 19271) / (373 × 60s) = 86.11%

	// Qualidade
	quality := float64(goodCount) / float64(totalCount)
	// 18848/19271 = 97.80%

	// OEE
	oee := availability * performance * quality
	// 88.81% × 86.11% × 97.80% = 74.79%

	fmt.Printf("Tempo Produção Planejado: %v\n", plannedProductionTime)
	fmt.Printf("Tempo de Operação:        %v\n", runTime)
	fmt.Printf("Disponibilidade:          %.2f%%\n", availability*100)
	fmt.Printf("Performance:              %.2f%%\n", performance*100)
	fmt.Printf("Qualidade:                %.2f%%\n", quality*100)
	fmt.Printf("OEE:                      %.2f%%\n", oee*100)
	fmt.Printf("Peças Boas:               %d\n", goodCount)
	fmt.Printf("Peças Rejeitadas:         %d\n", rejectCount)
}
```

Saída esperada:
```
Tempo Produção Planejado: 7h0m0s
Tempo de Operação:        6h13m0s
Disponibilidade:          88.81%
Performance:              86.11%
Qualidade:                97.80%
OEE:                      74.79%
Peças Boas:               18848
Peças Rejeitadas:         423
```

---

## 8. Mapeamento: Tópicos MQTT → Métricas

| Tópico MQTT | Payload | Métrica que alimenta |
|---|---|---|
| `machine/{id}/production` | PieceCount | Qualidade, Performance, Contagem por Ordem |
| `machine/{id}/state` | StateChange | Disponibilidade (cálculo de paradas) |
| `machine/{id}/cycle` | CycleTime | Performance (ciclo real vs ideal) |
| `machine/{id}/quality` | Reject | Qualidade, Contagem de rejeitos por Ordem |

---

## 9. Resumo dos Dados Mínimos Necessários por Métrica

| Métrica | Dados Necessários |
|---|---|
| **Disponibilidade** | Tempo do turno, pausas programadas, eventos de parada (state_change) |
| **Performance** | Tempo de ciclo ideal (cadastro), total de peças, tempo de operação |
| **Qualidade** | Total de peças, peças boas (ou rejeitadas) |
| **OEE** | Todos os acima combinados |
| **Performance Ponderada** | Tempo de ciclo ideal por produto, peças por produto, tempo por run |
| **Planejamento por Turno** | Tempo do turno, pausas, tempo de ciclo ideal, OEE alvo |
| **Ordens de Produção** | ID da ordem, produto, quantidade planejada, timestamps, contagens |
| **Vínculo Peça ↔ Ordem** | order_id no payload de piece_count |
| **Tempo Início/Fim Ordem** | Eventos de start/complete da ordem com timestamps |

---

## 10. Referências

- [OEE.com — Calculating OEE](https://www.oee.com/calculating-oee/) — Fórmulas detalhadas e exemplo completo
- [OEE.com — Six Big Losses](https://www.oee.com/oee-six-big-losses/) — Framework de categorização de perdas
- [OEE Academy — OEE with Mixed Output](https://oee.academy/oee-academy/oee-calculation-faq/oee-with-mixed-output/) — Performance ponderada para múltiplos produtos
- [Symestic — OEE Definition](https://www.symestic.com/en-us/blog/oee-definition-factors-calculation-with-examples) — Benchmarks world-class
- [Symestic — Manufacturing Order Management](https://www.symestic.com/en-us/what-is/manufacturing-order-management) — Ciclo de vida de ordens
- [Symestic — Downtime Periods](https://www.symestic.com/en-us/what-is/downtime-periods) — Categorização de paradas
- [Microsoft — Production Order Lifecycle](https://docs.microsoft.com/en-us/dynamics365/supply-chain/production-control/create-production-orders) — Modelo de dados de ordens
- [HiveMQ — MQTT Sparkplug](https://www.hivemq.com/blog/mqtt-sparkplug-essentials-part-1-introduction/) — Padrão de payload industrial
- [Symestic — MQTT in Manufacturing](https://www.symestic.com/en-us/what-is/mqtt) — MQTT no contexto de manufatura

Content was rephrased for compliance with licensing restrictions.
