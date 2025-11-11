# PEAD Research Module

## Overview

The **PEAD (Post-Earnings Announcement Drift)** research module analyzes companies based on their quarterly earnings reports to identify high-potential trading candidates. PEAD is a well-documented market anomaly where stocks tend to drift in the direction of an earnings surprise for weeks or months following the announcement.

## Key Features

### 📊 Comprehensive Scoring System

The module scores companies on a 0-100 scale based on seven key components:

1. **Earnings Surprise** (25% weight) - How much actual EPS exceeded/missed expectations
2. **Revenue Surprise** (15% weight) - Actual revenue vs expected revenue
3. **Earnings Growth** (20% weight) - Year-over-year EPS growth rate
4. **Revenue Growth** (15% weight) - Year-over-year revenue growth rate
5. **Margin Expansion** (10% weight) - Changes in profit margins (gross, operating, net)
6. **Consistency** (10% weight) - Track record of consecutive earnings beats
7. **Revenue Acceleration** (5% weight) - Quarter-over-quarter revenue momentum

### 🎯 Configurable Filtering

Filter companies based on:
- Minimum composite score threshold (configurable via `.env`)
- Earnings announcement recency (PEAD time window)
- Minimum earnings surprise percentage
- Minimum revenue and EPS growth rates

### 📈 Rating System

- **STRONG_BUY** (80-100): Exceptional earnings quality and growth
- **BUY** (65-79): Strong earnings performance
- **HOLD** (45-64): Average earnings quality
- **AVOID** (<45): Weak earnings performance

## Architecture

```
internal/research/pead/
├── types.go          # Data structures for earnings data and scores
├── fetcher.go        # Interface and implementations for fetching earnings data
├── scorer.go         # Scoring algorithm implementation
├── analyzer.go       # Main analyzer with filtering logic
├── peadobs/          # Observability wrapper for logging and tracing
│   └── peadobs.go
└── README.md         # This file

cmd/pead/
└── main.go           # Standalone CLI tool for running PEAD analysis

internal/interfaces/
└── pead.go           # PEADAnalyzer interface definition
```

## Usage

### Standalone CLI Tool

Run PEAD analysis from the command line:

```bash
# Run analysis with default settings
go run cmd/pead/main.go

# Save results to JSON file
go run cmd/pead/main.go --json
```

### Configuration

#### 1. Edit `config.yaml`

```yaml
pead:
  enabled: true
  min_days_since_earnings: 1
  max_days_since_earnings: 60
  min_composite_score: 40

  weights:
    earnings_surprise: 0.25
    revenue_surprise: 0.15
    earnings_growth: 0.20
    revenue_growth: 0.15
    margin_expansion: 0.10
    consistency: 0.10
    revenue_acceleration: 0.05
```

#### 2. Set Environment Variables (`.env`)

```bash
# Minimum PEAD score to qualify (0-100)
PEAD_MIN_SCORE=40

# Optional: API key for real earnings data
EARNINGS_API_KEY=your_api_key_here
```

### Programmatic Usage

```go
package main

import (
    "context"
    "llm-trading-bot/internal/research/pead"
    "llm-trading-bot/internal/research/pead/peadobs"
)

func main() {
    // Create configuration
    config := pead.GetDefaultConfig()
    config.MinCompositeScore = 60 // High-quality companies only

    // Create fetcher (use Mock for testing)
    fetcher := pead.NewMockEarningsDataFetcher()

    // Create analyzer
    analyzer := pead.NewAnalyzer(config, fetcher)

    // Wrap with observability
    analyzer = peadobs.Wrap(analyzer)

    // Run analysis
    symbols := []string{"RELIANCE", "TCS", "INFY", "HDFCBANK"}
    result, err := analyzer.Analyze(context.Background(), symbols)
    if err != nil {
        panic(err)
    }

    // Process results
    for _, score := range result.QualifiedSymbols {
        fmt.Printf("%s: %.1f (%s)\n",
            score.Symbol,
            score.CompositeScore,
            score.Rating)
    }
}
```

## Scoring Algorithm Details

### Earnings Surprise Score (0-100)

```
Negative surprise:  0 points
0-5% surprise:      0-50 points (linear)
5%+ surprise:       50-100 points (logarithmic decay)
```

### Earnings Growth Score (0-100)

```
<0% growth:         0-40 points (penalty)
0-20% growth:       40-70 points (linear)
20-50% growth:      70-90 points (linear)
50%+ growth:        90-100 points (logarithmic)
```

### Margin Expansion Score (0-100)

Weighted average of gross, operating, and net margin changes:
- Base score: 50 (neutral)
- Each 1% margin expansion adds 10 points
- Each 1% margin contraction subtracts 10 points

### Consistency Score (0-100)

