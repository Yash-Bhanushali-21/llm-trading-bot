package pead

import "time"

// EarningsData represents a company's quarterly earnings report
type EarningsData struct {
	Symbol           string    `json:"symbol"`
	Quarter          string    `json:"quarter"` // e.g., "Q1 2024"
	FiscalYear       int       `json:"fiscal_year"`
	AnnouncementDate time.Time `json:"announcement_date"`

	// Earnings metrics
	ActualEPS    float64 `json:"actual_eps"`
	ExpectedEPS  float64 `json:"expected_eps"`
	ActualRevenue float64 `json:"actual_revenue"`
	ExpectedRevenue float64 `json:"expected_revenue"`

	// Growth metrics (Year-over-Year)
	YoYEPSGrowth     float64 `json:"yoy_eps_growth"`      // in percentage
	YoYRevenueGrowth float64 `json:"yoy_revenue_growth"`  // in percentage

	// Quarter-over-Quarter growth
	QoQEPSGrowth     float64 `json:"qoq_eps_growth"`      // in percentage
	QoQRevenueGrowth float64 `json:"qoq_revenue_growth"`  // in percentage

	// Profit margins
	GrossMargin      float64 `json:"gross_margin"`        // in percentage
	OperatingMargin  float64 `json:"operating_margin"`    // in percentage
	NetMargin        float64 `json:"net_margin"`          // in percentage

	// Previous period comparison
	PrevGrossMargin     float64 `json:"prev_gross_margin"`
	PrevOperatingMargin float64 `json:"prev_operating_margin"`
	PrevNetMargin       float64 `json:"prev_net_margin"`

	// Beat streak (historical context)
	ConsecutiveBeats int `json:"consecutive_beats"` // Number of consecutive quarters beating estimates
}

// EarningSurprise calculates the earnings surprise percentage
func (e *EarningsData) EarningSurprise() float64 {
	if e.ExpectedEPS == 0 {
		return 0
	}
	return ((e.ActualEPS - e.ExpectedEPS) / abs(e.ExpectedEPS)) * 100
}

// RevenueSurprise calculates the revenue surprise percentage
func (e *EarningsData) RevenueSurprise() float64 {
	if e.ExpectedRevenue == 0 {
		return 0
	}
	return ((e.ActualRevenue - e.ExpectedRevenue) / e.ExpectedRevenue) * 100
}

// GrossMarginChange calculates change in gross margin
func (e *EarningsData) GrossMarginChange() float64 {
	return e.GrossMargin - e.PrevGrossMargin
}

// OperatingMarginChange calculates change in operating margin
func (e *EarningsData) OperatingMarginChange() float64 {
	return e.OperatingMargin - e.PrevOperatingMargin
}

// NetMarginChange calculates change in net margin
func (e *EarningsData) NetMarginChange() float64 {
	return e.NetMargin - e.PrevNetMargin
}

// PEADScore represents the complete analysis and score for a company
type PEADScore struct {
	Symbol           string    `json:"symbol"`
	Quarter          string    `json:"quarter"`
	AnnouncementDate time.Time `json:"announcement_date"`
	DaysSinceEarnings int      `json:"days_since_earnings"`

	// Individual component scores (0-100)
	EarningsSurpriseScore    float64 `json:"earnings_surprise_score"`
	RevenueSurpriseScore     float64 `json:"revenue_surprise_score"`
	EarningsGrowthScore      float64 `json:"earnings_growth_score"`
	RevenueGrowthScore       float64 `json:"revenue_growth_score"`
	MarginExpansionScore     float64 `json:"margin_expansion_score"`
	ConsistencyScore         float64 `json:"consistency_score"`
	RevenueAccelerationScore float64 `json:"revenue_acceleration_score"`

	// Overall composite score (0-100)
	CompositeScore float64 `json:"composite_score"`

	// Underlying data
	EarningsData EarningsData `json:"earnings_data"`

	// Qualitative assessment
	Rating     string `json:"rating"` // "STRONG_BUY", "BUY", "HOLD", "AVOID"
	Commentary string `json:"commentary"`
}

// PEADConfig holds configuration for PEAD analysis
type PEADConfig struct {
	// Enabled flag
	Enabled bool `yaml:"enabled"`

	// Minimum days since earnings announcement
	MinDaysSinceEarnings int `yaml:"min_days_since_earnings"`

	// Maximum days since earnings announcement (PEAD window)
	MaxDaysSinceEarnings int `yaml:"max_days_since_earnings"`

	// Minimum composite score threshold (0-100)
	MinCompositeScore float64 `yaml:"min_composite_score"`

	// Component weights (should sum to 1.0)
	Weights ScoringWeights `yaml:"weights"`

	// Minimum thresholds for individual metrics
	MinEarningsSurprise float64 `yaml:"min_earnings_surprise"` // in percentage
	MinRevenueGrowth    float64 `yaml:"min_revenue_growth"`    // in percentage
	MinEPSGrowth        float64 `yaml:"min_eps_growth"`        // in percentage

	// Data source configuration
	DataSource string `yaml:"data_source"`
	APIKeyEnv  string `yaml:"api_key_env"`
}

// ScoringWeights defines the weights for different scoring components
type ScoringWeights struct {
	EarningsSurprise    float64 `yaml:"earnings_surprise"`
	RevenueSurprise     float64 `yaml:"revenue_surprise"`
	EarningsGrowth      float64 `yaml:"earnings_growth"`
	RevenueGrowth       float64 `yaml:"revenue_growth"`
	MarginExpansion     float64 `yaml:"margin_expansion"`
	Consistency         float64 `yaml:"consistency"`
	RevenueAcceleration float64 `yaml:"revenue_acceleration"`
}

// PEADResult represents the final filtered results
type PEADResult struct {
	AnalysisDate     time.Time   `json:"analysis_date"`
	TotalAnalyzed    int         `json:"total_analyzed"`
	QualifiedCount   int         `json:"qualified_count"`
	QualifiedSymbols []PEADScore `json:"qualified_symbols"`
	Config           PEADConfig  `json:"config"`
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
