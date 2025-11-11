package news

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/trace"
	"llm-trading-bot/internal/types"
)

// SentimentAnalyzer analyzes news sentiment using LLM
type SentimentAnalyzer struct {
	cfg      *store.Config
	provider string // "OPENAI" or "CLAUDE"
}

// NewSentimentAnalyzer creates a new sentiment analyzer
func NewSentimentAnalyzer(cfg *store.Config) *SentimentAnalyzer {
	return &SentimentAnalyzer{
		cfg:      cfg,
		provider: cfg.LLM.Provider,
	}
}

// AnalyzeArticle analyzes sentiment of a single article
func (a *SentimentAnalyzer) AnalyzeArticle(ctx context.Context, article types.NewsArticle) (types.ArticleSentiment, error) {
	ctx, span := trace.StartSpan(ctx, "analyze-article-sentiment")
	defer span.End()

	sentiment := types.ArticleSentiment{
		ArticleTitle: article.Title,
		URL:          article.URL,
	}

	// Prepare prompt for LLM
	prompt := a.buildArticleAnalysisPrompt(article)

	// Call LLM based on provider
	var result map[string]interface{}
	var err error

	switch strings.ToUpper(a.provider) {
	case "OPENAI":
		result, err = a.analyzeWithOpenAI(ctx, prompt)
	case "CLAUDE":
		result, err = a.analyzeWithClaude(ctx, prompt)
	default:
		return sentiment, fmt.Errorf("unsupported LLM provider: %s", a.provider)
	}

	if err != nil {
		return sentiment, err
	}

	// Parse result
	if sent, ok := result["sentiment"].(string); ok {
		sentiment.Sentiment = strings.ToUpper(sent)
	}
	if score, ok := result["score"].(float64); ok {
		sentiment.Score = score
	}
	if reasoning, ok := result["reasoning"].(string); ok {
		sentiment.Reasoning = reasoning
	}
	if factors, ok := result["factors"].(map[string]interface{}); ok {
		if bo, ok := factors["business_outlook"].(float64); ok {
			sentiment.Factors.BusinessOutlook = bo
		}
		if mgmt, ok := factors["management"].(float64); ok {
			sentiment.Factors.Management = mgmt
		}
		if inv, ok := factors["investments"].(float64); ok {
			sentiment.Factors.Investments = inv
		}
	}

	return sentiment, nil
}

