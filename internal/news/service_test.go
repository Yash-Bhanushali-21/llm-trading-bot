package news

import (
	"context"
	"testing"
	"time"

	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/types"
)

func TestSentimentCache(t *testing.T) {
	cache := newSentimentCache(1 * time.Second)

	symbol := "RELIANCE"
	sentiment := types.NewsSentiment{
		Symbol:           symbol,
		OverallSentiment: "POSITIVE",
		OverallScore:     0.8,
		Confidence:       0.9,
		Timestamp:        time.Now().Unix(),
	}

	// Test set and get
	cache.set(symbol, sentiment)

	retrieved, found := cache.get(symbol)
	if !found {
		t.Fatal("Expected to find cached sentiment")
	}

	if retrieved.Symbol != symbol {
		t.Errorf("Expected symbol %s, got %s", symbol, retrieved.Symbol)
	}

	if retrieved.OverallScore != 0.8 {
		t.Errorf("Expected score 0.8, got %f", retrieved.OverallScore)
	}

	// Test expiration
	time.Sleep(2 * time.Second)
	_, found = cache.get(symbol)
	if found {
		t.Error("Expected cache entry to be expired")
	}
}

func TestServiceConfig(t *testing.T) {
	cfg := DefaultServiceConfig()

	if cfg.MaxArticles != 15 {
		t.Errorf("Expected MaxArticles to be 15, got %d", cfg.MaxArticles)
	}

	if cfg.CacheDuration != 1*time.Hour {
		t.Errorf("Expected CacheDuration to be 1 hour, got %v", cfg.CacheDuration)
	}

	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}
}

func TestNewService(t *testing.T) {
	botCfg := &store.Config{}
	botCfg.LLM.Provider = "OPENAI"
	botCfg.LLM.Model = "gpt-4o-mini"

	serviceCfg := DefaultServiceConfig()
	svc := NewService(botCfg, serviceCfg)

	if svc == nil {
		t.Fatal("Expected service to be created")
	}

	if svc.scraper == nil {
		t.Error("Expected scraper to be initialized")
	}

	if svc.analyzer == nil {
		t.Error("Expected analyzer to be initialized")
	}

	if svc.cache == nil {
		t.Error("Expected cache to be initialized")
	}
}

func TestServiceDisabled(t *testing.T) {
	botCfg := &store.Config{}
	serviceCfg := &ServiceConfig{
		Enabled: false,
	}

	svc := NewService(botCfg, serviceCfg)
	ctx := context.Background()

	sentiment, err := svc.GetSentiment(ctx, "RELIANCE")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if sentiment.OverallSentiment != "NEUTRAL" {
		t.Errorf("Expected NEUTRAL sentiment when disabled, got %s", sentiment.OverallSentiment)
	}

	if sentiment.Summary != "Sentiment analysis disabled" {
		t.Errorf("Expected disabled message, got %s", sentiment.Summary)
	}
}

func TestCacheCleanup(t *testing.T) {
	cache := newSentimentCache(100 * time.Millisecond)

	// Add some entries
	for i := 0; i < 5; i++ {
		sentiment := types.NewsSentiment{
			Symbol:     "SYM" + string(rune(i)),
			Timestamp:  time.Now().Unix(),
			Confidence: 0.5,
		}
		cache.set("SYM"+string(rune(i)), sentiment)
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Trigger cleanup
	cache.cleanup()

	// Check that all entries are removed
	cache.mu.RLock()
	count := len(cache.data)
	cache.mu.RUnlock()

	if count != 0 {
		t.Errorf("Expected 0 cache entries after cleanup, got %d", count)
	}
}

func TestGetCachedSymbols(t *testing.T) {
	botCfg := &store.Config{}
	botCfg.LLM.Provider = "OPENAI"
	serviceCfg := DefaultServiceConfig()

	svc := NewService(botCfg, serviceCfg)

	// Add some cached entries
	symbols := []string{"RELIANCE", "TCS", "INFY"}
	for _, sym := range symbols {
		sentiment := types.NewsSentiment{
			Symbol:    sym,
			Timestamp: time.Now().Unix(),
		}
		svc.cache.set(sym, sentiment)
	}

	cached := svc.GetCachedSymbols()

	if len(cached) != 3 {
		t.Errorf("Expected 3 cached symbols, got %d", len(cached))
	}
}

func TestClearCache(t *testing.T) {
	botCfg := &store.Config{}
	serviceCfg := DefaultServiceConfig()

	svc := NewService(botCfg, serviceCfg)

	// Add cached entry
	sentiment := types.NewsSentiment{
		Symbol:    "RELIANCE",
		Timestamp: time.Now().Unix(),
	}
	svc.cache.set("RELIANCE", sentiment)

	// Verify it's cached
	cached := svc.GetCachedSymbols()
	if len(cached) != 1 {
		t.Fatal("Expected 1 cached symbol")
	}

	// Clear cache
	svc.ClearCache()

	// Verify it's cleared
	cached = svc.GetCachedSymbols()
	if len(cached) != 0 {
		t.Errorf("Expected 0 cached symbols after clear, got %d", len(cached))
	}
}
