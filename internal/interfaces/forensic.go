package interfaces

import (
	"context"

	"llm-trading-bot/internal/types"
)

// ForensicChecker analyzes corporate governance and identifies red flags
type ForensicChecker interface {
	// Analyze performs comprehensive forensic analysis for a symbol
	Analyze(ctx context.Context, symbol string) (*types.ForensicReport, error)

	// CheckManagementChanges detects management changes and resignations
	CheckManagementChanges(ctx context.Context, symbol string) ([]types.ManagementChange, error)

	// CheckAuditorChanges detects auditor changes or qualifications
	CheckAuditorChanges(ctx context.Context, symbol string) ([]types.AuditorChange, error)

	// CheckRelatedPartyTxns analyzes related party transactions
	CheckRelatedPartyTxns(ctx context.Context, symbol string) ([]types.RelatedPartyTxn, error)

	// CheckPromoterPledges tracks pledge of promoter shares
	CheckPromoterPledges(ctx context.Context, symbol string) ([]types.PromoterPledge, error)

	// CheckRegulatoryActions monitors regulatory actions and penalties
	CheckRegulatoryActions(ctx context.Context, symbol string) ([]types.RegulatoryAction, error)

	// CheckInsiderTrading analyzes insider trading patterns
	CheckInsiderTrading(ctx context.Context, symbol string) ([]types.InsiderTrade, error)

	// CheckRestatements detects financial restatements
	CheckRestatements(ctx context.Context, symbol string) ([]types.FinancialRestatement, error)

	// CheckGovernanceScore monitors governance score degradation
	CheckGovernanceScore(ctx context.Context, symbol string) ([]types.GovernanceScore, error)

	// CalculateRiskScore computes overall risk score from all checks
	CalculateRiskScore(report *types.ForensicReport) float64
}

// CorporateDataSource provides access to corporate announcements and filings
type CorporateDataSource interface {
	// FetchAnnouncements retrieves corporate announcements for a symbol
	FetchAnnouncements(ctx context.Context, symbol string, fromDate, toDate string) ([]Announcement, error)

	// FetchShareholdingPattern retrieves shareholding pattern
	FetchShareholdingPattern(ctx context.Context, symbol string) (*ShareholdingPattern, error)

	// FetchInsiderTrades retrieves insider trading data
	FetchInsiderTrades(ctx context.Context, symbol string, fromDate, toDate string) ([]InsiderTradeData, error)

	// FetchFinancials retrieves financial statements
	FetchFinancials(ctx context.Context, symbol string, period string) (*FinancialData, error)

	// FetchRegulatoryFilings retrieves regulatory filings
	FetchRegulatoryFilings(ctx context.Context, symbol string, fromDate, toDate string) ([]RegulatoryFiling, error)
}

// Announcement represents a corporate announcement
type Announcement struct {
	Date        string
	Subject     string
	Category    string
	Description string
	AttachURL   string
}

// ShareholdingPattern represents shareholding data
type ShareholdingPattern struct {
	AsOfDate          string
	PromoterHolding   float64
	PublicHolding     float64
	PromoterPledged   float64
	PromoterDetails   []PromoterDetail
}

type PromoterDetail struct {
	Name              string
	SharesHeld        int64
	PercentageHolding float64
	SharesPledged     int64
	PledgePercentage  float64
}

// InsiderTradeData represents insider trading transaction
type InsiderTradeData struct {
	Date            string
	Name            string
	Designation     string
	TransactionType string
	Quantity        int64
	Value           float64
	Price           float64
}

// FinancialData represents financial statement data
type FinancialData struct {
	Period     string
	Revenue    float64
	Profit     float64
	Expenses   float64
	Assets     float64
	Liabilities float64
	IsRestated bool
}

// RegulatoryFiling represents a regulatory filing
type RegulatoryFiling struct {
	Date        string
	FilingType  string
	Description string
	URL         string
}
