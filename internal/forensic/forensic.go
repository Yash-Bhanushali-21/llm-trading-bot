package forensic

import (
	"context"
	"fmt"
	"time"

	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/types"
)

// Checker implements the ForensicChecker interface
type Checker struct {
	cfg        *types.ForensicConfig
	dataSource interfaces.CorporateDataSource
}

// NewChecker creates a new forensic checker
func NewChecker(cfg *types.ForensicConfig, dataSource interfaces.CorporateDataSource) *Checker {
	return &Checker{
		cfg:        cfg,
		dataSource: dataSource,
	}
}

// Analyze performs comprehensive forensic analysis for a symbol
func (c *Checker) Analyze(ctx context.Context, symbol string) (*types.ForensicReport, error) {
	logger.Info(ctx, "Starting forensic analysis", "symbol", symbol)

	report := &types.ForensicReport{
		Symbol:    symbol,
		Timestamp: time.Now(),
		RedFlags:  []types.RedFlag{},
	}

	// Run all enabled checks
	if c.cfg.CheckManagement {
		changes, err := c.CheckManagementChanges(ctx, symbol)
		if err != nil {
			logger.ErrorWithErr(ctx, "Failed to check management changes", err)
		} else {
			report.ManagementChanges = changes
			report.RedFlags = append(report.RedFlags, c.generateManagementRedFlags(changes)...)
		}
	}

	if c.cfg.CheckAuditor {
		changes, err := c.CheckAuditorChanges(ctx, symbol)
		if err != nil {
			logger.ErrorWithErr(ctx, "Failed to check auditor changes", err)
		} else {
			report.AuditorChanges = changes
			report.RedFlags = append(report.RedFlags, c.generateAuditorRedFlags(changes)...)
		}
	}

	if c.cfg.CheckRelatedParty {
		txns, err := c.CheckRelatedPartyTxns(ctx, symbol)
		if err != nil {
			logger.ErrorWithErr(ctx, "Failed to check related party transactions", err)
		} else {
			report.RelatedPartyTxns = txns
			report.RedFlags = append(report.RedFlags, c.generateRelatedPartyRedFlags(txns)...)
		}
	}

	if c.cfg.CheckPromoterPledge {
		pledges, err := c.CheckPromoterPledges(ctx, symbol)
		if err != nil {
			logger.ErrorWithErr(ctx, "Failed to check promoter pledges", err)
		} else {
			report.PromoterPledges = pledges
			report.RedFlags = append(report.RedFlags, c.generatePledgeRedFlags(pledges)...)
		}
	}

	if c.cfg.CheckRegulatory {
		actions, err := c.CheckRegulatoryActions(ctx, symbol)
		if err != nil {
			logger.ErrorWithErr(ctx, "Failed to check regulatory actions", err)
		} else {
			report.RegulatoryActions = actions
			report.RedFlags = append(report.RedFlags, c.generateRegulatoryRedFlags(actions)...)
		}
	}

	if c.cfg.CheckInsiderTrading {
		trades, err := c.CheckInsiderTrading(ctx, symbol)
		if err != nil {
			logger.ErrorWithErr(ctx, "Failed to check insider trading", err)
		} else {
			report.InsiderTrading = trades
			report.RedFlags = append(report.RedFlags, c.generateInsiderTradingRedFlags(trades)...)
		}
	}

	if c.cfg.CheckRestatements {
		restatements, err := c.CheckRestatements(ctx, symbol)
		if err != nil {
			logger.ErrorWithErr(ctx, "Failed to check restatements", err)
		} else {
			report.Restatements = restatements
			report.RedFlags = append(report.RedFlags, c.generateRestatementRedFlags(restatements)...)
		}
	}

	if c.cfg.CheckGovernance {
		scores, err := c.CheckGovernanceScore(ctx, symbol)
		if err != nil {
			logger.ErrorWithErr(ctx, "Failed to check governance scores", err)
		} else {
			report.GovernanceScores = scores
			report.RedFlags = append(report.RedFlags, c.generateGovernanceRedFlags(scores)...)
		}
	}

	// Calculate overall risk score
	report.OverallRiskScore = c.CalculateRiskScore(report)

	logger.Info(ctx, "Forensic analysis complete",
		"symbol", symbol,
		"risk_score", report.OverallRiskScore,
		"red_flags_count", len(report.RedFlags))

	return report, nil
}

