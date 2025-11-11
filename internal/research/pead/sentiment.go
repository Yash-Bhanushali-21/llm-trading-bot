package pead

import (
	"strings"
	"unicode"
)

// SentimentData represents NLP analysis of earnings communications
type SentimentData struct {
	Symbol string `json:"symbol"`
	Quarter string `json:"quarter"`

	// Overall sentiment scores (-1.0 to 1.0)
	OverallSentiment    float64 `json:"overall_sentiment"`
	ManagementTone      float64 `json:"management_tone"`
	QandASentiment      float64 `json:"q_and_a_sentiment"`

	// Linguistic features (0-1.0)
	CertaintyScore      float64 `json:"certainty_score"`      // Confidence in statements
	ForwardLookingScore float64 `json:"forward_looking_score"` // Future-oriented language
	OptimismScore       float64 `json:"optimism_score"`       // Positive outlook
	UncertaintyScore    float64 `json:"uncertainty_score"`    // Hedging language

	// Tone divergence
	ToneResultsDivergence float64 `json:"tone_results_divergence"` // Gap between tone and actual results

	// Textual complexity
	ReadabilityScore float64 `json:"readability_score"` // Flesch reading ease (0-100)

	// Word counts
	TotalWords          int     `json:"total_words"`
	PositiveWords       int     `json:"positive_words"`
	NegativeWords       int     `json:"negative_words"`
	UncertaintyWords    int     `json:"uncertainty_words"`
	LitigationWords     int     `json:"litigation_words"`

	// Derived metrics
	PositiveWordRatio   float64 `json:"positive_word_ratio"`
	NegativeWordRatio   float64 `json:"negative_word_ratio"`
	NetSentimentRatio   float64 `json:"net_sentiment_ratio"` // Positive - Negative

	// Source availability flags
	HasTranscript       bool    `json:"has_transcript"`
	HasPressRelease     bool    `json:"has_press_release"`
}

// SentimentAnalyzer analyzes textual data for sentiment and linguistic features
type SentimentAnalyzer struct {
	positiveWords   map[string]bool
	negativeWords   map[string]bool
	uncertaintyWords map[string]bool
	forwardWords    map[string]bool
	certaintyWords  map[string]bool
	litigationWords map[string]bool
}

// NewSentimentAnalyzer creates a new sentiment analyzer
func NewSentimentAnalyzer() *SentimentAnalyzer {
	return &SentimentAnalyzer{
		positiveWords:   loadPositiveWords(),
		negativeWords:   loadNegativeWords(),
		uncertaintyWords: loadUncertaintyWords(),
		forwardWords:    loadForwardLookingWords(),
		certaintyWords:  loadCertaintyWords(),
		litigationWords: loadLitigationWords(),
	}
}

// AnalyzeText performs comprehensive NLP analysis on earnings text
func (sa *SentimentAnalyzer) AnalyzeText(text string) *SentimentData {
	// Normalize text
	text = strings.ToLower(text)
	words := sa.tokenize(text)

	sentiment := &SentimentData{
		TotalWords: len(words),
	}

	// Count word categories
	for _, word := range words {
		if sa.positiveWords[word] {
			sentiment.PositiveWords++
		}
		if sa.negativeWords[word] {
			sentiment.NegativeWords++
		}
		if sa.uncertaintyWords[word] {
			sentiment.UncertaintyWords++
		}
		if sa.litigationWords[word] {
			sentiment.LitigationWords++
		}
	}

	// Calculate ratios
	if sentiment.TotalWords > 0 {
		sentiment.PositiveWordRatio = float64(sentiment.PositiveWords) / float64(sentiment.TotalWords)
		sentiment.NegativeWordRatio = float64(sentiment.NegativeWords) / float64(sentiment.TotalWords)
		sentiment.NetSentimentRatio = sentiment.PositiveWordRatio - sentiment.NegativeWordRatio
	}

	// Calculate uncertainty score
	sentiment.UncertaintyScore = sa.calculateUncertaintyScore(words)

	// Calculate certainty score (inverse of uncertainty)
	sentiment.CertaintyScore = 1.0 - sentiment.UncertaintyScore

	// Calculate forward-looking score
	sentiment.ForwardLookingScore = sa.calculateForwardLookingScore(words)

	// Calculate optimism score
	sentiment.OptimismScore = sa.calculateOptimismScore(sentiment)

	// Calculate overall sentiment
	sentiment.OverallSentiment = sa.calculateOverallSentiment(sentiment)

	// Calculate readability
	sentiment.ReadabilityScore = sa.calculateReadability(text)

	return sentiment
}

