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

type eodSummarizer struct{}


func (es *eodSummarizer) SummarizeDay(t time.Time) (string, error) {
	inPath := todaysTradeFile(t)

	if _, err := os.Stat(inPath); err != nil {
		return "", nil // No trades for this day
	}

	f, err := os.Open(inPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	aggs, err := es.parseTradeLog(f)
	if err != nil {
		return "", err
	}

	if len(aggs) == 0 {
		return "", nil
	}

	outPath := eodCSVPath(t)
	if err := es.writeCSVSummary(outPath, aggs); err != nil {
		return "", err
	}

	return outPath, nil
}

func (es *eodSummarizer) SummarizeToday() (string, error) {
	return es.SummarizeDay(istNow())
}

func (es *eodSummarizer) ShouldRunNow() (bool, string) {
	now := istNow()
	cutoff := marketCloseTime(now)
	outPath := eodCSVPath(now)

	if now.After(cutoff) {
		if _, err := os.Stat(outPath); errors.Is(err, os.ErrNotExist) {
			return true, outPath
		}
	}

	return false, outPath
}

func (es *eodSummarizer) parseTradeLog(f *os.File) (map[string]*aggRow, error) {
	aggs := make(map[string]*aggRow)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		var tl tradeLine
		if err := json.Unmarshal([]byte(scanner.Text()), &tl); err != nil {
			continue // Skip malformed lines
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

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return aggs, nil
}

func (es *eodSummarizer) writeCSVSummary(outPath string, aggs map[string]*aggRow) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	w := csv.NewWriter(out)
	defer w.Flush()

	headers := []string{"symbol", "buy_qty", "buy_avg", "sell_qty", "sell_avg", "realized_pnl", "gross_buy_value", "gross_sell_value"}
	if err := w.Write(headers); err != nil {
		return err
	}

	symbols := make([]string, 0, len(aggs))
	for symbol := range aggs {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)

	var totalBuy, totalSell, totalPnL float64

	for _, symbol := range symbols {
		row := aggs[symbol]

		var buyAvg, sellAvg float64
		if row.BuyQty > 0 {
			buyAvg = row.BuyValue / float64(row.BuyQty)
		}
		if row.SellQty > 0 {
			sellAvg = row.SellValue / float64(row.SellQty)
		}

		matchedQty := row.BuyQty
		if row.SellQty < matchedQty {
			matchedQty = row.SellQty
		}
		row.RealizedPnL = float64(matchedQty) * (sellAvg - buyAvg)

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

		totalBuy += row.BuyValue
		totalSell += row.SellValue
		totalPnL += row.RealizedPnL
	}

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
