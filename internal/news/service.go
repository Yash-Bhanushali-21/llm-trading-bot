package news

import (
	"context"
	"sync"
	"time"

	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/types"
)

// Service provides news sentiment analysis with caching
type Service struct {
	scraper  *Scraper
	analyzer *SentimentAnalyzer
	cache    *sentimentCache
	cfg      *ServiceConfig
}

// ServiceConfig configures the news sentiment service
type ServiceConfig struct {
	MaxArticles    int           // Maximum articles to scrape per symbol
	CacheDuration  time.Duration // How long to cache sentiment data
	ScraperTimeout time.Duration // Timeout for scraping operations
	Enabled        bool          // Whether sentiment analysis is enabled
}

// DefaultServiceConfig returns default configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxArticles:    15,
		CacheDuration:  1 * time.Hour,
		ScraperTimeout: 30 * time.Second,
		Enabled:        true,
	}
}

// sentimentCache stores sentiment results temporarily
type sentimentCache struct {
	mu    sync.RWMutex
	data  map[string]*cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	sentiment types.NewsSentiment
	timestamp time.Time
}

// newSentimentCache creates a new cache
func newSentimentCache(ttl time.Duration) *sentimentCache {
	cache := &sentimentCache{
		data: make(map[string]*cacheEntry),
		ttl:  ttl,
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// get retrieves cached sentiment if valid
func (c *sentimentCache) get(symbol string) (types.NewsSentiment, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.data[symbol]
	if !exists {
		return types.NewsSentiment{}, false
	}

	// Check if expired
	if time.Since(entry.timestamp) > c.ttl {
		return types.NewsSentiment{}, false
	}

	return entry.sentiment, true
}

// set stores sentiment in cache
func (c *sentimentCache) set(symbol string, sentiment types.NewsSentiment) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[symbol] = &cacheEntry{
		sentiment: sentiment,
		timestamp: time.Now(),
	}
}

// cleanupLoop periodically removes expired entries
func (c *sentimentCache) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired entries
func (c *sentimentCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for symbol, entry := range c.data {
		if now.Sub(entry.timestamp) > c.ttl {
			delete(c.data, symbol)
		}
	}
}

// NewService creates a new news sentiment service
func NewService(botCfg *store.Config, serviceCfg *ServiceConfig) *Service {
	if serviceCfg == nil {
		serviceCfg = DefaultServiceConfig()
	}

	return &Service{
		scraper:  NewScraper(serviceCfg.ScraperTimeout),
		analyzer: NewSentimentAnalyzer(botCfg),
		cache:    newSentimentCache(serviceCfg.CacheDuration),
		cfg:      serviceCfg,
	}
}

// GetSentiment retrieves news sentiment for a symbol (cached or fresh)
func (s *Service) GetSentiment(ctx context.Context, symbol string) (types.NewsSentiment, error) {
	if !s.cfg.Enabled {
		return types.NewsSentiment{
			Symbol:           symbol,
			OverallSentiment: "NEUTRAL",
			Summary:          "Sentiment analysis disabled",
			Timestamp:        time.Now().Unix(),
		}, nil
	}

	// Check cache first
	if cached, ok := s.cache.get(symbol); ok {
		logger.Info(ctx, "Using cached sentiment", "symbol", symbol, "age_minutes",
			time.Since(time.Unix(cached.Timestamp, 0)).Minutes())
		return cached, nil
	}

	// Fetch fresh sentiment
	logger.Info(ctx, "Fetching fresh news sentiment", "symbol", symbol)
	sentiment, err := s.fetchFreshSentiment(ctx, symbol)
	if err != nil {
		logger.ErrorWithErr(ctx, "Failed to fetch sentiment", err, "symbol", symbol)
		// Return neutral sentiment on error rather than failing
		return types.NewsSentiment{
			Symbol:           symbol,
			OverallSentiment: "NEUTRAL",
			Summary:          "Failed to fetch sentiment: " + err.Error(),
			Confidence:       0.0,
			Timestamp:        time.Now().Unix(),
		}, nil
	}

	// Cache the result
	s.cache.set(symbol, sentiment)

	return sentiment, nil
}

// fetchFreshSentiment scrapes and analyzes news for a symbol
func (s *Service) fetchFreshSentiment(ctx context.Context, symbol string) (types.NewsSentiment, error) {
	// Scrape news articles
	articles, err := s.scraper.ScrapeNews(ctx, symbol, s.cfg.MaxArticles)
	if err != nil {
		return types.NewsSentiment{}, err
	}

	// If no articles found, try Google News as fallback
	if len(articles) == 0 {
		logger.Info(ctx, "No articles from primary sources, trying Google News", "symbol", symbol)
		articles, err = s.scraper.ScrapeGoogleNews(ctx, symbol, s.cfg.MaxArticles)
		if err != nil {
			logger.ErrorWithErr(ctx, "Google News fallback failed", err, "symbol", symbol)
		}
	}

	// Analyze sentiment
	sentiment, err := s.analyzer.AnalyzeMultipleArticles(ctx, symbol, articles)
	if err != nil {
		return types.NewsSentiment{}, err
	}

	return sentiment, nil
}

// RefreshSentiment forces a refresh of sentiment data (bypasses cache)
func (s *Service) RefreshSentiment(ctx context.Context, symbol string) (types.NewsSentiment, error) {
	sentiment, err := s.fetchFreshSentiment(ctx, symbol)
	if err != nil {
		return types.NewsSentiment{}, err
	}

	s.cache.set(symbol, sentiment)
	return sentiment, nil
}

// ClearCache removes all cached sentiment data
func (s *Service) ClearCache() {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	s.cache.data = make(map[string]*cacheEntry)
}

// GetCachedSymbols returns list of symbols with cached sentiment
func (s *Service) GetCachedSymbols() []string {
	s.cache.mu.RLock()
	defer s.cache.mu.RUnlock()

	symbols := make([]string, 0, len(s.cache.data))
	for symbol := range s.cache.data {
		symbols = append(symbols, symbol)
	}
	return symbols
}
