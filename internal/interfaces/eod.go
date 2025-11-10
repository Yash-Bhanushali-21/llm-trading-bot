package interfaces

import "time"

type EodSummarizer interface {
	SummarizeDay(t time.Time) (csvPath string, err error)
	SummarizeToday() (csvPath string, err error)
	ShouldRunNow() (shouldRun bool, csvPath string)
}
