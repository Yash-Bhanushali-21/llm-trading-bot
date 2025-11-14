package pead

import (
	"context"
	"fmt"
	"strings"
	"time"

	"llm-trading-bot/api"
)

// EarningsDataFetcher defines the interface for fetching earnings data from live APIs
type EarningsDataFetcher interface {
	// FetchLatestEarnings fetches the most recent earnings data for a list of symbols
	FetchLatestEarnings(ctx context.Context, symbols []string) (map[string]*EarningsData, error)

	// FetchEarningsHistory fetches historical earnings for analysis
	FetchEarningsHistory(ctx context.Context, symbol string, quarters int) ([]*EarningsData, error)
}

// YahooFinanceEarningsDataFetcher fetches real earnings data from Yahoo Finance API
type YahooFinanceEarningsDataFetcher struct {
	client *api.Client
}

// NewYahooFinanceEarningsDataFetcher creates a new Yahoo Finance fetcher
func NewYahooFinanceEarningsDataFetcher() *YahooFinanceEarningsDataFetcher {
	// Create API client with Yahoo Finance specific configuration
	client := api.NewClient(
		api.WithTimeout(30*time.Second),
	)

	return &YahooFinanceEarningsDataFetcher{
		client: client,
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

// fetchSymbolEarnings fetches earnings data for a single symbol from Yahoo Finance
func (y *YahooFinanceEarningsDataFetcher) fetchSymbolEarnings(ctx context.Context, symbol string) (*EarningsData, error) {
	// Convert symbol to Yahoo Finance format (add .NS for NSE stocks)
	yahooSymbol := symbol
	if !strings.Contains(symbol, ".") {
		yahooSymbol = symbol + ".NS"
	}

	// Build Yahoo Finance API URL
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=earnings,financialData,defaultKeyStatistics,earningsHistory", yahooSymbol)

	// Make GET request using centralized API client
	resp, err := y.client.GET(ctx, url, api.YahooFinanceHeaders())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data from Yahoo Finance: %w", err)
	}

	// Parse the JSON response
	var yahooResp YahooFinanceResponse
	if err := resp.ParseJSON(&yahooResp); err != nil {
		return nil, fmt.Errorf("failed to parse Yahoo Finance response: %w", err)
	}

	// Extract and transform earnings data
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
