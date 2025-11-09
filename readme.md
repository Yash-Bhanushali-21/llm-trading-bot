# LLM Trading Bot (GoLang)

A **scalable, modular, and AI-driven trading engine** built in **Go**, designed to integrate seamlessly with multiple brokers, LLMs, and technical indicators.
The bot leverages **Large Language Models (LLMs)** like **OpenAI GPT** and **Claude** to interpret market signals, execute trades, and optimize decision-making in real time.

---

## Overview

The bot follows a **scalable plug-and-play architecture**, enabling flexible integrations across:
- **Brokers:** Zerodha (default), with easy extensions to others like AngelOne or Dhan.
- **LLMs:** OpenAI GPT (primary) and Anthropic Claude (fallback).
- **Indicators:** RSI, Bollinger Bands, SMA, VWAP, MACD, OBV, Donchian Channel, SuperTrend, and more.

It can analyze live market data, reason about entry/exit points, and place trades automatically — all while maintaining strict risk management and exposure limits.

---

## Features

### **Multi-Broker Architecture**
- Broker interface layer allowing plug-and-play support.
- Zerodha (default) with extensibility for AngelOne, Dhan, etc.

### **Multi-LLM Integration**
- OpenAI GPT as primary.
- Claude (Anthropic) or local models as fallback.
- Automatic retry, throttling, and failover logic.

### **Advanced Indicator Engine**
- Built-in indicators:
  - RSI, Bollinger Bands, SMA, VWAP, OBV, MACD, Donchian Channel, SuperTrend.
- Easy to add or remove indicators via modular packages.

### **Core Engine**
- Handles trade logic, risk checks, and LLM signal aggregation.
- DRY_RUN mode for simulation and backtesting.
- Supports stop-loss, take-profit, and daily risk control.

### ⚡ **Concurrent & Fault-Tolerant**
- Parallel routines for data streaming, order execution, and LLM inference.
- Retry and fallback for network/API errors.

### **Real-Time Data**
- Live WebSocket integration with Zerodha.
- Supports 1m, 5m, and 15m candle aggregation.

### **Risk Management**
- Enforces per-trade risk limits.
- Calculates exposure dynamically.
- Integrates with broker margin APIs or simulated capital.

### **Logging & Tracing**
- Structured logging with configurable formats (JSON or text)
- Distributed tracing with OpenTelemetry
- End-of-day trade summaries and performance reports
- Trace IDs for complete request flow tracking

## Architecture Overview

[ Broker Layer ] ->[ Core Engine ] -> [ LLM Layer ] -> [ Indicator Engine ]

### **Layered System Design**

### **Component Roles**
- **Broker Layer:** Handles trade execution, positions, and live data via APIs (Zerodha, etc.)
- **Core Engine:** Controls logic flow — from data → decision → order.
- **LLM Layer:** Interprets indicators, sentiment, and context to decide BUY/SELL/WAIT.
- **Indicator Engine:** Processes live candles and computes metrics like RSI, MACD, VWAP, etc.


## Modes of Operation

### **DRY_RUN Mode**
- Simulated trades for safe testing.
- Logs reasoning, confidence, and P&L.

### **LIVE Mode**
- Direct broker connection.
- Executes real trades with live capital and stop-loss control.

---

## Setup

### Prerequisites
- Go 1.22+
- Zerodha Kite Connect API credentials (for LIVE mode)
- OpenAI / Claude API keys (optional, bot can run without LLM in HOLD mode)

### Installation

```bash
# Clone repository
git clone https://github.com/yourusername/llm-trading-bot.git
cd llm-trading-bot

# Install dependencies
go mod tidy
```

### Configuration

#### 1. Environment Variables (.env)

Create a `.env` file from the example:

```bash
cp .env.example .env
```

Edit `.env` with your credentials:

