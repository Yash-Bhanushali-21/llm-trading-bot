package eod

import (
	"time"

	"llm-trading-bot/internal/interfaces"
)

var defaultSummarizer interfaces.EodSummarizer = &eodSummarizer{}

func SetDefaultSummarizer(summarizer interfaces.EodSummarizer) {
	defaultSummarizer = summarizer
}

func NewSummarizer() interfaces.EodSummarizer {
	return &eodSummarizer{}
}

func SummarizeDay(t time.Time) (string, error) {
	return defaultSummarizer.SummarizeDay(t)
}

func SummarizeToday() (string, error) {
	return defaultSummarizer.SummarizeToday()
}

func ShouldRunNow() (bool, string) {
	return defaultSummarizer.ShouldRunNow()
}
