package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"llm-trading-bot/internal/interfaces"
)

// BSEClient handles BSE India API interactions
type BSEClient struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
}

// NewBSEClient creates a new BSE API client
func NewBSEClient() *BSEClient {
	return &BSEClient{
		baseURL: "https://api.bseindia.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"Accept":     "application/json",
		},
	}
}

// FetchAnnouncements retrieves corporate announcements from BSE
func (b *BSEClient) FetchAnnouncements(ctx context.Context, scrip string, fromDate, toDate string) ([]interfaces.Announcement, error) {
	// BSE uses scrip code instead of symbol
	url := fmt.Sprintf("%s/BseIndiaAPI/api/AnnSubCategoryGetData/w", b.baseURL)

	// Note: BSE API requires proper authentication and may need API keys
	// This is a simplified version
	data, err := b.makeRequest(ctx, url, map[string]string{
		"scripcode": scrip,
		"fromdate":  fromDate,
		"todate":    toDate,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch BSE announcements: %w", err)
	}

	return b.parseAnnouncements(data)
}

// FetchCorporateActions retrieves corporate actions from BSE
func (b *BSEClient) FetchCorporateActions(ctx context.Context, scrip string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/BseIndiaAPI/api/ComHeader/w", b.baseURL)

	data, err := b.makeRequest(ctx, url, map[string]string{
		"quotetype": "EQ",
		"scripcode": scrip,
	})
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (b *BSEClient) makeRequest(ctx context.Context, url string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, err
	}

	// Add headers
	for key, value := range b.headers {
		req.Header.Set(key, value)
	}

	// Add query parameters (for POST, BSE uses form data)
	q := req.URL.Query()
	for key, value := range params {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BSE API returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (b *BSEClient) parseAnnouncements(data []byte) ([]interfaces.Announcement, error) {
	var rawData struct {
		Table []map[string]interface{} `json:"Table"`
	}

	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, err
	}

	announcements := []interfaces.Announcement{}
	for _, item := range rawData.Table {
		announcement := interfaces.Announcement{
			Date:        getString(item, "NEWS_DT"),
			Subject:     getString(item, "NEWSSUB"),
			Category:    getString(item, "SUBCATCODE"),
			Description: getString(item, "NEWS"),
			AttachURL:   getString(item, "ATTACHMENTNAME"),
		}

		announcements = append(announcements, announcement)
	}

	return announcements, nil
}

// SymbolToScripCode converts symbol to BSE scrip code
// In production, this should query a mapping database or API
func SymbolToScripCode(symbol string) string {
	// Common mappings (this should be in a database)
	mapping := map[string]string{
		"RELIANCE": "500325",
		"TCS":      "532540",
		"INFY":     "500209",
		"HDFCBANK": "500180",
		"ICICIBANK": "532174",
	}

	if scrip, ok := mapping[symbol]; ok {
		return scrip
	}

	// Return symbol as fallback
	return symbol
}
