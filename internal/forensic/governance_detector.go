package forensic

import (
	"llm-trading-bot/internal/interfaces"
	"context"
	"strconv"
	"strings"
	"time"

	"llm-trading-bot/internal/types"
)

// CheckGovernanceScore monitors governance score degradation
func (c *Checker) CheckGovernanceScore(ctx context.Context, symbol string) ([]types.GovernanceScore, error) {
	fromDate := time.Now().AddDate(0, 0, -c.cfg.LookbackDays).Format("2006-01-02")
	toDate := time.Now().Format("2006-01-02")

	announcements, err := c.dataSource.FetchAnnouncements(ctx, symbol, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	scores := []types.GovernanceScore{}

	for _, ann := range announcements {
		subject := strings.ToLower(ann.Subject)
		description := strings.ToLower(ann.Description)
		combined := subject + " " + description

		// Keywords for governance ratings
		if containsAny(combined, []string{
			"governance rating",
			"governance score",
			"esg rating",
			"corporate governance",
			"rating downgrade",
			"rating upgrade",
			"governance assessment",
		}) {
			score := c.parseGovernanceScore(ann)
			if score != nil {
				scores = append(scores, *score)
			}
		}
	}

	// If we have multiple scores, calculate changes
	if len(scores) > 1 {
		c.calculateGovernanceChanges(scores)
	}

	return scores, nil
}

func (c *Checker) parseGovernanceScore(ann interfaces.Announcement) *types.GovernanceScore {
	subject := strings.ToLower(ann.Subject)
	description := strings.ToLower(ann.Description)
	combined := subject + " " + description

	date, _ := time.Parse("2006-01-02", ann.Date)

	score := &types.GovernanceScore{
		Date:      date,
		Provider:  extractProvider(combined),
		Rationale: ann.Description,
		Change:    0,
	}

	// Extract score/grade
	score.Grade = extractGrade(combined)
	score.Score = gradeToScore(score.Grade)

	// Check if downgrade
	score.IsDegraded = containsAny(combined, []string{
		"downgrade",
		"degradation",
		"decline",
		"reduced",
		"lowered",
		"deteriorate",
	})

	// Calculate risk score
	score.RiskScore = c.calculateGovernanceRisk(score)

	return score
}

func (c *Checker) calculateGovernanceChanges(scores []types.GovernanceScore) {
	// Sort by date
	for i := 1; i < len(scores); i++ {
		if scores[i].Provider == scores[i-1].Provider {
			scores[i].Change = scores[i].Score - scores[i-1].Score
			if scores[i].Change < 0 {
				scores[i].IsDegraded = true
				scores[i].RiskScore = calculateGovernanceRiskFromChange(&scores[i])
			}
		}
	}
}

func (c *Checker) calculateGovernanceRisk(score *types.GovernanceScore) float64 {
	riskScore := 0.0

	// Low governance score is risky
	if score.Score < 30 {
		riskScore = 60.0
	} else if score.Score < 50 {
		riskScore = 40.0
	} else if score.Score < 70 {
		riskScore = 20.0
	} else {
		riskScore = 10.0
	}

	// Degradation is concerning
	if score.IsDegraded {
		riskScore += 30.0
	}

	// Recency
	daysSince := time.Since(score.Date).Hours() / 24
	if daysSince <= 90 {
		riskScore += 15.0
	} else if daysSince <= 180 {
		riskScore += 10.0
	} else if daysSince <= 365 {
		riskScore += 5.0
	}

	if riskScore > 100 {
		riskScore = 100
	}

	return riskScore
}

func calculateGovernanceRiskFromChange(score *types.GovernanceScore) float64 {
	riskScore := 50.0 // Base score for degradation

	// Magnitude of change
	if score.Change < -30 {
		riskScore += 40.0
	} else if score.Change < -20 {
		riskScore += 30.0
	} else if score.Change < -10 {
		riskScore += 20.0
	} else if score.Change < -5 {
		riskScore += 10.0
	}

	// Current score impact
	if score.Score < 40 {
		riskScore += 10.0
	}

	if riskScore > 100 {
		riskScore = 100
	}

	return riskScore
}

func extractProvider(text string) string {
	providers := map[string][]string{
		"CRISIL":     {"crisil"},
		"ICRA":       {"icra"},
		"CARE":       {"care rating"},
		"BSE":        {"bse"},
		"NSE":        {"nse"},
		"IiAS":       {"iias", "institutional investor advisory"},
		"Sustainalytics": {"sustainalytics"},
	}

	for provider, keywords := range providers {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				return provider
			}
		}
	}

	return "Rating Agency"
}

func extractGrade(text string) string {
	// Look for grade patterns like A+, A, B+, B, C, etc.
	grades := []string{"A++", "A+", "A", "A-", "B++", "B+", "B", "B-", "C+", "C", "C-", "D"}
	for _, grade := range grades {
		if strings.Contains(strings.ToUpper(text), grade) {
			return grade
		}
	}

	// Look for numeric scores
	words := strings.Fields(text)
	for i, word := range words {
		if containsAny(word, []string{"score", "rating"}) && i > 0 {
			prevWord := strings.TrimRight(words[i-1], ",.")
			if _, err := strconv.ParseFloat(prevWord, 64); err == nil {
				return prevWord
			}
		}
	}

	return "N/A"
}

func gradeToScore(grade string) float64 {
	// Convert letter grade to 0-100 score
	gradeMap := map[string]float64{
		"A++": 95.0,
		"A+":  90.0,
		"A":   85.0,
		"A-":  80.0,
		"B++": 75.0,
		"B+":  70.0,
		"B":   65.0,
		"B-":  60.0,
		"C+":  55.0,
		"C":   50.0,
		"C-":  45.0,
		"D":   35.0,
		"N/A": 50.0,
	}

	if score, ok := gradeMap[grade]; ok {
		return score
	}

	// Try to parse as number
	if val, err := strconv.ParseFloat(grade, 64); err == nil {
		if val <= 10 { // If score is on 1-10 scale, convert to 0-100
			return val * 10
		}
		return val
	}

	return 50.0 // Default middle score
}
