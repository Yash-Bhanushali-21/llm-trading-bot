package pead

import (
	"fmt"
	"math"
	"time"
)

// PEADScorer handles the scoring logic for PEAD analysis
type PEADScorer struct {
	config PEADConfig
}

// NewPEADScorer creates a new scorer with the given configuration
func NewPEADScorer(config PEADConfig) *PEADScorer {
	return &PEADScorer{
		config: config,
	}
}

// CalculateScore computes a comprehensive PEAD score for earnings data
func (s *PEADScorer) CalculateScore(data *EarningsData) *PEADScore {
	score := &PEADScore{
		Symbol:            data.Symbol,
		Quarter:           data.Quarter,
		AnnouncementDate:  data.AnnouncementDate,
		DaysSinceEarnings: int(time.Since(data.AnnouncementDate).Hours() / 24),
		EarningsData:      *data,
	}

	// Calculate individual component scores (0-100)
	score.EarningsSurpriseScore = s.scoreEarningsSurprise(data)
	score.RevenueSurpriseScore = s.scoreRevenueSurprise(data)
	score.EarningsGrowthScore = s.scoreEarningsGrowth(data)
	score.RevenueGrowthScore = s.scoreRevenueGrowth(data)
	score.MarginExpansionScore = s.scoreMarginExpansion(data)
	score.ConsistencyScore = s.scoreConsistency(data)
	score.RevenueAccelerationScore = s.scoreRevenueAcceleration(data)

	// Calculate weighted composite score
	score.CompositeScore = s.calculateCompositeScore(score)

	// Assign rating and commentary
	score.Rating = s.assignRating(score)
	score.Commentary = s.generateCommentary(score)

	return score
}

// scoreEarningsSurprise scores based on earnings surprise magnitude
func (s *PEADScorer) scoreEarningsSurprise(data *EarningsData) float64 {
	surprise := data.EarningSurprise()

	// Negative surprises get 0 score
	if surprise < 0 {
		return 0
	}

	// Score curve: 0-5% surprise = 0-50 points, 5-15%+ surprise = 50-100 points
	if surprise <= 5.0 {
		return surprise * 10 // 0-50 points
	}

	// Diminishing returns above 5%
	excessSurprise := surprise - 5.0
	additionalPoints := 50 * (1 - math.Exp(-excessSurprise/5.0))
	return math.Min(50+additionalPoints, 100)
}

// scoreRevenueSurprise scores based on revenue surprise
func (s *PEADScorer) scoreRevenueSurprise(data *EarningsData) float64 {
	surprise := data.RevenueSurprise()

	// Negative surprises get lower scores
	if surprise < 0 {
		// Penalty for revenue miss, but not zero (earnings might still be good)
		return math.Max(0, 30+surprise*10) // Down to 0 at -3% miss
	}

	// Positive surprise scoring (revenue surprises typically smaller than EPS)
	if surprise <= 3.0 {
		return 50 + surprise*10 // 50-80 points
	}

	// Exceptional revenue surprise
	excessSurprise := surprise - 3.0
	additionalPoints := 20 * (1 - math.Exp(-excessSurprise/2.0))
	return math.Min(80+additionalPoints, 100)
}

// scoreEarningsGrowth scores based on YoY EPS growth
func (s *PEADScorer) scoreEarningsGrowth(data *EarningsData) float64 {
	growth := data.YoYEPSGrowth

	// Negative growth gets penalized
	if growth < 0 {
		return math.Max(0, 40+growth) // 0 at -40% or worse
	}

	// Growth scoring: 0-20% = 40-70 points, 20-50% = 70-90 points, 50%+ = 90-100
	if growth <= 20.0 {
		return 40 + growth*1.5 // 40-70 points
	} else if growth <= 50.0 {
		return 70 + (growth-20.0)*0.67 // 70-90 points
	} else {
		// Exceptional growth (50%+)
		excessGrowth := growth - 50.0
		additionalPoints := 10 * (1 - math.Exp(-excessGrowth/20.0))
		return math.Min(90+additionalPoints, 100)
	}
}

// scoreRevenueGrowth scores based on YoY revenue growth
func (s *PEADScorer) scoreRevenueGrowth(data *EarningsData) float64 {
	growth := data.YoYRevenueGrowth

	// Negative growth is concerning
	if growth < 0 {
		return math.Max(0, 30+growth*2) // 0 at -15% or worse
	}

	// Revenue growth scoring: 0-15% = 30-60 points, 15-30% = 60-85 points, 30%+ = 85-100
	if growth <= 15.0 {
		return 30 + growth*2 // 30-60 points
	} else if growth <= 30.0 {
		return 60 + (growth-15.0)*1.67 // 60-85 points
	} else {
		// Exceptional revenue growth
		excessGrowth := growth - 30.0
		additionalPoints := 15 * (1 - math.Exp(-excessGrowth/15.0))
		return math.Min(85+additionalPoints, 100)
	}
}

