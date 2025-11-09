# Design Patterns & Architecture

## Overview
This trading bot follows clean architecture principles with clear separation of concerns.

## Core Design Patterns

### 1. **Separation of Concerns**
Each package has a single, well-defined responsibility:

```
internal/logger/    → Structured logging (using Zap)
internal/trace/     → Distributed tracing (using OpenTelemetry)
internal/broker/    → Broker communication & market data
internal/llm/       → LLM decision making (OpenAI, Claude, Noop)
internal/engine/    → Trading engine (decoupled into 7 modules)
  ├── IEngine.go           → Interface definition
  ├── engine.go            → Orchestration layer
  ├── position_manager.go  → Position tracking & updates
  ├── risk_manager.go      → Risk validation & exposure
  ├── order_executor.go    → Order placement & logging
  ├── stop_manager.go      → Stop-loss calculations
  └── helpers.go           → Utilities (indicators, quantity)
internal/eod/       → End-of-day reporting (decoupled into 4 modules)
  ├── IEod.go       → IEodSummarizer interface
  ├── eod.go        → Core implementation
  ├── types.go      → Data structures (tradeLine, aggRow)
  └── utils.go      → Utilities (paths, IST time)
cmd/bot/            → Application entry point & bootstrap
```

**Benefits:**
- Easy to test individual components in isolation
- Can swap implementations (e.g., different broker or LLM)
- Clear boundaries, minimal coupling
- Each module has a single responsibility

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

// IEngine interface - trading engine contract
type IEngine interface {
    Step(ctx context.Context, symbol string) (*StepResult, error)
}

// IEodSummarizer interface - end-of-day reporting
type IEodSummarizer interface {
    SummarizeDay(t time.Time) (csvPath string, err error)
    SummarizeToday() (csvPath string, err error)
    ShouldRunNow() (shouldRun bool, csvPath string)
}
```

**Benefits:**
- Easy to mock for testing
- Implementation doesn't need unnecessary methods
- Clear contract between components
- Enables multiple implementations (e.g., different reporting formats)

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
func New(cfg *Config, brk Broker, llm Decider) IEngine {
    return &Engine{
        cfg:    cfg,
        broker: brk,
        llm:    llm,

        // Internal components are created here
        positions: newPositionManager(),
        risk:      newRiskManager(),
        stop:      newStopManager(cfg.Stop.Mode, ...),
        executor:  newOrderExecutor(brk),
    }
}
```

**Initialization (bootstrap.go):**
```go
brk := initializeBroker(ctx, cfg)         // Create concrete implementation
decider := initializeDecider(ctx, cfg)    // Create concrete implementation
eng := initializeEngine(cfg, brk, decider) // Inject dependencies, returns IEngine
```

**Benefits:**
- Loose coupling - engine doesn't depend on Zerodha specifically
- Easy to test - can inject mocks for broker and LLM
- Configuration in one place
- Internal composition hidden from caller

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
┌──────────────────────────────────────────────────────────────┐
│  main.go (Event Loop)                                        │
│  - Tick processing (polling symbols)                        │
│  - Signal handling (SIGINT, SIGTERM)                        │
│  - EOD summary generation                                   │
└──────────────────────────────────────────────────────────────┘
                              ↓
┌──────────────────────────────────────────────────────────────┐
│  Engine (IEngine) - Trading Orchestration                    │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Step() Workflow:                                       │ │
│  │ 1. Fetch candles (broker)                              │ │
│  │ 2. Calculate indicators (helpers)                      │ │
│  │ 3. Check stop-loss (stop_manager + position_manager)   │ │
│  │ 4. Get LLM decision (decider)                          │ │
│  │ 5. Validate risk (risk_manager)                        │ │
│  │ 6. Execute orders (order_executor)                     │ │
│  │ 7. Update positions (position_manager)                 │ │
│  │ 8. Update trailing stop (stop_manager)                 │ │
│  └────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
         ↓                           ↓                  ↓
┌──────────────────┐  ┌──────────────────────┐  ┌──────────────┐
│  Broker          │  │  Decider (LLM)       │  │  EOD         │
│  - Zerodha       │  │  - OpenAI            │  │  - CSV Gen   │
│  - Market data   │  │  - Claude            │  │  - P&L Calc  │
│  - Order exec    │  │  - Noop (fallback)   │  │              │
└──────────────────┘  └──────────────────────┘  └──────────────┘
         ↓                      ↓                       ↓
