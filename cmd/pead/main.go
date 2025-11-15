package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"llm-trading-bot/internal/research/pead"
	"llm-trading-bot/internal/store"
)

func main() {
	// Load configuration
	cfg, err := store.LoadConfig("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Convert config struct to PEADConfig
	peadConfig := pead.PEADConfig{
		Enabled:              cfg.PEAD.Enabled,
		MinDaysSinceEarnings: cfg.PEAD.MinDaysSinceEarnings,
		MaxDaysSinceEarnings: cfg.PEAD.MaxDaysSinceEarnings,
		MinCompositeScore:    cfg.PEAD.MinCompositeScore,
		MinEarningsSurprise:  cfg.PEAD.MinEarningsSurprise,
		MinRevenueGrowth:     cfg.PEAD.MinRevenueGrowth,
		MinEPSGrowth:         cfg.PEAD.MinEPSGrowth,
		EnableNLP:            cfg.PEAD.EnableNLP,
		DataSource:           cfg.PEAD.DataSource,
		APIKeyEnv:            cfg.PEAD.APIKeyEnv,
		Weights: pead.ScoringWeights{
			EarningsSurprise:    cfg.PEAD.Weights.EarningsSurprise,
			RevenueSurprise:     cfg.PEAD.Weights.RevenueSurprise,
			EarningsGrowth:      cfg.PEAD.Weights.EarningsGrowth,
			RevenueGrowth:       cfg.PEAD.Weights.RevenueGrowth,
			MarginExpansion:     cfg.PEAD.Weights.MarginExpansion,
			Consistency:         cfg.PEAD.Weights.Consistency,
			RevenueAcceleration: cfg.PEAD.Weights.RevenueAcceleration,
			Sentiment:           cfg.PEAD.Weights.Sentiment,
			ToneDivergence:      cfg.PEAD.Weights.ToneDivergence,
			LinguisticQuality:   cfg.PEAD.Weights.LinguisticQuality,
		},
	}

	// Override min score from environment if set
	if envScore := os.Getenv("PEAD_MIN_SCORE"); envScore != "" {
		if score, err := strconv.ParseFloat(envScore, 64); err == nil {
			peadConfig.MinCompositeScore = score
		}
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║       PEAD Research Module - Post-Earnings Analysis         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create NSE fetcher - LIVE data only
	fmt.Println("📊 Fetching LIVE earnings data from NSE sources")
	fmt.Println("📡 Using Yahoo Finance + NSE API + Screener.in")
	fmt.Println("⏳ This may take a few moments...")
	fmt.Println()
	fetcher := pead.NewNSEDataFetcher()

	// Create analyzer
	analyzer := pead.NewAnalyzer(peadConfig, fetcher)

	// Get symbols from config (use universe_dynamic candidate_list)
	symbols := cfg.Universe.Dynamic.CandidateList
	if len(symbols) == 0 {
		// Fallback to static universe
		symbols = cfg.Universe.Static
	}

	// If still no symbols, fetch companies with recent earnings announcements
	if len(symbols) == 0 {
		fmt.Println("ℹ️  No symbols configured - fetching companies with recent earnings announcements")
		fmt.Println("📡 Querying NSE API for stocks that reported earnings in last 60 days...")
		fmt.Println()

		ctx := context.Background()
		recentSymbols, err := fetcher.FetchRecentEarningsAnnouncements(ctx, 60)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to fetch recent earnings: %v\n", err)
			fmt.Fprintln(os.Stderr, "💡 Ensure network connectivity and NSE API access")
			os.Exit(1)
		}

		symbols = recentSymbols
		fmt.Printf("✅ Discovered %d companies with recent earnings announcements\n", len(symbols))
		fmt.Println("🎯 These stocks reported quarterly results in last 60 days")
		fmt.Println()
	}

	if len(symbols) == 0 {
		fmt.Fprintln(os.Stderr, "❌ No symbols available for analysis")
		fmt.Fprintln(os.Stderr, "💡 Check network connectivity or configure symbols in config.yaml")
		os.Exit(1)
	}

	fmt.Printf("🔍 Analyzing %d symbols...\n\n", len(symbols))

	// Run analysis
	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, symbols)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Analysis failed: %v\n", err)
		os.Exit(1)
	}

	// Print results
	printResults(result)

	// Optionally save to JSON file
	if len(os.Args) > 1 && os.Args[1] == "--json" {
		saveResultsJSON(result, "pead_results.json")
	}
}

