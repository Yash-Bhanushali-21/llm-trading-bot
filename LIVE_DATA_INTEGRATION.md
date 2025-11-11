# Live Data Integration for Forensic Checker

This document describes the live data integration implementation for the forensic checker tool.

## Overview

The forensic checker now supports fetching real-time data from multiple Indian stock market sources:

- **NSE India API**: Corporate announcements, shareholding patterns
- **BSE India API**: Corporate announcements and actions
- **SEBI**: Insider trading data, regulatory actions and filings
- **Screener.in**: Financial data, shareholding patterns, peer comparisons

## Architecture

### Components

```
internal/forensic/datasource/
├── nse_client.go          # NSE India API client
├── bse_client.go          # BSE India API client
├── sebi_client.go         # SEBI data client
├── screener_client.go     # Screener.in client
├── cache.go               # File-based caching layer
├── rate_limiter.go        # Token bucket rate limiter
└── live_datasource.go     # Aggregated data source

internal/forensic/
├── datasource_factory.go  # Factory for creating data sources
└── mock_datasource.go     # Mock data for testing
```

### Data Flow

```
User Request
     ↓
[Forensic Checker]
     ↓
[Data Source Factory] → MOCK or LIVE?
     ↓
[Live Data Source]
     ↓
[Rate Limiter] → Enforces API rate limits
     ↓
[Cache Layer] → Checks cache first (24h TTL)
     ↓         ↓
  Cache Hit  Cache Miss
     ↓         ↓
   Return   [Multiple API Clients]
              ↓
         NSE | BSE | SEBI | Screener
              ↓
         Parse & Aggregate
              ↓
         Cache Result
              ↓
         Return to Checker
```

## Configuration

### config.yaml

```yaml
forensic:
  enabled: true
  data_source: LIVE              # MOCK or LIVE

  # Data source options (LIVE mode only)
  enable_nse: true               # NSE India API
  enable_bse: true               # BSE India API
  enable_sebi: true              # SEBI data
  enable_screener: true          # Screener.in

  # Caching
  cache_dir: cache/forensic
  cache_ttl_hours: 24            # Cache validity period

  # Other settings...
```

### Switching Between MOCK and LIVE

```yaml
# For testing with mock data
data_source: MOCK

# For production with real APIs
data_source: LIVE
```

## Features

### 1. Intelligent Caching

- **File-based cache**: Stores API responses locally
- **Configurable TTL**: Default 24 hours, adjustable
- **MD5 hashing**: Cache keys hashed for filesystem safety
- **Automatic cleanup**: Removes expired entries

```go
// Cache automatically used by live data source
cache := NewCache("cache/forensic", 24 * time.Hour)
```

### 2. Rate Limiting

- **Token bucket algorithm**: Smooth rate limiting
- **Per-source limits**: Different limits for each API
  - NSE: 10 requests/second
  - BSE: 5 requests/second
  - SEBI: 3 requests/second
  - Screener: 5 requests/second

```go
rateLimiter := NewMultiRateLimiter()
rateLimiter.AddLimiter("NSE", 10, 1*time.Second)
```

### 3. Multi-Source Aggregation

- **Primary + Fallback**: Try NSE first, fallback to others
- **Graceful degradation**: Continue if one source fails
- **Logging**: Track which sources succeeded/failed

### 4. Error Handling

- Non-blocking errors: One source failure doesn't stop analysis
- Detailed logging: Track API failures for debugging
- Retry logic: Built-in timeout handling

## API Clients

### NSE Client

```go
client := NewNSEClient()

// Fetch announcements
announcements, err := client.FetchAnnouncements(ctx, "RELIANCE", "2024-01-01", "2024-12-31")

// Fetch shareholding pattern
pattern, err := client.FetchShareholdingPattern(ctx, "RELIANCE")

// Search for symbol
symbols, err := client.SearchSymbol(ctx, "RELIANCE")
```

**Key Features**:
- Session management (NSE requires cookies)
- Corporate announcements
- Shareholding patterns
- Symbol search

### BSE Client

```go
client := NewBSEClient()

// Fetch announcements (requires scrip code)
announcements, err := client.FetchAnnouncements(ctx, "500325", "2024-01-01", "2024-12-31")

// Fetch corporate actions
actions, err := client.FetchCorporateActions(ctx, "500325")
```

