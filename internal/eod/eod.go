package eod

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

type tradeLine struct {
	Time, Symbol, Side string
	Qty                int
	Price              float64
	OrderID, Reason    string
	Confidence         float64
}
type aggRow struct {
	Symbol      string
	BuyQty      int
	BuyValue    float64
	SellQty     int
	SellValue   float64
	RealizedPnL float64
}

func logDir() string {
	if v := os.Getenv("TRADER_LOG_DIR"); v != "" {
		return v
	}
	return "logs"
}
func istNow() time.Time { return time.Now().In(time.FixedZone("IST", 19800)) }
func todaysTradeFile(t time.Time) string {
	d := t.Format("2006-01-02")
	return filepath.Join(logDir(), d+".txt")
}
func eodCSVPath(t time.Time) string {
	d := t.Format("2006-01-02")
	return filepath.Join(logDir(), "eod", d+".csv")
}
func SummarizeDay(t time.Time) (string, error) {
	inPath := todaysTradeFile(t)
	if _, err := os.Stat(inPath); err != nil {
		return "", nil
	}
	f, err := os.Open(inPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	aggs := map[string]*aggRow{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var tl tradeLine
		if err := json.Unmarshal([]byte(sc.Text()), &tl); err != nil {
			continue
		}
		row := aggs[tl.Symbol]
		if row == nil {
			row = &aggRow{Symbol: tl.Symbol}
			aggs[tl.Symbol] = row
		}
		if tl.Side == "BUY" {
			row.BuyQty += tl.Qty
			row.BuyValue += float64(tl.Qty) * tl.Price
		}
		if tl.Side == "SELL" {
			row.SellQty += tl.Qty
			row.SellValue += float64(tl.Qty) * tl.Price
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	if len(aggs) == 0 {
		return "", nil
	}
	keys := make([]string, 0, len(aggs))
	for k := range aggs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	outPath := eodCSVPath(t)
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", err
	}
	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer out.Close()
	w := csv.NewWriter(out)
	defer w.Flush()
	headers := []string{"symbol", "buy_qty", "buy_avg", "sell_qty", "sell_avg", "realized_pnl", "gross_buy_value", "gross_sell_value"}
	if err := w.Write(headers); err != nil {
		return "", err
	}
	var totalBuy, totalSell, totalPnL float64
	for _, k := range keys {
		r := aggs[k]
		var buyAvg, sellAvg float64
		if r.BuyQty > 0 {
			buyAvg = r.BuyValue / float64(r.BuyQty)
		}
		if r.SellQty > 0 {
			sellAvg = r.SellValue / float64(r.SellQty)
		}
		matched := r.BuyQty
		if r.SellQty < matched {
			matched = r.SellQty
		}
		r.RealizedPnL = float64(matched) * (sellAvg - buyAvg)
		rec := []string{r.Symbol, strconv.Itoa(r.BuyQty), fmt.Sprintf("%.4f", buyAvg), strconv.Itoa(r.SellQty), fmt.Sprintf("%.4f", sellAvg), fmt.Sprintf("%.2f", r.RealizedPnL), fmt.Sprintf("%.2f", r.BuyValue), fmt.Sprintf("%.2f", r.SellValue)}
		if err := w.Write(rec); err != nil {
			return "", err
		}
		totalBuy += r.BuyValue
		totalSell += r.SellValue
		totalPnL += r.RealizedPnL
	}
	_ = w.Write([]string{"TOTAL", "", "", "", "", fmt.Sprintf("%.2f", totalPnL), fmt.Sprintf("%.2f", totalBuy), fmt.Sprintf("%.2f", totalSell)})
	return outPath, nil
}
func SummarizeToday() (string, error) { return SummarizeDay(istNow()) }
func ShouldRunNow() (bool, string) {
	now := istNow()
	cutoff := time.Date(now.Year(), now.Month(), now.Day(), 15, 40, 0, 0, now.Location())
	outPath := eodCSVPath(now)
	if now.After(cutoff) {
		if _, err := os.Stat(outPath); errors.Is(err, os.ErrNotExist) {
			return true, outPath
		}
	}
	return false, outPath
}
