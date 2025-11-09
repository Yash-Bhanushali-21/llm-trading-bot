# Design Patterns & Architecture

## Overview
This trading bot follows clean architecture principles with clear separation of concerns.

## Core Design Patterns

### 1. **Separation of Concerns**
Each package has a single, well-defined responsibility:

```
internal/logger/    → Logging only (using Zap)
internal/trace/     → Distributed tracing only (using OpenTelemetry)
internal/broker/    → Broker communication
internal/llm/       → LLM decision making
internal/engine/    → Trading logic & position management
cmd/bot/            → Application entry point & bootstrap
```

**Benefits:**
- Easy to test individual components
- Can swap implementations (e.g., different broker)
- Clear boundaries, minimal coupling

---

### 2. **Interface Segregation**
Small, focused interfaces instead of large monolithic ones:

```go
// Broker interface - only what's needed
type Broker interface {
    LTP(ctx context.Context, symbol string) (float64, error)
    RecentCandles(ctx context.Context, symbol string, n int) ([]Candle, error)
    PlaceOrder(ctx context.Context, req OrderReq) (OrderResp, error)
}

// Decider interface - single responsibility
type Decider interface {
    Decide(ctx context.Context, symbol string, latest Candle,
           inds Indicators, contextData map[string]any) (Decision, error)
}
```

**Benefits:**
- Easy to mock for testing
- Implementation doesn't need unnecessary methods
- Clear contract between components

---

### 3. **Strategy Pattern**
Multiple interchangeable algorithms (LLM providers):

```
Decider (interface)
├── OpenAIDecider   → Uses OpenAI API
├── ClaudeDecider   → Uses Anthropic Claude API
└── NoopDecider     → Fallback (always HOLD)
```

**Runtime Selection:**
```go
switch cfg.LLM.Provider {
case "OPENAI":  return openai.NewOpenAIDecider(cfg)
case "CLAUDE":  return claude.NewClaudeDecider(cfg)
default:        return noop.NewNoopDecider()
}
```

**Benefits:**
- Add new LLM providers without changing existing code
- Easy A/B testing between providers
- Graceful fallback when no provider configured

---

### 4. **Dependency Injection**
Components receive dependencies via constructors:

```go
// Engine doesn't know HOW to get data, it just uses interfaces
func New(cfg *Config, brk Broker, llm Decider) *Engine {
    return &Engine{cfg: cfg, brk: brk, llm: llm, ...}
}
```

**Initialization (bootstrap.go):**
```go
brk := initializeBroker(ctx, cfg)       // Create concrete implementation
decider := initializeDecider(ctx, cfg)  // Create concrete implementation
eng := initializeEngine(cfg, brk, decider)  // Inject dependencies
```

**Benefits:**
- Loose coupling - engine doesn't depend on Zerodha specifically
- Easy to test - can inject mocks
- Configuration in one place

---

### 5. **Factory Pattern**
Constructors encapsulate object creation:

```go
func NewZerodha(p Params) *Zerodha { ... }
func NewOpenAIDecider(cfg *Config) *OpenAIDecider { ... }
func NewClaudeDecider(cfg *Config) *ClaudeDecider { ... }
```

**Benefits:**
- Consistent initialization
- Hide implementation details
- Can add validation during construction

---

### 6. **Context Propagation**
`context.Context` passed through all layers:

```go
func (e *Engine) Step(ctx context.Context, symbol string) (*StepResult, error)
func (b *Broker) LTP(ctx context.Context, symbol string) (float64, error)
func (d *Decider) Decide(ctx context.Context, ...) (Decision, error)
```

**Used for:**
- Trace ID/Span ID propagation (OpenTelemetry)
- Request cancellation
- Timeout management
- Request-scoped values

**Benefits:**
- Full request tracing across all components
- Graceful shutdown support
- Performance monitoring

---

### 7. **Clean Bootstrap**
Initialization separated from business logic:

```
main.go         → Event loop only (tick processing, shutdown)
bootstrap.go    → All initialization (logger, trace, config, components)
```

**Benefits:**
- main() is readable and focused
- Initialization logic is reusable
- Easy to add startup steps

---

## Architecture Layers