**Key Features**:
- Scrip code mapping
- Corporate announcements
- Corporate actions (dividends, splits, etc.)

### SEBI Client

```go
client := NewSEBIClient()

// Fetch insider trading data
trades, err := client.FetchInsiderTrading(ctx, "RELIANCE", "2024-01-01", "2024-12-31")

// Fetch regulatory actions
filings, err := client.FetchRegulatoryActions(ctx, "RELIANCE")

// Fetch annual reports
reports, err := client.FetchAnnualReports(ctx, "RELIANCE")
```

**Key Features**:
- Insider trading transactions
- Regulatory orders and penalties
- Annual report filings
- Multiple date format parsing

### Screener Client

```go
client := NewScreenerClient()

// Fetch comprehensive company data
data, err := client.FetchCompanyData(ctx, "reliance")

// Fetch shareholding pattern
pattern, err := client.FetchShareholdingPattern(ctx, "reliance")

// Fetch financials
financials, err := client.FetchFinancials(ctx, "reliance", "Q3FY24")

// Search companies
results, err := client.SearchCompany(ctx, "reliance")
```

**Key Features**:
- Aggregated financial data
- Shareholding patterns
- Peer comparisons
- Company search

## Usage

### Basic Usage

```bash
# Use MOCK data (default)
./forensic -symbol RELIANCE

# Use LIVE data (set in config.yaml: data_source: LIVE)
./forensic -symbol RELIANCE
```

### Programmatic Usage

```go
import (
    "llm-trading-bot/internal/forensic"
    "llm-trading-bot/internal/store"
)

// Load config
cfg, _ := store.LoadConfig("config.yaml")

// Create data source (automatically chooses MOCK or LIVE)
dataSource, _ := forensic.CreateDataSource(cfg)

// Create checker
forensicCfg := &types.ForensicConfig{...}
checker := forensic.NewChecker(forensicCfg, dataSource)

// Run analysis
report, _ := checker.Analyze(ctx, "RELIANCE")
```

### Direct API Client Usage

```go
import "llm-trading-bot/internal/forensic/datasource"

// Create NSE client
nse := datasource.NewNSEClient()

// Fetch data
announcements, err := nse.FetchAnnouncements(ctx, "RELIANCE", "2024-01-01", "2024-12-31")
```

## Cache Management

### Manual Cache Operations

```go
// Get cache instance
cache := datasource.NewCache("cache/forensic", 24*time.Hour)

// Get cached item
if data, ok := cache.Get("key"); ok {
    // Use cached data
}

// Set cache item
cache.Set("key", data)

// Delete specific item
cache.Delete("key")

// Clear all cache
cache.Clear()

// Cleanup expired entries only
cache.CleanupExpired()
```

### Cache Keys

Cache keys are automatically generated based on:
- API endpoint
- Symbol
- Date range
- Other parameters

Example: `announcements:RELIANCE:2024-01-01:2024-12-31`

## Rate Limiting

### Configuration

```go
limiter := datasource.NewRateLimiter(
    10,                    // Max 10 tokens
    100 * time.Millisecond // Refill every 100ms (10 req/sec)
)

// Wait for token
limiter.Wait(ctx)

// Make API call
response, err := apiCall()
```

### Multi-Source Limiting

```go
multiLimiter := datasource.NewMultiRateLimiter()
multiLimiter.AddLimiter("NSE", 10, 1*time.Second)
multiLimiter.AddLimiter("SEBI", 3, 1*time.Second)

// Wait for specific source
multiLimiter.Wait(ctx, "NSE")
```

## Error Handling

### Graceful Degradation

The system handles errors gracefully:

1. **Cache failures**: Fetch fresh data
2. **API failures**: Try fallback sources
3. **Rate limit exceeded**: Wait and retry
4. **Network errors**: Log and continue

### Logging

All operations are logged:

```
{"level":"info","msg":"Fetching announcements","symbol":"RELIANCE"}
{"level":"warn","msg":"Failed to fetch from NSE","error":"timeout"}
{"level":"info","msg":"Fetched from BSE","count":15}
```

## Symbol Normalization

### NSE Format

```go
symbol := datasource.NormalizeSymbol("RELIANCE.NS")
// Returns: "RELIANCE"
```

