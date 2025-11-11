package forensic

import (
	"llm-trading-bot/internal/interfaces"
	"context"
	"sort"
	"time"

	"llm-trading-bot/internal/types"
)

// CheckInsiderTrading analyzes insider trading patterns
func (c *Checker) CheckInsiderTrading(ctx context.Context, symbol string) ([]types.InsiderTrade, error) {
	fromDate := time.Now().AddDate(0, 0, -c.cfg.LookbackDays).Format("2006-01-02")
	toDate := time.Now().Format("2006-01-02")

	insiderData, err := c.dataSource.FetchInsiderTrades(ctx, symbol, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	trades := []types.InsiderTrade{}

	// Convert to our format and analyze
	for _, data := range insiderData {
		trade := c.parseInsiderTrade(data)
		if trade != nil {
			trades = append(trades, *trade)
		}
	}

	// Detect patterns
	c.detectInsiderTradingPatterns(trades)

	return trades, nil
}

func (c *Checker) parseInsiderTrade(data interfaces.InsiderTradeData) *types.InsiderTrade {
	date, _ := time.Parse("2006-01-02", data.Date)

	trade := &types.InsiderTrade{
		Date:            date,
		InsiderName:     data.Name,
		Designation:     data.Designation,
		TransactionType: data.TransactionType,
		Quantity:        data.Quantity,
		Value:           data.Value,
		AvgPrice:        data.Price,
		IsUnusual:       false,
		ClusteredTrades: false,
	}

	// Determine if unusual based on volume
	if trade.Value > 50000000 { // >50M
		trade.IsUnusual = true
	}

	// Check for senior management
	isKeyPersonnel := containsAny(trade.Designation, []string{
		"CEO",
		"CFO",
		"MD",
		"Managing Director",
		"Chairman",
		"Director",
		"Promoter",
	})

	// Calculate risk score
	trade.RiskScore = c.calculateInsiderTradingRisk(trade, isKeyPersonnel)

	return trade
}

func (c *Checker) detectInsiderTradingPatterns(trades []types.InsiderTrade) {
	if len(trades) < 2 {
		return
	}

	// Sort by date
	sort.Slice(trades, func(i, j int) bool {
		return trades[i].Date.Before(trades[j].Date)
	})

	// Detect clustered trades (multiple insiders trading within 30 days)
	for i := 0; i < len(trades); i++ {
		cluster := []types.InsiderTrade{trades[i]}

		for j := i + 1; j < len(trades); j++ {
			daysDiff := trades[j].Date.Sub(trades[i].Date).Hours() / 24
			if daysDiff <= 30 {
				// Same transaction type within 30 days
				if trades[j].TransactionType == trades[i].TransactionType {
					cluster = append(cluster, trades[j])
				}
			} else {
				break
			}
		}

		// If 3+ insiders traded in same direction within 30 days, flag as clustered
		if len(cluster) >= 3 {
			for idx := range cluster {
				// Find the trade in original slice and mark it
				for k := range trades {
					if trades[k].Date.Equal(cluster[idx].Date) &&
						trades[k].InsiderName == cluster[idx].InsiderName {
						trades[k].ClusteredTrades = true
						trades[k].RiskScore += 20.0
						if trades[k].RiskScore > 100 {
							trades[k].RiskScore = 100
						}
					}
				}
			}
		}
	}

	// Detect unusual timing (before major announcements)
	// This would need integration with announcement data
	// For now, flag large sells as potentially suspicious
	for i := range trades {
		if trades[i].TransactionType == "SELL" && trades[i].Value > 100000000 {
			trades[i].IsUnusual = true
			trades[i].RiskScore += 15.0
			if trades[i].RiskScore > 100 {
				trades[i].RiskScore = 100
			}
		}
	}
}

func (c *Checker) calculateInsiderTradingRisk(trade *types.InsiderTrade, isKeyPersonnel bool) float64 {
	score := 0.0

	// Base score for insider trades
	if trade.TransactionType == "SELL" {
		score = 30.0 // Sells are more concerning
	} else {
		score = 10.0 // Buys are generally positive
		return score // Buys are not risky
	}

	// Key personnel trading is more significant
	if isKeyPersonnel {
		score += 20.0
	} else {
		score += 10.0
	}

	// Volume impact
	if trade.Value > 100000000 { // >100M
		score += 25.0
	} else if trade.Value > 50000000 { // >50M
		score += 20.0
	} else if trade.Value > 10000000 { // >10M
		score += 15.0
	} else if trade.Value > 5000000 { // >5M
		score += 10.0
	}

	// Unusual flag
	if trade.IsUnusual {
		score += 15.0
	}

	// Recency
	daysSince := time.Since(trade.Date).Hours() / 24
	if daysSince <= 7 {
		score += 15.0
	} else if daysSince <= 30 {
		score += 10.0
	} else if daysSince <= 90 {
		score += 5.0
	}

	if score > 100 {
		score = 100
	}

	return score
}
