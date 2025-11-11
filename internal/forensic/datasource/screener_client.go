package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"llm-trading-bot/internal/interfaces"
)

// ScreenerClient handles Screener.in API interactions
// Screener.in aggregates financial data, shareholding patterns, and more
type ScreenerClient struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
}

// NewScreenerClient creates a new Screener.in client
func NewScreenerClient() *ScreenerClient {
	return &ScreenerClient{
		baseURL: "https://www.screener.in",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"Accept":     "text/html,application/json",
		},
	}
}

// FetchCompanyData retrieves comprehensive company data
func (sc *ScreenerClient) FetchCompanyData(ctx context.Context, symbol string) (map[string]interface{}, error) {
	// Screener uses company name/code in URL
	companySlug := strings.ToLower(symbol)
	url := fmt.Sprintf("%s/api/company/%s/", sc.baseURL, companySlug)

	data, err := sc.makeRequest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch company data: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// FetchShareholdingPattern retrieves shareholding pattern
func (sc *ScreenerClient) FetchShareholdingPattern(ctx context.Context, symbol string) (*interfaces.ShareholdingPattern, error) {
	companySlug := strings.ToLower(symbol)
	url := fmt.Sprintf("%s/company/%s/", sc.baseURL, companySlug)

	data, err := sc.makeRequest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch shareholding: %w", err)
	}

	return sc.parseShareholdingFromHTML(data)
}

// FetchFinancials retrieves financial statements
func (sc *ScreenerClient) FetchFinancials(ctx context.Context, symbol string, period string) (*interfaces.FinancialData, error) {
	companyData, err := sc.FetchCompanyData(ctx, symbol)
	if err != nil {
		return nil, err
	}

	return sc.parseFinancials(companyData, period)
}

// FetchPeerComparison retrieves peer comparison data
func (sc *ScreenerClient) FetchPeerComparison(ctx context.Context, symbol string) ([]map[string]interface{}, error) {
	companySlug := strings.ToLower(symbol)
	url := fmt.Sprintf("%s/api/company/%s/peers/", sc.baseURL, companySlug)

	data, err := sc.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (sc *ScreenerClient) makeRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range sc.headers {
		req.Header.Set(key, value)
	}

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Screener API returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (sc *ScreenerClient) parseShareholdingFromHTML(data []byte) (*interfaces.ShareholdingPattern, error) {
	html := string(data)

	pattern := &interfaces.ShareholdingPattern{
		AsOfDate:        time.Now().Format("2006-01-02"),
		PromoterDetails: []interfaces.PromoterDetail{},
	}

	// Extract promoter holding percentage
	promoterRegex := regexp.MustCompile(`Promoter.*?(\d+\.?\d*)%`)
	if matches := promoterRegex.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			pattern.PromoterHolding = val
		}
	}

	// Extract public holding
	publicRegex := regexp.MustCompile(`Public.*?(\d+\.?\d*)%`)
	if matches := publicRegex.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			pattern.PublicHolding = val
		}
	}

	// Extract promoter pledge percentage
	pledgeRegex := regexp.MustCompile(`Pledge.*?(\d+\.?\d*)%`)
	if matches := pledgeRegex.FindStringSubmatch(html); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			pattern.PromoterPledged = val
		}
	}

	return pattern, nil
}

func (sc *ScreenerClient) parseFinancials(data map[string]interface{}, period string) (*interfaces.FinancialData, error) {
	financials := &interfaces.FinancialData{
		Period:     period,
		IsRestated: false,
	}

	// Extract from company data structure
	if finances, ok := data["financials"].(map[string]interface{}); ok {
		// Revenue
		if revenue, ok := finances["sales"].([]interface{}); ok && len(revenue) > 0 {
			if val, ok := revenue[0].(float64); ok {
				financials.Revenue = val
			}
		}

		// Profit
		if profit, ok := finances["net_profit"].([]interface{}); ok && len(profit) > 0 {
			if val, ok := profit[0].(float64); ok {
				financials.Profit = val
			}
		}

		// Expenses (derived from revenue - profit)
		financials.Expenses = financials.Revenue - financials.Profit

		// Assets
		if assets, ok := finances["total_assets"].([]interface{}); ok && len(assets) > 0 {
			if val, ok := assets[0].(float64); ok {
				financials.Assets = val
			}
		}

		// Liabilities
		if liabilities, ok := finances["total_liabilities"].([]interface{}); ok && len(liabilities) > 0 {
			if val, ok := liabilities[0].(float64); ok {
				financials.Liabilities = val
			}
		}
	}

	return financials, nil
}

// SearchCompany searches for companies by name/symbol
func (sc *ScreenerClient) SearchCompany(ctx context.Context, query string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/search/?q=%s", sc.baseURL, query)

	data, err := sc.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetCompanyID retrieves the internal company ID for a symbol
func (sc *ScreenerClient) GetCompanyID(ctx context.Context, symbol string) (string, error) {
	results, err := sc.SearchCompany(ctx, symbol)
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "", fmt.Errorf("company not found: %s", symbol)
	}

	// Return the slug/ID of the first match
	if id, ok := results[0]["url"].(string); ok {
		return strings.TrimPrefix(id, "/company/"), nil
	}

	return symbol, nil
}
