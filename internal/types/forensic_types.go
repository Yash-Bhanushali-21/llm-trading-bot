package types

import "time"

// ForensicReport represents the complete forensic analysis for a symbol
type ForensicReport struct {
	Symbol            string                   `json:"symbol"`
	Timestamp         time.Time                `json:"timestamp"`
	OverallRiskScore  float64                  `json:"overall_risk_score"` // 0-100, higher = more risky
	RedFlags          []RedFlag                `json:"red_flags"`
	ManagementChanges []ManagementChange       `json:"management_changes,omitempty"`
	AuditorChanges    []AuditorChange          `json:"auditor_changes,omitempty"`
	RelatedPartyTxns  []RelatedPartyTxn        `json:"related_party_txns,omitempty"`
	PromoterPledges   []PromoterPledge         `json:"promoter_pledges,omitempty"`
	RegulatoryActions []RegulatoryAction       `json:"regulatory_actions,omitempty"`
	InsiderTrading    []InsiderTrade           `json:"insider_trading,omitempty"`
	Restatements      []FinancialRestatement   `json:"restatements,omitempty"`
	GovernanceScores  []GovernanceScore        `json:"governance_scores,omitempty"`
}

// RedFlag represents a detected governance/forensic issue
type RedFlag struct {
	Category    string    `json:"category"`    // e.g., "MANAGEMENT", "AUDITOR", "REGULATORY"
	Severity    string    `json:"severity"`    // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	Description string    `json:"description"`
	DetectedAt  time.Time `json:"detected_at"`
	Impact      float64   `json:"impact"` // Impact score on overall risk (0-100)
}

// ManagementChange tracks changes in key management personnel
type ManagementChange struct {
	Date        time.Time `json:"date"`
	Position    string    `json:"position"`    // CEO, CFO, MD, etc.
	PersonName  string    `json:"person_name"`
	ChangeType  string    `json:"change_type"` // "RESIGNATION", "APPOINTMENT", "REMOVAL"
	Reason      string    `json:"reason,omitempty"`
	IsAbrupt    bool      `json:"is_abrupt"` // Sudden resignation without succession plan
	RiskScore   float64   `json:"risk_score"`
}

// AuditorChange tracks changes in statutory auditors
type AuditorChange struct {
	Date              time.Time `json:"date"`
	OldAuditor        string    `json:"old_auditor"`
	NewAuditor        string    `json:"new_auditor"`
	Reason            string    `json:"reason"`
	HasQualification  bool      `json:"has_qualification"`  // Qualified opinion
	QualificationText string    `json:"qualification_text,omitempty"`
	IsMidTerm         bool      `json:"is_mid_term"` // Changed before term completion
	RiskScore         float64   `json:"risk_score"`
}

// RelatedPartyTxn represents related party transactions
type RelatedPartyTxn struct {
	Date             time.Time `json:"date"`
	PartyName        string    `json:"party_name"`
	Relationship     string    `json:"relationship"` // "PROMOTER", "SUBSIDIARY", "ASSOCIATE"
	TransactionType  string    `json:"transaction_type"` // "SALE", "PURCHASE", "LOAN", "GUARANTEE"
	Amount           float64   `json:"amount"`
	IsAtArmLength    bool      `json:"is_at_arm_length"`
	ExceedsThreshold bool      `json:"exceeds_threshold"` // Above materiality threshold
	RiskScore        float64   `json:"risk_score"`
}

// PromoterPledge tracks pledging of promoter shares
type PromoterPledge struct {
	Date               time.Time `json:"date"`
	PromoterName       string    `json:"promoter_name"`
	SharesPledged      int64     `json:"shares_pledged"`
	TotalShares        int64     `json:"total_shares"`
	PledgePercentage   float64   `json:"pledge_percentage"`
	IsIncrease         bool      `json:"is_increase"`
	ChangePercentage   float64   `json:"change_percentage,omitempty"`
	Lender             string    `json:"lender,omitempty"`
	RiskScore          float64   `json:"risk_score"`
}

// RegulatoryAction tracks actions by regulators (SEBI, NSE, BSE, etc.)
type RegulatoryAction struct {
	Date         time.Time `json:"date"`
	Regulator    string    `json:"regulator"`    // "SEBI", "NSE", "BSE", "ROC", "MCA"
	ActionType   string    `json:"action_type"`  // "PENALTY", "WARNING", "SUSPENSION", "INVESTIGATION"
	Description  string    `json:"description"`
	PenaltyAmount float64  `json:"penalty_amount,omitempty"`
	Status       string    `json:"status"`       // "ONGOING", "RESOLVED", "APPEALED"
	RiskScore    float64   `json:"risk_score"`
}

// InsiderTrade tracks insider trading patterns
type InsiderTrade struct {
	Date            time.Time `json:"date"`
	InsiderName     string    `json:"insider_name"`
	Designation     string    `json:"designation"`
	TransactionType string    `json:"transaction_type"` // "BUY", "SELL"
	Quantity        int64     `json:"quantity"`
	Value           float64   `json:"value"`
	AvgPrice        float64   `json:"avg_price"`
	IsUnusual       bool      `json:"is_unusual"` // Unusual timing or volume
	ClusteredTrades bool      `json:"clustered_trades"` // Multiple insiders trading together
	RiskScore       float64   `json:"risk_score"`
}

// FinancialRestatement tracks restatements of financial results
type FinancialRestatement struct {
	Date              time.Time `json:"date"`
	Period            string    `json:"period"`            // FY/Quarter being restated
	RestatementReason string    `json:"restatement_reason"`
	ItemsAffected     []string  `json:"items_affected"` // Revenue, Expenses, etc.
	OriginalValue     float64   `json:"original_value,omitempty"`
	RestatedValue     float64   `json:"restated_value,omitempty"`
	ImpactPercentage  float64   `json:"impact_percentage,omitempty"`
	IsMaterial        bool      `json:"is_material"`
	RiskScore         float64   `json:"risk_score"`
}

// GovernanceScore tracks changes in governance ratings
type GovernanceScore struct {
	Date       time.Time `json:"date"`
	Provider   string    `json:"provider"`   // Rating agency/provider
	Score      float64   `json:"score"`      // Normalized to 0-100
	Grade      string    `json:"grade,omitempty"` // A+, A, B, etc.
	Change     float64   `json:"change,omitempty"` // Change from previous score
	IsDegraded bool      `json:"is_degraded"`
	Rationale  string    `json:"rationale,omitempty"`
	RiskScore  float64   `json:"risk_score"`
}

// ForensicConfig holds configuration for forensic analysis
type ForensicConfig struct {
	Enabled               bool    `yaml:"enabled"`
	LookbackDays          int     `yaml:"lookback_days"`          // How far back to analyze
	MinRiskScore          float64 `yaml:"min_risk_score"`         // Minimum score to trigger alert
	CheckManagement       bool    `yaml:"check_management"`
	CheckAuditor          bool    `yaml:"check_auditor"`
	CheckRelatedParty     bool    `yaml:"check_related_party"`
	CheckPromoterPledge   bool    `yaml:"check_promoter_pledge"`
	CheckRegulatory       bool    `yaml:"check_regulatory"`
	CheckInsiderTrading   bool    `yaml:"check_insider_trading"`
	CheckRestatements     bool    `yaml:"check_restatements"`
	CheckGovernance       bool    `yaml:"check_governance"`
	PromoterPledgeThreshold float64 `yaml:"promoter_pledge_threshold"` // % above which to flag
}
