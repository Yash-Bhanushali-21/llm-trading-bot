package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"llm-trading-bot/internal/interfaces"
)

// SEBIClient handles SEBI India API interactions for insider trading and regulatory data
type SEBIClient struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
}

// NewSEBIClient creates a new SEBI API client
func NewSEBIClient() *SEBIClient {
	return &SEBIClient{
		baseURL: "https://www.sebi.gov.in",
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
		headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"Accept":     "application/json, text/plain, */*",
		},
	}
}

// FetchInsiderTrading retrieves insider trading data from SEBI
func (s *SEBIClient) FetchInsiderTrading(ctx context.Context, symbol string, fromDate, toDate string) ([]interfaces.InsiderTradeData, error) {
	// SEBI PIT (Prohibition of Insider Trading) endpoint
	// Note: SEBI endpoints may change, this is based on common patterns
	url := fmt.Sprintf("%s/sebiweb/other/TrackinsidertradeAjax.do", s.baseURL)

	// Parse dates
	from, _ := time.Parse("2006-01-02", fromDate)
	to, _ := time.Parse("2006-01-02", toDate)

	data, err := s.makeRequest(ctx, url, map[string]string{
		"companyName": symbol,
		"fromDate":    from.Format("02/01/2006"),
		"toDate":      to.Format("02/01/2006"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SEBI insider trading: %w", err)
	}

	return s.parseInsiderTrades(data)
}

// FetchRegulatoryActions retrieves regulatory actions and orders
func (s *SEBIClient) FetchRegulatoryActions(ctx context.Context, symbol string) ([]interfaces.RegulatoryFiling, error) {
	// SEBI orders and notices endpoint
	url := fmt.Sprintf("%s/sebiweb/other/ORAjax.do", s.baseURL)

	data, err := s.makeRequest(ctx, url, map[string]string{
		"companyName": symbol,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch regulatory actions: %w", err)
	}

	return s.parseRegulatoryFilings(data)
}

// FetchAnnualReports retrieves annual report filings
func (s *SEBIClient) FetchAnnualReports(ctx context.Context, companyCode string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/sebiweb/other/AnnualReportAjax.do", s.baseURL)

	data, err := s.makeRequest(ctx, url, map[string]string{
		"company_code": companyCode,
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

func (s *SEBIClient) makeRequest(ctx context.Context, url string, params map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, err
	}

	// Add headers
	for key, value := range s.headers {
		req.Header.Set(key, value)
	}

	// Add query parameters
	q := req.URL.Query()
	for key, value := range params {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SEBI API returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (s *SEBIClient) parseInsiderTrades(data []byte) ([]interfaces.InsiderTradeData, error) {
	var rawData []map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		// Try alternative format
		var wrapper struct {
			Data []map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return nil, err
		}
		rawData = wrapper.Data
	}

	trades := []interfaces.InsiderTradeData{}
	for _, item := range rawData {
		// Parse date
		dateStr := getString(item, "tdpTransDate")
		if dateStr == "" {
			dateStr = getString(item, "acqDate")
		}

		// Parse transaction type
		transType := "BUY"
		if strings.Contains(strings.ToLower(getString(item, "typeofSecurity")), "sale") ||
			strings.Contains(strings.ToLower(getString(item, "acqMode")), "sale") {
			transType = "SELL"
		}

		// Parse quantity
		qtyStr := getString(item, "befAcqSharesNo")
		if qtyStr == "" {
			qtyStr = getString(item, "afterAcqSharesNo")
		}
		qty, _ := parseNumber(qtyStr)

		// Parse value
		valueStr := getString(item, "valueTrans")
		value, _ := parseNumber(valueStr)

		// Calculate price
		price := 0.0
		if qty > 0 && value > 0 {
			price = float64(value) / float64(qty)
		}

		trade := interfaces.InsiderTradeData{
			Date:            parseDateString(dateStr),
			Name:            getString(item, "personName"),
			Designation:     getString(item, "personCategory"),
			TransactionType: transType,
			Quantity:        qty,
			Value:           float64(value),
			Price:           price,
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

func (s *SEBIClient) parseRegulatoryFilings(data []byte) ([]interfaces.RegulatoryFiling, error) {
	var rawData []map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, err
	}

	filings := []interfaces.RegulatoryFiling{}
	for _, item := range rawData {
		filing := interfaces.RegulatoryFiling{
			Date:        parseDateString(getString(item, "orderDate")),
			FilingType:  getString(item, "orderType"),
			Description: getString(item, "subject"),
			URL:         getString(item, "pdfURL"),
		}

		filings = append(filings, filing)
	}

	return filings, nil
}

func parseNumber(s string) (int64, error) {
	// Remove commas and spaces
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.TrimSpace(s)

	if s == "" || s == "-" {
		return 0, nil
	}

	return strconv.ParseInt(s, 10, 64)
}

func parseDateString(s string) string {
	// Try multiple date formats
	formats := []string{
		"02-01-2006",
		"02/01/2006",
		"2006-01-02",
		"02-Jan-2006",
		"02-January-2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t.Format("2006-01-02")
		}
	}

	return time.Now().Format("2006-01-02")
}