// calculateUncertaintyScore measures hedging and uncertainty language
func (sa *SentimentAnalyzer) calculateUncertaintyScore(words []string) float64 {
	uncertaintyCount := 0
	for _, word := range words {
		if sa.uncertaintyWords[word] {
			uncertaintyCount++
		}
	}

	if len(words) == 0 {
		return 0
	}

	// Normalize to 0-1 scale
	ratio := float64(uncertaintyCount) / float64(len(words))
	return min(ratio * 20, 1.0) // Scale up (typical uncertainty ~5% of words)
}

// calculateForwardLookingScore measures future-oriented language
func (sa *SentimentAnalyzer) calculateForwardLookingScore(words []string) float64 {
	forwardCount := 0
	for _, word := range words {
		if sa.forwardWords[word] {
			forwardCount++
		}
	}

	if len(words) == 0 {
		return 0
	}

	// Companies that talk more about the future tend to be more optimistic
	ratio := float64(forwardCount) / float64(len(words))
	return min(ratio * 10, 1.0) // Scale up (typical forward language ~10% of words)
}

// calculateOptimismScore combines multiple signals of optimism
func (sa *SentimentAnalyzer) calculateOptimismScore(sentiment *SentimentData) float64 {
	// Weighted combination of positive indicators
	optimism := 0.0

	// High positive word ratio
	optimism += sentiment.PositiveWordRatio * 0.4

	// Low negative word ratio
	optimism += (1.0 - sentiment.NegativeWordRatio*5) * 0.3

	// Low uncertainty
	optimism += (1.0 - sentiment.UncertaintyScore) * 0.15

	// High forward-looking language
	optimism += sentiment.ForwardLookingScore * 0.15

	return min(max(optimism, 0), 1.0)
}

// calculateOverallSentiment computes aggregate sentiment score
func (sa *SentimentAnalyzer) calculateOverallSentiment(sentiment *SentimentData) float64 {
	// Net sentiment ratio already captures positive vs negative balance
	// Scale to -1 to 1 range
	netSentiment := sentiment.NetSentimentRatio * 10 // Amplify the signal

	// Adjust for uncertainty (high uncertainty reduces confidence)
	netSentiment *= (1.0 - sentiment.UncertaintyScore * 0.5)

	return min(max(netSentiment, -1.0), 1.0)
}

// calculateReadability computes Flesch Reading Ease score
func (sa *SentimentAnalyzer) calculateReadability(text string) float64 {
	sentences := sa.countSentences(text)
	words := sa.tokenize(text)
	syllables := sa.countSyllables(words)

	if len(sentences) == 0 || len(words) == 0 {
		return 50.0 // Default mid-range
	}

	// Flesch Reading Ease: 206.835 - 1.015(words/sentences) - 84.6(syllables/words)
	wordsPerSentence := float64(len(words)) / float64(len(sentences))
	syllablesPerWord := float64(syllables) / float64(len(words))

	score := 206.835 - 1.015*wordsPerSentence - 84.6*syllablesPerWord

	// Clamp to 0-100
	return min(max(score, 0), 100)
}

// tokenize splits text into words
func (sa *SentimentAnalyzer) tokenize(text string) []string {
	var words []string
	var currentWord strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			currentWord.WriteRune(r)
		} else if currentWord.Len() > 0 {
			words = append(words, currentWord.String())
			currentWord.Reset()
		}
	}

	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

// countSentences counts sentences in text
func (sa *SentimentAnalyzer) countSentences(text string) []string {
	// Simple sentence detection based on punctuation
	sentences := strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '!' || r == '?'
	})
	return sentences
}

// countSyllables estimates syllable count for words
func (sa *SentimentAnalyzer) countSyllables(words []string) int {
	totalSyllables := 0
	for _, word := range words {
		syllables := sa.syllablesInWord(word)
		totalSyllables += syllables
	}
	return totalSyllables
}