┌──────────────────────────────────────────────────────────────┐
│  Cross-Cutting Concerns                                      │
│  - Logger (Zap) - Structured logging with trace IDs         │
│  - Trace (OpenTelemetry) - Distributed tracing & spans      │
│  - Config (YAML + Validation) - Fail-fast startup           │
│  - Tradelog - JSON trade history                            │
└──────────────────────────────────────────────────────────────┘
```

---

## Summary

The codebase follows **industry-standard patterns**:
- ✅ SOLID principles (especially S, I, D)
- ✅ Clean architecture (layered, decoupled)
- ✅ GoLang idioms (interfaces, context, errors)
- ✅ Fail-fast validation
- ✅ Modular decomposition (engine: 7 modules, eod: 4 modules)

**The design is clean, minimal, and production-ready.**

---

## Project Structure

```
llm-trading-bot/
├── cmd/
│   └── bot/
│       ├── main.go                 # Event loop, signal handling, tick processing
│       └── bootstrap.go            # Initialization (logger, trace, config, components)
│
├── internal/
│   ├── broker/
│   │   └── zerodha/
│   │       ├── zerodha.go          # Zerodha broker implementation
│   │       └── mock.go             # Mock broker for testing
│   │
│   ├── engine/                     # Trading engine (7 modules, 919 lines)
│   │   ├── IEngine.go              # Interface definition (IEngine)
│   │   ├── engine.go               # Orchestration layer (Step workflow)
│   │   ├── position_manager.go    # Position tracking & updates
│   │   ├── risk_manager.go         # Risk validation & exposure limits
│   │   ├── order_executor.go       # Order placement & trade logging
│   │   ├── stop_manager.go         # Stop-loss calculations (PCT/ATR)
│   │   └── helpers.go              # Utilities (indicators, quantity, time)
│   │
│   ├── llm/                        # LLM decision makers (Strategy pattern)
│   │   ├── openai/
│   │   │   └── openai.go           # OpenAI GPT integration
│   │   ├── claude/
│   │   │   └── claude.go           # Anthropic Claude integration
│   │   └── noop/
│   │       └── noop.go             # Fallback decider (always HOLD)
│   │
│   ├── eod/                        # End-of-day reporting (4 modules, 343 lines)
│   │   ├── IEod.go                 # Interface definition (IEodSummarizer)
│   │   ├── eod.go                  # Core implementation (parse, aggregate, CSV)
│   │   ├── types.go                # Data structures (tradeLine, aggRow)
│   │   └── utils.go                # Utilities (paths, IST time, market close)
│   │
│   ├── logger/
│   │   └── logger.go               # Structured logging (Zap with trace IDs)
│   │
│   ├── trace/
│   │   └── trace.go                # Distributed tracing (OpenTelemetry)
│   │
│   ├── store/
│   │   └── config.go               # Configuration loading & validation
│   │
│   ├── tradelog/
│   │   └── tradelog.go             # Trade history logging (JSON format)
│   │
│   ├── ta/
│   │   └── indicators.go           # Technical indicators (RSI, SMA, BB, ATR)
│   │
│   └── types/
│       └── types.go                # Shared types (Candle, Order, Decision, etc.)
│
├── config.yaml                     # Trading configuration (mode, symbols, risk)
├── .env                            # Environment variables (API keys, logging)
├── .env.example                    # Template for environment setup
├── go.mod                          # Go module definition
├── go.sum                          # Dependency checksums
├── README.md                       # User documentation & setup guide
└── DESIGN.md                       # Architecture & design patterns (this file)
```

### Key Files Explained

**Entry Point:**
- `cmd/bot/main.go` - Main event loop, handles ticks, signals, and graceful shutdown
- `cmd/bot/bootstrap.go` - Initializes all components (logger, tracer, broker, LLM, engine)

**Trading Engine (7 modules):**
- `IEngine.go` - Interface with `Step(ctx, symbol)` method
- `engine.go` - Orchestrates the 8-step trading workflow
- `position_manager.go` - Tracks positions, calculates averages, handles buy/sell updates
- `risk_manager.go` - Validates trades against risk limits
- `order_executor.go` - Places orders via broker, logs trades
- `stop_manager.go` - Calculates stop-loss (percentage or ATR-based)
- `helpers.go` - Indicator calculations, quantity selection, utilities

**EOD Reporting (4 modules):**
- `IEod.go` - Interface with `SummarizeDay()`, `SummarizeToday()`, `ShouldRunNow()`
- `eod.go` - Parses trade logs, aggregates by symbol, writes CSV
- `types.go` - `tradeLine` (JSON trade) and `aggRow` (aggregated stats)
- `utils.go` - Path helpers, IST timezone, market close time (3:40 PM)

**Configuration:**
- `config.yaml` - Trading parameters (mode, symbols, poll interval, risk limits)
- `.env` - API keys, logging format, tracing settings

### Module Responsibilities

| Module | Lines | Responsibility |
|--------|-------|----------------|
| **engine.go** | 269 | Orchestrates Step workflow (fetch → indicators → stop → LLM → execute) |
| **position_manager.go** | 170 | Tracks positions, calculates averages, manages open/close |
| **order_executor.go** | 163 | Places orders, logs trades to JSON |
| **helpers.go** | 104 | Calculates indicators, determines quantity, utilities |
| **stop_manager.go** | 93 | Calculates stop-loss, checks triggers |
| **risk_manager.go** | 81 | Validates exposure vs. risk limits |
| **IEngine.go** | 39 | Interface definition and public API |
| **eod.go** | 204 | Parses logs, aggregates trades, writes CSV |
| **utils.go** (eod) | 61 | Path helpers, IST time, market close |
| **IEod.go** | 53 | Interface for EOD summarization |
| **types.go** (eod) | 25 | Trade and aggregation data structures |

**Total:** Engine (919 lines), EOD (343 lines) - well-organized, maintainable, testable.
