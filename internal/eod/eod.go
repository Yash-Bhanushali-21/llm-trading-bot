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

// eodSummarizer is the default implementation of IEodSummarizer.
type eodSummarizer struct{}

// NewSummarizer creates a new EOD summarizer instance
func NewSummarizer() IEodSummarizer {
	return &eodSummarizer{}
}

// SummarizeDay generates an end-of-day CSV summary for a specific date.
func (es *eodSummarizer) SummarizeDay(t time.Time) (string, error) {
	inPath := todaysTradeFile(t)

	// Check if trade log exists
	if _, err := os.Stat(inPath); err != nil {
		return "", nil // No trades for this day
	}

	// Open trade log file
	f, err := os.Open(inPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Parse and aggregate trades
	aggs, err := es.parseTradeLog(f)
	if err != nil {
		return "", err
	}

	// No trades found
	if len(aggs) == 0 {
		return "", nil
	}

	// Write CSV summary
	outPath := eodCSVPath(t)
	if err := es.writeCSVSummary(outPath, aggs); err != nil {
		return "", err
	}

	return outPath, nil
}

// SummarizeToday generates an end-of-day summary for today.
func (es *eodSummarizer) SummarizeToday() (string, error) {
	return es.SummarizeDay(istNow())
}

// ShouldRunNow checks if EOD summary should be generated now.
func (es *eodSummarizer) ShouldRunNow() (bool, string) {
	now := istNow()
	cutoff := marketCloseTime(now)
	outPath := eodCSVPath(now)

	// Check if it's after market close
	if now.After(cutoff) {
		// Check if summary doesn't exist yet
		if _, err := os.Stat(outPath); errors.Is(err, os.ErrNotExist) {
			return true, outPath
		}
	}

	return false, outPath
}

// parseTradeLog reads and aggregates trades from the log file.
func (es *eodSummarizer) parseTradeLog(f *os.File) (map[string]*aggRow, error) {
	aggs := make(map[string]*aggRow)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		var tl tradeLine
		if err := json.Unmarshal([]byte(scanner.Text()), &tl); err != nil {
			continue // Skip malformed lines
		}

		// Get or create aggregation row for symbol
		row := aggs[tl.Symbol]
		if row == nil {
			row = &aggRow{Symbol: tl.Symbol}
			aggs[tl.Symbol] = row
		}

		// Aggregate by side
		if tl.Side == "BUY" {
			row.BuyQty += tl.Qty
			row.BuyValue += float64(tl.Qty) * tl.Price
		}
		if tl.Side == "SELL" {
			row.SellQty += tl.Qty
			row.SellValue += float64(tl.Qty) * tl.Price
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return aggs, nil
}

// writeCSVSummary writes the aggregated trade data to a CSV file.
func (es *eodSummarizer) writeCSVSummary(outPath string, aggs map[string]*aggRow) error {
	// Create output directory
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	// Create CSV file
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	w := csv.NewWriter(out)
	defer w.Flush()

	// Write headers
	headers := []string{"symbol", "buy_qty", "buy_avg", "sell_qty", "sell_avg", "realized_pnl", "gross_buy_value", "gross_sell_value"}
	if err := w.Write(headers); err != nil {
		return err
	}

	// Sort symbols for consistent output
	symbols := make([]string, 0, len(aggs))
	for symbol := range aggs {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)

	// Write trade data and calculate totals
	var totalBuy, totalSell, totalPnL float64

	for _, symbol := range symbols {
		row := aggs[symbol]

		// Calculate averages
		var buyAvg, sellAvg float64
		if row.BuyQty > 0 {
			buyAvg = row.BuyValue / float64(row.BuyQty)
		}
		if row.SellQty > 0 {
			sellAvg = row.SellValue / float64(row.SellQty)
		}

		// Calculate realized P&L from matched trades
		matchedQty := row.BuyQty
		if row.SellQty < matchedQty {
			matchedQty = row.SellQty
		}
		row.RealizedPnL = float64(matchedQty) * (sellAvg - buyAvg)

		// Write row
		record := []string{
			row.Symbol,
			strconv.Itoa(row.BuyQty),
			fmt.Sprintf("%.4f", buyAvg),
			strconv.Itoa(row.SellQty),
			fmt.Sprintf("%.4f", sellAvg),
			fmt.Sprintf("%.2f", row.RealizedPnL),
			fmt.Sprintf("%.2f", row.BuyValue),
			fmt.Sprintf("%.2f", row.SellValue),
		}

		if err := w.Write(record); err != nil {
			return err
		}

		// Update totals
		totalBuy += row.BuyValue
		totalSell += row.SellValue
		totalPnL += row.RealizedPnL
	}

	// Write total row
	totalRow := []string{
		"TOTAL",
		"",
		"",
		"",
		"",
		fmt.Sprintf("%.2f", totalPnL),
		fmt.Sprintf("%.2f", totalBuy),
		fmt.Sprintf("%.2f", totalSell),
	}

	if err := w.Write(totalRow); err != nil {
		return err
	}

	return nil
}
