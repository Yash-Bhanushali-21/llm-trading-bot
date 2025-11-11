# Forensic Checker Tool

A comprehensive corporate governance and forensic analysis tool for identifying red flags in Indian stocks.

## Overview

The Forensic Checker analyzes multiple aspects of corporate governance and identifies potential red flags that could indicate financial distress, fraud, or poor management. It provides a risk score (0-100) and detailed reports on various governance metrics.

## Features

The tool monitors and analyzes:

### 1. **Management Changes** 🏢
- Tracks resignations, appointments, and removals of key personnel
- Identifies abrupt changes (e.g., immediate resignations)
- Focuses on critical positions: CEO, CFO, MD, Directors
- Flags sudden departures without succession planning

### 2. **Auditor Changes** 📊
- Detects changes in statutory auditors
- Identifies qualified opinions and adverse reports
- Flags mid-term auditor changes (before tenure completion)
- Monitors audit qualifications and disclaimers

### 3. **Related Party Transactions** 🤝
- Analyzes transactions with promoters, subsidiaries, and associates
- Identifies non-arm's length transactions
- Flags transactions exceeding materiality thresholds
- Monitors loans, guarantees, and other high-risk transactions

### 4. **Promoter Pledge** 💰
- Tracks pledging of promoter shares
- Monitors pledge percentage increases
- Alerts on high pledge levels (configurable threshold)
- Detects invocation of pledged shares

### 5. **Regulatory Actions** ⚖️
- Monitors SEBI, NSE, BSE, and other regulatory actions
- Tracks penalties, suspensions, and investigations
- Records show-cause notices and warnings
- Analyzes violation patterns

### 6. **Insider Trading** 📈
- Analyzes insider buying and selling patterns
- Detects clustered trades (multiple insiders trading together)
- Identifies unusual timing or volumes
- Focuses on key management personnel transactions

### 7. **Financial Restatements** 📋
- Detects restatements of financial results
- Analyzes impact on revenue, profits, and assets
- Identifies material restatements
- Flags accounting errors and corrections

### 8. **Governance Scores** ⭐
- Monitors governance ratings from agencies
- Tracks score degradations over time
- Compares ratings across providers
- Identifies declining governance standards

## Installation

### Prerequisites
- Go 1.21 or higher
- Access to corporate data sources (API keys/credentials)

### Build

```bash
# Build the forensic checker CLI
go build -o forensic ./cmd/forensic/

# Or build with the main bot
go build -o llm-trading-bot ./cmd/bot/
```

## Configuration

Add forensic configuration to your `config.yaml`:

```yaml
forensic:
  enabled: true                      # enable/disable forensic analysis
  lookback_days: 365                 # how far back to analyze (days)
  min_risk_score: 40.0               # minimum score to trigger alert (0-100)

  # Enable/disable specific checks
  check_management: true             # detect management changes & resignations
  check_auditor: true                # detect auditor changes or qualifications
  check_related_party: true          # analyze related party transactions
  check_promoter_pledge: true        # track pledge of promoter shares
  check_regulatory: true             # monitor regulatory actions & penalties
  check_insider_trading: true        # analyze insider trading patterns
  check_restatements: true           # detect financial restatements
  check_governance: true             # monitor governance score degradation

  promoter_pledge_threshold: 50.0    # alert if pledge exceeds this % (0-100)
  output_dir: logs/forensic          # where to save reports
```

## Usage

### Standalone CLI

Run forensic analysis on a specific stock:

```bash
# Basic usage
./forensic -symbol RELIANCE

# Specify output format (text, json, csv)
./forensic -symbol RELIANCE -format json

# Save to specific file
./forensic -symbol RELIANCE -output reliance_report.txt

# Use custom config
./forensic -config my_config.yaml -symbol TCS
```

### Command-line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-symbol` | Stock symbol to analyze (required) | - |
| `-format` | Output format: text, json, or csv | text |
| `-output` | Save report to file (optional) | auto-generated |
| `-config` | Path to config file | config.yaml |

### Exit Codes

- `0`: Analysis complete, risk below threshold
- `1`: Error occurred
- `2`: Risk score exceeds configured threshold

## Output Formats

### Text Format
Human-readable report with sections for each check type:
- Summary with overall risk score
- Sorted list of red flags by severity
- Detailed findings for each category
- Visual indicators (⚠️) for critical issues

### JSON Format
Structured JSON output suitable for:
- Integration with other tools
- Programmatic analysis
- Data pipelines
- API responses

### CSV Format
Tabular format for:
- Spreadsheet analysis
- Data exports
- Quick summaries

## Risk Scoring

### Overall Risk Score (0-100)
- **0-39**: 🟢 Low Risk - Good governance
- **40-59**: 🟡 Medium Risk - Some concerns
- **60-74**: 🟠 High Risk - Significant red flags
- **75-100**: 🔴 Critical Risk - Serious issues

### Severity Levels
Each red flag is assigned a severity:
- **LOW**: Minor issue, monitor
- **MEDIUM**: Notable concern, investigate
- **HIGH**: Significant red flag, caution advised
- **CRITICAL**: Severe issue, avoid or exit

### Risk Calculation
The overall risk score is calculated by:
1. Computing individual risk scores for each finding
2. Applying weights based on category importance
3. Aggregating scores across all categories
4. Normalizing to 0-100 scale

Categories with higher weights:
- Auditor changes (1.5x)
- Regulatory actions (1.8x)
- Financial restatements (1.5x)

