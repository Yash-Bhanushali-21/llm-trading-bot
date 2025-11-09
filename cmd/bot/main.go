package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"llm-trading-bot/internal/broker"
	"llm-trading-bot/internal/engine"
	"llm-trading-bot/internal/eod"
	"llm-trading-bot/internal/llm"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/tradelog"
	"llm-trading-bot/internal/types"

	"github.com/joho/godotenv"
)

func must(ctx context.Context, err error) {
	if err != nil {
		logger.ErrorWithErr(ctx, "Fatal error", err)
		log.Fatal(err)
	}
}

func main() {
	// Load environment variables
	_ = godotenv.Load()

	// Initialize logger first
	if err := logger.Init(); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	// Create root context with tracing span for the entire session
	ctx := context.Background()
	ctx, mainSpan := logger.StartSpan(ctx, "trading-bot-session")
	defer mainSpan.End()

	logger.Info(ctx, "=== LLM Trading Bot Starting ===")

	// Ensure graceful shutdown of tracer
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := logger.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down tracer: %v", err)
		}
	}()

	// Load configuration
	cfg, err := store.LoadConfig("config.yaml")
	must(ctx, err)

	// Setup cancellation context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Compress old logs if retention is configured
	if v := os.Getenv("TRADER_LOG_RETENTION_DAYS"); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		if err := tradelog.CompressOlder(n); err != nil {
			logger.Warn(ctx, "Failed to compress old logs", "error", err)
		}
	}

	// Setup signal handling for graceful shutdown
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	// Initialize broker
	brk := broker.NewZerodha(cpf(cfg))
	if cfg.Mode == "DRY_RUN" {
		logger.Warn(ctx, "Running in DRY_RUN mode - orders will be simulated")
	}

	// Initialize LLM decider
	var decider types.Decider
	if cfg.LLM.Provider == "OPENAI" {
		decider = llm.NewOpenAIDecider(cfg)
	} else if cfg.LLM.Provider == "CLAUDE" {
		decider = llm.NewClaudeDecider(cfg)
	} else {
		decider = llm.NewNoopDecider()
		logger.Warn(ctx, "No LLM provider configured - using Noop decider (always HOLD)")
	}

	// Initialize trading engine
	eng := engine.New(cfg, brk, decider)

	// Setup tickers
	tick := time.NewTicker(time.Duration(cfg.PollSeconds) * time.Second)
	defer tick.Stop()
	eodTick := time.NewTicker(60 * time.Second)
	defer eodTick.Stop()

	logger.Info(ctx, "Bot started - entering main loop",
		"poll_interval_seconds", cfg.PollSeconds,
		"symbols", cfg.UniverseStatic,
	)

	// Main event loop
	for {
		select {
		case <-tick.C:
			// Create a new span for this tick
			tickCtx, tickSpan := logger.StartSpan(ctx, "tick-processing")
			logger.Debug(tickCtx, "Tick - processing symbols", "count", len(cfg.UniverseStatic))

			for _, sym := range cfg.UniverseStatic {
				// Create symbol-specific span
				symCtx, symSpan := logger.StartSpan(tickCtx, "process-symbol")
				logger.Debug(symCtx, "Processing symbol", "symbol", sym)

				timer := logger.StartOperation(symCtx, "symbol_processing", "symbol", sym)
				st, err := eng.Step(timer.GetContext(), sym)
				if err != nil {
					timer.EndWithError(err)
					logger.ErrorWithErr(symCtx, "Symbol processing failed", err, "symbol", sym)
					symSpan.End()
					continue
				}
				timer.End("status", "success")

				if st != nil {
					logger.Debug(symCtx, "Symbol state updated", "symbol", sym, "state", st)
					if logger.IsDebugEnabled() {
						b, _ := json.Marshal(st)
						fmt.Println(string(b))
					}
				}
				symSpan.End()
			}
			tickSpan.End()

		case <-eodTick.C:
			eodCtx, eodSpan := logger.StartSpan(ctx, "eod-check")
			logger.Debug(eodCtx, "Checking if EOD summary should run")
			if ok, _ := eod.ShouldRunNow(); ok {
				logger.Info(eodCtx, "Running end-of-day summary")
				timer := logger.StartOperation(eodCtx, "eod_summary")
				if p, err := eod.SummarizeToday(); err == nil && p != "" {
					timer.End("path", p)
					logger.Info(eodCtx, "EOD CSV written successfully", "path", p)
				} else if err != nil {
					timer.EndWithError(err)
					logger.ErrorWithErr(eodCtx, "Failed to write EOD CSV", err)
				}
			}
			eodSpan.End()

		case <-sigc:
			shutdownCtx, shutdownSpan := logger.StartSpan(ctx, "graceful-shutdown")
			logger.Info(shutdownCtx, "Shutdown signal received - gracefully shutting down")

			// Generate final EOD summary
			logger.Info(shutdownCtx, "Generating final end-of-day summary")
			if p, err := eod.SummarizeToday(); err == nil && p != "" {
				logger.Info(shutdownCtx, "Final EOD CSV written", "path", p)
			} else if err != nil {
				logger.ErrorWithErr(shutdownCtx, "Failed to write final EOD CSV", err)
			}

			logger.Info(shutdownCtx, "=== LLM Trading Bot Shutdown Complete ===")
			shutdownSpan.End()
			return

		case <-ctx.Done():
			logger.Info(ctx, "Context cancelled - exiting")
			return
		}
	}
}

func cpf(c *store.Config) broker.Params {
	return broker.Params{Mode: c.Mode, APIKey: os.Getenv("KITE_API_KEY"), AccessToken: os.Getenv("KITE_ACCESS_TOKEN"), Exchange: c.Exchange}
}
