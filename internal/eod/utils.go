package eod

import (
	"os"
	"path/filepath"
	"time"
)

// logDir returns the directory where trade logs are stored.
// Checks TRADER_LOG_DIR environment variable, defaults to "logs".
func logDir() string {
	if v := os.Getenv("TRADER_LOG_DIR"); v != "" {
		return v
	}
	return "logs"
}

// istNow returns the current time in Indian Standard Time (IST).
// IST is UTC+5:30 (19800 seconds offset).
func istNow() time.Time {
	return time.Now().In(time.FixedZone("IST", 19800))
}

// todaysTradeFile returns the path to the trade log file for a given date.
// Format: logs/YYYY-MM-DD.txt
//
// Parameters:
//   - t: The date to get the trade file for
//
// Returns:
//   - path: Full path to the trade log file
func todaysTradeFile(t time.Time) string {
	dateStr := t.Format("2006-01-02")
	return filepath.Join(logDir(), dateStr+".txt")
}

// eodCSVPath returns the path where the EOD CSV summary should be written.
// Format: logs/eod/YYYY-MM-DD.csv
//
// Parameters:
//   - t: The date for the EOD summary
//
// Returns:
//   - path: Full path to the EOD CSV file
func eodCSVPath(t time.Time) string {
	dateStr := t.Format("2006-01-02")
	return filepath.Join(logDir(), "eod", dateStr+".csv")
}

// marketCloseTime returns the market close time for a given date.
// Indian markets close at 3:30 PM IST, but we wait until 3:40 PM
// to ensure all trades are logged.
//
// Parameters:
//   - t: The date to get market close time for
//
// Returns:
//   - closeTime: 15:40:00 IST on the given date
func marketCloseTime(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 15, 40, 0, 0, t.Location())
}
