package forensic

import (
	"llm-trading-bot/internal/interfaces"
	"context"
	"strings"
	"time"

	"llm-trading-bot/internal/types"
)

// CheckManagementChanges detects management changes and resignations
func (c *Checker) CheckManagementChanges(ctx context.Context, symbol string) ([]types.ManagementChange, error) {
	fromDate := time.Now().AddDate(0, 0, -c.cfg.LookbackDays).Format("2006-01-02")
	toDate := time.Now().Format("2006-01-02")

	announcements, err := c.dataSource.FetchAnnouncements(ctx, symbol, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	changes := []types.ManagementChange{}

	for _, ann := range announcements {
		// Parse announcement for management changes
		subject := strings.ToLower(ann.Subject)
		description := strings.ToLower(ann.Description)
		combined := subject + " " + description

		// Keywords for management changes
		if containsAny(combined, []string{"resignation", "resign", "cessation", "appointment", "removal", "director", "ceo", "cfo", "md", "managing director", "chief"}) {
			change := c.parseManagementChange(ann)
			if change != nil {
				changes = append(changes, *change)
			}
		}
	}

	return changes, nil
}

func (c *Checker) parseManagementChange(ann interfaces.Announcement) *types.ManagementChange {
	subject := strings.ToLower(ann.Subject)
	description := strings.ToLower(ann.Description)
	combined := subject + " " + description

	date, _ := time.Parse("2006-01-02", ann.Date)

	change := &types.ManagementChange{
		Date:       date,
		PersonName: extractPersonName(combined),
		Reason:     ann.Description,
	}

	// Determine change type
	if containsAny(combined, []string{"resignation", "resign", "cessation"}) {
		change.ChangeType = "RESIGNATION"
	} else if containsAny(combined, []string{"appointment", "appointed", "appoint"}) {
		change.ChangeType = "APPOINTMENT"
	} else if containsAny(combined, []string{"removal", "removed", "terminate"}) {
		change.ChangeType = "REMOVAL"
	} else {
		return nil // Not a relevant change
	}

	// Determine position
	if containsAny(combined, []string{"ceo", "chief executive"}) {
		change.Position = "CEO"
	} else if containsAny(combined, []string{"cfo", "chief financial"}) {
		change.Position = "CFO"
	} else if containsAny(combined, []string{"md", "managing director"}) {
		change.Position = "MD"
	} else if containsAny(combined, []string{"chairman"}) {
		change.Position = "CHAIRMAN"
	} else if containsAny(combined, []string{"director", "board"}) {
		change.Position = "DIRECTOR"
	} else {
		change.Position = "EXECUTIVE"
	}

	// Determine if abrupt (key indicators)
	change.IsAbrupt = containsAny(combined, []string{
		"immediate effect",
		"with immediate",
		"sudden",
		"unexpect",
		"health reason",
		"personal reason",
		"without successor",
	})

	// Calculate risk score
	change.RiskScore = c.calculateManagementRisk(change)

	return change
}

func (c *Checker) calculateManagementRisk(change *types.ManagementChange) float64 {
	score := 0.0

	// Base score for resignations/removals
	if change.ChangeType == "RESIGNATION" {
		score = 40.0
	} else if change.ChangeType == "REMOVAL" {
		score = 60.0
	} else if change.ChangeType == "APPOINTMENT" {
		return 10.0 // Low risk for appointments
	}

	// Position impact
	positionWeights := map[string]float64{
		"CEO":      30.0,
		"CFO":      25.0,
		"MD":       30.0,
		"CHAIRMAN": 20.0,
		"DIRECTOR": 15.0,
		"EXECUTIVE": 10.0,
	}
	score += positionWeights[change.Position]

	// Abrupt resignation is high risk
	if change.IsAbrupt {
		score += 25.0
	}

	// Recency impact (more recent = higher risk)
	daysSince := time.Since(change.Date).Hours() / 24
	if daysSince <= 30 {
		score += 15.0
	} else if daysSince <= 90 {
		score += 10.0
	} else if daysSince <= 180 {
		score += 5.0
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

func extractPersonName(text string) string {
	// Simple extraction - in production, use NLP
	// For now, return "Person" as placeholder
	return "Management Personnel"
}
