package forensic

import (
	"llm-trading-bot/internal/interfaces"
	"context"
	"strings"
	"time"

	"llm-trading-bot/internal/types"
)

// CheckAuditorChanges detects auditor changes or qualifications
func (c *Checker) CheckAuditorChanges(ctx context.Context, symbol string) ([]types.AuditorChange, error) {
	fromDate := time.Now().AddDate(0, 0, -c.cfg.LookbackDays).Format("2006-01-02")
	toDate := time.Now().Format("2006-01-02")

	announcements, err := c.dataSource.FetchAnnouncements(ctx, symbol, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	changes := []types.AuditorChange{}

	for _, ann := range announcements {
		subject := strings.ToLower(ann.Subject)
		description := strings.ToLower(ann.Description)
		combined := subject + " " + description

		// Keywords for auditor changes
		if containsAny(combined, []string{
			"auditor",
			"statutory auditor",
			"audit",
			"qualification",
			"qualified opinion",
			"adverse opinion",
			"disclaimer",
		}) {
			change := c.parseAuditorChange(ann)
			if change != nil {
				changes = append(changes, *change)
			}
		}
	}

	return changes, nil
}

func (c *Checker) parseAuditorChange(ann interfaces.Announcement) *types.AuditorChange {
	subject := strings.ToLower(ann.Subject)
	description := strings.ToLower(ann.Description)
	combined := subject + " " + description

	// Check if it's an auditor change
	if !containsAny(combined, []string{
		"appointment of auditor",
		"change of auditor",
		"resignation of auditor",
		"auditor change",
	}) && !containsAny(combined, []string{
		"qualified opinion",
		"adverse opinion",
		"disclaimer of opinion",
		"qualification",
	}) {
		return nil
	}

	date, _ := time.Parse("2006-01-02", ann.Date)

	change := &types.AuditorChange{
		Date:      date,
		Reason:    ann.Description,
		OldAuditor: "Previous Auditor",
		NewAuditor: "New Auditor",
	}

	// Check for qualifications
	change.HasQualification = containsAny(combined, []string{
		"qualified opinion",
		"adverse opinion",
		"disclaimer",
		"qualification",
		"emphasis of matter",
		"material uncertainty",
	})

	if change.HasQualification {
		change.QualificationText = ann.Description
	}

	// Check if mid-term change (not at year-end)
	change.IsMidTerm = !containsAny(combined, []string{
		"term completion",
		"completion of tenure",
		"end of term",
		"expiry",
	})

	// Calculate risk score
	change.RiskScore = c.calculateAuditorRisk(change)

	return change
}

func (c *Checker) calculateAuditorRisk(change *types.AuditorChange) float64 {
	score := 50.0 // Base score for any auditor change

	// Qualification is very high risk
	if change.HasQualification {
		score += 40.0

		// Adverse opinion or disclaimer is critical
		qual := strings.ToLower(change.QualificationText)
		if containsAny(qual, []string{"adverse", "disclaimer"}) {
			score += 10.0
		}
	}

	// Mid-term change is suspicious
	if change.IsMidTerm {
		score += 25.0
	}

	// Recency impact
	daysSince := time.Since(change.Date).Hours() / 24
	if daysSince <= 90 {
		score += 15.0
	} else if daysSince <= 180 {
		score += 10.0
	} else if daysSince <= 365 {
		score += 5.0
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}
