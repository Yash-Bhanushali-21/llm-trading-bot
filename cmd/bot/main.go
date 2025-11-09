package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"llm-trading-bot/internal/eod"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/trace"
)

func main() {
	// Initialize system (logger, tracer, env)
	if err := initializeSystem(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Create root context with tracing span
	ctx := context.Background()
	ctx, mainSpan := trace.StartSpan(ctx, "trading-bot-session")
	defer mainSpan.End()

	logger.Info(ctx, "=== LLM Trading Bot Starting ===")

	// Ensure graceful shutdown of tracer
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = trace.Shutdown(shutdownCtx)
	}()

	// Load configuration
	cfg, err := loadConfig(ctx)
	if err != nil {
		os.Exit(1)
	}

	// Setup cancellation context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Compress old logs
	compressOldLogs(ctx)

	// Setup signal handling
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	// Initialize components
	brk := initializeBroker(ctx, cfg)
	decider := initializeDecider(ctx, cfg)
	eng := initializeEngine(cfg, brk, decider)

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
			tickCtx, tickSpan := trace.StartSpan(ctx, "tick-processing")
			logger.Debug(tickCtx, "Tick - processing symbols", "count", len(cfg.UniverseStatic))

			for _, sym := range cfg.UniverseStatic {
				symCtx, symSpan := trace.StartSpan(tickCtx, "process-symbol")

				st, err := eng.Step(symCtx, sym)
				if err != nil {
					logger.ErrorWithErr(symCtx, "Symbol processing failed", err, "symbol", sym)
					symSpan.End()
					continue
				}

				if st != nil {
					logger.Debug(symCtx, "Symbol state updated", "symbol", sym, "state", st)
					b, _ := json.Marshal(st)
					fmt.Println(string(b))
				}
				symSpan.End()
			}
			tickSpan.End()

		case <-eodTick.C:
			eodCtx, eodSpan := trace.StartSpan(ctx, "eod-check")
			if ok, _ := eod.ShouldRunNow(); ok {
				logger.Info(eodCtx, "Running end-of-day summary")
				if p, err := eod.SummarizeToday(); err == nil && p != "" {
					logger.Info(eodCtx, "EOD CSV written successfully", "path", p)
				} else if err != nil {
					logger.ErrorWithErr(eodCtx, "Failed to write EOD CSV", err)
				}
			}
			eodSpan.End()

		case <-sigc:
			shutdownCtx, shutdownSpan := trace.StartSpan(ctx, "graceful-shutdown")
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
