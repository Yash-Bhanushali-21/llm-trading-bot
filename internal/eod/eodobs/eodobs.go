package eodobs

import (
	"context"
	"time"

	"llm-trading-bot/internal/eod"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/trace"
)

// observableEodSummarizer wraps an IEodSummarizer with observability (logging & tracing)
type observableEodSummarizer struct {
	summarizer eod.IEodSummarizer
}

// Compile-time interface check
var _ eod.IEodSummarizer = (*observableEodSummarizer)(nil)

// Wrap wraps an EOD summarizer with observability middleware
func Wrap(summarizer eod.IEodSummarizer) eod.IEodSummarizer {
	return &observableEodSummarizer{
		summarizer: summarizer,
	}
}

// SummarizeDay generates end-of-day summary with observability
func (oes *observableEodSummarizer) SummarizeDay(t time.Time) (string, error) {
	ctx := context.Background()
	ctx, span := trace.StartSpan(ctx, "eod.SummarizeDay")
	defer span.End()

	// Use InfoSkip(1) to report the actual caller, not this middleware wrapper
	logger.InfoSkip(ctx, 1, "Starting EOD summary generation",
		"date", t.Format("2006-01-02"),
	)

	// Call underlying summarizer
	csvPath, err := oes.summarizer.SummarizeDay(t)
	if err != nil {
		// Use ErrorWithErrSkip(1) to report the actual caller
		logger.ErrorWithErrSkip(ctx, 1, "EOD summary generation failed", err,
			"date", t.Format("2006-01-02"),
		)
		return "", err
	}

	// No trades for the day
	if csvPath == "" {
		logger.InfoSkip(ctx, 1, "No trades found for EOD summary",
			"date", t.Format("2006-01-02"),
		)
		return "", nil
	}

	// Log successful summary generation
	logger.InfoSkip(ctx, 1, "EOD summary generated successfully",
		"date", t.Format("2006-01-02"),
		"csv_path", csvPath,
	)

	return csvPath, nil
}

// SummarizeToday generates today's EOD summary with observability
func (oes *observableEodSummarizer) SummarizeToday() (string, error) {
	ctx := context.Background()
	ctx, span := trace.StartSpan(ctx, "eod.SummarizeToday")
	defer span.End()

	logger.InfoSkip(ctx, 1, "Starting today's EOD summary generation")

	csvPath, err := oes.summarizer.SummarizeToday()
	if err != nil {
		logger.ErrorWithErrSkip(ctx, 1, "Today's EOD summary generation failed", err)
		return "", err
	}

	if csvPath == "" {
		logger.InfoSkip(ctx, 1, "No trades found for today's EOD summary")
		return "", nil
	}

	logger.InfoSkip(ctx, 1, "Today's EOD summary generated successfully",
		"csv_path", csvPath,
	)

	return csvPath, nil
}

// ShouldRunNow checks if EOD should run with observability
func (oes *observableEodSummarizer) ShouldRunNow() (bool, string) {
	ctx := context.Background()
	ctx, span := trace.StartSpan(ctx, "eod.ShouldRunNow")
	defer span.End()

	shouldRun, csvPath := oes.summarizer.ShouldRunNow()

	logger.DebugSkip(ctx, 1, "EOD check completed",
		"should_run", shouldRun,
		"csv_path", csvPath,
	)

	return shouldRun, csvPath
}
