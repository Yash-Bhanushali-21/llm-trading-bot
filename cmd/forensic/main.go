package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"llm-trading-bot/internal/forensic"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/types"
)

func main() {
	// Command-line flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	symbol := flag.String("symbol", "", "stock symbol to analyze (required)")
	format := flag.String("format", "text", "output format: text, json, or csv")
	outputFile := flag.String("output", "", "save report to file (optional)")
	flag.Parse()

	if *symbol == "" {
		fmt.Println("Error: -symbol is required")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := store.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Check if forensic analysis is enabled
	if !cfg.Forensic.Enabled {
		fmt.Println("Forensic analysis is disabled in config. Set 'forensic.enabled: true' to enable.")
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(); err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔍 Starting Forensic Analysis for %s\n", *symbol)
	fmt.Println("─────────────────────────────────────────────────────────────────────────────")

	// Create forensic config from main config
	forensicCfg := &types.ForensicConfig{
		Enabled:                 cfg.Forensic.Enabled,
		LookbackDays:            cfg.Forensic.LookbackDays,
		MinRiskScore:            cfg.Forensic.MinRiskScore,
		CheckManagement:         cfg.Forensic.CheckManagement,
		CheckAuditor:            cfg.Forensic.CheckAuditor,
		CheckRelatedParty:       cfg.Forensic.CheckRelatedParty,
		CheckPromoterPledge:     cfg.Forensic.CheckPromoterPledge,
		CheckRegulatory:         cfg.Forensic.CheckRegulatory,
		CheckInsiderTrading:     cfg.Forensic.CheckInsiderTrading,
		CheckRestatements:       cfg.Forensic.CheckRestatements,
		CheckGovernance:         cfg.Forensic.CheckGovernance,
		PromoterPledgeThreshold: cfg.Forensic.PromoterPledgeThreshold,
	}

	// Initialize data source based on configuration
	dataSource, err := forensic.CreateDataSource(cfg)
	if err != nil {
		fmt.Printf("Error creating data source: %v\n", err)
		os.Exit(1)
	}

	// Create forensic checker
	checker := forensic.NewChecker(forensicCfg, dataSource)

	// Run analysis
	ctx := context.Background()
	report, err := checker.Analyze(ctx, *symbol)
	if err != nil {
		fmt.Printf("Error running analysis: %v\n", err)
		os.Exit(1)
	}

	// Create reporter
	outputDir := cfg.Forensic.OutputDir
	if outputDir == "" {
		outputDir = "logs/forensic"
	}
	reporter := forensic.NewReporter(outputDir)

	// Generate report in specified format
	var reportFormat forensic.ReportFormat
	switch *format {
	case "json":
		reportFormat = forensic.FormatJSON
	case "csv":
		reportFormat = forensic.FormatCSV
	case "text":
		reportFormat = forensic.FormatText
	default:
		fmt.Printf("Unknown format: %s. Using text format.\n", *format)
		reportFormat = forensic.FormatText
	}

	reportContent, err := reporter.GenerateReport(report, reportFormat)
	if err != nil {
		fmt.Printf("Error generating report: %v\n", err)
		os.Exit(1)
	}

	// Output to console
	fmt.Println(reportContent)

	// Save to file if requested
	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, []byte(reportContent), 0644); err != nil {
			fmt.Printf("Error saving report to file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n✅ Report saved to: %s\n", *outputFile)
	} else {
		// Auto-save to default location
		savedPath, err := reporter.SaveReport(report, reportFormat)
		if err != nil {
			fmt.Printf("Warning: Could not auto-save report: %v\n", err)
		} else {
			fmt.Printf("\n✅ Report auto-saved to: %s\n", savedPath)
		}
	}

	// Summary
	fmt.Println("\n─────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("Analysis complete for %s\n", *symbol)
	fmt.Printf("Overall Risk Score: %.2f/100\n", report.OverallRiskScore)
	fmt.Printf("Red Flags Detected: %d\n", len(report.RedFlags))

	riskLevel := "LOW"
	if report.OverallRiskScore >= 75 {
		riskLevel = "🔴 CRITICAL"
	} else if report.OverallRiskScore >= 60 {
		riskLevel = "🟠 HIGH"
	} else if report.OverallRiskScore >= 40 {
		riskLevel = "🟡 MEDIUM"
	} else {
		riskLevel = "🟢 LOW"
	}
	fmt.Printf("Risk Level: %s\n", riskLevel)

	// Exit with appropriate code
	if report.OverallRiskScore >= cfg.Forensic.MinRiskScore {
		fmt.Printf("\n⚠️  Risk score exceeds threshold (%.2f). Review the red flags carefully.\n", cfg.Forensic.MinRiskScore)
		os.Exit(2) // Exit code 2 indicates high risk
	}

	os.Exit(0)
}
