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

	// Create fetcher based on config
	var fetcher pead.EarningsDataFetcher
	if peadConfig.DataSource == "MOCK" {
		fmt.Println("📊 Using MOCK earnings data for testing")
		fetcher = pead.NewMockEarningsDataFetcher()
	} else {
		fmt.Println("📊 Fetching LIVE earnings data from Yahoo Finance")
		fmt.Println("⏳ This may take a few moments...")
		fetcher = pead.NewYahooFinanceEarningsDataFetcher()
	}

	// Create analyzer
	analyzer := pead.NewAnalyzer(peadConfig, fetcher)

	// Get symbols from config (use universe_dynamic candidate_list)
	symbols := cfg.Universe.Dynamic.CandidateList
	if len(symbols) == 0 {
		// Fallback to static universe
		symbols = cfg.Universe.Static
	}

	if len(symbols) == 0 {
		fmt.Println("⚠️  No symbols configured for analysis")
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
	fmt.Printf("Qualified:          %d companies (%.1f%%)\n",
		result.QualifiedCount,
		float64(result.QualifiedCount)/float64(result.TotalAnalyzed)*100)
	fmt.Printf("Min Score Filter:   %.1f\n", result.Config.MinCompositeScore)
	fmt.Println()

	if result.QualifiedCount == 0 {
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
		printCompanyScore(i+1, &score)
		fmt.Println()
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("💡 Next Steps:")
	fmt.Println("  1. Review qualified companies above")
	fmt.Println("  2. Add top picks to universe_static in config.yaml")
	fmt.Println("  3. Run the trading bot to analyze these symbols")
	fmt.Println("  4. Monitor PEAD drift over the next 30-60 days")
	fmt.Println()
}

func printCompanyScore(rank int, score *pead.PEADScore) {
	data := &score.EarningsData

	// Rating emoji
	emoji := "📊"
	switch score.Rating {
	case "STRONG_BUY":
		emoji = "🔥"
	case "BUY":
		emoji = "✅"
	case "HOLD":
		emoji = "⚠️"
	case "AVOID":
		emoji = "❌"
	}

	fmt.Printf("%s Rank #%d: %s (%.1f/100 - %s)\n",
		emoji, rank, score.Symbol, score.CompositeScore, score.Rating)
	fmt.Println("─────────────────────────────────────────────────────────────")

	// Earnings announcement details
	fmt.Printf("  📅 Quarter:           %s (announced %d days ago)\n",
		data.Quarter, score.DaysSinceEarnings)

	// Surprises
	fmt.Printf("  💰 EPS Surprise:      %.2f%% (Actual: %.2f vs Expected: %.2f)\n",
		data.EarningSurprise(), data.ActualEPS, data.ExpectedEPS)
	fmt.Printf("  💵 Revenue Surprise:  %.2f%%\n", data.RevenueSurprise())

	// Growth metrics
	fmt.Printf("  📈 YoY EPS Growth:    %.1f%%\n", data.YoYEPSGrowth)
	fmt.Printf("  📈 YoY Revenue Growth: %.1f%%\n", data.YoYRevenueGrowth)

	// Margins
	if data.NetMarginChange() > 0 {
		fmt.Printf("  💹 Net Margin:        %.1f%% (↑ %.1f%%)\n",
			data.NetMargin, data.NetMarginChange())
	} else if data.NetMarginChange() < 0 {
		fmt.Printf("  💹 Net Margin:        %.1f%% (↓ %.1f%%)\n",
			data.NetMargin, abs(data.NetMarginChange()))
	} else {
		fmt.Printf("  💹 Net Margin:        %.1f%% (unchanged)\n", data.NetMargin)
	}

	// Consistency
	if data.ConsecutiveBeats > 0 {
		fmt.Printf("  🎯 Consistency:       %d consecutive beats\n", data.ConsecutiveBeats)
	}

	// Component scores
	fmt.Println()
	fmt.Println("  Component Scores:")
	fmt.Printf("    • Earnings Surprise:    %.1f/100\n", score.EarningsSurpriseScore)
	fmt.Printf("    • Earnings Growth:      %.1f/100\n", score.EarningsGrowthScore)
	fmt.Printf("    • Revenue Growth:       %.1f/100\n", score.RevenueGrowthScore)
	fmt.Printf("    • Margin Expansion:     %.1f/100\n", score.MarginExpansionScore)
	fmt.Printf("    • Consistency:          %.1f/100\n", score.ConsistencyScore)

	// Commentary
	fmt.Println()
	fmt.Printf("  📝 %s\n", score.Commentary)
}

func saveResultsJSON(result *pead.PEADResult, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create JSON file: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write JSON: %v\n", err)
		return
	}

	fmt.Printf("💾 Results saved to %s\n", filename)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
