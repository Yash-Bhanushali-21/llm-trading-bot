package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/logger"
)

// LiveDataSource implements CorporateDataSource using real APIs
type LiveDataSource struct {
	nseClient      *NSEClient
	bseClient      *BSEClient
	sebiClient     *SEBIClient
	screenerClient *ScreenerClient
	cache          *Cache
	rateLimiter    *MultiRateLimiter
	config         *LiveDataSourceConfig
}

// LiveDataSourceConfig holds configuration for live data source
type LiveDataSourceConfig struct {
	EnableNSE      bool
	EnableBSE      bool
	EnableSEBI     bool
	EnableScreener bool
	CacheDir       string
	CacheTTL       time.Duration
}

// NewLiveDataSource creates a new live data source with all clients
func NewLiveDataSource(config *LiveDataSourceConfig) *LiveDataSource {
	if config == nil {
		config = &LiveDataSourceConfig{
			EnableNSE:      true,
			EnableBSE:      true,
			EnableSEBI:     true,
			EnableScreener: true,
			CacheDir:       "cache/forensic",
			CacheTTL:       24 * time.Hour,
		}
	}

	// Initialize rate limiters for each source
	rateLimiter := NewMultiRateLimiter()
	rateLimiter.AddLimiter("NSE", 10, 1*time.Second)     // 10 requests per second
	rateLimiter.AddLimiter("BSE", 5, 1*time.Second)      // 5 requests per second
	rateLimiter.AddLimiter("SEBI", 3, 1*time.Second)     // 3 requests per second
	rateLimiter.AddLimiter("SCREENER", 5, 1*time.Second) // 5 requests per second

	return &LiveDataSource{
		nseClient:      NewNSEClient(),
		bseClient:      NewBSEClient(),
		sebiClient:     NewSEBIClient(),
		screenerClient: NewScreenerClient(),
		cache:          NewCache(config.CacheDir, config.CacheTTL),
		rateLimiter:    rateLimiter,
		config:         config,
	}
}

// FetchAnnouncements retrieves corporate announcements from multiple sources
func (lds *LiveDataSource) FetchAnnouncements(ctx context.Context, symbol string, fromDate, toDate string) ([]interfaces.Announcement, error) {
	logger.Info(ctx, "Fetching announcements", "symbol", symbol, "from", fromDate, "to", toDate)

	cacheKey := fmt.Sprintf("announcements:%s:%s:%s", symbol, fromDate, toDate)

	// Try cache first
	if cached, ok := lds.cache.Get(cacheKey); ok {
		var announcements []interfaces.Announcement
		if err := json.Unmarshal(cached, &announcements); err == nil {
			logger.Info(ctx, "Returning cached announcements", "count", len(announcements))
			return announcements, nil
		}
	}

	announcements := []interfaces.Announcement{}

	// Fetch from NSE
	if lds.config.EnableNSE {
		if err := lds.rateLimiter.Wait(ctx, "NSE"); err != nil {
			logger.Warn(ctx, "Rate limit wait cancelled for NSE", "error", err)
		} else {
			nseAnn, err := lds.nseClient.FetchAnnouncements(ctx, NormalizeSymbol(symbol), fromDate, toDate)
			if err != nil {
				logger.Warn(ctx, "Failed to fetch NSE announcements", "error", err)
			} else {
				announcements = append(announcements, nseAnn...)
				logger.Info(ctx, "Fetched NSE announcements", "count", len(nseAnn))
			}
		}
	}

	// Fetch from BSE (if enabled)
	if lds.config.EnableBSE {
		if err := lds.rateLimiter.Wait(ctx, "BSE"); err != nil {
			logger.Warn(ctx, "Rate limit wait cancelled for BSE", "error", err)
		} else {
			scripCode := SymbolToScripCode(symbol)
			bseAnn, err := lds.bseClient.FetchAnnouncements(ctx, scripCode, fromDate, toDate)
			if err != nil {
				logger.Warn(ctx, "Failed to fetch BSE announcements", "error", err)
			} else {
				announcements = append(announcements, bseAnn...)
				logger.Info(ctx, "Fetched BSE announcements", "count", len(bseAnn))
			}
		}
	}

	// Cache the results
	if data, err := json.Marshal(announcements); err == nil {
		lds.cache.Set(cacheKey, data)
	}

	logger.Info(ctx, "Total announcements fetched", "count", len(announcements))
	return announcements, nil
}

// FetchShareholdingPattern retrieves shareholding pattern
func (lds *LiveDataSource) FetchShareholdingPattern(ctx context.Context, symbol string) (*interfaces.ShareholdingPattern, error) {
	logger.Info(ctx, "Fetching shareholding pattern", "symbol", symbol)

	cacheKey := fmt.Sprintf("shareholding:%s", symbol)

	// Try cache first (24-hour TTL for shareholding data)
	if cached, ok := lds.cache.Get(cacheKey); ok {
		var pattern interfaces.ShareholdingPattern
		if err := json.Unmarshal(cached, &pattern); err == nil {
			logger.Info(ctx, "Returning cached shareholding pattern")
			return &pattern, nil
		}
	}

	var pattern *interfaces.ShareholdingPattern
	var err error

	// Try NSE first
	if lds.config.EnableNSE {
		if waitErr := lds.rateLimiter.Wait(ctx, "NSE"); waitErr == nil {
			pattern, err = lds.nseClient.FetchShareholdingPattern(ctx, NormalizeSymbol(symbol))
			if err != nil {
				logger.Warn(ctx, "Failed to fetch from NSE", "error", err)
			} else {
				logger.Info(ctx, "Fetched shareholding from NSE")
			}
		}
	}

	// Fallback to Screener if NSE failed
	if pattern == nil && lds.config.EnableScreener {
		if waitErr := lds.rateLimiter.Wait(ctx, "SCREENER"); waitErr == nil {
			pattern, err = lds.screenerClient.FetchShareholdingPattern(ctx, symbol)
			if err != nil {
				logger.Warn(ctx, "Failed to fetch from Screener", "error", err)
			} else {
				logger.Info(ctx, "Fetched shareholding from Screener")
			}
		}
	}

	if pattern == nil {
		return nil, fmt.Errorf("failed to fetch shareholding pattern from all sources")
	}

	// Cache the result
	if data, err := json.Marshal(pattern); err == nil {
		lds.cache.Set(cacheKey, data)
	}

	return pattern, nil
}

