package forensic

import (
	"context"
	"fmt"
	"time"

	"llm-trading-bot/internal/interfaces"
)

// MockDataSource provides mock data for testing
type MockDataSource struct {
	symbol string
}

// NewMockDataSource creates a new mock data source
func NewMockDataSource() *MockDataSource {
	return &MockDataSource{}
}

// FetchAnnouncements returns mock announcements
func (m *MockDataSource) FetchAnnouncements(ctx context.Context, symbol string, fromDate, toDate string) ([]interfaces.Announcement, error) {
	m.symbol = symbol

	announcements := []interfaces.Announcement{
		{
			Date:        time.Now().AddDate(0, 0, -15).Format("2006-01-02"),
			Subject:     "Resignation of Chief Financial Officer",
			Category:    "Management",
			Description: "Mr. John Doe has resigned from the position of CFO with immediate effect due to personal reasons. The company is in the process of identifying a successor.",
			AttachURL:   "",
		},
		{
			Date:        time.Now().AddDate(0, 0, -45).Format("2006-01-02"),
			Subject:     "Change in Statutory Auditors",
			Category:    "Auditor",
			Description: "The company has appointed XYZ & Associates as statutory auditors replacing ABC & Co. The change is due to completion of tenure. The previous auditor had given a qualified opinion on inventory valuation.",
			AttachURL:   "",
		},
		{
			Date:        time.Now().AddDate(0, 0, -30).Format("2006-01-02"),
			Subject:     "Related Party Transaction - Material",
			Category:    "Related Party",
			Description: "The company has entered into a sale transaction with promoter group entity ABC Ltd for Rs. 50 crore. The transaction exceeds materiality threshold and has been approved by the audit committee.",
			AttachURL:   "",
		},
		{
			Date:        time.Now().AddDate(0, 0, -20).Format("2006-01-02"),
			Subject:     "Pledging of Promoter Shares",
			Category:    "Shareholding",
			Description: "Promoter Mr. Rajesh Kumar has pledged additional 15% of his shareholding, bringing total pledge to 65% of his holdings.",
			AttachURL:   "",
		},
		{
			Date:        time.Now().AddDate(0, 0, -60).Format("2006-01-02"),
			Subject:     "SEBI Penalty",
			Category:    "Regulatory",
			Description: "The company has received a penalty of Rs. 2 crore from SEBI for delay in disclosure of material events. The company has paid the penalty and strengthened its compliance systems.",
			AttachURL:   "",
		},
		{
			Date:        time.Now().AddDate(0, 0, -10).Format("2006-01-02"),
			Subject:     "Revision of Financial Results",
			Category:    "Financial",
			Description: "The company has restated its Q2 FY24 results due to an accounting error in revenue recognition. Revenue has been revised from Rs. 500 crore to Rs. 480 crore (4% reduction).",
			AttachURL:   "",
		},
	}

	return announcements, nil
}

// FetchShareholdingPattern returns mock shareholding data
func (m *MockDataSource) FetchShareholdingPattern(ctx context.Context, symbol string) (*interfaces.ShareholdingPattern, error) {
	return &interfaces.ShareholdingPattern{
		AsOfDate:        time.Now().AddDate(0, -1, 0).Format("2006-01-02"),
		PromoterHolding: 51.5,
		PublicHolding:   48.5,
		PromoterPledged: 33.2,
		PromoterDetails: []interfaces.PromoterDetail{
			{
				Name:              "Mr. Rajesh Kumar",
				SharesHeld:        5000000,
				PercentageHolding: 25.0,
				SharesPledged:     3250000,
				PledgePercentage:  65.0,
			},
			{
				Name:              "Mrs. Priya Kumar",
				SharesHeld:        3000000,
				PercentageHolding: 15.0,
				SharesPledged:     900000,
				PledgePercentage:  30.0,
			},
			{
				Name:              "Kumar Family Trust",
				SharesHeld:        2300000,
				PercentageHolding: 11.5,
				SharesPledged:     0,
				PledgePercentage:  0,
			},
		},
	}, nil
}

// FetchInsiderTrades returns mock insider trading data
func (m *MockDataSource) FetchInsiderTrades(ctx context.Context, symbol string, fromDate, toDate string) ([]interfaces.InsiderTradeData, error) {
	trades := []interfaces.InsiderTradeData{
		{
			Date:            time.Now().AddDate(0, 0, -5).Format("2006-01-02"),
			Name:            "Mr. Amit Sharma",
			Designation:     "CEO",
			TransactionType: "SELL",
			Quantity:        100000,
			Value:           15000000,
			Price:           150.0,
		},
		{
			Date:            time.Now().AddDate(0, 0, -7).Format("2006-01-02"),
			Name:            "Mrs. Sunita Verma",
			Designation:     "Director",
			TransactionType: "SELL",
			Quantity:        50000,
			Value:           7500000,
			Price:           150.0,
		},
		{
			Date:            time.Now().AddDate(0, 0, -8).Format("2006-01-02"),
			Name:            "Mr. Vijay Singh",
			Designation:     "CFO",
			TransactionType: "SELL",
			Quantity:        75000,
			Value:           11250000,
			Price:           150.0,
		},
		{
			Date:            time.Now().AddDate(0, 0, -40).Format("2006-01-02"),
			Name:            "Mr. Rahul Gupta",
			Designation:     "VP Operations",
			TransactionType: "BUY",
			Quantity:        10000,
			Value:           1400000,
			Price:           140.0,
		},
	}

	return trades, nil
}

// FetchFinancials returns mock financial data
func (m *MockDataSource) FetchFinancials(ctx context.Context, symbol string, period string) (*interfaces.FinancialData, error) {
	return &interfaces.FinancialData{
		Period:      "Q3FY24",
		Revenue:     50000000000,  // 50000 Cr
		Profit:      5000000000,   // 5000 Cr
		Expenses:    45000000000,  // 45000 Cr
		Assets:      100000000000, // 100000 Cr
		Liabilities: 60000000000,  // 60000 Cr
		IsRestated:  false,
	}, nil
}

// FetchRegulatoryFilings returns mock regulatory filings
func (m *MockDataSource) FetchRegulatoryFilings(ctx context.Context, symbol string, fromDate, toDate string) ([]interfaces.RegulatoryFiling, error) {
	filings := []interfaces.RegulatoryFiling{
		{
			Date:        time.Now().AddDate(0, 0, -35).Format("2006-01-02"),
			FilingType:  "Show Cause Notice",
			Description: "Response to SEBI show cause notice regarding delayed disclosure of material information",
			URL:         fmt.Sprintf("https://example.com/filings/%s/scn.pdf", symbol),
		},
		{
			Date:        time.Now().AddDate(0, 0, -90).Format("2006-01-02"),
			FilingType:  "Compliance Certificate",
			Description: "Quarterly compliance certificate submitted to stock exchanges",
			URL:         fmt.Sprintf("https://example.com/filings/%s/compliance.pdf", symbol),
		},
	}

	return filings, nil
}
