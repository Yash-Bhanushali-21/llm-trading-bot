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

### 8. **Fail-Fast Validation**
Configuration validated at startup:

```go
func (c *Config) Validate() error {
    if c.Mode != "DRY_RUN" && c.Mode != "LIVE" {
        return fmt.Errorf("invalid mode '%s'", c.Mode)
    }
    if len(c.UniverseStatic) == 0 {
        return errors.New("universe_static cannot be empty")
    }
    // ... critical checks only
}
```

**Benefits:**
- Catches misconfigurations before trading starts
- Clear error messages on startup
- Prevents runtime surprises

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
│  - Config (YAML + Validation)           │
└─────────────────────────────────────────┘
```

---

## Summary

The codebase follows **industry-standard patterns**:
- ✅ SOLID principles (especially S, I, D)
- ✅ Clean architecture (layered, decoupled)
- ✅ GoLang idioms (interfaces, context, errors)
- ✅ Fail-fast validation

**The design is clean, minimal, and production-ready.**
