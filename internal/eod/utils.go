package eod

import (
	"os"
	"path/filepath"
	"time"
)

func logDir() string {
	if v := os.Getenv("TRADER_LOG_DIR"); v != "" {
		return v
	}
	return "logs"
}

func istNow() time.Time {
	return time.Now().In(time.FixedZone("IST", 19800))
}

//
//
func todaysTradeFile(t time.Time) string {
	dateStr := t.Format("2006-01-02")
	return filepath.Join(logDir(), dateStr+".txt")
}

//
//
func eodCSVPath(t time.Time) string {
	dateStr := t.Format("2006-01-02")
	return filepath.Join(logDir(), "eod", dateStr+".csv")
}

//
//
func marketCloseTime(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 15, 40, 0, 0, t.Location())
}
