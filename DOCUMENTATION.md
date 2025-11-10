# LLM Trading Bot - Code Documentation

## Architecture Overview

The bot follows a clean architecture with observability middleware pattern:
- **Business Logic**: Pure implementation without logging
- **Observability Middleware**: Wraps interfaces with logging and tracing
- **Interfaces**: Centralized in `internal/interfaces/`

## Core Components

### Engine (`internal/engine/`)

#### Engine.New()
Creates a new trading engine instance. Initializes all sub-components (position manager, risk manager, stop manager, order executor).

#### Engine.Step()
Executes one complete trading cycle for a symbol:
1. Fetches recent candles from broker
2. Calculates technical indicators (RSI, SMA, BB, ATR)
3. Checks stop-loss triggers
4. Gets LLM trading decision
5. Determines trade quantity
6. Executes trading action (BUY/SELL/HOLD)
7. Updates trailing stop-loss if enabled

Returns StepResult with decision details and orders.

#### fetchCandles()
Retrieves recent candle data from broker. Returns error if insufficient data available (minimum 50 candles required).

#### logIndicators()
Empty stub - actual logging handled by observability middleware.

#### handleStopLoss()
Checks if stop-loss is triggered for current position. If triggered, places SELL order and closes position. Returns StepResult if stop executed, nil otherwise.

#### executeDecision()
Executes the LLM trading decision:
- **BUY**: Checks risk limits, calculates stop-loss, places order, updates position
- **SELL**: Places order, updates position, calculates realized P&L
- **HOLD**: No action

Returns StepResult with execution details.

#### updateTrailingStop()
Updates trailing stop-loss if enabled in config. Only trails upward, never down. Returns boolean indicating if stop was updated.

---

### Order Executor (`internal/engine/order_executor.go`)

#### placeBuyOrder()
Executes BUY order via broker. Logs trade to tradelog on success. Returns OrderResp with order details.

#### placeSellOrder()
Executes SELL order via broker. Supports multiple exit reasons via tag parameter. Logs trade to tradelog on success.

#### logDecision()
Logs LLM trading decision to decision log file for analysis.

---

### Position Manager (`internal/engine/position_manager.go`)

#### get()
Retrieves position for a symbol. Returns nil if no position exists.

#### has()
Checks if position exists for symbol.

#### addBuy()
Updates position after BUY execution. Calculates new average price using weighted average. Updates stop-loss and ATR. Creates new position if first buy.

#### reduceSell()
Updates position after SELL execution. Calculates realized P&L. Closes position if fully sold.

#### updateTrailingStop()
Updates trailing stop for a position. Only trails upward based on new ATR calculation. Returns true if stop was updated.

---

### Risk Manager (`internal/engine/risk_manager.go`)

#### canTrade()
Checks if new trade is allowed based on risk limits:
- Per-symbol max quantity limit
- Daily trade count cap

Returns boolean indicating if trade allowed.

---

### Stop Manager (`internal/engine/stop_manager.go`)

#### shouldTrigger()
Checks if stop-loss should trigger based on configured mode:
- **FIXED**: Percentage-based stop (e.g. 2% below entry)
- **ATR**: ATR-based dynamic stop (e.g. 2x ATR below entry)
- **TIME**: Time-based exit after N minutes
- **HYBRID**: Combines ATR + time-based stops

Returns true if stop should trigger.

#### calculateInitial()
Calculates initial stop-loss price for new position based on mode. Rounds to configured tick size.

#### calculateTrailing()
Calculates new trailing stop price. Only moves upward, never down. Ensures minimum tick movement.

---

### Helpers (`internal/engine/helpers.go`)

#### roundToTick()
Rounds price to nearest tick size. Returns original if tick is 0 or negative.

#### midnightIST()
Returns midnight timestamp in Indian Standard Time for current day. Used for day boundary tracking.

#### calculateIndicators()
Computes all technical indicators from candle data:
- RSI (Relative Strength Index)
- SMA (Simple Moving Average) for multiple windows
- Bollinger Bands (Middle, Upper, Lower)
- ATR (Average True Range)

Returns Indicators struct with all calculated values.

#### pickQuantity()
Determines trade quantity using priority order:
1. Quantity from LLM decision (if > 0)
2. Per-symbol configuration
3. Default buy/sell quantity

---

## Broker (`internal/broker/zerodha/`)

#### NewZerodha()
Creates Zerodha broker instance. Initializes ticker manager for live data if configured.

#### newTickerManager()
Creates WebSocket ticker manager for live candle streaming. Initializes candle cache and token mapping.

#### LTP()
Returns last traded price for symbol. Currently returns mock price for testing.

#### RecentCandles()
Fetches recent candles. Routes to live ticker or static mock data based on configuration.

#### fetchStaticCandles()
Generates mock candle data for testing. Creates realistic OHLC bars with random variation.

#### fetchLiveCandles()
Retrieves candles from WebSocket ticker cache. Falls back to static if unavailable.

#### PlaceOrder()
Places order. Returns simulated response in DRY_RUN mode, otherwise places live order.

#### Start()
Initializes broker for trading session. Starts WebSocket ticker and subscribes to symbols if live mode.

#### Stop()
Stops broker and closes WebSocket connections.

---

### Ticker Manager (`internal/broker/zerodha/ticker_manager.go`)

#### Start()
Initializes WebSocket connection to Zerodha. Sets up event handlers and starts ticker in goroutine.

#### Stop()
Closes WebSocket connection gracefully.

#### Subscribe()
Subscribes to symbols for live data streaming. Sets ticker mode to FULL for OHLC data.

