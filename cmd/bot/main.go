package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/trace"

	// Commented out imports - will be needed when trading logic is re-enabled
	// "encoding/json"
	// "os/signal"
	// "syscall"
	// "llm-trading-bot/internal/eod"
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

	// Run PEAD pre-filter to select high-quality stocks
	// This runs BEFORE the bot starts and filters the universe based on earnings quality
	// If PEAD fails (e.g., network issues), the bot continues with the original universe
	if err := runPEADPrefilter(ctx, cfg); err != nil {
		logger.Warn(ctx, "PEAD pre-filter failed - continuing with original universe", "error", err)
	}

	// Compress old logs
	compressOldLogs(ctx)

	// ═══════════════════════════════════════════════════════════════════════════
	// TRADING LOGIC DISABLED - Currently only running PEAD analysis
	// ═══════════════════════════════════════════════════════════════════════════
	// Uncomment the section below to enable actual trading with Zerodha
	// For now, we only run PEAD analysis to select stocks

	logger.Info(ctx, "═══════════════════════════════════════════════════════════════")
	logger.Info(ctx, "Trading logic is DISABLED - PEAD analysis complete")
	logger.Info(ctx, "Check 'pead_results.json' for detailed analysis results")
	logger.Info(ctx, "Selected stocks for trading:", "symbols", cfg.Universe.Static)
	logger.Info(ctx, "═══════════════════════════════════════════════════════════════")
	logger.Info(ctx, "To enable trading, uncomment the trading logic in cmd/bot/main.go")

	// Exit gracefully after showing PEAD results
	logger.Info(ctx, "=== LLM Trading Bot Shutdown (PEAD Analysis Only) ===")
	return

	/*
	// Setup signal handling
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	// Initialize components
	brk := initializeBroker(ctx, cfg)
	decider := initializeDecider(ctx, cfg)
	eng := initializeEngine(cfg, brk, decider)

	// Start broker (WebSocket connections if in LIVE mode)
	if err := brk.Start(ctx, cfg.UniverseStatic); err != nil {
		logger.ErrorWithErr(ctx, "Failed to start broker", err)
		os.Exit(1)
	}
	defer brk.Stop(ctx)

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

			// Stop broker connections
			logger.Info(shutdownCtx, "Stopping broker connections")
			brk.Stop(shutdownCtx)

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
	*/
}