### BSE Scrip Codes

```go
scrip := datasource.SymbolToScripCode("RELIANCE")
// Returns: "500325"
```

## Best Practices

### 1. Use Caching Effectively

```go
// Good: Let the system handle caching
dataSource := datasource.NewLiveDataSource(config)
data, _ := dataSource.FetchAnnouncements(ctx, symbol, from, to)

// Bad: Bypass cache unnecessarily
cache.Delete(key)  // Don't delete cache unless necessary
```

### 2. Handle Rate Limits

```go
// Good: Use built-in rate limiting
err := rateLimiter.Wait(ctx)
if err != nil {
    return err // Context cancelled or deadline exceeded
}

// Bad: Make rapid successive calls without rate limiting
for i := 0; i < 100; i++ {
    client.FetchData()  // Will hit rate limits
}
```

### 3. Enable Multiple Sources

```yaml
# Good: Enable all sources for redundancy
enable_nse: true
enable_bse: true
enable_sebi: true
enable_screener: true

# Acceptable: Disable if not needed
enable_bse: false  # If only NSE data is sufficient
```

### 4. Configure Appropriate Cache TTL

```yaml
# For frequently changing data
cache_ttl_hours: 1

# For daily updates
cache_ttl_hours: 24

# For static/historical data
cache_ttl_hours: 168  # 1 week
```

## Troubleshooting

### Issue: API Returns 403/401

**Cause**: Missing authentication or rate limiting

**Solution**:
1. Check if API requires authentication keys
2. Verify rate limits aren't exceeded
3. Check if User-Agent headers are set

### Issue: Cache Growing Too Large

**Solution**:
```bash
# Manual cleanup
rm -rf cache/forensic/*

# Or programmatically
dataSource.CleanupExpiredCache()
```

### Issue: Slow Performance

**Causes**:
- Cache misses
- Multiple API calls
- Network latency

**Solutions**:
1. Increase cache TTL
2. Reduce lookback days
3. Disable unnecessary sources

### Issue: No Data Returned

**Debugging steps**:
```bash
# Check logs for errors
./forensic -symbol RELIANCE 2>&1 | grep "error\|warn"

# Test with MOCK data first
# Set data_source: MOCK in config.yaml
./forensic -symbol RELIANCE

# Enable all data sources
enable_nse: true
enable_bse: true
enable_sebi: true
enable_screener: true
```

## Limitations

### Current Limitations

1. **No API Keys**: Implementation doesn't include API key support (add if needed)
2. **Symbol Mapping**: Limited hardcoded BSE scrip code mappings
3. **Date Format Parsing**: May not cover all possible formats
4. **HTML Parsing**: Screener client uses regex, may break with UI changes

### Future Enhancements

- [ ] Add API key support for authenticated endpoints
- [ ] Implement symbol-to-scrip code database
- [ ] Add more robust HTML/JSON parsing
- [ ] Implement exponential backoff for retries
- [ ] Add Prometheus metrics for monitoring
- [ ] Support for more data sources (MoneyControl, Trendlyne)
- [ ] Real-time WebSocket data streaming
- [ ] Bulk symbol processing
- [ ] Advanced query filtering

## Production Deployment

### Checklist

- [ ] Set `data_source: LIVE` in config.yaml
- [ ] Configure cache directory with appropriate permissions
- [ ] Set reasonable cache TTL (24 hours recommended)
- [ ] Enable required data sources
- [ ] Monitor API rate limits
- [ ] Set up log aggregation
- [ ] Configure disk space monitoring (for cache)
- [ ] Test with actual symbols
- [ ] Document any custom API keys/credentials
- [ ] Set up periodic cache cleanup cron job

### Monitoring

Monitor these metrics:
- API success/failure rates
- Cache hit/miss ratios
- Response times
- Rate limit violations
- Cache size

### Security

- Store API keys in environment variables (if needed)
- Don't commit cache files to git
- Use HTTPS for all API calls
- Sanitize symbol inputs
- Validate date ranges

## Support

For issues or questions:
1. Check logs for detailed error messages
2. Test with MOCK data first
3. Verify network connectivity to APIs
4. Check API status pages (NSE, BSE, SEBI)
5. Create an issue in the repository

## License

See main project LICENSE file.