func printResults(result *pead.PEADResult) {
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("                      ANALYSIS SUMMARY")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Printf("Analysis Date:      %s\n", result.AnalysisDate.Format("2006-01-02 15:04:05"))
	fmt.Printf("Total Analyzed:     %d companies\n", result.TotalAnalyzed)
	fmt.Printf("Qualified:          %d companies (%.1f%%)\n", result.QualifiedCount,
		float64(result.QualifiedCount)/float64(result.TotalAnalyzed)*100)
	fmt.Printf("Min Score Filter:   %.1f\n\n", result.Config.MinCompositeScore)

	if len(result.QualifiedSymbols) == 0 {
		fmt.Println("⚠️  No companies met the qualification criteria")
		fmt.Println()
		fmt.Println("Consider:")
		fmt.Println("  - Lowering PEAD_MIN_SCORE in .env file")
		fmt.Println("  - Adjusting min_eps_growth or min_revenue_growth in config.yaml")
		fmt.Println("  - Expanding the PEAD time window (max_days_since_earnings)")
		return
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("                    QUALIFIED COMPANIES")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	for i, score := range result.QualifiedSymbols {
		fmt.Printf("✅ Rank #%d: %s (%.1f/100 - %s)\n", i+1, score.Symbol, score.CompositeScore, score.Rating)
		fmt.Println("─────────────────────────────────────────────────────────────")
		fmt.Printf("  📅 Quarter:           %s (announced %d days ago)\n", score.Quarter, score.DaysSinceEarnings)
		fmt.Printf("  💰 EPS Surprise:      %.2f%% (Actual: %.2f vs Expected: %.2f)\n",
			score.EarningsData.EarningSurprise(), score.EarningsData.ActualEPS, score.EarningsData.ExpectedEPS)
		fmt.Printf("  💵 Revenue Surprise:  %.2f%%\n", score.EarningsData.RevenueSurprise())
		fmt.Printf("  📈 YoY EPS Growth:    %.1f%%\n", score.EarningsData.YoYEPSGrowth)
		fmt.Printf("  📈 YoY Revenue Growth: %.1f%%\n", score.EarningsData.YoYRevenueGrowth)
		fmt.Printf("  💹 Net Margin:        %.1f%% (↑ %.1f%%)\n",
			score.EarningsData.NetMargin, score.EarningsData.NetMargin-score.EarningsData.PrevNetMargin)
		fmt.Printf("  🎯 Consistency:       %d consecutive beats\n", score.EarningsData.ConsecutiveBeats)
		fmt.Println()

		fmt.Println("  Component Scores:")
		fmt.Printf("    • Earnings Surprise:    %.1f/100\n", score.EarningsSurpriseScore)
		fmt.Printf("    • Earnings Growth:      %.1f/100\n", score.EarningsGrowthScore)
		fmt.Printf("    • Revenue Growth:       %.1f/100\n", score.RevenueGrowthScore)
		fmt.Printf("    • Margin Expansion:     %.1f/100\n", score.MarginExpansionScore)
		fmt.Printf("    • Consistency:          %.1f/100\n", score.ConsistencyScore)
		fmt.Println()

		fmt.Printf("  📝 %s\n\n", score.Commentary)
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("💡 Next Steps:")
	fmt.Println("  1. Review qualified companies above")
	fmt.Println("  2. Add top picks to universe_static in config.yaml")
	fmt.Println("  3. Run the trading bot to analyze these symbols")
	fmt.Println("  4. Monitor PEAD drift over the next 30-60 days")
}

func saveResultsJSON(result *pead.PEADResult, filename string) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal results: %v\n", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write file: %v\n", err)
		return
	}

	fmt.Printf("💾 Results saved to %s\n", filename)
}
