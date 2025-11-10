package eodobs

import (
	"context"
	"time"

	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/trace"
)

type observableEodSummarizer struct {
	summarizer interfaces.EodSummarizer
}

var _ interfaces.EodSummarizer = (*observableEodSummarizer)(nil)

func Wrap(summarizer interfaces.EodSummarizer) interfaces.EodSummarizer {
	return &observableEodSummarizer{
		summarizer: summarizer,
	}
}

func (oes *observableEodSummarizer) SummarizeDay(t time.Time) (string, error) {
	ctx := context.Background()
	ctx, span := trace.StartSpan(ctx, "eod.SummarizeDay")
	defer span.End()

	logger.InfoSkip(ctx, 1, "Starting EOD summary generation",
		"date", t.Format("2006-01-02"),
	)

	csvPath, err := oes.summarizer.SummarizeDay(t)
	if err != nil {
		logger.ErrorWithErrSkip(ctx, 1, "EOD summary generation failed", err,
			"date", t.Format("2006-01-02"),
		)
		return "", err
	}

	if csvPath == "" {
		logger.InfoSkip(ctx, 1, "No trades found for EOD summary",
			"date", t.Format("2006-01-02"),
		)
		return "", nil
	}

	logger.InfoSkip(ctx, 1, "EOD summary generated successfully",
		"date", t.Format("2006-01-02"),
		"csv_path", csvPath,
	)

	return csvPath, nil
}

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
