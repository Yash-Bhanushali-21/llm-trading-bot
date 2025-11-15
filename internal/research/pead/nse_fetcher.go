package pead

import (
	"context"
	"fmt"
	"strings"
	"time"

	"llm-trading-bot/internal/api"
)

// NSEDataFetcher fetches earnings data specifically for NSE-listed stocks
// Uses multiple fallback sources for reliability
type NSEDataFetcher struct {
	client       *api.Client
	yahooFetcher *YahooFinanceEarningsDataFetcher
	useYahoo     bool
	useScreener  bool
}

// NewNSEDataFetcher creates a fetcher optimized for NSE stocks
func NewNSEDataFetcher() *NSEDataFetcher {
	// Create API client with longer timeout for NSE APIs
	client := api.NewClient(
		api.WithTimeout(45*time.Second),
		api.WithLogging(true), // Enable API logging
	)

	return &NSEDataFetcher{
		client:       client,
		yahooFetcher: NewYahooFinanceEarningsDataFetcher(),
		useYahoo:     true,
		useScreener:  true,
	}
}

// FetchLatestEarnings fetches earnings for NSE stocks with fallback sources
func (n *NSEDataFetcher) FetchLatestEarnings(ctx context.Context, symbols []string) (map[string]*EarningsData, error) {
	result := make(map[string]*EarningsData)

	fmt.Println("📍 Fetching data for NSE stocks...")

	for i, symbol := range symbols {
		var data *EarningsData
		var err error

		// Try Yahoo Finance first (primary source)
		if n.useYahoo {
			data, err = n.yahooFetcher.fetchSymbolEarnings(ctx, symbol)
			if err == nil && data != nil {
				result[symbol] = data
				fmt.Printf("  ✓ %s: Fetched from Yahoo Finance\n", symbol)

				// Rate limiting
				if i < len(symbols)-1 {
					time.Sleep(2 * time.Second)
				}
				continue
			}
			fmt.Printf("  ⚠ %s: Yahoo Finance failed (%v), trying alternatives...\n", symbol, err)
		}

		// Fallback to NSE-specific screener data
		if n.useScreener {
			data, err = n.fetchFromScreener(ctx, symbol)
			if err == nil && data != nil {
				result[symbol] = data
				fmt.Printf("  ✓ %s: Fetched from Screener.in\n", symbol)

				if i < len(symbols)-1 {
					time.Sleep(3 * time.Second)
				}
				continue
			}
			fmt.Printf("  ⚠ %s: Screener failed (%v)\n", symbol, err)
		}

		// If all sources fail, log warning
		fmt.Printf("  ✗ %s: Could not fetch data from any source\n", symbol)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("failed to fetch data for any symbols - check network and data sources")
	}

	fmt.Printf("\n📊 Successfully fetched data for %d/%d stocks\n\n", len(result), len(symbols))
	return result, nil
}

// FetchEarningsHistory fetches historical earnings for NSE stocks
func (n *NSEDataFetcher) FetchEarningsHistory(ctx context.Context, symbol string, quarters int) ([]*EarningsData, error) {
	// Use Yahoo Finance for historical data
	return n.yahooFetcher.FetchEarningsHistory(ctx, symbol, quarters)
}

// fetchFromScreener fetches data from Screener.in (Indian stock screener)
func (n *NSEDataFetcher) fetchFromScreener(ctx context.Context, symbol string) (*EarningsData, error) {
	// Screener.in provides fundamental data for NSE stocks
	// Format: https://www.screener.in/api/company/{company_id}/

	// This is a simplified implementation
	// In production, you would:
	// 1. First resolve symbol to company ID
	// 2. Fetch quarterly results
	// 3. Parse and structure the data

	// For now, return an informative error
	return nil, fmt.Errorf("screener.in integration not yet implemented - use Yahoo Finance or add API key")
}

// EarningsAnnouncement represents a company's earnings announcement
type EarningsAnnouncement struct {
	Symbol           string    `json:"symbol"`
	CompanyName      string    `json:"company_name"`
	AnnouncementDate time.Time `json:"announcement_date"`
	Quarter          string    `json:"quarter"`
	FiscalYear       int       `json:"fiscal_year"`
}

// FetchRecentEarningsAnnouncements fetches companies that announced earnings in the last N days
// This is the dynamic approach - discovers stocks with fresh earnings instead of hardcoded lists
func (n *NSEDataFetcher) FetchRecentEarningsAnnouncements(ctx context.Context, daysBack int) ([]string, error) {
	symbols := make([]string, 0)

	// Try multiple sources for earnings announcements

	// Source 1: NSE Corporate Announcements API
	nseSymbols, err := n.fetchNSECorporateAnnouncements(ctx, daysBack)
	if err == nil && len(nseSymbols) > 0 {
		symbols = append(symbols, nseSymbols...)
	}

	// Source 2: MoneyControl earnings calendar
	mcSymbols, err := n.fetchMoneyControlEarnings(ctx, daysBack)
	if err == nil && len(mcSymbols) > 0 {
		symbols = append(symbols, mcSymbols...)
	}

	// Source 3: Screener.in recent results
	screenerSymbols, err := n.fetchScreenerRecentResults(ctx, daysBack)
	if err == nil && len(screenerSymbols) > 0 {
		symbols = append(symbols, screenerSymbols...)
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		if !seen[symbol] {
			seen[symbol] = true
			unique = append(unique, symbol)
		}
	}

	if len(unique) == 0 {
		return nil, fmt.Errorf("no recent earnings announcements found from any source")
	}

	return unique, nil
}

