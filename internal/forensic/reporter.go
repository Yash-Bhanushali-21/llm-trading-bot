package forensic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"llm-trading-bot/internal/types"
)

// ReportFormat specifies the output format for forensic reports
type ReportFormat string

const (
	FormatJSON ReportFormat = "json"
	FormatText ReportFormat = "text"
	FormatCSV  ReportFormat = "csv"
)

// Reporter handles generation and storage of forensic reports
type Reporter struct {
	outputDir string
}

// NewReporter creates a new reporter
func NewReporter(outputDir string) *Reporter {
	return &Reporter{
		outputDir: outputDir,
	}
}

// GenerateReport creates a forensic report in the specified format
func (r *Reporter) GenerateReport(report *types.ForensicReport, format ReportFormat) (string, error) {
	switch format {
	case FormatJSON:
		return r.generateJSONReport(report)
	case FormatText:
		return r.generateTextReport(report)
	case FormatCSV:
		return r.generateCSVReport(report)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// SaveReport saves the report to disk
func (r *Reporter) SaveReport(report *types.ForensicReport, format ReportFormat) (string, error) {
	content, err := r.GenerateReport(report, format)
	if err != nil {
		return "", err
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return "", err
	}

	// Generate filename
	timestamp := report.Timestamp.Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_forensic_%s.%s", report.Symbol, timestamp, format)
	filepath := filepath.Join(r.outputDir, filename)

	// Write to file
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return "", err
	}

	return filepath, nil
}

func (r *Reporter) generateJSONReport(report *types.ForensicReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *Reporter) generateTextReport(report *types.ForensicReport) (string, error) {
	var sb strings.Builder

	sb.WriteString("=" + strings.Repeat("=", 78) + "\n")
	sb.WriteString(fmt.Sprintf("FORENSIC ANALYSIS REPORT - %s\n", report.Symbol))
	sb.WriteString("=" + strings.Repeat("=", 78) + "\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n", report.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Overall Risk Score: %.2f/100\n", report.OverallRiskScore))
	sb.WriteString("\n")

	// Risk level
	riskLevel := "LOW"
	if report.OverallRiskScore >= 75 {
		riskLevel = "CRITICAL"
	} else if report.OverallRiskScore >= 60 {
		riskLevel = "HIGH"
	} else if report.OverallRiskScore >= 40 {
		riskLevel = "MEDIUM"
	}
	sb.WriteString(fmt.Sprintf("Risk Level: %s\n", riskLevel))
	sb.WriteString("\n")

	// Summary of red flags
	sb.WriteString(fmt.Sprintf("RED FLAGS DETECTED: %d\n", len(report.RedFlags)))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	if len(report.RedFlags) > 0 {
		// Sort by severity and impact
		sortedFlags := make([]types.RedFlag, len(report.RedFlags))
		copy(sortedFlags, report.RedFlags)
		sort.Slice(sortedFlags, func(i, j int) bool {
			severityOrder := map[string]int{"CRITICAL": 4, "HIGH": 3, "MEDIUM": 2, "LOW": 1}
			if severityOrder[sortedFlags[i].Severity] != severityOrder[sortedFlags[j].Severity] {
				return severityOrder[sortedFlags[i].Severity] > severityOrder[sortedFlags[j].Severity]
			}
			return sortedFlags[i].Impact > sortedFlags[j].Impact
		})

		for i, flag := range sortedFlags {
			sb.WriteString(fmt.Sprintf("\n%d. [%s] %s\n", i+1, flag.Severity, flag.Category))
			sb.WriteString(fmt.Sprintf("   %s\n", flag.Description))
			sb.WriteString(fmt.Sprintf("   Impact: %.2f/100\n", flag.Impact))
		}
	} else {
		sb.WriteString("\nNo significant red flags detected.\n")
	}

	// Detailed sections
	r.addManagementSection(&sb, report)
	r.addAuditorSection(&sb, report)
	r.addRelatedPartySection(&sb, report)
	r.addPledgeSection(&sb, report)
	r.addRegulatorySection(&sb, report)
	r.addInsiderTradingSection(&sb, report)
	r.addRestatementSection(&sb, report)
	r.addGovernanceSection(&sb, report)

	sb.WriteString("\n" + strings.Repeat("=", 80) + "\n")
	sb.WriteString("END OF REPORT\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n")

	return sb.String(), nil
}

func (r *Reporter) addManagementSection(sb *strings.Builder, report *types.ForensicReport) {
	if len(report.ManagementChanges) == 0 {
		return
	}

	sb.WriteString("\n\n" + strings.Repeat("=", 80) + "\n")
	sb.WriteString("MANAGEMENT CHANGES\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n")

	for _, change := range report.ManagementChanges {
		sb.WriteString(fmt.Sprintf("\n• %s - %s (%s)\n", change.Date.Format("2006-01-02"), change.Position, change.ChangeType))
		sb.WriteString(fmt.Sprintf("  Risk Score: %.2f/100\n", change.RiskScore))
		if change.IsAbrupt {
			sb.WriteString("  ⚠ ABRUPT CHANGE\n")
		}
	}
}

func (r *Reporter) addAuditorSection(sb *strings.Builder, report *types.ForensicReport) {
	if len(report.AuditorChanges) == 0 {
		return
	}

	sb.WriteString("\n\n" + strings.Repeat("=", 80) + "\n")
	sb.WriteString("AUDITOR CHANGES\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n")

	for _, change := range report.AuditorChanges {
		sb.WriteString(fmt.Sprintf("\n• %s: %s → %s\n", change.Date.Format("2006-01-02"), change.OldAuditor, change.NewAuditor))
		sb.WriteString(fmt.Sprintf("  Risk Score: %.2f/100\n", change.RiskScore))
		if change.HasQualification {
			sb.WriteString("  ⚠ HAS QUALIFICATION\n")
		}
		if change.IsMidTerm {
			sb.WriteString("  ⚠ MID-TERM CHANGE\n")
		}
	}
}

func (r *Reporter) addRelatedPartySection(sb *strings.Builder, report *types.ForensicReport) {
	if len(report.RelatedPartyTxns) == 0 {
		return
	}

	sb.WriteString("\n\n" + strings.Repeat("=", 80) + "\n")
	sb.WriteString("RELATED PARTY TRANSACTIONS\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n")

	for _, txn := range report.RelatedPartyTxns {
		sb.WriteString(fmt.Sprintf("\n• %s: %s with %s (%s)\n", txn.Date.Format("2006-01-02"), txn.TransactionType, txn.PartyName, txn.Relationship))
		sb.WriteString(fmt.Sprintf("  Amount: ₹%.2fM\n", txn.Amount/1000000))
		sb.WriteString(fmt.Sprintf("  Risk Score: %.2f/100\n", txn.RiskScore))
		if !txn.IsAtArmLength {
			sb.WriteString("  ⚠ NOT AT ARM'S LENGTH\n")
		}
		if txn.ExceedsThreshold {
			sb.WriteString("  ⚠ EXCEEDS MATERIALITY THRESHOLD\n")
		}
	}
}

func (r *Reporter) addPledgeSection(sb *strings.Builder, report *types.ForensicReport) {
	if len(report.PromoterPledges) == 0 {
		return
	}

	sb.WriteString("\n\n" + strings.Repeat("=", 80) + "\n")
	sb.WriteString("PROMOTER PLEDGES\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n")

	for _, pledge := range report.PromoterPledges {
		sb.WriteString(fmt.Sprintf("\n• %s: %s\n", pledge.Date.Format("2006-01-02"), pledge.PromoterName))
		sb.WriteString(fmt.Sprintf("  Pledged: %.2f%% of holdings\n", pledge.PledgePercentage))
		sb.WriteString(fmt.Sprintf("  Risk Score: %.2f/100\n", pledge.RiskScore))
		if pledge.IsIncrease {
			sb.WriteString("  ⚠ PLEDGE INCREASED\n")
		}
	}
}

func (r *Reporter) addRegulatorySection(sb *strings.Builder, report *types.ForensicReport) {
	if len(report.RegulatoryActions) == 0 {
		return
	}

	sb.WriteString("\n\n" + strings.Repeat("=", 80) + "\n")
	sb.WriteString("REGULATORY ACTIONS\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n")

	for _, action := range report.RegulatoryActions {
		sb.WriteString(fmt.Sprintf("\n• %s: %s by %s\n", action.Date.Format("2006-01-02"), action.ActionType, action.Regulator))
		sb.WriteString(fmt.Sprintf("  Status: %s\n", action.Status))
		if action.PenaltyAmount > 0 {
			sb.WriteString(fmt.Sprintf("  Penalty: ₹%.2fM\n", action.PenaltyAmount/1000000))
		}
		sb.WriteString(fmt.Sprintf("  Risk Score: %.2f/100\n", action.RiskScore))
	}
}

func (r *Reporter) addInsiderTradingSection(sb *strings.Builder, report *types.ForensicReport) {
	if len(report.InsiderTrading) == 0 {
		return
	}

	sb.WriteString("\n\n" + strings.Repeat("=", 80) + "\n")
	sb.WriteString("INSIDER TRADING\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n")

	for _, trade := range report.InsiderTrading {
		if trade.RiskScore < 40 {
			continue // Only show high-risk trades
		}
		sb.WriteString(fmt.Sprintf("\n• %s: %s by %s (%s)\n", trade.Date.Format("2006-01-02"), trade.TransactionType, trade.InsiderName, trade.Designation))
		sb.WriteString(fmt.Sprintf("  Quantity: %d shares, Value: ₹%.2fM\n", trade.Quantity, trade.Value/1000000))
		sb.WriteString(fmt.Sprintf("  Risk Score: %.2f/100\n", trade.RiskScore))
		if trade.IsUnusual {
			sb.WriteString("  ⚠ UNUSUAL TRADE\n")
		}
		if trade.ClusteredTrades {
			sb.WriteString("  ⚠ CLUSTERED WITH OTHER INSIDERS\n")
		}
	}
}

func (r *Reporter) addRestatementSection(sb *strings.Builder, report *types.ForensicReport) {
	if len(report.Restatements) == 0 {
		return
	}

	sb.WriteString("\n\n" + strings.Repeat("=", 80) + "\n")
	sb.WriteString("FINANCIAL RESTATEMENTS\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n")

	for _, restatement := range report.Restatements {
		sb.WriteString(fmt.Sprintf("\n• %s: Period %s\n", restatement.Date.Format("2006-01-02"), restatement.Period))
		sb.WriteString(fmt.Sprintf("  Items Affected: %s\n", strings.Join(restatement.ItemsAffected, ", ")))
		if restatement.ImpactPercentage > 0 {
			sb.WriteString(fmt.Sprintf("  Impact: %.2f%%\n", restatement.ImpactPercentage))
		}
		sb.WriteString(fmt.Sprintf("  Risk Score: %.2f/100\n", restatement.RiskScore))
		if restatement.IsMaterial {
			sb.WriteString("  ⚠ MATERIAL RESTATEMENT\n")
		}
	}
}

func (r *Reporter) addGovernanceSection(sb *strings.Builder, report *types.ForensicReport) {
	if len(report.GovernanceScores) == 0 {
		return
	}

	sb.WriteString("\n\n" + strings.Repeat("=", 80) + "\n")
	sb.WriteString("GOVERNANCE SCORES\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n")

	for _, score := range report.GovernanceScores {
		sb.WriteString(fmt.Sprintf("\n• %s: %s by %s\n", score.Date.Format("2006-01-02"), score.Grade, score.Provider))
		sb.WriteString(fmt.Sprintf("  Score: %.2f/100\n", score.Score))
		if score.Change != 0 {
			sb.WriteString(fmt.Sprintf("  Change: %.2f\n", score.Change))
		}
		sb.WriteString(fmt.Sprintf("  Risk Score: %.2f/100\n", score.RiskScore))
		if score.IsDegraded {
			sb.WriteString("  ⚠ DEGRADED\n")
		}
	}
}

func (r *Reporter) generateCSVReport(report *types.ForensicReport) (string, error) {
	var sb strings.Builder

	// Header
	sb.WriteString("Symbol,Timestamp,OverallRiskScore,RedFlagCount\n")
	sb.WriteString(fmt.Sprintf("%s,%s,%.2f,%d\n\n",
		report.Symbol,
		report.Timestamp.Format("2006-01-02 15:04:05"),
		report.OverallRiskScore,
		len(report.RedFlags)))

	// Red flags summary
	sb.WriteString("Category,Severity,Description,Impact\n")
	for _, flag := range report.RedFlags {
		sb.WriteString(fmt.Sprintf("%s,%s,\"%s\",%.2f\n",
			flag.Category,
			flag.Severity,
			strings.ReplaceAll(flag.Description, "\"", "\"\""), // Escape quotes
			flag.Impact))
	}

	return sb.String(), nil
}