## Integration with Trading Bot

The forensic checker can be integrated into the main trading bot to:
- Screen stocks before trading
- Monitor existing positions
- Trigger exit signals on high risk
- Generate periodic governance reports

### Example Integration

```go
// In engine or decision flow
checker := forensic.NewChecker(cfg, dataSource, log)
report, err := checker.Analyze(ctx, symbol)

if report.OverallRiskScore >= cfg.Forensic.MinRiskScore {
    // High risk detected - avoid trading or exit position
    log.Warn("High forensic risk detected",
        "symbol", symbol,
        "risk_score", report.OverallRiskScore)
    return Decision{Action: "HOLD"}, nil
}
```

## Data Sources

### Current Implementation
The tool currently uses a mock data source for demonstration. To use in production:

1. Implement the `CorporateDataSource` interface:
   ```go
   type CorporateDataSource interface {
       FetchAnnouncements(ctx, symbol, from, to string) ([]Announcement, error)
       FetchShareholdingPattern(ctx, symbol string) (*ShareholdingPattern, error)
       FetchInsiderTrades(ctx, symbol, from, to string) ([]InsiderTradeData, error)
       FetchFinancials(ctx, symbol, period string) (*FinancialData, error)
       FetchRegulatoryFilings(ctx, symbol, from, to string) ([]RegulatoryFiling, error)
   }
   ```

2. Integrate with data providers:
   - NSE/BSE announcements
   - SEBI filings
   - Shareholding pattern data
   - Insider trading databases
   - Rating agencies

### Recommended Data Providers
- **NSE/BSE APIs**: Corporate announcements, shareholding patterns
- **SEBI APIs**: Regulatory filings, insider trading data
- **Trendlyne/Screener**: Aggregated financial and governance data
- **Bloomberg/Reuters**: Premium data feeds
- **Custom scrapers**: For public disclosures

## Examples

### Example 1: High Risk Detection

```bash
$ ./forensic -symbol EXAMPLE

🔍 Starting Forensic Analysis for EXAMPLE
─────────────────────────────────────────────────────────────────────────────
================================================================================
FORENSIC ANALYSIS REPORT - EXAMPLE
================================================================================
Generated: 2024-01-15 10:30:45
Overall Risk Score: 78.50/100

Risk Level: CRITICAL

RED FLAGS DETECTED: 8
────────────────────────────────────────────────────────────────────────────

1. [CRITICAL] AUDITOR
   Auditor changed from ABC & Co to XYZ & Associates on 2024-01-01
   Impact: 85.00/100

2. [HIGH] REGULATORY
   PENALTY action by SEBI on 2023-12-15: Penalty for delayed disclosure
   Impact: 75.00/100

[... more red flags ...]

⚠️  Risk score exceeds threshold (40.00). Review the red flags carefully.
```

### Example 2: Clean Company

```bash
$ ./forensic -symbol GOODSTOCK

Overall Risk Score: 15.20/100
Risk Level: 🟢 LOW
Red Flags Detected: 0

No significant red flags detected.
```

## Architecture

```
internal/forensic/
├── forensic.go              # Main checker implementation
├── management_detector.go   # Management change detection
├── auditor_detector.go      # Auditor change detection
├── related_party_detector.go# Related party transaction analysis
├── pledge_detector.go       # Promoter pledge tracking
├── regulatory_detector.go   # Regulatory action monitoring
├── insider_trading_detector.go # Insider trading analysis
├── restatement_detector.go  # Financial restatement detection
├── governance_detector.go   # Governance score tracking
├── reporter.go              # Report generation
├── mock_datasource.go       # Mock data for testing
└── utils.go                 # Utility functions

internal/interfaces/
└── forensic.go              # Interface definitions

internal/types/
└── forensic_types.go        # Type definitions

cmd/forensic/
└── main.go                  # CLI entry point
```

## Testing

Run with mock data source to test the tool:

```bash
# Test with various scenarios
./forensic -symbol TEST1  # Clean company
./forensic -symbol TEST2  # Medium risk
./forensic -symbol TEST3  # High risk
```

## Limitations

1. **Data Quality**: Analysis quality depends on data source accuracy
2. **Timeliness**: Detection depends on announcement/filing delays
3. **False Positives**: Some legitimate events may trigger flags
4. **Coverage**: Limited to available public disclosures
5. **Mock Data**: Current implementation uses mock data for demonstration

## Future Enhancements

- [ ] Real-time data source integration
- [ ] Machine learning for pattern detection
- [ ] Historical trend analysis
- [ ] Peer comparison and benchmarking
- [ ] Alert notifications (email, Slack, etc.)
- [ ] Web dashboard for visualization
- [ ] Bulk analysis for multiple stocks
- [ ] Scheduled periodic scans
- [ ] Custom rule engine for alerts
- [ ] Integration with more data providers

## Contributing

To add a new check:

1. Create detector file in `internal/forensic/`
2. Implement check method following existing patterns
3. Add types to `forensic_types.go`
4. Update config structs
5. Add red flag generation logic
6. Update documentation

## License

See main project LICENSE file.

## Support

For issues or questions:
- Check existing issues in the repository
- Create a new issue with detailed description
- Include sample data (anonymized)
- Provide configuration used

## Disclaimer

This tool is for informational purposes only. It does not constitute financial advice. Always perform your own due diligence and consult with financial professionals before making investment decisions.
