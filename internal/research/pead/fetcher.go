package pead

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// EarningsDataFetcher defines the interface for fetching earnings data
// This allows for multiple implementations (API-based, database, mock, etc.)
type EarningsDataFetcher interface {
	// FetchLatestEarnings fetches the most recent earnings data for a list of symbols
	FetchLatestEarnings(ctx context.Context, symbols []string) (map[string]*EarningsData, error)

	// FetchEarningsHistory fetches historical earnings for analysis
	FetchEarningsHistory(ctx context.Context, symbol string, quarters int) ([]*EarningsData, error)
}

// MockEarningsDataFetcher provides mock earnings data for testing and development
type MockEarningsDataFetcher struct {
	seed int64
}

// NewMockEarningsDataFetcher creates a new mock fetcher
func NewMockEarningsDataFetcher() *MockEarningsDataFetcher {
	return &MockEarningsDataFetcher{
		seed: time.Now().UnixNano(),
	}
}

// FetchLatestEarnings generates mock earnings data for testing
func (m *MockEarningsDataFetcher) FetchLatestEarnings(ctx context.Context, symbols []string) (map[string]*EarningsData, error) {
	result := make(map[string]*EarningsData)
	r := rand.New(rand.NewSource(m.seed))

	for _, symbol := range symbols {
		result[symbol] = m.generateMockEarnings(symbol, r)
	}

	return result, nil
}

// FetchEarningsHistory generates mock historical earnings data
func (m *MockEarningsDataFetcher) FetchEarningsHistory(ctx context.Context, symbol string, quarters int) ([]*EarningsData, error) {
	r := rand.New(rand.NewSource(m.seed + int64(len(symbol))))
	history := make([]*EarningsData, quarters)

	for i := 0; i < quarters; i++ {
		history[i] = m.generateMockEarnings(symbol, r)
		// Adjust date for each quarter
		history[i].AnnouncementDate = time.Now().AddDate(0, -3*i, 0)
		history[i].Quarter = fmt.Sprintf("Q%d %d", ((int(time.Now().Month())-1)/3+1-i)%4+1, time.Now().Year())
	}

	return history, nil
}

// generateMockEarnings creates realistic mock earnings data
func (m *MockEarningsDataFetcher) generateMockEarnings(symbol string, r *rand.Rand) *EarningsData {
	// Base values that vary by symbol
	symbolSeed := 0
	for _, c := range symbol {
		symbolSeed += int(c)
	}
	r.Seed(int64(symbolSeed))

	baseEPS := 10.0 + r.Float64()*50.0
	baseRevenue := 1000.0 + r.Float64()*10000.0

	// Generate earnings surprise (positive or negative)
	// 60% chance of positive surprise for mock data
	surpriseMultiplier := 1.0
	if r.Float64() < 0.6 {
		surpriseMultiplier = 1.0 + r.Float64()*0.15 // Up to 15% positive surprise
	} else {
		surpriseMultiplier = 1.0 - r.Float64()*0.10 // Up to 10% negative surprise
	}

	expectedEPS := baseEPS
	actualEPS := baseEPS * surpriseMultiplier

	// Revenue surprise (usually smaller than EPS surprise)
	revenueSurpriseMultiplier := 1.0 + (surpriseMultiplier-1.0)*0.5
	expectedRevenue := baseRevenue
	actualRevenue := baseRevenue * revenueSurpriseMultiplier

	// Growth rates (can be negative or positive)
	yoyEPSGrowth := -20.0 + r.Float64()*100.0 // -20% to 80% growth
	yoyRevenueGrowth := -10.0 + r.Float64()*60.0 // -10% to 50% growth

	qoqEPSGrowth := -10.0 + r.Float64()*40.0
	qoqRevenueGrowth := -5.0 + r.Float64()*30.0

	// Profit margins
	grossMargin := 30.0 + r.Float64()*40.0
	operatingMargin := 10.0 + r.Float64()*30.0
	netMargin := 5.0 + r.Float64()*25.0

	prevGrossMargin := grossMargin - 5.0 + r.Float64()*10.0
	prevOperatingMargin := operatingMargin - 5.0 + r.Float64()*10.0
	prevNetMargin := netMargin - 5.0 + r.Float64()*10.0

	// Consecutive beats (0-8 quarters)
	consecutiveBeats := 0
	if surpriseMultiplier > 1.0 {
		consecutiveBeats = r.Intn(9)
	}

	now := time.Now()
	quarter := fmt.Sprintf("Q%d %d", (int(now.Month())-1)/3+1, now.Year())

	return &EarningsData{
		Symbol:              symbol,
		Quarter:             quarter,
		FiscalYear:          now.Year(),
		AnnouncementDate:    now.AddDate(0, 0, -r.Intn(60)), // 0-60 days ago
		ActualEPS:           actualEPS,
		ExpectedEPS:         expectedEPS,
		ActualRevenue:       actualRevenue,
		ExpectedRevenue:     expectedRevenue,
		YoYEPSGrowth:        yoyEPSGrowth,
		YoYRevenueGrowth:    yoyRevenueGrowth,
		QoQEPSGrowth:        qoqEPSGrowth,
		QoQRevenueGrowth:    qoqRevenueGrowth,
		GrossMargin:         grossMargin,
		OperatingMargin:     operatingMargin,
		NetMargin:           netMargin,
		PrevGrossMargin:     prevGrossMargin,
		PrevOperatingMargin: prevOperatingMargin,
		PrevNetMargin:       prevNetMargin,
		ConsecutiveBeats:    consecutiveBeats,
	}
}

// APIEarningsDataFetcher is a placeholder for real API implementation
// TODO: Implement with actual financial data API (e.g., Alpha Vantage, Financial Modeling Prep, etc.)
type APIEarningsDataFetcher struct {
	APIKey  string
	BaseURL string
}

// NewAPIEarningsDataFetcher creates a new API-based fetcher
func NewAPIEarningsDataFetcher(apiKey, baseURL string) *APIEarningsDataFetcher {
	return &APIEarningsDataFetcher{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
}

// FetchLatestEarnings fetches real earnings data from API
func (a *APIEarningsDataFetcher) FetchLatestEarnings(ctx context.Context, symbols []string) (map[string]*EarningsData, error) {
	// TODO: Implement actual API call
	// Example APIs to integrate:
	// - Alpha Vantage: https://www.alphavantage.co/documentation/#earnings
	// - Financial Modeling Prep: https://financialmodelingprep.com/developer/docs/#Earnings-Calendar
	// - Yahoo Finance (unofficial): yfinance library
	return nil, fmt.Errorf("API implementation not yet available - use mock fetcher for testing")
}

// FetchEarningsHistory fetches historical earnings from API
func (a *APIEarningsDataFetcher) FetchEarningsHistory(ctx context.Context, symbol string, quarters int) ([]*EarningsData, error) {
	// TODO: Implement actual API call
	return nil, fmt.Errorf("API implementation not yet available - use mock fetcher for testing")
}