Based on consecutive quarterly earnings beats:
```
Current miss:       0 points
0 beats:            40 points
1 beat:             50 points
2 beats:            60 points
3 beats:            70 points
4 beats:            80 points
5+ beats:           80-100 points (logarithmic)
```

## Data Sources

### Mock Data (Default)

For testing and development, the module generates realistic mock earnings data with:
- Randomized earnings surprises (60% positive, 40% negative)
- Variable growth rates (-20% to 80% for EPS, -10% to 50% for revenue)
- Realistic profit margins and consecutive beat counts

### API Integration (TODO)

The module is designed to integrate with real earnings data APIs:
- **Alpha Vantage**: Earnings Calendar API
- **Financial Modeling Prep**: Earnings API
- **Yahoo Finance**: Unofficial yfinance library

To add API support:
1. Implement `FetchLatestEarnings()` in `APIEarningsDataFetcher`
2. Set `EARNINGS_API_KEY` in `.env`
3. Change `data_source: API` in `config.yaml`

## Integration with Trading Bot

The PEAD module can be integrated into the main trading bot workflow:

1. **Pre-market screening**: Run PEAD analysis before market open
2. **Universe filtering**: Use qualified symbols as trading candidates
3. **Signal enhancement**: Boost confidence for PEAD-qualified stocks
4. **Position sizing**: Allocate more capital to higher PEAD scores

Example integration in `cmd/bot/bootstrap.go`:

```go
// Run PEAD analysis to filter universe
if cfg.PEAD.Enabled {
    peadAnalyzer := initializePEADAnalyzer(cfg)
    topPicks, _ := peadAnalyzer.GetTopPicks(ctx, symbols, 10)

    // Use top PEAD picks as trading universe
    symbols = extractSymbols(topPicks)
}
```

## Research Background

### What is PEAD?

Post-Earnings Announcement Drift (PEAD) is a market anomaly where stocks that beat (miss) earnings expectations tend to continue drifting upward (downward) for weeks or months after the announcement. This contradicts the Efficient Market Hypothesis.

### Why Does PEAD Exist?

1. **Underreaction**: Investors initially underreact to earnings information
2. **Gradual Information Diffusion**: News spreads slowly through the market
3. **Anchoring Bias**: Investors anchor to previous prices
4. **Limited Attention**: Not all investors analyze earnings immediately

### Academic Research

- **Ball & Brown (1968)**: First documented PEAD
- **Bernard & Thomas (1989)**: Confirmed drift persists 60+ days
- **Chordia & Shivakumar (2006)**: PEAD stronger in less liquid stocks
- **Livnat & Mendenhall (2006)**: Revenue surprises also cause drift

### Typical PEAD Time Window

- **Days 0-3**: Initial price reaction (earnings announcement)
- **Days 4-60**: Drift period (PEAD effect strongest here)
- **Days 60+**: Effect diminishes

## Performance Tuning

### High Precision (Fewer, Higher Quality Picks)

```yaml
min_composite_score: 70
min_eps_growth: 20
min_earnings_surprise: 5
```

### High Recall (More Picks, Lower Quality)

```yaml
min_composite_score: 30
min_eps_growth: 0
min_earnings_surprise: 0
```

### Balanced (Recommended)

```yaml
min_composite_score: 40
min_eps_growth: 0
min_earnings_surprise: 0
```

## Limitations

1. **Mock Data**: Currently uses mock earnings data - integrate real API for production
2. **Historical Bias**: Past earnings beats don't guarantee future performance
3. **Market Conditions**: PEAD may be weaker in bear markets or high volatility periods
4. **Transaction Costs**: Frequent trading based on PEAD can incur significant costs

## Future Enhancements

- [ ] Real-time earnings data API integration
- [ ] Machine learning model for score weighting optimization
- [ ] Historical backtesting framework
- [ ] Sentiment analysis integration (earnings call transcripts)
- [ ] Options strategies for PEAD (long calls/puts based on direction)
- [ ] Sector-specific scoring adjustments
- [ ] Institutional ownership and analyst coverage metrics

## Contributing

To extend the PEAD module:

1. Implement new scoring components in `scorer.go`
2. Add new data sources in `fetcher.go`
3. Update scoring weights in `config.yaml`
4. Add tests in `pead_test.go`

## References

- Ball, R., & Brown, P. (1968). An empirical evaluation of accounting income numbers. *Journal of Accounting Research*, 6(2), 159-178.
- Bernard, V. L., & Thomas, J. K. (1989). Post-earnings-announcement drift: Delayed price response or risk premium? *Journal of Accounting Research*, 27, 1-36.
- Livnat, J., & Mendenhall, R. R. (2006). Comparing the post–earnings announcement drift for surprises calculated from analyst and time series forecasts. *Journal of Accounting Research*, 44(1), 177-205.
