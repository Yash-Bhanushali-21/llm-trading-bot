package tradelog

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var mu sync.Mutex

type Entry struct {
	Time, Symbol, Side, OrderID, Reason string
	Qty                                 int
	Price                               float64
	Confidence                          float64
	Extra                               map[string]any `json:"extra,omitempty"`
}
type DecisionEntry struct {
	Time, Symbol, Action, Reason string
	Confidence                   float64
	Price                        float64
	Indicators                   map[string]float64
	Extra                        map[string]any
}

func logDir() string {
	if v := os.Getenv("TRADER_LOG_DIR"); v != "" {
		return v
	}
	return "logs"
}
func dailyFilepath(t time.Time) string {
	d := t.In(time.FixedZone("IST", 19800)).Format("2006-01-02")
	return filepath.Join(logDir(), d+".txt")
}
func decisionsFilepath(t time.Time) string {
	d := t.In(time.FixedZone("IST", 19800)).Format("2006-01-02")
	return filepath.Join(logDir(), "decisions", d+".txt")
}
func Append(e Entry) error {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now().In(time.FixedZone("IST", 19800))
	e.Time = now.Format("2006-01-02 15:04:05")
	p := dailyFilepath(now)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, _ := json.Marshal(e)
	_, err = fmt.Fprintln(f, string(b))
	return err
}
func AppendDecision(e DecisionEntry) error {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now().In(time.FixedZone("IST", 19800))
	e.Time = now.Format("2006-01-02 15:04:05")
	p := decisionsFilepath(now)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, _ := json.Marshal(e)
	_, err = fmt.Fprintln(f, string(b))
	return err
}
func CompressOlder(retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	root := logDir()
	return filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(p) != ".txt" {
			return nil
		}
		info, er := os.Stat(p)
		if er != nil {
			return nil
		}
		cutoff := time.Now().AddDate(0, 0, -retentionDays)
		if info.ModTime().Before(cutoff) {
			gz := p + ".gz"
			// if already gz exists, remove original .txt
			if _, e2 := os.Stat(gz); e2 == nil {
				_ = os.Remove(p)
				return nil
			}

			in, e3 := os.Open(p)
			if e3 != nil {
				return nil
			}
			defer in.Close()

			out, e4 := os.OpenFile(gz, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if e4 != nil {
				return nil
			}
			// ensure writer is closed and file closed
			gw := gzip.NewWriter(out)
			// copy and handle error
			if _, e5 := io.Copy(gw, in); e5 == nil {
				_ = gw.Close()
				_ = out.Close()
				_ = os.Remove(p)
			} else {
				// close writer and file even on error
				_ = gw.Close()
				_ = out.Close()
			}
		}
		return nil
	})
}
