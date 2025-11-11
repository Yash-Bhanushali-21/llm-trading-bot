# PEAD NLP Enhancement

## Overview

This module implements **NLP-enhanced PEAD (Post-Earnings Announcement Drift) analysis** based on academic research showing that sentiment analysis of earnings communications can improve prediction of post-earnings drift by 10-30%.

## Research Background

Traditional PEAD analysis focuses on quantitative metrics (earnings surprise, growth rates). However, research has shown that **linguistic analysis** of earnings calls, press releases, and management commentary provides additional predictive power.

### Key Findings from Research:

1. **Tone-Result Divergence**: When management tone conflicts with actual results (e.g., positive results but cautious tone, or vice versa), it predicts future drift direction
2. **Uncertainty Language**: High uncertainty in earnings calls correlates with negative drift
3. **Forward-Looking Optimism**: Confident, future-oriented language predicts positive drift
4. **Linguistic Complexity**: Overly complex language may hide bad news
5. **Sentiment Amplification**: Positive sentiment amplifies positive surprises (and vice versa)

## NLP Components

### 1. Sentiment Analysis

**Metrics Analyzed:**
- **Overall Sentiment** (-1 to 1): Net positive/negative tone
- **Management Tone**: Confidence level in prepared remarks
- **Q&A Sentiment**: Tone in response to analyst questions
- **Net Sentiment Ratio**: (Positive words - Negative words) / Total words

**Scoring (0-100):**
- Maps sentiment (-1 to 1) to 0-100 scale
- Boosts for strong net positive sentiment
- Adjusts for optimism score

### 2. Tone-Result Divergence

**Purpose**: Detect misalignment between what management says and actual results

**Scoring Logic:**
- **Alignment** (70-100 pts): Positive results + positive tone OR negative results + negative tone
- **Cautious Management** (50-65 pts): Beat earnings but negative/cautious tone (conservative)
- **Red Flag** (0-40 pts): Miss earnings but overly positive tone (spin/misleading)

**Why It Matters:**
- Perfect alignment suggests transparent management
- Divergence may indicate future revisions or hidden issues

### 3. Linguistic Quality

**Metrics:**
- **Certainty Score** (0-1): Use of confident vs. uncertain language
- **Readability** (Flesch score 0-100): Complexity of language
- **Forward-Looking** (0-1): Future-oriented statements
- **Optimism Score** (0-1): Overall positive outlook

**Scoring (0-100):**
- 35 pts: Certainty (confident language)
- 25 pts: Readability (clear, transparent communication)
- 25 pts: Forward-looking (vision for future)
- 15 pts: Optimism bonus
- -15 pts: Uncertainty penalty

## Implementation

### Word Dictionaries

Based on **Loughran-McDonald Financial Sentiment Word Lists**:

- **Positive Words**: achieve, excel, growth, improve, success, etc.
- **Negative Words**: decline, loss, concern, risk, weakness, etc.
- **Uncertainty Words**: may, could, perhaps, uncertain, etc.
- **Forward-Looking**: future, expect, guidance, target, etc.
- **Certainty Words**: absolutely, confident, clear, definite, etc.
- **Litigation Words**: lawsuit, investigation, violation, etc.

### Text Analysis

**Features Extracted:**
1. Word counts (positive, negative, uncertainty, etc.)
2. Word ratios (% of each category)
3. Sentence complexity
4. Syllable counts (for readability)
5. Flesch Reading Ease score

### Integration with Traditional PEAD

**Composite Score Calculation:**

```
When NLP Disabled (default):
  Score = 0.25×Earnings_Surprise + 0.15×Revenue_Surprise +
          0.20×Earnings_Growth + 0.15×Revenue_Growth +
          0.10×Margin_Expansion + 0.10×Consistency +
          0.05×Revenue_Acceleration

When NLP Enabled:
  Score = Traditional_Components (reduced weights) +
          0.10×Sentiment + 0.10×Tone_Divergence +
          0.05×Linguistic_Quality
```

**Recommended NLP-Enhanced Weights:**
```yaml
weights:
  earnings_surprise: 0.20      # reduced from 0.25
  revenue_surprise: 0.12       # reduced from 0.15
  earnings_growth: 0.18        # reduced from 0.20
  revenue_growth: 0.12         # reduced from 0.15
  margin_expansion: 0.08       # reduced from 0.10
  consistency: 0.08            # reduced from 0.10
  revenue_acceleration: 0.02   # reduced from 0.05
  sentiment: 0.10              # NEW
  tone_divergence: 0.10        # NEW
  linguistic_quality: 0.05     # NEW
```

## Configuration

### Enable NLP Analysis

**In `config.yaml`:**
```yaml
pead:
  enable_nlp: true              # Enable NLP features

  weights:
    # Traditional weights (reduce these when enabling NLP)
    earnings_surprise: 0.20
    revenue_surprise: 0.12
    earnings_growth: 0.18
    revenue_growth: 0.12
    margin_expansion: 0.08
    consistency: 0.08
    revenue_acceleration: 0.02

    # NLP weights (set these when enabling NLP)
    sentiment: 0.10
    tone_divergence: 0.10
    linguistic_quality: 0.05
```

