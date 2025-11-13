package pead

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// NSEDataFetcher fetches earnings data specifically for NSE-listed stocks
// Uses multiple fallback sources for reliability
type NSEDataFetcher struct {
	client         *http.Client
	yahooFetcher   *YahooFinanceEarningsDataFetcher
	useYahoo       bool
	useScreener    bool
}

// NewNSEDataFetcher creates a fetcher optimized for NSE stocks
func NewNSEDataFetcher() *NSEDataFetcher {
	return &NSEDataFetcher{
		client: &http.Client{
			Timeout: 45 * time.Second,
		},
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

// ValidateNSESymbol checks if a symbol is a valid NSE stock code
func ValidateNSESymbol(symbol string) bool {
	// NSE symbols are typically:
	// - All uppercase
	// - Alphanumeric
	// - 1-10 characters
	// - No special characters except & (for some stocks)

	if len(symbol) == 0 || len(symbol) > 10 {
		return false
	}

	// Check if already has exchange suffix
	if strings.HasSuffix(symbol, ".NS") || strings.HasSuffix(symbol, ".BO") {
		return true
	}

	// Common NSE stocks
	commonNSEStocks := map[string]bool{
		"RELIANCE": true, "TCS": true, "HDFCBANK": true, "INFY": true,
		"ICICIBANK": true, "HINDUNILVR": true, "SBIN": true, "BHARTIARTL": true,
		"BAJFINANCE": true, "ITC": true, "KOTAKBANK": true, "LT": true,
		"AXISBANK": true, "ASIANPAINT": true, "MARUTI": true, "TITAN": true,
		"SUNPHARMA": true, "ULTRACEMCO": true, "NESTLEIND": true, "HCLTECH": true,
		"WIPRO": true, "TATAMOTORS": true, "TATASTEEL": true, "POWERGRID": true,
		"NTPC": true, "ONGC": true, "COALINDIA": true, "JSWSTEEL": true,
		"GRASIM": true, "TECHM": true, "BAJAJFINSV": true, "HDFCLIFE": true,
		"DIVISLAB": true, "BRITANNIA": true, "INDUSINDBK": true, "ADANIENT": true,
		"ADANIPORTS": true, "SHREECEM": true, "DRREDDY": true, "APOLLOHOSP": true,
		"BPCL": true, "CIPLA": true, "EICHERMOT": true, "HEROMOTOCO": true,
		"HINDALCO": true, "M&M": true, "TATACONSUM": true, "HAL": true,
		"UPL": true, "VEDL": true,
	}

	return commonNSEStocks[symbol]
}

// GetNSETop50 returns the NSE Nifty 50 stock symbols
func GetNSETop50() []string {
	return []string{
		"RELIANCE", "TCS", "HDFCBANK", "INFY", "ICICIBANK",
		"HINDUNILVR", "SBIN", "BHARTIARTL", "BAJFINANCE", "ITC",
		"KOTAKBANK", "LT", "AXISBANK", "ASIANPAINT", "MARUTI",
		"TITAN", "SUNPHARMA", "ULTRACEMCO", "NESTLEIND", "HCLTECH",
		"WIPRO", "TATAMOTORS", "TATASTEEL", "POWERGRID", "NTPC",
		"ONGC", "COALINDIA", "JSWSTEEL", "GRASIM", "TECHM",
		"BAJAJFINSV", "HDFCLIFE", "DIVISLAB", "BRITANNIA", "INDUSINDBK",
		"ADANIENT", "ADANIPORTS", "SHREECEM", "DRREDDY", "APOLLOHOSP",
		"BPCL", "CIPLA", "EICHERMOT", "HEROMOTOCO", "HINDALCO",
		"M&M", "TATACONSUM", "UPL", "VEDL", "HAL",
	}
}

// GetNSEMidcap returns popular NSE midcap stocks
func GetNSEMidcap() []string {
	return []string{
		"GODREJCP", "PIDILITIND", "BERGEPAINT", "HAVELLS", "SBICARD",
		"BANDHANBNK", "MCDOWELL-N", "BANKBARODA", "GAIL", "INDIGO",
		"SIEMENS", "DLF", "AMBUJACEM", "TORNTPHARM", "LUPIN",
		"NAUKRI", "MOTHERSON", "ABCAPITAL", "ESCORTS", "TVSMOTOR",
		"IDEA", "DIXON", "PERSISTENT", "LTIM", "COFORGE",
		"SAIL", "MUTHOOTFIN", "NMDC", "CHOLAFIN", "LICHSGFIN",
	}
}

// GetNSENext50 returns the NSE Nifty Next 50 stock symbols
func GetNSENext50() []string {
	return []string{
		"ADANIGREEN", "ADANIPOWER", "ATGL", "BAJAJHLDNG", "BEL",
		"BOSCHLTD", "CANBK", "COLPAL", "DMART", "GLAND",
		"GODREJCP", "HAVELLS", "HINDPETRO", "ICICIPRULI", "INDUSTOWER",
		"INDUSINDBK", "IOC", "IRCTC", "ITC", "JINDALSTEL",
		"LAURUSLABS", "LICHSGFIN", "LTIM", "LUPIN", "MARICO",
		"NAUKRI", "NMDC", "OBEROIRLTY", "OFSS", "PAGEIND",
		"PERSISTENT", "PETRONET", "PIDILITIND", "PNB", "RECLTD",
		"SBILIFE", "SHRIRAMFIN", "SIEMENS", "SRF", "TATACOMM",
		"TATAPOWER", "TORNTPHARM", "TRENT", "VOLTAS", "ZOMATO",
		"ABB", "ALKEM", "AMBUJACEM", "ASTRAL", "AUROPHARMA",
	}
}

// GetNSESmallcap returns popular NSE smallcap stocks with growth potential
func GetNSESmallcap() []string {
	return []string{
		"ZYDUSLIFE", "CROMPTON", "MPHASIS", "JINDALSTEL", "INDHOTEL",
		"CHAMBLFERT", "GMRINFRA", "IDFCFIRSTB", "PFC", "IRFC",
		"NATIONALUM", "RBLBANK", "CONCOR", "BALKRISIND", "SUPREMEIND",
		"PIIND", "CUMMINSIND", "DEEPAKNTR", "FLUOROCHEM", "AARTIIND",
		"APLAPOLLO", "ASHOKLEY", "BATAINDIA", "CANFINHOME", "CREDITACC",
		"DELTACORP", "FINEORG", "GODREJAGRO", "GRINDWELL", "HFCL",
		"IIFL", "INTELLECT", "JKCEMENT", "JUBLPHARMA", "KPITTECH",
		"LALPATHLAB", "MANAPPURAM", "NATIONALUM", "NAVINFLUOR", "NHPC",
		"PAGEIND", "PFIZER", "POLYCAB", "RAMCOCEM", "RATNAMANI",
		"SOLARINDS", "SUMICHEM", "TATACHEM", "THYROCARE", "TORNTPOWER",
	}
}

// GetNSEBroadUniverse returns a comprehensive NSE universe for PEAD discovery
// This includes Nifty 50, Next 50, Midcap, and Smallcap stocks (~200+ stocks)
// Use this for discovering new opportunities with sudden earnings growth
// DEPRECATED: Use FetchRecentEarningsAnnouncements() for dynamic discovery
func GetNSEBroadUniverse() []string {
	universe := make([]string, 0, 250)

	// Add all segments
	universe = append(universe, GetNSETop50()...)
	universe = append(universe, GetNSENext50()...)
	universe = append(universe, GetNSEMidcap()...)
	universe = append(universe, GetNSESmallcap()...)

	// Remove duplicates
	seen := make(map[string]bool)
	unique := make([]string, 0, len(universe))
	for _, symbol := range universe {
		if !seen[symbol] {
			seen[symbol] = true
			unique = append(unique, symbol)
		}
	}

	return unique
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
	// https://www.nseindia.com/api/corporates-financial-results

	url := "https://www.nseindia.com/api/corporates-financial-results?index=equities"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NSE request: %w", err)
	}

	// NSE requires specific headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.nseindia.com/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("NSE API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NSE API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read NSE response: %w", err)
	}

	// Parse NSE response
	var data struct {
		Data []struct {
			Symbol  string `json:"symbol"`
			XDate   string `json:"xdate"` // Announcement date
			Purpose string `json:"purpose"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse NSE response: %w", err)
	}

	// Filter for recent earnings announcements
	cutoffDate := time.Now().AddDate(0, 0, -daysBack)
	symbols := make([]string, 0)

	for _, item := range data.Data {
		// Check if it's a financial results announcement
		if !strings.Contains(strings.ToLower(item.Purpose), "result") &&
			!strings.Contains(strings.ToLower(item.Purpose), "financial") {
			continue
		}

		// Parse date
		announcementDate, err := time.Parse("02-Jan-2006", item.XDate)
		if err != nil {
			continue
		}

		// Check if within our timeframe
		if announcementDate.After(cutoffDate) {
			symbols = append(symbols, item.Symbol)
		}
	}

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
		Quarter     string  `json:"quarter"`
		Sales       float64 `json:"sales"`
		NetProfit   float64 `json:"netProfit"`
		EPS         float64 `json:"eps"`
		Date        string  `json:"date"`
	} `json:"results"`
}

// fetchFromNSEAPI fetches data from NSE India's official API
// Note: NSE API requires proper headers and may have rate limits
func (n *NSEDataFetcher) fetchFromNSEAPI(ctx context.Context, symbol string) (*EarningsData, error) {
	// NSE Corporate API endpoint
	url := fmt.Sprintf("https://www.nseindia.com/api/quote-equity?symbol=%s", symbol)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// NSE requires these specific headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.nseindia.com/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from NSE: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NSE API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read NSE response: %w", err)
	}

	// Parse NSE response
	var nseData map[string]interface{}
	if err := json.Unmarshal(body, &nseData); err != nil {
		return nil, fmt.Errorf("failed to parse NSE response: %w", err)
	}

	// Extract earnings data from NSE response
	// This would need proper parsing based on actual NSE API response structure
	return nil, fmt.Errorf("NSE API parsing not yet implemented")
}
