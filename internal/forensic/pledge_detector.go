package forensic

import (
	"llm-trading-bot/internal/interfaces"
	"context"
	"strconv"
	"strings"
	"time"

	"llm-trading-bot/internal/types"
)

// CheckPromoterPledges tracks pledge of promoter shares
func (c *Checker) CheckPromoterPledges(ctx context.Context, symbol string) ([]types.PromoterPledge, error) {
	// Fetch current shareholding pattern
	pattern, err := c.dataSource.FetchShareholdingPattern(ctx, symbol)
	if err != nil {
		return nil, err
	}

	pledges := []types.PromoterPledge{}

	// Analyze each promoter's pledge status
	for _, promoter := range pattern.PromoterDetails {
		if promoter.SharesPledged > 0 {
			pledge := types.PromoterPledge{
				Date:             parseDate(pattern.AsOfDate),
				PromoterName:     promoter.Name,
				SharesPledged:    promoter.SharesPledged,
				TotalShares:      promoter.SharesHeld,
				PledgePercentage: promoter.PledgePercentage,
				IsIncrease:       false, // Would need historical data to determine
				ChangePercentage: 0,
			}

			// Calculate risk score
			pledge.RiskScore = c.calculatePledgeRisk(&pledge)

			pledges = append(pledges, pledge)
		}
	}

	// Also check announcements for pledge changes
	fromDate := time.Now().AddDate(0, 0, -c.cfg.LookbackDays).Format("2006-01-02")
	toDate := time.Now().Format("2006-01-02")

	announcements, err := c.dataSource.FetchAnnouncements(ctx, symbol, fromDate, toDate)
	if err == nil {
		for _, ann := range announcements {
			subject := strings.ToLower(ann.Subject)
			description := strings.ToLower(ann.Description)
			combined := subject + " " + description

			// Keywords for pledge changes
			if containsAny(combined, []string{
				"pledge",
				"pledging",
				"encumbrance",
				"invocation of pledge",
			}) {
				pledge := c.parsePledgeAnnouncement(ann)
				if pledge != nil {
					pledges = append(pledges, *pledge)
				}
			}
		}
	}

	return pledges, nil
}

func (c *Checker) parsePledgeAnnouncement(ann interfaces.Announcement) *types.PromoterPledge {
	subject := strings.ToLower(ann.Subject)
	description := strings.ToLower(ann.Description)
	combined := subject + " " + description

	date, _ := time.Parse("2006-01-02", ann.Date)

	pledge := &types.PromoterPledge{
		Date:         date,
		PromoterName: extractPromoterName(combined),
	}

	// Extract pledge percentage if mentioned
	pledge.PledgePercentage = extractPercentage(combined)

	// Check if increase or decrease
	pledge.IsIncrease = containsAny(combined, []string{
		"increase",
		"additional",
		"further pledge",
		"more shares",
	})

	// Check for invocation (very high risk)
	if containsAny(combined, []string{"invocation", "invoked"}) {
		pledge.RiskScore = 95.0
	} else {
		pledge.RiskScore = c.calculatePledgeRisk(pledge)
	}

	return pledge
}

func (c *Checker) calculatePledgeRisk(pledge *types.PromoterPledge) float64 {
	score := 0.0

	// Pledge percentage is the primary risk factor
	if pledge.PledgePercentage >= 90 {
		score = 90.0
	} else if pledge.PledgePercentage >= 75 {
		score = 75.0
	} else if pledge.PledgePercentage >= 60 {
		score = 60.0
	} else if pledge.PledgePercentage >= 50 {
		score = 45.0
	} else if pledge.PledgePercentage >= 25 {
		score = 30.0
	} else {
		score = 15.0
	}

	// Increasing pledge is more risky
	if pledge.IsIncrease {
		score += 15.0
	}

	// Recent pledges are more concerning
	daysSince := time.Since(pledge.Date).Hours() / 24
	if daysSince <= 30 {
		score += 10.0
	} else if daysSince <= 90 {
		score += 5.0
	}

	if score > 100 {
		score = 100
	}

	return score
}

func extractPromoterName(text string) string {
	// Simple extraction
	return "Promoter"
}

func extractPercentage(text string) float64 {
	// Look for percentage patterns
	words := strings.Fields(text)
	for i, word := range words {
		word = strings.TrimSuffix(word, "%")
		if val, err := strconv.ParseFloat(word, 64); err == nil {
			// Check if next word is "percent" or "%"
			if i+1 < len(words) && (words[i+1] == "percent" || words[i+1] == "%") {
				return val
			}
			// If the word ends with %, return it
			if strings.HasSuffix(words[i], "%") {
				return val
			}
		}
	}
	return 0
}

func parseDate(dateStr string) time.Time {
	// Try different date formats
	formats := []string{
		"2006-01-02",
		"02-01-2006",
		"02/01/2006",
		"2006/01/02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	return time.Now()
}
