package peadobs

import (
	"context"
	"time"

	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/research/pead"
	"llm-trading-bot/internal/trace"
)

// observableAnalyzer wraps PEADAnalyzer with logging and tracing
type observableAnalyzer struct {
	inner interfaces.PEADAnalyzer
}

// Wrap wraps a PEADAnalyzer with observability middleware
func Wrap(analyzer interfaces.PEADAnalyzer) interfaces.PEADAnalyzer {
	return &observableAnalyzer{inner: analyzer}
}

// Analyze wraps the Analyze method with logging and tracing
func (o *observableAnalyzer) Analyze(ctx context.Context, symbols []string) (*pead.PEADResult, error) {
	ctx, span := trace.StartSpan(ctx, "pead.Analyze")
	defer span.End()

	fields := trace.GetTraceFields(ctx)
	fields["symbol_count"] = len(symbols)

	logger.InfoSkip(ctx, 1, "Starting PEAD analysis", fields)
	start := time.Now()

	result, err := o.inner.Analyze(ctx, symbols)

	duration := time.Since(start)
	fields["duration_ms"] = duration.Milliseconds()

	if err != nil {
		fields["error"] = err.Error()
		logger.ErrorSkip(ctx, 1, "PEAD analysis failed", fields)
		span.RecordError(err)
		return nil, err
	}

	fields["total_analyzed"] = result.TotalAnalyzed
	fields["qualified_count"] = result.QualifiedCount
	fields["qualification_rate"] = float64(result.QualifiedCount) / float64(result.TotalAnalyzed) * 100

	logger.InfoSkip(ctx, 1, "PEAD analysis completed", fields)

	return result, nil
}

// AnalyzeSymbol wraps the AnalyzeSymbol method with logging and tracing
func (o *observableAnalyzer) AnalyzeSymbol(ctx context.Context, symbol string) (*pead.PEADScore, error) {
	ctx, span := trace.StartSpan(ctx, "pead.AnalyzeSymbol")
	defer span.End()

	fields := trace.GetTraceFields(ctx)
	fields["symbol"] = symbol

	logger.DebugSkip(ctx, 1, "Analyzing symbol for PEAD", fields)
	start := time.Now()

	score, err := o.inner.AnalyzeSymbol(ctx, symbol)

	duration := time.Since(start)
	fields["duration_ms"] = duration.Milliseconds()

	if err != nil {
		fields["error"] = err.Error()
		logger.ErrorSkip(ctx, 1, "Symbol PEAD analysis failed", fields)
		span.RecordError(err)
		return nil, err
	}

	fields["composite_score"] = score.CompositeScore
	fields["rating"] = score.Rating
	fields["days_since_earnings"] = score.DaysSinceEarnings

	logger.DebugSkip(ctx, 1, "Symbol PEAD analysis completed", fields)

	return score, nil
}

// GetTopPicks wraps the GetTopPicks method with logging and tracing
func (o *observableAnalyzer) GetTopPicks(ctx context.Context, symbols []string, topN int) ([]pead.PEADScore, error) {
	ctx, span := trace.StartSpan(ctx, "pead.GetTopPicks")
	defer span.End()

	fields := trace.GetTraceFields(ctx)
	fields["symbol_count"] = len(symbols)
	fields["top_n"] = topN

	logger.InfoSkip(ctx, 1, "Getting top PEAD picks", fields)
	start := time.Now()

	picks, err := o.inner.GetTopPicks(ctx, symbols, topN)

	duration := time.Since(start)
	fields["duration_ms"] = duration.Milliseconds()

	if err != nil {
		fields["error"] = err.Error()
		logger.ErrorSkip(ctx, 1, "Failed to get top PEAD picks", fields)
		span.RecordError(err)
		return nil, err
	}

	fields["picks_count"] = len(picks)
	if len(picks) > 0 {
		// Log top pick details
		topPick := picks[0]
		fields["top_pick_symbol"] = topPick.Symbol
		fields["top_pick_score"] = topPick.CompositeScore
	}

	logger.InfoSkip(ctx, 1, "Top PEAD picks retrieved", fields)

	return picks, nil
}