// FetchInsiderTrades retrieves insider trading data from SEBI
func (lds *LiveDataSource) FetchInsiderTrades(ctx context.Context, symbol string, fromDate, toDate string) ([]interfaces.InsiderTradeData, error) {
	logger.Info(ctx, "Fetching insider trades", "symbol", symbol)

	if !lds.config.EnableSEBI {
		return []interfaces.InsiderTradeData{}, nil
	}

	cacheKey := fmt.Sprintf("insider:%s:%s:%s", symbol, fromDate, toDate)

	// Try cache first
	if cached, ok := lds.cache.Get(cacheKey); ok {
		var trades []interfaces.InsiderTradeData
		if err := json.Unmarshal(cached, &trades); err == nil {
			logger.Info(ctx, "Returning cached insider trades", "count", len(trades))
			return trades, nil
		}
	}

	if err := lds.rateLimiter.Wait(ctx, "SEBI"); err != nil {
		return nil, err
	}

	trades, err := lds.sebiClient.FetchInsiderTrading(ctx, symbol, fromDate, toDate)
	if err != nil {
		logger.Warn(ctx, "Failed to fetch insider trades", "error", err)
		return []interfaces.InsiderTradeData{}, nil
	}

	// Cache the result
	if data, err := json.Marshal(trades); err == nil {
		lds.cache.Set(cacheKey, data)
	}

	logger.Info(ctx, "Fetched insider trades", "count", len(trades))
	return trades, nil
}

// FetchFinancials retrieves financial statements
func (lds *LiveDataSource) FetchFinancials(ctx context.Context, symbol string, period string) (*interfaces.FinancialData, error) {
	logger.Info(ctx, "Fetching financials", "symbol", symbol, "period", period)

	if !lds.config.EnableScreener {
		return &interfaces.FinancialData{Period: period}, nil
	}

	cacheKey := fmt.Sprintf("financials:%s:%s", symbol, period)

	// Try cache first
	if cached, ok := lds.cache.Get(cacheKey); ok {
		var financials interfaces.FinancialData
		if err := json.Unmarshal(cached, &financials); err == nil {
			logger.Info(ctx, "Returning cached financials")
			return &financials, nil
		}
	}

	if err := lds.rateLimiter.Wait(ctx, "SCREENER"); err != nil {
		return nil, err
	}

	financials, err := lds.screenerClient.FetchFinancials(ctx, symbol, period)
	if err != nil {
		logger.Warn(ctx, "Failed to fetch financials", "error", err)
		return &interfaces.FinancialData{Period: period}, nil
	}

	// Cache the result
	if data, err := json.Marshal(financials); err == nil {
		lds.cache.Set(cacheKey, data)
	}

	logger.Info(ctx, "Fetched financials")
	return financials, nil
}

// FetchRegulatoryFilings retrieves regulatory filings from SEBI
func (lds *LiveDataSource) FetchRegulatoryFilings(ctx context.Context, symbol string, fromDate, toDate string) ([]interfaces.RegulatoryFiling, error) {
	logger.Info(ctx, "Fetching regulatory filings", "symbol", symbol)

	if !lds.config.EnableSEBI {
		return []interfaces.RegulatoryFiling{}, nil
	}

	cacheKey := fmt.Sprintf("regulatory:%s:%s:%s", symbol, fromDate, toDate)

	// Try cache first
	if cached, ok := lds.cache.Get(cacheKey); ok {
		var filings []interfaces.RegulatoryFiling
		if err := json.Unmarshal(cached, &filings); err == nil {
			logger.Info(ctx, "Returning cached regulatory filings", "count", len(filings))
			return filings, nil
		}
	}

	if err := lds.rateLimiter.Wait(ctx, "SEBI"); err != nil {
		return nil, err
	}

	filings, err := lds.sebiClient.FetchRegulatoryActions(ctx, symbol)
	if err != nil {
		logger.Warn(ctx, "Failed to fetch regulatory filings", "error", err)
		return []interfaces.RegulatoryFiling{}, nil
	}

	// Cache the result
	if data, err := json.Marshal(filings); err == nil {
		lds.cache.Set(cacheKey, data)
	}

	logger.Info(ctx, "Fetched regulatory filings", "count", len(filings))
	return filings, nil
}

// ClearCache clears all cached data
func (lds *LiveDataSource) ClearCache() error {
	return lds.cache.Clear()
}

// CleanupExpiredCache removes expired cache entries
func (lds *LiveDataSource) CleanupExpiredCache() error {
	return lds.cache.CleanupExpired()
}
