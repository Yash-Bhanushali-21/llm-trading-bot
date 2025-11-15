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

	// Calculate traditional component scores (0-100)
	score.EarningsSurpriseScore = s.scoreEarningsSurprise(data)
	score.RevenueSurpriseScore = s.scoreRevenueSurprise(data)
	score.EarningsGrowthScore = s.scoreEarningsGrowth(data)
	score.RevenueGrowthScore = s.scoreRevenueGrowth(data)
	score.MarginExpansionScore = s.scoreMarginExpansion(data)
	score.ConsistencyScore = s.scoreConsistency(data)
	score.RevenueAccelerationScore = s.scoreRevenueAcceleration(data)

	// Calculate NLP-enhanced scores if sentiment data available
	if data.Sentiment != nil && s.config.EnableNLP {
		score.SentimentScore = s.scoreSentiment(data.Sentiment)
		score.ToneDivergenceScore = s.scoreToneDivergence(data)
		score.LinguisticQualityScore = s.scoreLinguisticQuality(data.Sentiment)
	}

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

	// Traditional PEAD components
	composite := score.EarningsSurpriseScore*weights.EarningsSurprise +
		score.RevenueSurpriseScore*weights.RevenueSurprise +
		score.EarningsGrowthScore*weights.EarningsGrowth +
		score.RevenueGrowthScore*weights.RevenueGrowth +
		score.MarginExpansionScore*weights.MarginExpansion +
		score.ConsistencyScore*weights.Consistency +
		score.RevenueAccelerationScore*weights.RevenueAcceleration

	// Add NLP components if enabled
	if s.config.EnableNLP && score.EarningsData.Sentiment != nil {
		composite += score.SentimentScore * weights.Sentiment
		composite += score.ToneDivergenceScore * weights.ToneDivergence
		composite += score.LinguisticQualityScore * weights.LinguisticQuality
	}

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

	// Add NLP insights if available
	if s.config.EnableNLP && data.Sentiment != nil {
		sentiment := data.Sentiment

		if sentiment.OverallSentiment > 0.3 {
			commentary += "Management tone is positive and optimistic. "
		} else if sentiment.OverallSentiment < -0.3 {
			commentary += "Management tone shows caution or concern. "
		}

		if sentiment.CertaintyScore > 0.7 {
			commentary += "High confidence in forward guidance. "
		}

		// Flag tone-result divergence
		earningSurprise := data.EarningSurprise()
		if earningSurprise > 0 && sentiment.OverallSentiment < 0 {
			commentary += "Note: Positive results but cautious management tone. "
		} else if earningSurprise < 0 && sentiment.OverallSentiment > 0 {
			commentary += "⚠️ Warning: Negative results but overly positive tone - potential red flag. "
		}
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

// scoreSentiment scores based on overall sentiment from NLP analysis
func (s *PEADScorer) scoreSentiment(sentiment *SentimentData) float64 {
	// Overall sentiment ranges from -1 to 1
	// Convert to 0-100 scale
	// Positive sentiment = higher score
	baseScore := (sentiment.OverallSentiment + 1) * 50 // Maps -1..1 to 0..100

	// Boost for strong net positive sentiment
	if sentiment.NetSentimentRatio > 0 {
		baseScore += sentiment.NetSentimentRatio * 100 * 0.2 // Up to 20 point boost
	}

	// Boost for optimism
	baseScore += sentiment.OptimismScore * 10 // Up to 10 point boost

	return math.Min(100, math.Max(0, baseScore))
}

// scoreToneDivergence scores based on alignment between tone and results
func (s *PEADScorer) scoreToneDivergence(data *EarningsData) float64 {
	if data.Sentiment == nil {
		return 50 // Neutral if no sentiment data
	}

	// Calculate earnings quality (beat or miss)
	earningSurprise := data.EarningSurprise()

	// Expected alignment:
	// - Positive results should have positive tone
	// - Negative results should have negative tone
	// - Divergence is suspicious (managing expectations or hiding problems)

	sentiment := data.Sentiment.OverallSentiment

	// Perfect alignment scores high
	if (earningSurprise > 0 && sentiment > 0) || (earningSurprise < 0 && sentiment < 0) {
		// Both positive or both negative = good alignment
		alignment := 1.0 - math.Abs(earningSurprise/10.0-sentiment*10.0)/20.0
		return math.Max(0, math.Min(100, 70+alignment*30)) // 70-100 range
	}

	// Divergence scenarios:
	// 1. Positive results, negative tone = cautious management (slightly negative)
	// 2. Negative results, positive tone = spin/misleading (very negative)

	if earningSurprise > 0 && sentiment < 0 {
		// Beat but negative tone - management cautious, could be good (conservative)
		return 55 + earningSurprise // Slight boost from results
	}

	if earningSurprise < 0 && sentiment > 0 {
		// Miss but positive tone - trying to spin bad news (red flag)
		return math.Max(0, 40-math.Abs(earningSurprise)) // Penalty for divergence
	}

	return 50 // Neutral
}

// scoreLinguisticQuality scores based on certainty, clarity, and forward-looking language
func (s *PEADScorer) scoreLinguisticQuality(sentiment *SentimentData) float64 {
	// High quality = confident, clear, forward-looking
	// Low quality = uncertain, complex, evasive

	qualityScore := 0.0

	// Certainty (confident language is positive)
	qualityScore += sentiment.CertaintyScore * 35 // Max 35 points

	// Readability (clearer is better)
	// Flesch score: 60-70 = standard, >70 = easy, <60 = difficult
	readabilityScore := 0.0
	if sentiment.ReadabilityScore > 70 {
		readabilityScore = 25 // Easy to read = transparent
	} else if sentiment.ReadabilityScore > 60 {
		readabilityScore = 20 // Standard
	} else if sentiment.ReadabilityScore > 50 {
		readabilityScore = 15 // Somewhat difficult
	} else {
		readabilityScore = 10 // Very complex (potentially hiding issues)
	}
	qualityScore += readabilityScore

	// Forward-looking (companies with vision score higher)
	qualityScore += sentiment.ForwardLookingScore * 25 // Max 25 points

	// Penalty for excessive uncertainty
	qualityScore -= sentiment.UncertaintyScore * 15 // Max 15 point penalty

	// Bonus for optimism
	qualityScore += sentiment.OptimismScore * 15 // Max 15 points

	return math.Max(0, math.Min(100, qualityScore))
}
