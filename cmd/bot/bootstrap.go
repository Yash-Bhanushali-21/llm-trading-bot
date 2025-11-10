package main

import (
	"context"
	"fmt"
	"os"

	"llm-trading-bot/internal/broker/brokerobs"
	"llm-trading-bot/internal/broker/zerodha"
	"llm-trading-bot/internal/engine"
	"llm-trading-bot/internal/engine/engineobs"
	"llm-trading-bot/internal/eod"
	"llm-trading-bot/internal/eod/eodobs"
	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/llm/claude"
	"llm-trading-bot/internal/llm/llmobs"
	"llm-trading-bot/internal/llm/noop"
	"llm-trading-bot/internal/llm/openai"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/trace"
	"llm-trading-bot/internal/tradelog"

	"github.com/joho/godotenv"
)

// initializeSystem initializes logger, tracer, and EOD summarizer
func initializeSystem() error {
	// Load environment variables
	_ = godotenv.Load()

	// Initialize logger
	if err := logger.Init(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize tracer
	if err := trace.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize tracer: %v\n", err)
	}

	// Initialize EOD summarizer with observability
	initializeEOD()

	return nil
}

// loadConfig loads and returns the configuration
func loadConfig(ctx context.Context) (*store.Config, error) {
	cfg, err := store.LoadConfig("config.yaml")
	if err != nil {
		logger.ErrorWithErr(ctx, "Failed to load config", err)
		return nil, err
	}
	return cfg, nil
}

// compressOldLogs compresses old tradelog files if retention is configured
func compressOldLogs(ctx context.Context) {
	if v := os.Getenv("TRADER_LOG_RETENTION_DAYS"); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		if err := tradelog.CompressOlder(n); err != nil {
			logger.Warn(ctx, "Failed to compress old logs", "error", err)
		}
	}
}

// initializeBroker initializes and returns the broker instance with observability
func initializeBroker(ctx context.Context, cfg *store.Config) interfaces.Broker {
	// Create base broker
	brk := zerodha.NewZerodha(zerodha.Params{
		Mode:         cfg.Mode,
		APIKey:       os.Getenv("KITE_API_KEY"),
		AccessToken:  os.Getenv("KITE_ACCESS_TOKEN"),
		Exchange:     cfg.Exchange,
		CandleSource: cfg.DataSource,
	})

	// Log initialization info
	if cfg.Mode == "DRY_RUN" {
		logger.Warn(ctx, "Running in DRY_RUN mode - orders will be simulated")
	}

	if cfg.DataSource == "LIVE" {
		logger.Info(ctx, "Using LIVE candle data from Zerodha")
	} else {
		logger.Info(ctx, "Using STATIC mock candle data for testing")
	}

	// Wrap with observability middleware
	return brokerobs.Wrap(brk)
}

// initializeDecider initializes and returns the LLM decider with observability
func initializeDecider(ctx context.Context, cfg *store.Config) interfaces.Decider {
	var decider interfaces.Decider

	switch cfg.LLM.Provider {
	case "OPENAI":
		decider = openai.NewOpenAIDecider(cfg)
	case "CLAUDE":
		decider = claude.NewClaudeDecider(cfg)
	default:
		decider = noop.NewNoopDecider()
		logger.Warn(ctx, "No LLM provider configured - using Noop decider (always HOLD)")
	}

	// Wrap with observability middleware
	return llmobs.Wrap(decider)
}

// initializeEngine initializes and returns the trading engine with observability
func initializeEngine(cfg *store.Config, brk interfaces.Broker, decider interfaces.Decider) interfaces.Engine {
	// Create base engine
	eng := engine.New(cfg, brk, decider)

	// Wrap with observability middleware
	return engineobs.Wrap(eng)
}

// initializeEOD wraps the default EOD summarizer with observability
func initializeEOD() {
	// Create base summarizer
	baseSummarizer := eod.NewSummarizer()

	// Wrap with observability middleware
	observableSummarizer := eodobs.Wrap(baseSummarizer)

	// Set as default summarizer
	eod.SetDefaultSummarizer(observableSummarizer)
}