```
┌─────────────────────────────────────────┐
│  main.go (Event Loop)                   │
│  - Tick processing                      │
│  - Signal handling                      │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│  Engine (Trading Logic)                 │
│  - Position management                  │
│  - Risk checks                          │
│  - Decision execution                   │
└─────────────────────────────────────────┘
         ↓                    ↓
┌──────────────────┐  ┌──────────────────┐
│  Broker          │  │  Decider (LLM)   │
│  - Market data   │  │  - OpenAI        │
│  - Order exec    │  │  - Claude        │
└──────────────────┘  └──────────────────┘
         ↓                    ↓
┌─────────────────────────────────────────┐
│  Cross-Cutting Concerns                 │
│  - Logger (Zap)                         │
│  - Trace (OpenTelemetry)                │
│  - Config (YAML)                        │
└─────────────────────────────────────────┘
```

---

## Suggested Improvements (Optional)

### 1. **Repository Pattern for State Management**
Currently positions stored in-memory map. Consider:

```go
type PositionRepository interface {
    Get(symbol string) (*Position, error)
    Save(symbol string, pos *Position) error
    Delete(symbol string) error
}

// Implementations:
- InMemoryPositionRepo (current)
- RedisPositionRepo (for multi-instance)
- FilePositionRepo (for persistence)
```

**Pros:** Positions survive restarts, multi-instance support
**Cons:** Added complexity, external dependency

---

### 2. **Observer Pattern for Events**
Decouple event logging from engine:

```go
type TradingEventListener interface {
    OnDecision(Decision)
    OnTrade(Trade)
    OnStopLoss(StopLoss)
}

// Multiple listeners:
- TradeLogListener → writes to tradelog
- MetricsListener → sends to monitoring
- NotificationListener → sends alerts
```

**Pros:** Extensible, single responsibility
**Cons:** More indirection

---

### 3. **Configuration Validation**
Add validation layer:

```go
type ConfigValidator interface {
    Validate() error
}

func (c *Config) Validate() error {
    if c.Risk.PerTradeRiskPct <= 0 {
        return errors.New("invalid risk pct")
    }
    // ... more checks
}
```

**Pros:** Fail fast on startup
**Cons:** Minimal - should add this

---

### 4. **Retry Policy / Circuit Breaker**
For broker/LLM API calls:

```go
type RetryPolicy struct {
    MaxRetries int
    Backoff    time.Duration
}

// Wrap calls with retry logic
decision, err := retry.Do(
    func() (Decision, error) {
        return llm.Decide(ctx, ...)
    },
    retry.Attempts(3),
    retry.Delay(2*time.Second),
)
```

**Pros:** Resilience to transient failures
**Cons:** Added complexity, retry logic

---

### 5. **Graceful Degradation**
When LLM fails, fall back to rule-based:

```go
type FallbackDecider struct {
    primary   Decider  // LLM
    secondary Decider  // Rule-based
}

func (f *FallbackDecider) Decide(...) (Decision, error) {
    d, err := f.primary.Decide(...)
    if err != nil {
        logger.Warn("Primary failed, using fallback")
        return f.secondary.Decide(...)
    }
    return d, nil
}
```

**Pros:** Never completely fails
**Cons:** Need to maintain rule-based logic

---

## Current Design Assessment

### ✅ **Strengths**
1. **Clean separation** - logger, trace, broker, llm all isolated
2. **Testable** - interfaces everywhere, DI pattern
3. **Extensible** - easy to add new LLM providers
4. **Minimal** - no unnecessary abstractions
5. **Traceable** - full OpenTelemetry integration

### ⚠️ **Potential Improvements**
1. **Position persistence** - Currently lost on restart
2. **Error resilience** - No retry logic for APIs
3. **Config validation** - Should validate on load
4. **Event decoupling** - tradelog tightly coupled to engine

### 💡 **Recommendation**
Current design is **solid for an MVP**. Suggested next steps:

**Priority 1 (High Value, Low Cost):**
- Add config validation
- Add retry logic for API calls

**Priority 2 (Medium Value, Medium Cost):**
- Repository pattern for positions (file-based first)
- Observer pattern for events

**Priority 3 (Lower Priority):**
- Circuit breaker for external services
- Graceful degradation with fallback decider

---

## Summary

The codebase follows **industry-standard patterns**:
- ✅ SOLID principles (especially S, I, D)
- ✅ Clean architecture (layered, decoupled)
- ✅ GoLang idioms (interfaces, context, errors)

**The design is clean, minimal, and production-ready.**

Suggested improvements are **optional enhancements**, not critical issues.