// CalculateRiskScore computes overall risk score from all checks
func (c *Checker) CalculateRiskScore(report *types.ForensicReport) float64 {
	if report == nil {
		return 0
	}

	totalScore := 0.0
	count := 0

	// Weight and aggregate all individual risk scores
	for _, change := range report.ManagementChanges {
		totalScore += change.RiskScore
		count++
	}

	for _, change := range report.AuditorChanges {
		totalScore += change.RiskScore * 1.5 // Higher weight
		count++
	}

	for _, txn := range report.RelatedPartyTxns {
		totalScore += txn.RiskScore
		count++
	}

	for _, pledge := range report.PromoterPledges {
		totalScore += pledge.RiskScore
		count++
	}

	for _, action := range report.RegulatoryActions {
		totalScore += action.RiskScore * 1.8 // Higher weight
		count++
	}

	for _, trade := range report.InsiderTrading {
		totalScore += trade.RiskScore
		count++
	}

	for _, restatement := range report.Restatements {
		totalScore += restatement.RiskScore * 1.5 // Higher weight
		count++
	}

	for _, score := range report.GovernanceScores {
		totalScore += score.RiskScore
		count++
	}

	if count == 0 {
		return 0
	}

	// Normalize to 0-100 scale
	avgScore := totalScore / float64(count)
	if avgScore > 100 {
		avgScore = 100
	}

	return avgScore
}

// Helper functions to generate red flags from each check

func (c *Checker) generateManagementRedFlags(changes []types.ManagementChange) []types.RedFlag {
	flags := []types.RedFlag{}
	for _, change := range changes {
		if change.RiskScore >= 50 {
			severity := "MEDIUM"
			if change.RiskScore >= 75 {
				severity = "HIGH"
			} else if change.IsAbrupt {
				severity = "HIGH"
			}

			flags = append(flags, types.RedFlag{
				Category:    "MANAGEMENT",
				Severity:    severity,
				Description: fmt.Sprintf("%s %s: %s on %s", change.Position, change.ChangeType, change.PersonName, change.Date.Format("2006-01-02")),
				DetectedAt:  time.Now(),
				Impact:      change.RiskScore,
			})
		}
	}
	return flags
}

func (c *Checker) generateAuditorRedFlags(changes []types.AuditorChange) []types.RedFlag {
	flags := []types.RedFlag{}
	for _, change := range changes {
		if change.RiskScore >= 40 {
			severity := "MEDIUM"
			if change.HasQualification || change.IsMidTerm {
				severity = "HIGH"
			}
			if change.RiskScore >= 80 {
				severity = "CRITICAL"
			}

			flags = append(flags, types.RedFlag{
				Category:    "AUDITOR",
				Severity:    severity,
				Description: fmt.Sprintf("Auditor changed from %s to %s on %s", change.OldAuditor, change.NewAuditor, change.Date.Format("2006-01-02")),
				DetectedAt:  time.Now(),
				Impact:      change.RiskScore,
			})
		}
	}
	return flags
}

func (c *Checker) generateRelatedPartyRedFlags(txns []types.RelatedPartyTxn) []types.RedFlag {
	flags := []types.RedFlag{}
	for _, txn := range txns {
		if txn.RiskScore >= 50 {
			severity := "MEDIUM"
			if !txn.IsAtArmLength || txn.ExceedsThreshold {
				severity = "HIGH"
			}

			flags = append(flags, types.RedFlag{
				Category:    "RELATED_PARTY",
				Severity:    severity,
				Description: fmt.Sprintf("%s transaction with %s (%.2fM) on %s", txn.TransactionType, txn.PartyName, txn.Amount/1000000, txn.Date.Format("2006-01-02")),
				DetectedAt:  time.Now(),
				Impact:      txn.RiskScore,
			})
		}
	}
	return flags
}