```bash
# LLM API Keys (choose one or both)
OPENAI_API_KEY=your-openai-key-here
CLAUDE_API_KEY=your-claude-key-here

# Zerodha API (required for LIVE mode only)
KITE_API_KEY=your-kite-api-key
KITE_ACCESS_TOKEN=your-kite-access-token

# Logging Configuration
LOG_LEVEL=INFO              # DEBUG, INFO, WARN, ERROR
LOG_FORMAT=text             # text (readable) or json (production)
LOG_DETAILED=true           # Include file:line numbers and timings
LOG_TRACING_ENABLED=true    # Enable distributed tracing with trace IDs
```

#### 2. Trading Configuration (config.yaml)

Edit `config.yaml` to configure trading parameters:

```yaml
mode: DRY_RUN              # DRY_RUN (safe) or LIVE (real trading)
exchange: NSE              # NSE or BSE
poll_seconds: 120          # How often to check symbols (in seconds)
universe_static:           # Symbols to trade
  - RELIANCE
  - TCS
llm:
  provider: OPENAI         # OPENAI, CLAUDE, or leave empty for HOLD-only
```

### Running the Bot

#### Development (Quick Start)

```bash
# Run directly without building (recommended for development)
go run ./cmd/bot

# Alternative: specify all files in the package
go run cmd/bot/*.go
```

**Important:** Do NOT use `go run cmd/bot/main.go` - this will fail because Go needs all files in the package (main.go and bootstrap.go).

#### Production (Build Binary)

```bash
# Build optimized binary
go build -o bot ./cmd/bot

# Run the binary
./bot

# Run in background
nohup ./bot > bot.log 2>&1 &
```

#### Graceful Shutdown

The bot handles `SIGINT` (Ctrl+C) and `SIGTERM` gracefully:
- Generates final end-of-day summary
- Closes all positions (if configured)
- Flushes logs and traces
- Shuts down cleanly

Press `Ctrl+C` to stop the bot gracefully.

### Viewing Logs

**Text Format (Development):**
```
2025-11-09T15:32:16.872Z	info	bot/main.go:29	=== LLM Trading Bot Starting ===
2025-11-09T15:32:16.875Z	info	bot/main.go:66	Bot started - entering main loop
```

**JSON Format (Production):**
```json
{"level":"info","time":"2025-11-09T15:32:16.872Z","msg":"Bot started","trace_id":"4aa465911ea91eff980156f4325a89fe"}
```

Enable detailed logging in `.env`:
```bash
LOG_LEVEL=DEBUG
LOG_DETAILED=true
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/engine/...

# Verbose test output
go test -v ./...
```

---

## Project Structure

```
llm-trading-bot/
├── cmd/
│   └── bot/
│       ├── main.go         # Main entry point and event loop
│       └── bootstrap.go    # Initialization logic
├── internal/
│   ├── broker/            # Broker integrations (Zerodha, etc.)
│   ├── engine/            # Core trading engine
│   ├── llm/               # LLM integrations (OpenAI, Claude)
│   ├── indicators/        # Technical indicators
│   ├── logger/            # Structured logging
│   ├── trace/             # Distributed tracing
│   ├── store/             # Configuration and state management
│   ├── tradelog/          # Trade logging and history
│   └── types/             # Shared types and interfaces
├── config.yaml            # Trading configuration
├── .env                   # Environment variables (create from .env.example)
└── go.mod                 # Go module definition
```

---

## Common Issues

### "undefined: initializeSystem" error

This happens when you run only `main.go`:
```bash
# ❌ Wrong - only compiles main.go
go run cmd/bot/main.go

# ✅ Correct - compiles all files in the package
go run ./cmd/bot
```

### Bot exits immediately

Check that `config.yaml` exists and is valid YAML. The bot will log errors if configuration is missing or invalid.

### No trades executing

- Verify you're in the correct mode (DRY_RUN vs LIVE)
- Check that symbols in `universe_static` are valid
- Ensure `poll_seconds` isn't too long (try 10 for testing)
- Check logs for errors or warnings

---

## License

MIT License - see LICENSE file for details.
