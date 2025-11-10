package engine

import (
	"context"

	"llm-trading-bot/internal/logger"
)

type riskManager struct {
	accountValue float64
}

func newRiskManager() *riskManager {
	return &riskManager{
		accountValue: 100.0, // Placeholder value
	}
}

//
//
func (rm *riskManager) validateTrade(ctx context.Context, symbol string, price float64, qty int, maxRiskPct float64) (exceeded bool, exposure float64) {
	if maxRiskPct <= 0 {
		return false, 0
	}

	exposure = price * float64(qty)

	exposurePct := (exposure / rm.accountValue) * 100.0

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

func (rm *riskManager) calculateExposure(price float64, qty int) float64 {
	return price * float64(qty)
}

func (rm *riskManager) setAccountValue(value float64) {
	rm.accountValue = value
}

func (rm *riskManager) getAccountValue() float64 {
	return rm.accountValue
}
