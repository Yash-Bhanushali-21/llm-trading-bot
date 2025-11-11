package forensic

import (
	"llm-trading-bot/internal/interfaces"
	"context"
	"strconv"
	"strings"
	"time"

	"llm-trading-bot/internal/types"
)

// CheckRelatedPartyTxns analyzes related party transactions
func (c *Checker) CheckRelatedPartyTxns(ctx context.Context, symbol string) ([]types.RelatedPartyTxn, error) {
	fromDate := time.Now().AddDate(0, 0, -c.cfg.LookbackDays).Format("2006-01-02")
	toDate := time.Now().Format("2006-01-02")

	announcements, err := c.dataSource.FetchAnnouncements(ctx, symbol, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	txns := []types.RelatedPartyTxn{}

	for _, ann := range announcements {
		subject := strings.ToLower(ann.Subject)
		description := strings.ToLower(ann.Description)
		combined := subject + " " + description

		// Keywords for related party transactions
		if containsAny(combined, []string{
			"related party",
			"related-party",
			"promoter transaction",
			"associate transaction",
			"subsidiary transaction",
		}) {
			txn := c.parseRelatedPartyTxn(ann)
			if txn != nil {
				txns = append(txns, *txn)
			}
		}
	}

	return txns, nil
}

func (c *Checker) parseRelatedPartyTxn(ann interfaces.Announcement) *types.RelatedPartyTxn {
	subject := strings.ToLower(ann.Subject)
	description := strings.ToLower(ann.Description)
	combined := subject + " " + description

	date, _ := time.Parse("2006-01-02", ann.Date)

	txn := &types.RelatedPartyTxn{
		Date:      date,
		PartyName: extractPartyName(combined),
	}

	// Determine relationship
	if containsAny(combined, []string{"promoter", "promoter group"}) {
		txn.Relationship = "PROMOTER"
	} else if containsAny(combined, []string{"subsidiary", "subsidiaries"}) {
		txn.Relationship = "SUBSIDIARY"
	} else if containsAny(combined, []string{"associate", "associated"}) {
		txn.Relationship = "ASSOCIATE"
	} else {
		txn.Relationship = "OTHER"
	}

	// Determine transaction type
	if containsAny(combined, []string{"sale", "sold", "selling"}) {
		txn.TransactionType = "SALE"
	} else if containsAny(combined, []string{"purchase", "bought", "buying"}) {
		txn.TransactionType = "PURCHASE"
	} else if containsAny(combined, []string{"loan", "lending", "advance"}) {
		txn.TransactionType = "LOAN"
	} else if containsAny(combined, []string{"guarantee", "security"}) {
		txn.TransactionType = "GUARANTEE"
	} else {
		txn.TransactionType = "OTHER"
	}

	// Extract amount (simple extraction)
	txn.Amount = extractAmount(combined)

	// Check if at arm's length
	txn.IsAtArmLength = containsAny(combined, []string{
		"arm's length",
		"arms length",
		"market rate",
		"prevailing rate",
	})

	// Check if exceeds threshold
	txn.ExceedsThreshold = containsAny(combined, []string{
		"material",
		"exceeds",
		"threshold",
		"approval required",
	}) || txn.Amount > 10000000 // 10M threshold

	// Calculate risk score
	txn.RiskScore = c.calculateRelatedPartyRisk(txn)

	return txn
}

func (c *Checker) calculateRelatedPartyRisk(txn *types.RelatedPartyTxn) float64 {
	score := 30.0 // Base score

	// Relationship risk
	relationshipWeights := map[string]float64{
		"PROMOTER":   25.0,
		"SUBSIDIARY": 15.0,
		"ASSOCIATE":  20.0,
		"OTHER":      10.0,
	}
	score += relationshipWeights[txn.Relationship]

	// Transaction type risk
	typeWeights := map[string]float64{
		"SALE":      15.0,
		"PURCHASE":  15.0,
		"LOAN":      25.0,
		"GUARANTEE": 30.0,
		"OTHER":     10.0,
	}
	score += typeWeights[txn.TransactionType]

	// Not at arm's length is high risk
	if !txn.IsAtArmLength {
		score += 20.0
	}

	// Exceeds threshold
	if txn.ExceedsThreshold {
		score += 15.0
	}

	// Amount impact
	if txn.Amount > 100000000 { // >100M
		score += 15.0
	} else if txn.Amount > 50000000 { // >50M
		score += 10.0
	} else if txn.Amount > 10000000 { // >10M
		score += 5.0
	}

	// Recency
	daysSince := time.Since(txn.Date).Hours() / 24
	if daysSince <= 90 {
		score += 10.0
	} else if daysSince <= 180 {
		score += 5.0
	}

	if score > 100 {
		score = 100
	}

	return score
}

func extractPartyName(text string) string {
	// Simple extraction - in production, use NLP
	return "Related Party"
}

func extractAmount(text string) float64 {
	// Simple amount extraction
	// Look for patterns like "Rs. 10 crore", "INR 100 lakhs", etc.
	text = strings.ReplaceAll(text, ",", "")

	// Try to find numbers
	words := strings.Fields(text)
	for i, word := range words {
		if val, err := strconv.ParseFloat(word, 64); err == nil {
			// Check for units
			if i+1 < len(words) {
				unit := strings.ToLower(words[i+1])
				if strings.Contains(unit, "crore") || strings.Contains(unit, "cr") {
					return val * 10000000 // 1 crore = 10M
				} else if strings.Contains(unit, "lakh") {
					return val * 100000 // 1 lakh = 100K
				} else if strings.Contains(unit, "million") {
					return val * 1000000
				}
			}
			return val
		}
	}
	return 0
}
