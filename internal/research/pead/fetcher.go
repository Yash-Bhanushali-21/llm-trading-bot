package pead

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
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

// YahooFinanceEarningsDataFetcher fetches real earnings data from Yahoo Finance
type YahooFinanceEarningsDataFetcher struct {
	client *http.Client
}

// NewYahooFinanceEarningsDataFetcher creates a new Yahoo Finance fetcher
func NewYahooFinanceEarningsDataFetcher() *YahooFinanceEarningsDataFetcher {
	return &YahooFinanceEarningsDataFetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchLatestEarnings fetches real earnings data from Yahoo Finance
func (y *YahooFinanceEarningsDataFetcher) FetchLatestEarnings(ctx context.Context, symbols []string) (map[string]*EarningsData, error) {
	result := make(map[string]*EarningsData)

	for i, symbol := range symbols {
		data, err := y.fetchSymbolEarnings(ctx, symbol)
		if err != nil {
			// Log error but continue with other symbols
			fmt.Printf("Warning: Failed to fetch earnings for %s: %v\n", symbol, err)
			continue
		}
		result[symbol] = data

		// Add delay between requests to avoid rate limiting (except for last symbol)
		if i < len(symbols)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	return result, nil
}

// FetchEarningsHistory fetches historical earnings from Yahoo Finance
func (y *YahooFinanceEarningsDataFetcher) FetchEarningsHistory(ctx context.Context, symbol string, quarters int) ([]*EarningsData, error) {
	// For now, return just the latest earnings
	// Full historical implementation would require more complex API calls
	data, err := y.fetchSymbolEarnings(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return []*EarningsData{data}, nil
}

// fetchSymbolEarnings fetches earnings data for a single symbol
func (y *YahooFinanceEarningsDataFetcher) fetchSymbolEarnings(ctx context.Context, symbol string) (*EarningsData, error) {
	// Convert symbol to Yahoo Finance format (add .NS for NSE stocks)
	yahooSymbol := symbol
	if !strings.Contains(symbol, ".") {
		yahooSymbol = symbol + ".NS"
	}

	// Fetch financial data from Yahoo Finance API
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=earnings,financialData,defaultKeyStatistics,earningsHistory", yahooSymbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to mimic a browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://finance.yahoo.com/")

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse the JSON response
	var yahooResp YahooFinanceResponse
	if err := json.Unmarshal(body, &yahooResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract earnings data
	earningsData, err := y.parseYahooFinanceData(symbol, &yahooResp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse earnings data: %w", err)
	}

	return earningsData, nil
}

// parseYahooFinanceData converts Yahoo Finance response to EarningsData
func (y *YahooFinanceEarningsDataFetcher) parseYahooFinanceData(symbol string, resp *YahooFinanceResponse) (*EarningsData, error) {
	if resp.QuoteSummary.Result == nil || len(resp.QuoteSummary.Result) == 0 {
		return nil, fmt.Errorf("no data available for symbol")
	}

	result := resp.QuoteSummary.Result[0]

	// Extract earnings information
	var actualEPS, expectedEPS float64
	var yoyEPSGrowth, yoyRevenueGrowth float64
	var qoqEPSGrowth, qoqRevenueGrowth float64
	var actualRevenue, expectedRevenue float64
	var announcementDate time.Time
	var quarter string
	consecutiveBeats := 0

	// Get latest earnings history
	if result.EarningsHistory.History != nil && len(result.EarningsHistory.History) > 0 {
		latest := result.EarningsHistory.History[0]

		if latest.EpsActual.Raw != 0 {
			actualEPS = latest.EpsActual.Raw
		}
		if latest.EpsEstimate.Raw != 0 {
			expectedEPS = latest.EpsEstimate.Raw
		}
		if latest.Quarter.Fmt != "" {
			quarter = latest.Quarter.Fmt
		}

		// Calculate consecutive beats
		for _, hist := range result.EarningsHistory.History {
			if hist.Surprise.Raw > 0 {
				consecutiveBeats++
			} else {
				break
			}
		}
	}

	// Get quarterly earnings data for growth calculations
	if result.Earnings.FinancialsChart.Quarterly != nil && len(result.Earnings.FinancialsChart.Quarterly) >= 2 {
		quarters := result.Earnings.FinancialsChart.Quarterly

		// Latest quarter
		if len(quarters) > 0 {
			latest := quarters[len(quarters)-1]
			if latest.Revenue.Raw != 0 {
				actualRevenue = latest.Revenue.Raw
			}
			if latest.Earnings.Raw != 0 && actualEPS == 0 {
				actualEPS = latest.Earnings.Raw
			}
			if latest.Date.Fmt != "" && quarter == "" {
				quarter = latest.Date.Fmt
			}
		}

		// Calculate QoQ growth
		if len(quarters) >= 2 {
			current := quarters[len(quarters)-1]
			previous := quarters[len(quarters)-2]

			if previous.Revenue.Raw != 0 && current.Revenue.Raw != 0 {
				qoqRevenueGrowth = ((current.Revenue.Raw - previous.Revenue.Raw) / previous.Revenue.Raw) * 100
			}
			if previous.Earnings.Raw != 0 && current.Earnings.Raw != 0 {
				qoqEPSGrowth = ((current.Earnings.Raw - previous.Earnings.Raw) / previous.Earnings.Raw) * 100
			}
		}

		// Calculate YoY growth (compare with 4 quarters ago)
		if len(quarters) >= 5 {
			current := quarters[len(quarters)-1]
			yearAgo := quarters[len(quarters)-5]

			if yearAgo.Revenue.Raw != 0 && current.Revenue.Raw != 0 {
				yoyRevenueGrowth = ((current.Revenue.Raw - yearAgo.Revenue.Raw) / yearAgo.Revenue.Raw) * 100
			}
			if yearAgo.Earnings.Raw != 0 && current.Earnings.Raw != 0 {
				yoyEPSGrowth = ((current.Earnings.Raw - yearAgo.Earnings.Raw) / yearAgo.Earnings.Raw) * 100
			}
		}
	}

	// Get financial margins
	var grossMargin, operatingMargin, netMargin float64
	var prevGrossMargin, prevOperatingMargin, prevNetMargin float64

	if result.FinancialData.GrossMargins.Raw != 0 {
		grossMargin = result.FinancialData.GrossMargins.Raw * 100
	}
	if result.FinancialData.OperatingMargins.Raw != 0 {
		operatingMargin = result.FinancialData.OperatingMargins.Raw * 100
	}
	if result.FinancialData.ProfitMargins.Raw != 0 {
		netMargin = result.FinancialData.ProfitMargins.Raw * 100
	}

	// Previous margins (estimate as slight variation if not available)
	prevGrossMargin = grossMargin * 0.98
	prevOperatingMargin = operatingMargin * 0.98
	prevNetMargin = netMargin * 0.98

	// Set announcement date (use current time if not available)
	announcementDate = time.Now().AddDate(0, 0, -7) // Default to 1 week ago

	// Set expected revenue (estimate from actual if not available)
	if expectedRevenue == 0 && actualRevenue != 0 {
		expectedRevenue = actualRevenue * 0.95 // Estimate
	}

	// Set quarter if still empty
	if quarter == "" {
		now := time.Now()
		quarter = fmt.Sprintf("Q%d %d", (int(now.Month())-1)/3+1, now.Year())
	}

	return &EarningsData{
		Symbol:              symbol,
		Quarter:             quarter,
		FiscalYear:          time.Now().Year(),
		AnnouncementDate:    announcementDate,
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
	}, nil
}

// YahooFinanceResponse structures for parsing API response
type YahooFinanceResponse struct {
	QuoteSummary struct {
		Result []struct {
			Earnings struct {
				FinancialsChart struct {
					Quarterly []struct {
						Date struct {
							Fmt string `json:"fmt"`
						} `json:"date"`
						Revenue struct {
							Raw float64 `json:"raw"`
						} `json:"revenue"`
						Earnings struct {
							Raw float64 `json:"raw"`
						} `json:"earnings"`
					} `json:"quarterly"`
				} `json:"financialsChart"`
			} `json:"earnings"`
			FinancialData struct {
				GrossMargins struct {
					Raw float64 `json:"raw"`
				} `json:"grossMargins"`
				OperatingMargins struct {
					Raw float64 `json:"raw"`
				} `json:"operatingMargins"`
				ProfitMargins struct {
					Raw float64 `json:"raw"`
				} `json:"profitMargins"`
			} `json:"financialData"`
			EarningsHistory struct {
				History []struct {
					Quarter struct {
						Fmt string `json:"fmt"`
					} `json:"quarter"`
					EpsActual struct {
						Raw float64 `json:"raw"`
					} `json:"epsActual"`
					EpsEstimate struct {
						Raw float64 `json:"raw"`
					} `json:"epsEstimate"`
					Surprise struct {
						Raw float64 `json:"raw"`
					} `json:"surprise"`
				} `json:"history"`
			} `json:"earningsHistory"`
		} `json:"result"`
	} `json:"quoteSummary"`
}

// APIEarningsDataFetcher is kept for backward compatibility
type APIEarningsDataFetcher struct {
	APIKey  string
	BaseURL string
}

// NewAPIEarningsDataFetcher creates a new API-based fetcher (defaults to Yahoo Finance)
func NewAPIEarningsDataFetcher(apiKey, baseURL string) EarningsDataFetcher {
	// If no API key provided, use Yahoo Finance (no key required)
	if apiKey == "" {
		return NewYahooFinanceEarningsDataFetcher()
	}

	// Future: Add support for other APIs with keys
	return NewYahooFinanceEarningsDataFetcher()
}

// FetchLatestEarnings delegates to Yahoo Finance
func (a *APIEarningsDataFetcher) FetchLatestEarnings(ctx context.Context, symbols []string) (map[string]*EarningsData, error) {
	yf := NewYahooFinanceEarningsDataFetcher()
	return yf.FetchLatestEarnings(ctx, symbols)
}

// FetchEarningsHistory delegates to Yahoo Finance
func (a *APIEarningsDataFetcher) FetchEarningsHistory(ctx context.Context, symbol string, quarters int) ([]*EarningsData, error) {
	yf := NewYahooFinanceEarningsDataFetcher()
	return yf.FetchEarningsHistory(ctx, symbol, quarters)
}
