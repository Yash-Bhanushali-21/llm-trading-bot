package openai

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
	"llm-trading-bot/internal/trace"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/types"
)

// OpenAIDecider implements the Decider interface using OpenAI's API
type OpenAIDecider struct {
	cfg *store.Config
}

// NewOpenAIDecider creates a new OpenAI-based decider
func NewOpenAIDecider(cfg *store.Config) *OpenAIDecider {
	return &OpenAIDecider{cfg: cfg}
}

// Decide makes a trading decision using OpenAI's API
func (d *OpenAIDecider) Decide(ctx context.Context, symbol string, latest types.Candle, inds types.Indicators, ctxmap map[string]any) (types.Decision, error) {
	logger.Debug(ctx, "OpenAI decider called", "symbol", symbol, "model", d.cfg.LLM.Model)

	// Create span for LLM API call
	ctx, span := trace.StartSpan(ctx, "openai-decide")
	defer span.End()

	// Validate API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		err := errors.New("OPENAI_API_KEY missing")
		logger.ErrorWithErr(ctx, "OpenAI API key not configured", err)
		return types.Decision{}, err
	}

	// Prepare request
	logger.Debug(ctx, "Preparing OpenAI API request", "symbol", symbol)
	user := map[string]any{"symbol": symbol, "latest": latest, "indicators": inds, "context": ctxmap}
	ub, _ := json.Marshal(user)
	prompt := fmt.Sprintf("You will receive state as JSON. Respond ONLY with compact JSON matching the schema.\nSchema:%s\nState:%s", d.cfg.LLM.Schema, string(ub))
	body := map[string]any{
		"model":       d.cfg.LLM.Model,
		"messages":    []map[string]string{{"role": "system", "content": d.cfg.LLM.System}, {"role": "user", "content": prompt}},
		"temperature": d.cfg.LLM.Temperature,
		"max_tokens":  d.cfg.LLM.MaxTokens,
	}
	bb, _ := json.Marshal(body)

	// Make API request
	logger.Debug(ctx, "Sending request to OpenAI", "model", d.cfg.LLM.Model, "temperature", d.cfg.LLM.Temperature)
	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(bb))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		logger.ErrorWithErr(ctx, "OpenAI API request failed", err, "symbol", symbol, "latency_ms", latency.Milliseconds())
		return types.Decision{}, err
	}
	defer resp.Body.Close()

	logger.Debug(ctx, "Received response from OpenAI",
		"symbol", symbol,
		"status_code", resp.StatusCode,
		"latency_ms", latency.Milliseconds(),
	)

	if resp.StatusCode >= 300 {
		err := fmt.Errorf("openai http %d", resp.StatusCode)
		logger.ErrorWithErr(ctx, "OpenAI API returned error status", err, "symbol", symbol, "status_code", resp.StatusCode)
		return types.Decision{}, err
	}

	// Parse response
	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		logger.ErrorWithErr(ctx, "Failed to decode OpenAI response", err, "symbol", symbol)
		return types.Decision{}, err
	}

	if len(r.Choices) == 0 {
		err := errors.New("no choices")
		logger.Error(ctx, "OpenAI returned no choices", "symbol", symbol)
		return types.Decision{}, err
	}

	out := strings.TrimSpace(r.Choices[0].Message.Content)
	logger.Debug(ctx, "OpenAI raw response", "symbol", symbol, "content", out)

	// Parse decision JSON
	var dres types.Decision
	if err := json.Unmarshal([]byte(out), &dres); err != nil {
		logger.Warn(ctx, "Failed to parse OpenAI decision JSON, defaulting to HOLD",
			"symbol", symbol,
			"error", err,
			"raw_content", out,
		)
		return types.Decision{Action: "HOLD", Reason: "invalid_json", Confidence: 0.0}, nil
	}

	// Normalize and validate decision
	dres.Action = strings.ToUpper(strings.TrimSpace(dres.Action))
	valid := map[string]bool{"BUY": true, "SELL": true, "HOLD": true}
	if !valid[dres.Action] {
		logger.Warn(ctx, "Invalid action from OpenAI, defaulting to HOLD",
			"symbol", symbol,
			"invalid_action", dres.Action,
		)
		dres.Action = "HOLD"
	}
	if dres.Confidence < 0 || dres.Confidence > 1 {
		logger.Warn(ctx, "Invalid confidence from OpenAI, clamping to 0",
			"symbol", symbol,
			"invalid_confidence", dres.Confidence,
		)
		dres.Confidence = 0.0
	}

	logger.Info(ctx, "OpenAI decision received",
		"symbol", symbol,
		"action", dres.Action,
		"confidence", dres.Confidence,
		"reason", dres.Reason,
		"latency_ms", latency.Milliseconds(),
	)

	return dres, nil
}
