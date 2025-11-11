package pead

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Analyzer performs PEAD analysis on a universe of stocks
type Analyzer struct {
	config  PEADConfig
	fetcher EarningsDataFetcher
	scorer  *PEADScorer
}

// NewAnalyzer creates a new PEAD analyzer
func NewAnalyzer(config PEADConfig, fetcher EarningsDataFetcher) *Analyzer {
	return &Analyzer{
		config:  config,
		fetcher: fetcher,
		scorer:  NewPEADScorer(config),
	}
}

// Analyze performs complete PEAD analysis on a list of symbols
func (a *Analyzer) Analyze(ctx context.Context, symbols []string) (*PEADResult, error) {
	// Fetch earnings data for all symbols
	earningsData, err := a.fetcher.FetchLatestEarnings(ctx, symbols)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch earnings data: %w", err)
	}

	// Score each company
	scores := make([]*PEADScore, 0, len(symbols))
	for _, data := range earningsData {
		if data == nil {
			continue
		}

		score := a.scorer.CalculateScore(data)
		scores = append(scores, score)
	}

	// Filter based on criteria
	qualified := a.filterQualified(scores)

	// Sort by composite score (descending)
	sort.Slice(qualified, func(i, j int) bool {
		return qualified[i].CompositeScore > qualified[j].CompositeScore
	})

	result := &PEADResult{
		AnalysisDate:     time.Now(),
		TotalAnalyzed:    len(scores),
		QualifiedCount:   len(qualified),
		QualifiedSymbols: qualified,
		Config:           a.config,
	}

	return result, nil
}

// filterQualified filters companies based on configuration thresholds
func (a *Analyzer) filterQualified(scores []*PEADScore) []PEADScore {
	qualified := make([]PEADScore, 0)

	for _, score := range scores {
		if a.meetsQualificationCriteria(score) {
			qualified = append(qualified, *score)
		}
	}

	return qualified
}

// meetsQualificationCriteria checks if a score meets all qualification criteria
func (a *Analyzer) meetsQualificationCriteria(score *PEADScore) bool {
	// Check composite score threshold
	if score.CompositeScore < a.config.MinCompositeScore {
		return false
	}

	// Check PEAD time window
	if score.DaysSinceEarnings < a.config.MinDaysSinceEarnings {
		return false
	}
	if score.DaysSinceEarnings > a.config.MaxDaysSinceEarnings {
		return false
	}

	// Check minimum earnings surprise
	if score.EarningsData.EarningSurprise() < a.config.MinEarningsSurprise {
		return false
	}

	// Check minimum EPS growth
	if score.EarningsData.YoYEPSGrowth < a.config.MinEPSGrowth {
		return false
	}

	// Check minimum revenue growth
	if score.EarningsData.YoYRevenueGrowth < a.config.MinRevenueGrowth {
		return false
	}

	return true
}

// AnalyzeSymbol performs detailed analysis on a single symbol
func (a *Analyzer) AnalyzeSymbol(ctx context.Context, symbol string) (*PEADScore, error) {
	// Fetch latest earnings
	earningsData, err := a.fetcher.FetchLatestEarnings(ctx, []string{symbol})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch earnings for %s: %w", symbol, err)
	}

	data, exists := earningsData[symbol]
	if !exists || data == nil {
		return nil, fmt.Errorf("no earnings data found for %s", symbol)
	}

	// Calculate score
	score := a.scorer.CalculateScore(data)

	return score, nil
}

// GetTopPicks returns the top N qualified symbols by composite score
func (a *Analyzer) GetTopPicks(ctx context.Context, symbols []string, topN int) ([]PEADScore, error) {
	result, err := a.Analyze(ctx, symbols)
	if err != nil {
		return nil, err
	}

	// Return top N (already sorted by score)
	if len(result.QualifiedSymbols) <= topN {
		return result.QualifiedSymbols, nil
	}

	return result.QualifiedSymbols[:topN], nil
}

// GetDefaultConfig returns a sensible default PEAD configuration
func GetDefaultConfig() PEADConfig {
	return PEADConfig{
		MinDaysSinceEarnings: 1,
		MaxDaysSinceEarnings: 60, // PEAD typically occurs within 60 days
		MinCompositeScore:    40,  // Changed to default 40 (user can override in .env)
		EnableNLP:            false, // NLP disabled by default
		Weights: ScoringWeights{
			// Traditional PEAD weights (sum = 1.0)
			EarningsSurprise:    0.25,
			RevenueSurprise:     0.15,
			EarningsGrowth:      0.20,
			RevenueGrowth:       0.15,
			MarginExpansion:     0.10,
			Consistency:         0.10,
			RevenueAcceleration: 0.05,
			// NLP weights (0.0 by default)
			Sentiment:           0.00,
			ToneDivergence:      0.00,
			LinguisticQuality:   0.00,
		},
		MinEarningsSurprise: 0,    // No minimum by default (can be negative)
		MinRevenueGrowth:    -10,  // Allow up to -10% revenue decline
		MinEPSGrowth:        0,    // Minimum 0% EPS growth (no decline)
		DataSource:          "LIVE",
	}
}