// syllablesInWord estimates syllables in a single word
func (sa *SentimentAnalyzer) syllablesInWord(word string) int {
	word = strings.ToLower(word)
	syllables := 0
	previousWasVowel := false

	vowels := "aeiouy"

	for _, r := range word {
		isVowel := strings.ContainsRune(vowels, r)
		if isVowel && !previousWasVowel {
			syllables++
		}
		previousWasVowel = isVowel
	}

	// Adjust for silent e
	if strings.HasSuffix(word, "e") {
		syllables--
	}

	// Minimum 1 syllable per word
	if syllables < 1 {
		syllables = 1
	}

	return syllables
}

// Helper functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// Word lists based on financial sentiment dictionaries
// (Loughran-McDonald financial sentiment word lists)

func loadPositiveWords() map[string]bool {
	words := []string{
		"achieve", "attain", "benefit", "better", "competitive", "delight",
		"enhance", "excellent", "exceptional", "extraordinary", "favorable",
		"gain", "good", "great", "grew", "growth", "improve", "improved",
		"improvement", "increasing", "innovation", "innovative", "leader",
		"leading", "opportunity", "optimal", "optimistic", "outperform",
		"positive", "profitable", "progress", "prosper", "record", "remarkable",
		"robust", "solid", "strength", "strong", "succeed", "success",
		"successful", "superior", "surpass", "tremendous", "upbeat", "valuable",
		"well-positioned", "winning",
	}
	m := make(map[string]bool)
	for _, w := range words {
		m[w] = true
	}
	return m
}

func loadNegativeWords() map[string]bool {
	words := []string{
		"abandon", "adverse", "challenge", "challenging", "concern", "concerns",
		"crisis", "damage", "debt", "decline", "decrease", "deficit", "deteriorate",
		"difficult", "difficulty", "disappoint", "disappointing", "disadvantage",
		"downturn", "erode", "fail", "failure", "falling", "fear", "headwind",
		"impair", "impairment", "inability", "inadequate", "increase", "ineffective",
		"loss", "losses", "negative", "obstacle", "poor", "problem", "recession",
		"restructuring", "risk", "risks", "slow", "slowdown", "uncertain",
		"uncertainty", "underperform", "unfavorable", "unprofitable", "volatile",
		"volatility", "weak", "weakness", "worse", "worsen", "worst",
	}
	m := make(map[string]bool)
	for _, w := range words {
		m[w] = true
	}
	return m
}

func loadUncertaintyWords() map[string]bool {
	words := []string{
		"almost", "anticipate", "anticipates", "appear", "appears", "approximately",
		"assume", "assumes", "believe", "believes", "could", "depend", "depending",
		"estimate", "estimates", "expect", "expects", "forecast", "forecasts",
		"if", "intend", "intends", "likely", "may", "maybe", "might", "outlook",
		"pending", "perhaps", "plan", "plans", "possible", "possibly", "potential",
		"predict", "predicts", "project", "projects", "should", "somewhat",
		"suggest", "suggests", "uncertain", "uncertainty", "unclear", "unlikely",
		"variable", "will", "would",
	}
	m := make(map[string]bool)
	for _, w := range words {
		m[w] = true
	}
	return m
}

func loadForwardLookingWords() map[string]bool {
	words := []string{
		"ahead", "anticipate", "expect", "forecast", "future", "going forward",
		"guidance", "intend", "looking ahead", "next quarter", "next year",
		"outlook", "pipeline", "plan", "project", "projections", "prospects",
		"roadmap", "target", "targets", "upcoming", "vision", "will",
	}
	m := make(map[string]bool)
	for _, w := range words {
		m[w] = true
	}
	return m
}

func loadCertaintyWords() map[string]bool {
	words := []string{
		"absolute", "absolutely", "always", "assure", "assures", "certain",
		"certainly", "clarity", "clear", "clearly", "commit", "committed",
		"confident", "confidently", "definite", "definitely", "ensure",
		"ensures", "evident", "guaranteed", "inevitable", "must", "never",
		"obvious", "obviously", "positive", "sure", "surely", "undoubtedly",
		"unquestionable", "unquestionably", "will",
	}
	m := make(map[string]bool)
	for _, w := range words {
		m[w] = true
	}
	return m
}

func loadLitigationWords() map[string]bool {
	words := []string{
		"allege", "alleged", "allegation", "amend", "appeal", "attorney",
		"claim", "claims", "complaint", "defendant", "investigation",
		"lawsuit", "legal", "litigat", "plaintiff", "regulatory",
		"sec", "settlement", "sue", "sued", "trial", "violation",
	}
	m := make(map[string]bool)
	for _, w := range words {
		m[w] = true
	}
	return m
}
