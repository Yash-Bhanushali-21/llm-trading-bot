package eod

import "time"

// IEodSummarizer defines the interface for end-of-day trade summarization.
// Implementations should parse trade logs and generate CSV summaries.
type IEodSummarizer interface {
	// SummarizeDay generates an end-of-day CSV summary for a specific date.
	// Reads the trade log file, aggregates trades by symbol, and writes a CSV report.
	//
	// Parameters:
	//   - t: The date to summarize (in IST timezone)
	//
	// Returns:
	//   - csvPath: Path to the generated CSV file
	//   - error: Error if summarization fails, or nil if no trades exist
	SummarizeDay(t time.Time) (csvPath string, err error)

	// SummarizeToday generates an end-of-day summary for the current date.
	// Convenience wrapper around SummarizeDay(istNow()).
	//
	// Returns:
	//   - csvPath: Path to the generated CSV file
	//   - error: Error if summarization fails
	SummarizeToday() (csvPath string, err error)

	// ShouldRunNow checks if EOD summary should be generated now.
	// Returns true if current time is after market close (3:40 PM IST)
	// and the summary file doesn't exist yet.
	//
	// Returns:
	//   - shouldRun: true if EOD summary should be generated
	//   - csvPath: Path where the CSV would be written
	ShouldRunNow() (shouldRun bool, csvPath string)
}

// Default implementation is package-level for backwards compatibility
var defaultSummarizer IEodSummarizer = &eodSummarizer{}

// SetDefaultSummarizer allows setting a custom default summarizer (e.g., wrapped with observability)
func SetDefaultSummarizer(summarizer IEodSummarizer) {
	defaultSummarizer = summarizer
}

// SummarizeDay uses the default summarizer to generate EOD summary.
func SummarizeDay(t time.Time) (string, error) {
	return defaultSummarizer.SummarizeDay(t)
}

// SummarizeToday uses the default summarizer for today's summary.
func SummarizeToday() (string, error) {
	return defaultSummarizer.SummarizeToday()
}

// ShouldRunNow uses the default summarizer to check if EOD should run.
func ShouldRunNow() (bool, string) {
	return defaultSummarizer.ShouldRunNow()
}