func (c *Checker) generatePledgeRedFlags(pledges []types.PromoterPledge) []types.RedFlag {
	flags := []types.RedFlag{}
	for _, pledge := range pledges {
		if pledge.PledgePercentage >= c.cfg.PromoterPledgeThreshold {
			severity := "MEDIUM"
			if pledge.PledgePercentage >= 75 {
				severity = "HIGH"
			}
			if pledge.PledgePercentage >= 90 {
				severity = "CRITICAL"
			}

			flags = append(flags, types.RedFlag{
				Category:    "PROMOTER_PLEDGE",
				Severity:    severity,
				Description: fmt.Sprintf("%s pledged %.2f%% shares on %s", pledge.PromoterName, pledge.PledgePercentage, pledge.Date.Format("2006-01-02")),
				DetectedAt:  time.Now(),
				Impact:      pledge.RiskScore,
			})
		}
	}
	return flags
}

func (c *Checker) generateRegulatoryRedFlags(actions []types.RegulatoryAction) []types.RedFlag {
	flags := []types.RedFlag{}
	for _, action := range actions {
		if action.RiskScore >= 40 {
			severity := "HIGH"
			if action.ActionType == "SUSPENSION" || action.PenaltyAmount > 10000000 {
				severity = "CRITICAL"
			}

			flags = append(flags, types.RedFlag{
				Category:    "REGULATORY",
				Severity:    severity,
				Description: fmt.Sprintf("%s action by %s on %s: %s", action.ActionType, action.Regulator, action.Date.Format("2006-01-02"), action.Description),
				DetectedAt:  time.Now(),
				Impact:      action.RiskScore,
			})
		}
	}
	return flags
}

func (c *Checker) generateInsiderTradingRedFlags(trades []types.InsiderTrade) []types.RedFlag {
	flags := []types.RedFlag{}
	for _, trade := range trades {
		if trade.IsUnusual || trade.ClusteredTrades {
			severity := "MEDIUM"
			if trade.ClusteredTrades && trade.TransactionType == "SELL" {
				severity = "HIGH"
			}

			flags = append(flags, types.RedFlag{
				Category:    "INSIDER_TRADING",
				Severity:    severity,
				Description: fmt.Sprintf("%s by %s (%s): %d shares (%.2fM) on %s", trade.TransactionType, trade.InsiderName, trade.Designation, trade.Quantity, trade.Value/1000000, trade.Date.Format("2006-01-02")),
				DetectedAt:  time.Now(),
				Impact:      trade.RiskScore,
			})
		}
	}
	return flags
}

func (c *Checker) generateRestatementRedFlags(restatements []types.FinancialRestatement) []types.RedFlag {
	flags := []types.RedFlag{}
	for _, restatement := range restatements {
		if restatement.IsMaterial {
			severity := "HIGH"
			if restatement.RiskScore >= 80 {
				severity = "CRITICAL"
			}

			flags = append(flags, types.RedFlag{
				Category:    "RESTATEMENT",
				Severity:    severity,
				Description: fmt.Sprintf("Financial restatement for %s on %s: %s", restatement.Period, restatement.Date.Format("2006-01-02"), restatement.RestatementReason),
				DetectedAt:  time.Now(),
				Impact:      restatement.RiskScore,
			})
		}
	}
	return flags
}

func (c *Checker) generateGovernanceRedFlags(scores []types.GovernanceScore) []types.RedFlag {
	flags := []types.RedFlag{}
	for _, score := range scores {
		if score.IsDegraded && score.Change < -10 {
			severity := "MEDIUM"
			if score.Change < -20 {
				severity = "HIGH"
			}

			flags = append(flags, types.RedFlag{
				Category:    "GOVERNANCE",
				Severity:    severity,
				Description: fmt.Sprintf("Governance score degraded by %.1f points to %.1f (%s) by %s on %s", -score.Change, score.Score, score.Grade, score.Provider, score.Date.Format("2006-01-02")),
				DetectedAt:  time.Now(),
				Impact:      score.RiskScore,
			})
		}
	}
	return flags
}
