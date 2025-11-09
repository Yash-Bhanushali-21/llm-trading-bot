package engine

import (
	"context"

	"llm-trading-bot/internal/logger"
)

// riskManager handles risk validation and exposure calculations.
type riskManager struct {
	// In a real implementation, this would track actual account balance
	// For now, using a placeholder value
	accountValue float64
}

// newRiskManager creates a new risk manager.
// TODO: Integrate with broker to fetch real account balance
func newRiskManager() *riskManager {
	return &riskManager{
		accountValue: 100.0, // Placeholder value
	}
}

// validateTrade checks if a trade exceeds the allowed risk limit.
//
// Parameters:
//   - ctx: Context for logging
//   - symbol: Trading symbol
//   - price: Execution price
//   - qty: Quantity to trade
//   - maxRiskPct: Maximum allowed risk as percentage of account
//
// Returns:
//   - exceeded: true if trade exceeds risk limit
//   - exposure: Total exposure amount for the trade
func (rm *riskManager) validateTrade(ctx context.Context, symbol string, price float64, qty int, maxRiskPct float64) (exceeded bool, exposure float64) {
	// If no risk limit is configured, allow all trades
	if maxRiskPct <= 0 {
		return false, 0
	}

	// Calculate trade exposure
	exposure = price * float64(qty)

	// Calculate exposure as percentage of account
	exposurePct := (exposure / rm.accountValue) * 100.0

	// Check if it exceeds the limit
	exceeded = exposurePct > maxRiskPct

	if exceeded {
		logger.Warn(ctx, "Trade blocked by risk cap",
			"symbol", symbol,
			"event", "TRADE_BLOCKED_RISK_CAP",
			"qty", qty,
			"price", price,
			"exposure", exposure,
			"exposure_pct", exposurePct,
			"risk_limit_pct", maxRiskPct,
			"account_value", rm.accountValue,
		)
	}

	return exceeded, exposure
}

// calculateExposure returns the total exposure for a trade.
func (rm *riskManager) calculateExposure(price float64, qty int) float64 {
	return price * float64(qty)
}

// setAccountValue updates the account value.
// TODO: This should be automatically fetched from the broker
func (rm *riskManager) setAccountValue(value float64) {
	rm.accountValue = value
}

// getAccountValue returns the current account value.
func (rm *riskManager) getAccountValue() float64 {
	return rm.accountValue
}