// scoreMarginExpansion scores based on profit margin improvements
func (s *PEADScorer) scoreMarginExpansion(data *EarningsData) float64 {
	// Calculate weighted margin change
	grossMarginChange := data.GrossMarginChange()
	operatingMarginChange := data.OperatingMarginChange()
	netMarginChange := data.NetMarginChange()

	// Weighted average (net margin most important)
	avgMarginChange := (grossMarginChange*0.2 + operatingMarginChange*0.3 + netMarginChange*0.5)

	// Base score starts at 50 (neutral)
	baseScore := 50.0

	// Each 1% margin expansion = 10 points (up or down)
	marginScore := baseScore + avgMarginChange*10

	return math.Max(0, math.Min(100, marginScore))
}

// scoreConsistency scores based on consecutive earnings beats
func (s *PEADScorer) scoreConsistency(data *EarningsData) float64 {
	beats := float64(data.ConsecutiveBeats)

	// First check if current quarter is a beat
	if data.EarningSurprise() < 0 {
		return 0 // Current miss = 0 consistency score
	}

	// Score based on streak: 0 beats = 40, 1 = 50, 2 = 60, 3 = 70, 4 = 80, 5+ = 90-100
	if beats == 0 {
		return 40
	} else if beats <= 4 {
		return 40 + beats*10
	} else {
		// 5+ beats = exceptional consistency
		excessBeats := beats - 4
		additionalPoints := 20 * (1 - math.Exp(-excessBeats/3.0))
		return math.Min(80+additionalPoints, 100)
	}
}

// scoreRevenueAcceleration scores based on QoQ vs YoY revenue growth trends
func (s *PEADScorer) scoreRevenueAcceleration(data *EarningsData) float64 {
	// If QoQ growth is accelerating faster than YoY trend, it's bullish
	qoqGrowth := data.QoQRevenueGrowth
	yoyGrowth := data.YoYRevenueGrowth

	// Expected QoQ should be ~1/4 of YoY (rough approximation)
	expectedQoQ := yoyGrowth / 4.0
	acceleration := qoqGrowth - expectedQoQ

	// Base score 50 (neutral)
	baseScore := 50.0

	// Each 1% acceleration = 8 points
	accelerationScore := baseScore + acceleration*8

	return math.Max(0, math.Min(100, accelerationScore))
}

// calculateCompositeScore computes the weighted average of all component scores
func (s *PEADScorer) calculateCompositeScore(score *PEADScore) float64 {
	weights := s.config.Weights

	composite := score.EarningsSurpriseScore*weights.EarningsSurprise +
		score.RevenueSurpriseScore*weights.RevenueSurprise +
		score.EarningsGrowthScore*weights.EarningsGrowth +
		score.RevenueGrowthScore*weights.RevenueGrowth +
		score.MarginExpansionScore*weights.MarginExpansion +
		score.ConsistencyScore*weights.Consistency +
		score.RevenueAccelerationScore*weights.RevenueAcceleration

	return math.Min(100, math.Max(0, composite))
}

// assignRating assigns a qualitative rating based on composite score
func (s *PEADScorer) assignRating(score *PEADScore) string {
	composite := score.CompositeScore

	if composite >= 80 {
		return "STRONG_BUY"
	} else if composite >= 65 {
		return "BUY"
	} else if composite >= 45 {
		return "HOLD"
	} else {
		return "AVOID"
	}
}

// generateCommentary creates a human-readable summary of the analysis
func (s *PEADScorer) generateCommentary(score *PEADScore) string {
	data := &score.EarningsData

	commentary := fmt.Sprintf("%s reported %s earnings with %.1f%% EPS surprise and %.1f%% revenue surprise. ",
		data.Symbol, data.Quarter, data.EarningSurprise(), data.RevenueSurprise())

	// Highlight key strengths
	if score.EarningsGrowthScore >= 80 {
		commentary += fmt.Sprintf("Strong earnings growth of %.1f%% YoY. ", data.YoYEPSGrowth)
	}

	if score.MarginExpansionScore >= 70 {
		commentary += "Expanding profit margins indicate improving operational efficiency. "
	}

	if score.ConsistencyScore >= 70 {
		commentary += fmt.Sprintf("Consistent track record with %d consecutive beats. ", data.ConsecutiveBeats)
	}

	// Add concerns if any
	if data.YoYRevenueGrowth < 0 {
		commentary += "Revenue declining YoY is a concern. "
	}

	if data.EarningSurprise() < 0 {
		commentary += "Missed earnings expectations. "
	}

	commentary += fmt.Sprintf("Overall PEAD score: %.1f (%s).", score.CompositeScore, score.Rating)

	return commentary
}