### Data Sources for NLP

**Currently Supported:**
- ✅ Mock sentiment data (for testing)
- 🔜 Earnings call transcript scraping
- 🔜 Press release analysis
- 🔜 SEC filing analysis (10-Q, 10-K MD&A)

**To Add Real Data:**
1. Scrape earnings call transcripts from sources like:
   - Seeking Alpha transcripts
   - Company investor relations pages
   - SEC EDGAR filings
2. Pass transcript text to `SentimentAnalyzer.AnalyzeText()`
3. Attach sentiment data to `EarningsData.Sentiment`

## Usage Example

###Python Pseudocode (Conceptual):
```python
# Fetch earnings call transcript
transcript = fetch_earnings_call("RELIANCE", "Q2 2024")

# Analyze sentiment
analyzer = SentimentAnalyzer()
sentiment = analyzer.analyze_text(transcript)

# Attach to earnings data
earnings_data.sentiment = sentiment

# Score with NLP enhancement
scorer = PEADScorer(config_with_nlp=True)
score = scorer.calculate_score(earnings_data)

# Result includes NLP scores
print(f"Sentiment Score: {score.sentiment_score}")
print(f"Tone Divergence: {score.tone_divergence_score}")
print(f"Linguistic Quality: {score.linguistic_quality_score}")
```

### Go Usage:
```go
// Analyze text from earnings call
analyzer := pead.NewSentimentAnalyzer()
sentiment := analyzer.AnalyzeText(transcriptText)

// Attach to earnings data
earningsData.Sentiment = sentiment

// Score will automatically include NLP if enabled in config
config := pead.GetDefaultConfig()
config.EnableNLP = true
config.Weights.Sentiment = 0.10
config.Weights.ToneDivergence = 0.10
config.Weights.LinguisticQuality = 0.05

scorer := pead.NewPEADScorer(config)
score := scorer.CalculateScore(earningsData)
```

## Enhanced Commentary

When NLP is enabled, commentary includes linguistic insights:

```
RELIANCE reported Q2 2024 earnings with 7.2% EPS surprise and 7.2%
revenue surprise. Strong earnings growth of 42.8% YoY. Management tone
is positive and optimistic. High confidence in forward guidance. Overall
PEAD score: 82.5 (STRONG_BUY).
```

**Tone Divergence Warnings:**
```
⚠️ Warning: Negative results but overly positive tone - potential red flag.
```

```
Note: Positive results but cautious management tone.
```

## Performance Improvement

**Expected Benefits:**
- **10-30% improvement** in PEAD prediction accuracy
- Better detection of "hidden" issues (management spin)
- Earlier identification of confidence shifts
- Reduced false positives from accounting manipulation

**When NLP Helps Most:**
1. **Divergence Detection**: Company beats but management sounds cautious
2. **Guidance Quality**: High certainty in forward-looking statements
3. **Transparency**: Clear, simple language vs. complex obfuscation
4. **Sentiment Momentum**: Increasingly optimistic tone over quarters

## Limitations

1. **Data Availability**: Requires earnings call transcripts (not always available)
2. **Language Dependency**: Currently English-only
3. **Context Sensitivity**: Sarcasm, idioms may be misinterpreted
4. **Industry Variance**: Financial language varies by sector
5. **Computational Cost**: Text analysis adds processing time

## Future Enhancements

- [ ] Automatic transcript fetching from Seeking Alpha
- [ ] Multi-language support (Hindi for Indian companies)
- [ ] Deep learning models (BERT, FinBERT) for sentiment
- [ ] Topic modeling (what are they talking about?)
- [ ] Emotion analysis (fear, confidence, evasiveness)
- [ ] Q&A vs prepared remarks comparison
- [ ] CEO vs CFO tone differences
- [ ] Year-over-year tone change tracking

## Research References

1. **Loughran, T., & McDonald, B.** (2011). "When is a liability not a liability? Textual analysis, dictionaries, and 10-Ks." *Journal of Finance*, 66(1), 35-65.

2. **Price, S. M., Doran, J. S., Peterson, D. R., & Bliss, B. A.** (2012). "Earnings conference calls and stock returns: The incremental informativeness of textual tone." *Journal of Banking & Finance*, 36(4), 992-1011.

3. **Mayew, W. J., & Venkatachalam, M.** (2012). "The power of voice: Managerial affective states and future firm performance." *Journal of Finance*, 67(1), 1-43.

4. **Huang, X., Teoh, S. H., & Zhang, Y.** (2014). "Tone management." *The Accounting Review*, 89(3), 1083-1113.

5. **Jegadeesh, N., & Wu, D.** (2013). "Word power: A new approach for content analysis." *Journal of Financial Economics*, 110(3), 712-729.

## Contributing

To add new linguistic features:
1. Update word dictionaries in `sentiment.go`
2. Add new scoring method in `scorer.go`
3. Update `SentimentData` struct in `sentiment.go`
4. Update composite score calculation
5. Add tests

## License

Based on Loughran-McDonald sentiment word lists (freely available for academic research).