// fetchNSECorporateAnnouncements fetches recent earnings from NSE corporate actions
func (n *NSEDataFetcher) fetchNSECorporateAnnouncements(ctx context.Context, daysBack int) ([]string, error) {
	// NSE Corporate Announcements endpoint
	url := "https://www.nseindia.com/api/corporates-financial-results?index=equities"

	fmt.Printf("\n🔍 [NSE API] Making request to: %s\n", url)

	// Make GET request using centralized API client with NSE-specific headers
	resp, err := n.client.GET(ctx, url, api.NSEHeaders())
	if err != nil {
		fmt.Printf("❌ [NSE API] Request failed: %v\n", err)
		return nil, fmt.Errorf("NSE API request failed: %w", err)
	}

	fmt.Printf("✅ [NSE API] Response received: Status=%d, Size=%d bytes\n", resp.StatusCode, len(resp.Body))

	// Parse NSE response
	var data struct {
		Data []struct {
			Symbol  string `json:"symbol"`
			XDate   string `json:"xdate"` // Announcement date
			Purpose string `json:"purpose"`
		} `json:"data"`
	}

	if err := resp.ParseJSON(&data); err != nil {
		fmt.Printf("❌ [NSE API] JSON parsing failed: %v\n", err)
		maxLen := 500
		if len(resp.Body) < maxLen {
			maxLen = len(resp.Body)
		}
		fmt.Printf("📄 [NSE API] Raw response: %s\n", string(resp.Body[:maxLen]))
		return nil, fmt.Errorf("failed to parse NSE response: %w", err)
	}

	fmt.Printf("📊 [NSE API] Total announcements received: %d\n", len(data.Data))

	// Filter for recent earnings announcements
	cutoffDate := time.Now().AddDate(0, 0, -daysBack)
	symbols := make([]string, 0)

	matchedPurpose := 0
	matchedDate := 0

	for _, item := range data.Data {
		// Check if it's a financial results announcement
		if !strings.Contains(strings.ToLower(item.Purpose), "result") &&
			!strings.Contains(strings.ToLower(item.Purpose), "financial") {
			continue
		}
		matchedPurpose++

		// Parse date
		announcementDate, err := time.Parse("02-Jan-2006", item.XDate)
		if err != nil {
			fmt.Printf("⚠️  [NSE API] Failed to parse date '%s' for %s\n", item.XDate, item.Symbol)
			continue
		}

		// Check if within our timeframe
		if announcementDate.After(cutoffDate) {
			symbols = append(symbols, item.Symbol)
			matchedDate++
			if len(symbols) <= 5 {
				fmt.Printf("  ✓ %s - %s (%s)\n", item.Symbol, item.XDate, item.Purpose)
			}
		}
	}

	fmt.Printf("📌 [NSE API] Filtered results:\n")
	fmt.Printf("  - Matched purpose filter (result/financial): %d\n", matchedPurpose)
	fmt.Printf("  - Within %d days timeframe: %d\n", daysBack, matchedDate)
	fmt.Printf("  - Final symbols: %d\n", len(symbols))

	return symbols, nil
}

// fetchMoneyControlEarnings fetches from MoneyControl earnings calendar
func (n *NSEDataFetcher) fetchMoneyControlEarnings(ctx context.Context, daysBack int) ([]string, error) {
	// MoneyControl has an earnings calendar API
	// This would need to be implemented with proper scraping or API access
	// For now, return empty to indicate not implemented
	return nil, fmt.Errorf("MoneyControl earnings calendar not yet implemented")
}

// fetchScreenerRecentResults fetches from Screener.in recent results page
func (n *NSEDataFetcher) fetchScreenerRecentResults(ctx context.Context, daysBack int) ([]string, error) {
	// Screener.in shows recent quarterly results
	// This would need web scraping or API access
	return nil, fmt.Errorf("Screener.in recent results not yet implemented")
}

// NSEQuarterlyResultsAPI represents NSE quarterly results structure
// Based on NSE India's official corporate API
type NSEQuarterlyResults struct {
	Symbol  string `json:"symbol"`
	Results []struct {
		Quarter   string  `json:"quarter"`
		Sales     float64 `json:"sales"`
		NetProfit float64 `json:"netProfit"`
		EPS       float64 `json:"eps"`
		Date      string  `json:"date"`
	} `json:"results"`
}

// fetchFromNSEAPI fetches data from NSE India's official API
// Note: NSE API requires proper headers and may have rate limits
func (n *NSEDataFetcher) fetchFromNSEAPI(ctx context.Context, symbol string) (*EarningsData, error) {
	// NSE Corporate API endpoint
	url := fmt.Sprintf("https://www.nseindia.com/api/quote-equity?symbol=%s", symbol)

	// Make GET request using centralized API client with NSE-specific headers
	resp, err := n.client.GET(ctx, url, api.NSEHeaders())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from NSE: %w", err)
	}

	// Parse NSE response
	var nseData map[string]interface{}
	if err := resp.ParseJSON(&nseData); err != nil {
		return nil, fmt.Errorf("failed to parse NSE response: %w", err)
	}

	// Extract earnings data from NSE response
	// This would need proper parsing based on actual NSE API response structure
	return nil, fmt.Errorf("NSE API parsing not yet implemented")
}
