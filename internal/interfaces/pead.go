package interfaces

import (
	"context"

	"llm-trading-bot/internal/research/pead"
)

// PEADAnalyzer defines the interface for PEAD (Post-Earnings Announcement Drift) analysis
type PEADAnalyzer interface {
	// Analyze performs PEAD analysis on a list of symbols
	Analyze(ctx context.Context, symbols []string) (*pead.PEADResult, error)

	// AnalyzeSymbol performs detailed analysis on a single symbol
	AnalyzeSymbol(ctx context.Context, symbol string) (*pead.PEADScore, error)

	// GetTopPicks returns the top N qualified symbols by composite score
	GetTopPicks(ctx context.Context, symbols []string, topN int) ([]pead.PEADScore, error)
}
