package eod

// tradeLine represents a single trade entry from the trade log file.
// This structure matches the JSON format written by the tradelog package.
type tradeLine struct {
	Time       string  // Timestamp of the trade
	Symbol     string  // Trading symbol (e.g., "RELIANCE")
	Side       string  // "BUY" or "SELL"
	Qty        int     // Quantity traded
	Price      float64 // Execution price
	OrderID    string  // Broker order ID
	Reason     string  // Trade reason (LLM decision or STOP_LOSS)
	Confidence float64 // LLM confidence level (0.0 to 1.0)
}

// aggRow represents aggregated trading statistics for a symbol.
// Used to calculate EOD summary metrics across all trades for a symbol.
type aggRow struct {
	Symbol      string  // Trading symbol
	BuyQty      int     // Total quantity bought
	BuyValue    float64 // Total value of buy orders (qty * price)
	SellQty     int     // Total quantity sold
	SellValue   float64 // Total value of sell orders (qty * price)
	RealizedPnL float64 // Realized profit/loss (calculated from matched trades)
}
