package main

import (
	"context"
	"encoding/json"
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
	"llm-trading-bot/internal/research/pead"
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

// runPEADPrefilter runs PEAD analysis to generate a filtered list of qualified stocks
// This runs BEFORE the bot starts trading to ensure only high-quality stocks are traded
func runPEADPrefilter(ctx context.Context, cfg *store.Config) error {
	// Skip if PEAD is disabled
	if !cfg.PEAD.Enabled {
		logger.Info(ctx, "PEAD pre-filter disabled in config")
		return nil
	}

	logger.Info(ctx, "╔══════════════════════════════════════════════════════════════╗")
	logger.Info(ctx, "║       Running PEAD Pre-Filter for Stock Selection           ║")
	logger.Info(ctx, "╚══════════════════════════════════════════════════════════════╝")

	// Determine symbols to analyze
	symbols := cfg.Universe.Static
	if len(symbols) == 0 {
		logger.Info(ctx, "No symbols in config, using NSE Nifty 50 for PEAD analysis")
		symbols = pead.GetNSETop50()
	}

	logger.Info(ctx, "Analyzing stocks for PEAD qualification", "count", len(symbols))

	// Create data fetcher based on config
	var fetcher pead.EarningsDataFetcher
	if cfg.PEAD.DataSource == "MOCK" {
		logger.Info(ctx, "Using MOCK earnings data for PEAD analysis")
		fetcher = pead.NewMockEarningsDataFetcher()
	} else {
		logger.Info(ctx, "Using LIVE earnings data from NSE sources")
		fetcher = pead.NewNSEDataFetcher()
	}

	// Create PEAD config from main config
	peadConfig := pead.PEADConfig{
		MinCompositeScore:    cfg.PEAD.MinCompositeScore,
		MinDaysSinceEarnings: cfg.PEAD.MinDaysSinceEarnings,
		MaxDaysSinceEarnings: cfg.PEAD.MaxDaysSinceEarnings,
		MinEarningsSurprise:  cfg.PEAD.MinEarningsSurprise,
		MinRevenueGrowth:     cfg.PEAD.MinRevenueGrowth,
		MinEPSGrowth:         cfg.PEAD.MinEPSGrowth,
		EnableNLP:            cfg.PEAD.EnableNLP,
		DataSource:           cfg.PEAD.DataSource,
		APIKeyEnv:            cfg.PEAD.APIKeyEnv,
		Weights:              cfg.PEAD.Weights,
	}

	// Create analyzer and run analysis
	analyzer := pead.NewAnalyzer(peadConfig, fetcher)
	result, err := analyzer.Analyze(ctx, symbols)
	if err != nil {
		logger.Warn(ctx, "PEAD analysis failed - bot will use original universe", "error", err)
		logger.Warn(ctx, "This is often due to network restrictions or API rate limits")
		logger.Warn(ctx, "To use PEAD filtering, ensure network access or set data_source: MOCK for testing")
		return nil // Don't fail the bot startup, just use original universe
	}

	// Extract qualified symbols
	qualifiedSymbols := make([]string, 0, len(result.QualifiedSymbols))
	for _, score := range result.QualifiedSymbols {
		qualifiedSymbols = append(qualifiedSymbols, score.Symbol)
	}

	logger.Info(ctx, "")
	logger.Info(ctx, "═══════════════════════════════════════════════════════════════")
	logger.Info(ctx, "              PEAD ANALYSIS RESULTS")
	logger.Info(ctx, "═══════════════════════════════════════════════════════════════")
	logger.Info(ctx, "Analysis Summary",
		"total_analyzed", result.TotalAnalyzed,
		"qualified", result.QualifiedCount,
		"qualification_rate", fmt.Sprintf("%.1f%%", float64(result.QualifiedCount)/float64(result.TotalAnalyzed)*100),
		"min_score_threshold", result.Config.MinCompositeScore,
	)
	logger.Info(ctx, "")

	// Show detailed scores for each qualified stock
	if len(result.QualifiedSymbols) > 0 {
		logger.Info(ctx, "🎯 QUALIFIED STOCKS (Ranked by Score)")
		logger.Info(ctx, "───────────────────────────────────────────────────────────────")

		for i, score := range result.QualifiedSymbols {
			logger.Info(ctx, fmt.Sprintf("Rank #%d: %s", i+1, score.Symbol),
				"composite_score", fmt.Sprintf("%.1f/100", score.CompositeScore),
				"rating", score.Rating,
				"quarter", score.Quarter,
				"days_since_earnings", score.DaysSinceEarnings,
			)

			// Show component scores
			logger.Info(ctx, fmt.Sprintf("  Component Scores for %s:", score.Symbol),
				"earnings_surprise", fmt.Sprintf("%.1f", score.EarningsSurpriseScore),
				"earnings_growth", fmt.Sprintf("%.1f", score.EarningsGrowthScore),
				"revenue_growth", fmt.Sprintf("%.1f", score.RevenueGrowthScore),
				"margin_expansion", fmt.Sprintf("%.1f", score.MarginExpansionScore),
				"consistency", fmt.Sprintf("%.1f", score.ConsistencyScore),
			)

			// Show fundamental metrics
			eps := score.EarningsData
			logger.Info(ctx, fmt.Sprintf("  Fundamentals for %s:", score.Symbol),
				"eps_surprise", fmt.Sprintf("%.2f%%", eps.EarningSurprise()),
				"yoy_eps_growth", fmt.Sprintf("%.1f%%", eps.YoYEPSGrowth),
				"yoy_revenue_growth", fmt.Sprintf("%.1f%%", eps.YoYRevenueGrowth),
				"net_margin", fmt.Sprintf("%.1f%%", eps.NetMargin),
				"consecutive_beats", eps.ConsecutiveBeats,
			)
			logger.Info(ctx, "")
		}
	}

	// Save results to file for reference
	if err := savePEADResults(ctx, result); err != nil {
		logger.Warn(ctx, "Failed to save PEAD results to file", "error", err)
	} else {
		logger.Info(ctx, "📄 Full results saved to: pead_results.json")
	}

	logger.Info(ctx, "")
	logger.Info(ctx, "═══════════════════════════════════════════════════════════════")

	// Update universe with qualified stocks
	if len(qualifiedSymbols) > 0 {
		cfg.Universe.Static = qualifiedSymbols
		logger.Info(ctx, "✅ Universe updated with PEAD-qualified stocks", "symbols", qualifiedSymbols)
	} else {
		logger.Warn(ctx, "⚠️  No stocks qualified from PEAD analysis - using original universe")
		logger.Warn(ctx, "Consider lowering PEAD_MIN_SCORE threshold or adjusting filters")
	}

	logger.Info(ctx, "")
	return nil
}

// savePEADResults saves PEAD analysis results to a JSON file
func savePEADResults(ctx context.Context, result *pead.PEADResult) error {
	filename := "pead_results.json"

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal PEAD results: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write PEAD results file: %w", err)
	}

	logger.Info(ctx, "PEAD results saved", "file", filename)
	return nil
}