// AnalyzeMultipleArticles analyzes sentiment from multiple articles and aggregates
func (a *SentimentAnalyzer) AnalyzeMultipleArticles(ctx context.Context, symbol string, articles []types.NewsArticle) (types.NewsSentiment, error) {
	logger.Info(ctx, "Analyzing sentiment for multiple articles", "symbol", symbol, "count", len(articles))

	if len(articles) == 0 {
		return types.NewsSentiment{
			Symbol:           symbol,
			OverallSentiment: "NEUTRAL",
			OverallScore:     0.0,
			ArticleCount:     0,
			Summary:          "No articles found for analysis",
			Recommendation:   "Insufficient data for recommendation",
			Confidence:       0.0,
			Timestamp:        time.Now().Unix(),
		}, nil
	}

	// Analyze each article
	articleSentiments := []types.ArticleSentiment{}
	for i, article := range articles {
		sentiment, err := a.AnalyzeArticle(ctx, article)
		if err != nil {
			logger.ErrorWithErr(ctx, "Failed to analyze article", err, "article", article.Title)
			continue
		}
		articleSentiments = append(articleSentiments, sentiment)

		// Rate limiting
		if i < len(articles)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	// Aggregate sentiments
	aggregated := a.aggregateSentiments(ctx, symbol, articleSentiments)

	logger.Info(ctx, "Sentiment analysis completed", "symbol", symbol,
		"overall", aggregated.OverallSentiment, "score", aggregated.OverallScore)

	return aggregated, nil
}

// aggregateSentiments combines multiple article sentiments into overall sentiment
func (a *SentimentAnalyzer) aggregateSentiments(ctx context.Context, symbol string, articles []types.ArticleSentiment) types.NewsSentiment {
	if len(articles) == 0 {
		return types.NewsSentiment{
			Symbol:           symbol,
			OverallSentiment: "NEUTRAL",
			ArticleCount:     0,
			Timestamp:        time.Now().Unix(),
		}
	}

	// Calculate average scores
	totalScore := 0.0
	totalBusinessOutlook := 0.0
	totalManagement := 0.0
	totalInvestments := 0.0

	sentimentCounts := map[string]int{
		"POSITIVE": 0,
		"NEGATIVE": 0,
		"NEUTRAL":  0,
	}

	for _, article := range articles {
		totalScore += article.Score
		totalBusinessOutlook += article.Factors.BusinessOutlook
		totalManagement += article.Factors.Management
		totalInvestments += article.Factors.Investments

		sentimentCounts[article.Sentiment]++
	}

	count := float64(len(articles))
	avgScore := totalScore / count
	avgBusinessOutlook := totalBusinessOutlook / count
	avgManagement := totalManagement / count
	avgInvestments := totalInvestments / count

	// Determine overall sentiment
	overallSentiment := "NEUTRAL"
	if sentimentCounts["POSITIVE"] > sentimentCounts["NEGATIVE"]*2 {
		overallSentiment = "POSITIVE"
	} else if sentimentCounts["NEGATIVE"] > sentimentCounts["POSITIVE"]*2 {
		overallSentiment = "NEGATIVE"
	} else if sentimentCounts["POSITIVE"] > 0 && sentimentCounts["NEGATIVE"] > 0 {
		overallSentiment = "MIXED"
	}

	// Generate summary
	summary := fmt.Sprintf("Analyzed %d articles. Sentiment breakdown: %d positive, %d negative, %d neutral. ",
		len(articles), sentimentCounts["POSITIVE"], sentimentCounts["NEGATIVE"], sentimentCounts["NEUTRAL"])

	// Generate recommendation
	recommendation := a.generateRecommendation(overallSentiment, avgScore, avgBusinessOutlook, avgManagement, avgInvestments)

	// Calculate confidence based on article count and sentiment consistency
	confidence := a.calculateConfidence(len(articles), sentimentCounts, avgScore)

	return types.NewsSentiment{
		Symbol:           symbol,
		OverallSentiment: overallSentiment,
		OverallScore:     avgScore,
		ArticleCount:     len(articles),
		Articles:         articles,
		Summary:          summary,
		Recommendation:   recommendation,
		Confidence:       confidence,
		Timestamp:        time.Now().Unix(),
	}
}

// generateRecommendation creates investment recommendation based on sentiment factors
func (a *SentimentAnalyzer) generateRecommendation(sentiment string, score, businessOutlook, management, investments float64) string {
	if score >= 0.5 && businessOutlook > 0.3 && management > 0.2 {
		return "STRONG_BUY: Positive news sentiment with good business outlook and management"
	} else if score >= 0.3 {
		return "BUY: Generally positive sentiment, consider buying"
	} else if score <= -0.5 && businessOutlook < -0.3 {
		return "STRONG_SELL: Negative sentiment with poor business outlook"
	} else if score <= -0.3 {
		return "SELL: Negative sentiment, consider selling"
	} else if sentiment == "MIXED" {
		return "HOLD: Mixed sentiment, wait for clearer signals"
	} else {
		return "HOLD: Neutral sentiment, no strong signal"
	}
}

// calculateConfidence determines confidence level based on data quality
func (a *SentimentAnalyzer) calculateConfidence(articleCount int, sentimentCounts map[string]int, avgScore float64) float64 {
	// Base confidence on article count
	confidence := 0.0
	if articleCount >= 10 {
		confidence = 0.9
	} else if articleCount >= 5 {
		confidence = 0.7
	} else if articleCount >= 3 {
		confidence = 0.5
	} else {
		confidence = 0.3
	}

	// Reduce confidence if sentiments are very mixed
	total := float64(sentimentCounts["POSITIVE"] + sentimentCounts["NEGATIVE"] + sentimentCounts["NEUTRAL"])
	if total > 0 {
		maxCount := float64(max(sentimentCounts["POSITIVE"], sentimentCounts["NEGATIVE"], sentimentCounts["NEUTRAL"]))
		consistency := maxCount / total
		confidence *= consistency
	}

	return confidence
}

// buildArticleAnalysisPrompt creates the prompt for analyzing a single article
func (a *SentimentAnalyzer) buildArticleAnalysisPrompt(article types.NewsArticle) string {
	schema := `{
  "sentiment": "POSITIVE|NEGATIVE|NEUTRAL",
  "score": -1.0 to 1.0 (float),
  "reasoning": "brief explanation",
  "factors": {
    "business_outlook": -1.0 to 1.0,
    "management": -1.0 to 1.0,
    "investments": -1.0 to 1.0
  }
}`

	content := article.Content
	if len(content) > 2000 {
		content = content[:2000] + "..."
	}

	prompt := fmt.Sprintf(`Analyze the sentiment of this news article about %s stock for investment purposes.

Article Title: %s
Source: %s
Content: %s

Evaluate:
1. Overall sentiment (POSITIVE, NEGATIVE, or NEUTRAL)
2. Sentiment score from -1.0 (very negative) to 1.0 (very positive)
3. Business outlook: How the news affects company's future prospects
4. Management quality: Any indication of management decisions/competence
5. Investment attractiveness: Impact on investment appeal

Respond ONLY with valid JSON matching this schema:
%s`, article.Symbol, article.Title, article.Source, content, schema)

	return prompt
}

// analyzeWithOpenAI performs sentiment analysis using OpenAI
func (a *SentimentAnalyzer) analyzeWithOpenAI(ctx context.Context, prompt string) (map[string]interface{}, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY missing")
	}

	systemPrompt := "You are a financial analyst expert at analyzing news sentiment for investment decisions. Respond ONLY with valid JSON."

	body := map[string]any{
		"model": a.cfg.LLM.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.1,
		"max_tokens":  500,
	}
	bb, _ := json.Marshal(body)

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(bb))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai http %d", resp.StatusCode)
	}

	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	if len(r.Choices) == 0 {
		return nil, errors.New("no choices")
	}

	content := strings.TrimSpace(r.Choices[0].Message.Content)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	return result, nil
}

// analyzeWithClaude performs sentiment analysis using Claude
func (a *SentimentAnalyzer) analyzeWithClaude(ctx context.Context, prompt string) (map[string]interface{}, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, errors.New("ANTHROPIC_API_KEY missing")
	}

	systemPrompt := "You are a financial analyst expert at analyzing news sentiment for investment decisions. Respond ONLY with valid JSON."

	body := map[string]any{
		"model":      a.cfg.LLM.Model,
		"max_tokens": 500,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	bb, _ := json.Marshal(body)

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bb))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("claude http %d", resp.StatusCode)
	}

	var r struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	if len(r.Content) == 0 {
		return nil, errors.New("no content")
	}

	content := strings.TrimSpace(r.Content[0].Text)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	return result, nil
}

func max(a, b, c int) int {
	if a > b && a > c {
		return a
	}
	if b > c {
		return b
	}
	return c
}
