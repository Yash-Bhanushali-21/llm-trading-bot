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

### **Logging & EOD Reports**
- Structured trade logs with LLM reasoning and confidence.
- EOD summaries with trade outcomes and performance.

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

## 🛠️Setup

### Prerequisites
- Go 1.22+
- Zerodha Kite Connect API credentials
- OpenAI / Claude API keys

### Installation
```bash
# Clone repository
git clone https://github.com/yourusername/llm-trading-bot.git
cd llm-trading-bot

# Install dependencies
go mod tidy

# Run in dry-run mode
go run main.go --mode=DRY_RUN

# Run live (use with caution)
go run main.go --mode=LIVE
