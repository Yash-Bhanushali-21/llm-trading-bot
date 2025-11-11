package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"llm-trading-bot/internal/interfaces"
)

// NSEClient handles NSE India API interactions
type NSEClient struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
}

// NewNSEClient creates a new NSE API client
func NewNSEClient() *NSEClient {
	return &NSEClient{
		baseURL: "https://www.nseindia.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		headers: map[string]string{
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"Accept":          "*/*",
			"Accept-Language": "en-US,en;q=0.9",
		},
	}
}

// FetchAnnouncements retrieves corporate announcements from NSE
func (n *NSEClient) FetchAnnouncements(ctx context.Context, symbol string, fromDate, toDate string) ([]interfaces.Announcement, error) {
	// NSE corporate announcements endpoint
	url := fmt.Sprintf("%s/api/corporates-corporateActions?index=equities&symbol=%s", n.baseURL, symbol)

	data, err := n.makeRequest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NSE announcements: %w", err)
	}

	return n.parseAnnouncements(data, fromDate, toDate)
}

// FetchCorporateActions retrieves corporate actions like dividends, splits, etc.
func (n *NSEClient) FetchCorporateActions(ctx context.Context, symbol string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/corporate-announcements?index=equities&symbol=%s", n.baseURL, symbol)

	data, err := n.makeRequest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch corporate actions: %w", err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// FetchShareholdingPattern retrieves shareholding pattern from NSE
func (n *NSEClient) FetchShareholdingPattern(ctx context.Context, symbol string) (*interfaces.ShareholdingPattern, error) {
	url := fmt.Sprintf("%s/api/quote-equity?symbol=%s", n.baseURL, symbol)

	data, err := n.makeRequest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch shareholding pattern: %w", err)
	}

	return n.parseShareholdingPattern(data)
}

func (n *NSEClient) makeRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add headers
	for key, value := range n.headers {
		req.Header.Set(key, value)
	}

	// First, make a request to get cookies (NSE requires session)
	homeReq, _ := http.NewRequestWithContext(ctx, "GET", n.baseURL, nil)
	for key, value := range n.headers {
		homeReq.Header.Set(key, value)
	}
	_, _ = n.httpClient.Do(homeReq)

	// Now make the actual request
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NSE API returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (n *NSEClient) parseAnnouncements(data []byte, fromDate, toDate string) ([]interfaces.Announcement, error) {
	var rawData []map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, err
	}

	from, _ := time.Parse("2006-01-02", fromDate)
	to, _ := time.Parse("2006-01-02", toDate)

	announcements := []interfaces.Announcement{}
	for _, item := range rawData {
		dateStr, ok := item["date"].(string)
		if !ok {
			continue
		}

		annDate, err := time.Parse("02-Jan-2006", dateStr)
		if err != nil {
			continue
		}

		if annDate.Before(from) || annDate.After(to) {
			continue
		}

		announcement := interfaces.Announcement{
			Date:        annDate.Format("2006-01-02"),
			Subject:     getString(item, "subject"),
			Category:    getString(item, "series"),
			Description: getString(item, "desc"),
			AttachURL:   getString(item, "attchmntFile"),
		}

		announcements = append(announcements, announcement)
	}

	return announcements, nil
}

func (n *NSEClient) parseShareholdingPattern(data []byte) (*interfaces.ShareholdingPattern, error) {
	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, err
	}

	// Extract shareholding data from NSE response
	pattern := &interfaces.ShareholdingPattern{
		AsOfDate:        time.Now().Format("2006-01-02"),
		PromoterHolding: 0,
		PublicHolding:   0,
		PromoterPledged: 0,
		PromoterDetails: []interfaces.PromoterDetail{},
	}

	// NSE stores shareholding in "shareholdingData" key
	if shareholding, ok := rawData["shareholdingData"].(map[string]interface{}); ok {
		if promoter, ok := shareholding["promoter"].(float64); ok {
			pattern.PromoterHolding = promoter
		}
		if public, ok := shareholding["public"].(float64); ok {
			pattern.PublicHolding = public
		}
	}

	return pattern, nil
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// SearchSymbol searches for a symbol on NSE
func (n *NSEClient) SearchSymbol(ctx context.Context, query string) ([]string, error) {
	url := fmt.Sprintf("%s/api/search/autocomplete?q=%s", n.baseURL, query)

	data, err := n.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var result struct {
		Symbols []struct {
			Symbol string `json:"symbol"`
		} `json:"symbols"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	symbols := make([]string, 0, len(result.Symbols))
	for _, s := range result.Symbols {
		symbols = append(symbols, s.Symbol)
	}

	return symbols, nil
}

// GetSymbolInfo retrieves detailed info about a symbol
func (n *NSEClient) GetSymbolInfo(ctx context.Context, symbol string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/quote-equity?symbol=%s", n.baseURL, symbol)

	data, err := n.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// NormalizeSymbol converts various symbol formats to NSE format
func NormalizeSymbol(symbol string) string {
	// Remove exchange suffix if present
	symbol = strings.TrimSuffix(symbol, ".NS")
	symbol = strings.TrimSuffix(symbol, ".BO")
	return strings.ToUpper(strings.TrimSpace(symbol))
}