#### GetRecentCandles()
Retrieves recent candles from internal cache. Returns error if no data available.

#### addCandle()
Adds candle to symbol's buffer. Maintains max buffer size of 200 candles per symbol.

#### getPlaceholderToken()
Returns instrument token for symbol. Currently uses hardcoded token map for testing.

---

### Ticker Events (`internal/broker/zerodha/ticker_events.go`)

#### setupEventHandlers()
Configures all WebSocket event callbacks.

#### onConnect()
Connection established to WebSocket.

#### onError()
WebSocket error occurred. Logged by ticker.

#### onClose()
WebSocket connection closed.

#### onReconnect()
WebSocket attempting reconnection.

#### onNoReconnect()
WebSocket reconnection failed after max attempts.

#### onTick()
Receives tick data from WebSocket. Converts to candle and adds to cache.

#### onOrderUpdate()
Receives order update from WebSocket (not yet implemented).

---

## LLM Deciders (`internal/llm/`)

### OpenAI Decider (`openai/openai.go`)

#### NewOpenAIDecider()
Creates OpenAI-based decider instance with configured model and parameters.

#### Decide()
Makes trading decision using OpenAI API. Sends market data and indicators as prompt. Parses JSON response into Decision struct.

---

### Claude Decider (`claude/claude.go`)

#### NewClaudeDecider()
Creates Claude-based decider instance.

#### Decide()
Makes trading decision using Anthropic Claude API. Sends structured prompt with market context. Parses response (JSON or natural language).

#### parseDecisionFromText()
Parses Claude's response. Handles both JSON format and natural language. Returns HOLD decision if parsing fails.

---

### Noop Decider (`noop/noop.go`)

#### NewNoopDecider()
Creates no-op decider for testing.

#### Decide()
Always returns HOLD decision with zero confidence. Used as fallback when no LLM configured.

---

## Observability Middleware

### Broker Observability (`brokerobs/brokerobs.go`)

#### Wrap()
Wraps Broker interface with logging and tracing. All broker operations logged with context.

Methods wrapped: LTP, RecentCandles, PlaceOrder, Start, Stop

---

### LLM Observability (`llmobs/llmobs.go`)

#### Wrap()
Wraps Decider interface with logging and tracing. Logs decision requests and responses.

---

### Engine Observability (`engineobs/engineobs.go`)

#### Wrap()
Wraps Engine interface with logging and tracing. Logs trading cycle start, completion, and duration metrics.

---

### EOD Observability (`eodobs/eodobs.go`)

#### Wrap()
Wraps EodSummarizer interface with logging. Tracks summary generation success/failure.

---

## EOD Summarizer (`internal/eod/`)

#### NewSummarizer()
Creates end-of-day summarizer instance.

#### SetDefaultSummarizer()
Sets custom default summarizer (used for middleware injection).

#### SummarizeDay()
Generates CSV summary for specific date. Aggregates trades by symbol, calculates P&L.

#### SummarizeToday()
Convenience wrapper for today's summary.

#### ShouldRunNow()
Checks if EOD should run based on time (after 3:40 PM IST) and whether summary already exists.

#### parseTradeLog()
Parses JSON trade log file. Aggregates buy/sell volumes and values by symbol.

#### writeCSVSummary()
Writes aggregated trade data to CSV. Includes per-symbol stats and total row.

---

## Trade Logging (`internal/tradelog/`)

#### Append()
Appends trade entry to daily log file.

#### AppendDecision()
Appends LLM decision to decision log.

#### CompressOlder()
Compresses log files older than N days using gzip.

---

## Configuration (`internal/store/`)

#### LoadConfig()
Loads configuration from YAML file. Validates all required fields. Returns Config struct.

---

## Logger (`internal/logger/`)

#### Init()
Initializes global logger with configured level and format.

#### Debug/Info/Warn/Error()
Standard logging functions with context and structured fields.

#### DebugSkip/InfoSkip/WarnSkip/ErrorSkip()
Logging functions with caller skip for middleware use. Reports actual caller instead of wrapper.

#### ErrorWithErr()
Logs error with tracing integration. Records error in OpenTelemetry span.

---

## Tracer (`internal/trace/`)

#### Init()
Initializes OpenTelemetry tracing exporter.

#### StartSpan()
Creates new trace span with operation name.

#### GetTraceFields()
Extracts trace ID and span ID from context for logging.

---

## Interfaces (`internal/interfaces/`)

All interface definitions centralized:
- **Broker**: Market data and order execution
- **Decider**: LLM trading decisions
- **Engine**: Trading engine orchestration
- **EodSummarizer**: End-of-day reporting
- **TickerManager**: WebSocket ticker management

---

## Bootstrap (`cmd/bot/`)

#### initializeSystem()
Initializes logger, tracer, and EOD summarizer with observability wrappers.

#### loadConfig()
Loads and validates configuration from config.yaml.

#### compressOldLogs()
Compresses old tradelog files based on retention policy.

#### initializeBroker()
Creates broker instance and wraps with observability middleware.

#### initializeDecider()
Creates LLM decider based on provider config. Wraps with observability middleware.

#### initializeEngine()
Creates trading engine and wraps with observability middleware.

#### initializeEOD()
Wraps default EOD summarizer with observability middleware.

---

## Key Design Patterns

### Decorator Pattern
Observability middleware wraps interfaces to add cross-cutting concerns without modifying business logic.

### Factory Pattern
New() functions provide clean interface-based construction of components.

### Strategy Pattern
Multiple LLM providers (OpenAI, Claude, Noop) implement same Decider interface.

### Repository Pattern
Position manager encapsulates position storage and updates.
