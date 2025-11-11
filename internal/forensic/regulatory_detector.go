package forensic

import (
	"llm-trading-bot/internal/interfaces"
	"context"
	"strings"
	"time"

	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/types"
)

// CheckRegulatoryActions monitors regulatory actions and penalties
func (c *Checker) CheckRegulatoryActions(ctx context.Context, symbol string) ([]types.RegulatoryAction, error) {
	fromDate := time.Now().AddDate(0, 0, -c.cfg.LookbackDays).Format("2006-01-02")
	toDate := time.Now().Format("2006-01-02")

	// Check announcements
	announcements, err := c.dataSource.FetchAnnouncements(ctx, symbol, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	// Check regulatory filings
	filings, err := c.dataSource.FetchRegulatoryFilings(ctx, symbol, fromDate, toDate)
	if err != nil {
		logger.Warn(ctx, "Failed to fetch regulatory filings", "error", err)
	}

	actions := []types.RegulatoryAction{}

	// Parse announcements
	for _, ann := range announcements {
		subject := strings.ToLower(ann.Subject)
		description := strings.ToLower(ann.Description)
		combined := subject + " " + description

		// Keywords for regulatory actions
		if containsAny(combined, []string{
			"sebi",
			"penalty",
			"fine",
			"investigation",
			"suspension",
			"nse action",
			"bse action",
			"regulatory action",
			"show cause",
			"violation",
			"non-compliance",
			"warning",
		}) {
			action := c.parseRegulatoryAnnouncement(ann)
			if action != nil {
				actions = append(actions, *action)
			}
		}
	}

	// Parse filings
	for _, filing := range filings {
		action := c.parseRegulatoryFiling(filing)
		if action != nil {
			actions = append(actions, *action)
		}
	}

	return actions, nil
}

func (c *Checker) parseRegulatoryAnnouncement(ann interfaces.Announcement) *types.RegulatoryAction {
	subject := strings.ToLower(ann.Subject)
	description := strings.ToLower(ann.Description)
	combined := subject + " " + description

	date, _ := time.Parse("2006-01-02", ann.Date)

	action := &types.RegulatoryAction{
		Date:        date,
		Description: ann.Description,
		Status:      "ONGOING",
	}

	// Identify regulator
	if containsAny(combined, []string{"sebi"}) {
		action.Regulator = "SEBI"
	} else if containsAny(combined, []string{"nse"}) {
		action.Regulator = "NSE"
	} else if containsAny(combined, []string{"bse"}) {
		action.Regulator = "BSE"
	} else if containsAny(combined, []string{"roc", "registrar of companies"}) {
		action.Regulator = "ROC"
	} else if containsAny(combined, []string{"mca", "ministry of corporate affairs"}) {
		action.Regulator = "MCA"
	} else {
		action.Regulator = "OTHER"
	}

	// Determine action type
	if containsAny(combined, []string{"penalty", "fine", "penalised"}) {
		action.ActionType = "PENALTY"
	} else if containsAny(combined, []string{"suspension", "suspended"}) {
		action.ActionType = "SUSPENSION"
	} else if containsAny(combined, []string{"investigation", "investigating", "inquiry"}) {
		action.ActionType = "INVESTIGATION"
	} else if containsAny(combined, []string{"warning", "cautioned", "show cause"}) {
		action.ActionType = "WARNING"
	} else if containsAny(combined, []string{"violation", "non-compliance", "breach"}) {
		action.ActionType = "VIOLATION"
	} else {
		return nil // Not a relevant action
	}

	// Extract penalty amount
	action.PenaltyAmount = extractAmount(combined)

	// Determine status
	if containsAny(combined, []string{"resolved", "settled", "completed"}) {
		action.Status = "RESOLVED"
	} else if containsAny(combined, []string{"appeal", "appealed"}) {
		action.Status = "APPEALED"
	}

	// Calculate risk score
	action.RiskScore = c.calculateRegulatoryRisk(action)

	return action
}

func (c *Checker) parseRegulatoryFiling(filing interfaces.RegulatoryFiling) *types.RegulatoryAction {
	filingType := strings.ToLower(filing.FilingType)
	description := strings.ToLower(filing.Description)
	combined := filingType + " " + description

	// Only process penalty/action related filings
	if !containsAny(combined, []string{
		"penalty",
		"action",
		"violation",
		"non-compliance",
		"investigation",
	}) {
		return nil
	}

	date, _ := time.Parse("2006-01-02", filing.Date)

	action := &types.RegulatoryAction{
		Date:        date,
		Description: filing.Description,
		ActionType:  "VIOLATION",
		Regulator:   "REGULATORY_BODY",
		Status:      "ONGOING",
	}

	action.RiskScore = c.calculateRegulatoryRisk(action)

	return action
}

func (c *Checker) calculateRegulatoryRisk(action *types.RegulatoryAction) float64 {
	score := 40.0 // Base score

	// Action type weights
	typeWeights := map[string]float64{
		"PENALTY":       30.0,
		"SUSPENSION":    40.0,
		"INVESTIGATION": 25.0,
		"WARNING":       15.0,
		"VIOLATION":     20.0,
	}
	score += typeWeights[action.ActionType]

	// Regulator importance
	regulatorWeights := map[string]float64{
		"SEBI": 20.0,
		"NSE":  15.0,
		"BSE":  15.0,
		"ROC":  10.0,
		"MCA":  10.0,
		"OTHER": 5.0,
	}
	score += regulatorWeights[action.Regulator]

	// Penalty amount impact
	if action.PenaltyAmount > 10000000 { // >10M
		score += 20.0
	} else if action.PenaltyAmount > 5000000 { // >5M
		score += 15.0
	} else if action.PenaltyAmount > 1000000 { // >1M
		score += 10.0
	} else if action.PenaltyAmount > 0 {
		score += 5.0
	}

	// Status impact
	if action.Status == "ONGOING" {
		score += 10.0
	} else if action.Status == "RESOLVED" {
		score -= 15.0 // Lower risk if resolved
	}

	// Recency
	daysSince := time.Since(action.Date).Hours() / 24
	if daysSince <= 30 {
		score += 15.0
	} else if daysSince <= 90 {
		score += 10.0
	} else if daysSince <= 180 {
		score += 5.0
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}
