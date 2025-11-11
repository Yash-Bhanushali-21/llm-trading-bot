package forensic

import (
	"llm-trading-bot/internal/interfaces"
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"llm-trading-bot/internal/types"
)

// CheckRestatements detects financial restatements
func (c *Checker) CheckRestatements(ctx context.Context, symbol string) ([]types.FinancialRestatement, error) {
	fromDate := time.Now().AddDate(0, 0, -c.cfg.LookbackDays).Format("2006-01-02")
	toDate := time.Now().Format("2006-01-02")

	announcements, err := c.dataSource.FetchAnnouncements(ctx, symbol, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	restatements := []types.FinancialRestatement{}

	for _, ann := range announcements {
		subject := strings.ToLower(ann.Subject)
		description := strings.ToLower(ann.Description)
		combined := subject + " " + description

		// Keywords for restatements
		if containsAny(combined, []string{
			"restatement",
			"restated",
			"revision of results",
			"revised results",
			"correction",
			"accounting error",
			"prior period adjustment",
		}) {
			restatement := c.parseRestatement(ann)
			if restatement != nil {
				restatements = append(restatements, *restatement)
			}
		}
	}

	return restatements, nil
}

func (c *Checker) parseRestatement(ann interfaces.Announcement) *types.FinancialRestatement {
	subject := strings.ToLower(ann.Subject)
	description := strings.ToLower(ann.Description)
	combined := subject + " " + description

	date, _ := time.Parse("2006-01-02", ann.Date)

	restatement := &types.FinancialRestatement{
		Date:              date,
		RestatementReason: ann.Description,
		ItemsAffected:     []string{},
		IsMaterial:        false,
	}

	// Extract period being restated
	restatement.Period = extractPeriod(combined)

	// Identify affected items
	if containsAny(combined, []string{"revenue", "sales", "income"}) {
		restatement.ItemsAffected = append(restatement.ItemsAffected, "Revenue")
	}
	if containsAny(combined, []string{"expense", "cost"}) {
		restatement.ItemsAffected = append(restatement.ItemsAffected, "Expenses")
	}
	if containsAny(combined, []string{"profit", "loss", "net income"}) {
		restatement.ItemsAffected = append(restatement.ItemsAffected, "Profit/Loss")
	}
	if containsAny(combined, []string{"asset", "balance sheet"}) {
		restatement.ItemsAffected = append(restatement.ItemsAffected, "Assets")
	}
	if containsAny(combined, []string{"liability", "liabilities"}) {
		restatement.ItemsAffected = append(restatement.ItemsAffected, "Liabilities")
	}
	if containsAny(combined, []string{"equity", "reserves"}) {
		restatement.ItemsAffected = append(restatement.ItemsAffected, "Equity")
	}

	// Determine materiality
	restatement.IsMaterial = containsAny(combined, []string{
		"material",
		"significant",
		"substantial",
	}) || len(restatement.ItemsAffected) > 2

	// Try to extract values if mentioned
	amounts := extractMultipleAmounts(combined)
	if len(amounts) >= 2 {
		restatement.OriginalValue = amounts[0]
		restatement.RestatedValue = amounts[1]
		if restatement.OriginalValue != 0 {
			restatement.ImpactPercentage = math.Abs((restatement.RestatedValue - restatement.OriginalValue) / restatement.OriginalValue * 100)
		}
	}

	// Calculate risk score
	restatement.RiskScore = c.calculateRestatementRisk(restatement)

	return restatement
}

func (c *Checker) calculateRestatementRisk(restatement *types.FinancialRestatement) float64 {
	score := 50.0 // Base score for any restatement

	// Materiality
	if restatement.IsMaterial {
		score += 25.0
	}

	// Number of items affected
	score += float64(len(restatement.ItemsAffected)) * 5.0

	// Impact percentage
	if restatement.ImpactPercentage > 20 {
		score += 20.0
	} else if restatement.ImpactPercentage > 10 {
		score += 15.0
	} else if restatement.ImpactPercentage > 5 {
		score += 10.0
	}

	// Critical items affected
	reason := strings.ToLower(restatement.RestatementReason)
	if containsAny(reason, []string{"fraud", "misstatement", "irregularity"}) {
		score += 30.0
	}

	// Revenue/profit restatements are more serious
	for _, item := range restatement.ItemsAffected {
		if item == "Revenue" || item == "Profit/Loss" {
			score += 10.0
		}
	}

	// Recency
	daysSince := time.Since(restatement.Date).Hours() / 24
	if daysSince <= 90 {
		score += 15.0
	} else if daysSince <= 180 {
		score += 10.0
	} else if daysSince <= 365 {
		score += 5.0
	}

	if score > 100 {
		score = 100
	}

	return score
}

func extractPeriod(text string) string {
	// Look for period patterns like "FY2023", "Q1FY24", etc.
	words := strings.Fields(text)
	for _, word := range words {
		word = strings.ToUpper(word)
		if strings.Contains(word, "FY") || strings.Contains(word, "Q") {
			return word
		}
	}
	return "Unknown Period"
}

func extractMultipleAmounts(text string) []float64 {
	// Extract multiple amounts from text
	amounts := []float64{}
	words := strings.Fields(text)

	for i, word := range words {
		word = strings.ReplaceAll(word, ",", "")
		if val, err := strconv.ParseFloat(word, 64); err == nil {
			// Check for units
			if i+1 < len(words) {
				unit := strings.ToLower(words[i+1])
				if strings.Contains(unit, "crore") {
					amounts = append(amounts, val*10000000)
				} else if strings.Contains(unit, "lakh") {
					amounts = append(amounts, val*100000)
				} else if strings.Contains(unit, "million") {
					amounts = append(amounts, val*1000000)
				} else {
					amounts = append(amounts, val)
				}
			} else {
				amounts = append(amounts, val)
			}
		}
	}

	return amounts
}
