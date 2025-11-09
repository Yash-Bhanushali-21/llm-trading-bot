package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"llm-trading-bot/internal/logger"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/types"
)

// ClaudeDecider calls Anthropic Claude Messages API and returns a types.Decision.
type ClaudeDecider struct {
	cfg      *store.Config
	endpoint string
}

func NewClaudeDecider(cfg *store.Config) *ClaudeDecider {
	// default messages endpoint (public Anthropic)
	endpoint := "https://api.anthropic.com/v1/messages"
	// If you use a proxy/bedrock/vertex, set endpoint via CLAUDE_API_ENDPOINT env var
	if ep := os.Getenv("CLAUDE_API_ENDPOINT"); ep != "" {
		endpoint = ep
	}
	return &ClaudeDecider{cfg: cfg, endpoint: endpoint}
}

func (d *ClaudeDecider) Decide(ctx context.Context, symbol string, latest types.Candle, inds types.Indicators, ctxmap map[string]any) (types.Decision, error) {
	logger.Debug(ctx, "Claude decider called", "symbol", symbol, "model", d.cfg.LLM.Model, "endpoint", d.endpoint)

	// Create span for LLM API call
	ctx, span := logger.StartSpan(ctx, "claude-decide")
	defer span.End()

	// Validate API key
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		err := errors.New("CLAUDE_API_KEY missing")
		logger.ErrorWithErr(ctx, "Claude API key not configured", err)
		return types.Decision{}, err
	}

	// Build the state object the model will see
	logger.Debug(ctx, "Preparing Claude API request", "symbol", symbol)
	state := map[string]any{
		"symbol":     symbol,
		"latest":     latest,
		"indicators": inds,
		"context":    ctxmap,
	}
	stateB, _ := json.Marshal(state)

	// Compose messages (system + user)
	system := d.cfg.LLM.System
	if system == "" {
		system = "You are a disciplined equities trader. Output STRICT JSON with BUY/SELL/HOLD."
	}
	// User prompt asks model to respond with compact JSON matching schema
	user := fmt.Sprintf("Schema:%s\nState:%s\n\nRespond ONLY with compact JSON matching the schema.", d.cfg.LLM.Schema, string(stateB))

	reqBody := map[string]any{
		"model": d.cfg.LLM.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"max_tokens":  d.cfg.LLM.MaxTokens,
		"temperature": d.cfg.LLM.Temperature,
	}

	// Make API request
	bb, _ := json.Marshal(reqBody)
	logger.Debug(ctx, "Sending request to Claude", "model", d.cfg.LLM.Model, "temperature", d.cfg.LLM.Temperature, "endpoint", d.endpoint)
	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, "POST", d.endpoint, bytes.NewReader(bb))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start)

	if err != nil {
		logger.ErrorWithErr(ctx, "Claude API request failed", err, "symbol", symbol, "latency_ms", latency.Milliseconds())
		return types.Decision{}, err
	}
	defer resp.Body.Close()

	logger.Debug(ctx, "Received response from Claude",
		"symbol", symbol,
		"status_code", resp.StatusCode,
		"latency_ms", latency.Milliseconds(),
	)

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("claude http %d: %s", resp.StatusCode, string(body))
		logger.ErrorWithErr(ctx, "Claude API returned error status", err, "symbol", symbol, "status_code", resp.StatusCode)
		return types.Decision{}, err
	}

	// Read body and try to extract assistant content robustly
	respBytes, _ := io.ReadAll(resp.Body)
	logger.Debug(ctx, "Claude raw response received", "symbol", symbol, "response_length", len(respBytes))

	// Try to parse JSON and drill common fields
	var anyResp any
	if err := json.Unmarshal(respBytes, &anyResp); err != nil {
		// Not JSON? treat full body as the text response
		logger.Warn(ctx, "Claude response is not JSON, parsing as text", "symbol", symbol)
		decision, parseErr := parseDecisionFromText(ctx, string(respBytes))
		if parseErr != nil {
			logger.ErrorWithErr(ctx, "Failed to parse Claude response", parseErr, "symbol", symbol)
			return decision, parseErr
		}
		logger.Info(ctx, "Claude decision received (from text)",
			"symbol", symbol,
			"action", decision.Action,
			"confidence", decision.Confidence,
			"reason", decision.Reason,
			"latency_ms", latency.Milliseconds(),
		)
		return decision, nil
	}

	// Try common Claude messages structures: { "completion": "..."} or { "messages":[{ "role":"assistant","content":"..."}] } etc
	if m, ok := anyResp.(map[string]any); ok {
		// 1) messages array
		if msgs, found := m["messages"]; found {
			if arr, ok2 := msgs.([]any); ok2 && len(arr) > 0 {
				if first, ok3 := arr[0].(map[string]any); ok3 {
					if cont, ok4 := first["content"].(string); ok4 && strings.TrimSpace(cont) != "" {
						logger.Debug(ctx, "Extracting Claude decision from messages array", "symbol", symbol)
						decision, parseErr := parseDecisionFromText(ctx, cont)
						if parseErr != nil {
							return decision, parseErr
						}
						logger.Info(ctx, "Claude decision received",
							"symbol", symbol,
							"action", decision.Action,
							"confidence", decision.Confidence,
							"reason", decision.Reason,
							"latency_ms", latency.Milliseconds(),
						)
						return decision, nil
					}
				}
			}
		}
		// 2) completion / output_text / completion_text
		for _, k := range []string{"completion", "output", "output_text", "completion_text", "result"} {
			if v, exists := m[k]; exists {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					logger.Debug(ctx, "Extracting Claude decision from completion field", "symbol", symbol, "field", k)
					decision, parseErr := parseDecisionFromText(ctx, s)
					if parseErr != nil {
						return decision, parseErr
					}
					logger.Info(ctx, "Claude decision received",
						"symbol", symbol,
						"action", decision.Action,
						"confidence", decision.Confidence,
						"reason", decision.Reason,
						"latency_ms", latency.Milliseconds(),
					)
					return decision, nil
				}
			}
		}
		// 3) fields used by newer messages API
		if choices, ok := m["choices"]; ok {
			if arr, ok2 := choices.([]any); ok2 && len(arr) > 0 {
				if c0, ok3 := arr[0].(map[string]any); ok3 {
					// try message content in choice
					if msg, ex := c0["message"]; ex {
						if mm, ok4 := msg.(map[string]any); ok4 {
							if cont, ex2 := mm["content"]; ex2 {
								if s, ok5 := cont.(string); ok5 {
									logger.Debug(ctx, "Extracting Claude decision from choices/message", "symbol", symbol)
									decision, parseErr := parseDecisionFromText(ctx, s)
									if parseErr != nil {
										return decision, parseErr
									}
									logger.Info(ctx, "Claude decision received",
										"symbol", symbol,
										"action", decision.Action,
										"confidence", decision.Confidence,
										"reason", decision.Reason,
										"latency_ms", latency.Milliseconds(),
									)
									return decision, nil
								}
							}
						}
					}
					// fallback to text field
					if txt, ex := c0["text"]; ex {
						if s, ok5 := txt.(string); ok5 {
							logger.Debug(ctx, "Extracting Claude decision from choices/text", "symbol", symbol)
							decision, parseErr := parseDecisionFromText(ctx, s)
							if parseErr != nil {
								return decision, parseErr
							}
							logger.Info(ctx, "Claude decision received",
								"symbol", symbol,
								"action", decision.Action,
								"confidence", decision.Confidence,
								"reason", decision.Reason,
								"latency_ms", latency.Milliseconds(),
							)
							return decision, nil
						}
					}
				}
			}
		}
	}

	// final fallback: raw text
	logger.Warn(ctx, "Using fallback raw text parsing for Claude response", "symbol", symbol)
	decision, parseErr := parseDecisionFromText(ctx, string(respBytes))
	if parseErr != nil {
		logger.ErrorWithErr(ctx, "Failed to parse Claude response", parseErr, "symbol", symbol)
		return decision, parseErr
	}
	logger.Info(ctx, "Claude decision received (fallback)",
		"symbol", symbol,
		"action", decision.Action,
		"confidence", decision.Confidence,
		"reason", decision.Reason,
		"latency_ms", latency.Milliseconds(),
	)
	return decision, nil
}

// parseDecisionFromText tries to locate a JSON object in text and unmarshal into types.Decision
func parseDecisionFromText(ctx context.Context, text string) (types.Decision, error) {
	// Trim and try to find first { ... } JSON substring
	t := strings.TrimSpace(text)
	logger.Debug(ctx, "Parsing decision from text", "text_length", len(t), "text_preview", t[:min(100, len(t))])

	// If it already looks like JSON object, unmarshal directly
	if strings.HasPrefix(t, "{") {
		var d types.Decision
		if err := json.Unmarshal([]byte(t), &d); err == nil {
			normalizeDecision(&d)
			logger.Debug(ctx, "Successfully parsed decision from JSON", "action", d.Action, "confidence", d.Confidence)
			return d, nil
		}
		// try to find first {...} substring
	}
	// Search for first '{' and matching '}' (simple)
	start := strings.Index(t, "{")
	end := strings.LastIndex(t, "}")
	if start >= 0 && end > start {
		sub := t[start : end+1]
		var d types.Decision
		if err := json.Unmarshal([]byte(sub), &d); err == nil {
			normalizeDecision(&d)
			logger.Debug(ctx, "Successfully parsed decision from extracted JSON", "action", d.Action, "confidence", d.Confidence)
			return d, nil
		}
	}
	// If still not parsable, return HOLD
	logger.Warn(ctx, "Unable to parse decision from text, defaulting to HOLD", "text", t)
	return types.Decision{Action: "HOLD", Reason: "unable_to_parse_claude_output", Confidence: 0.0}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func normalizeDecision(d *types.Decision) {
	d.Action = strings.ToUpper(strings.TrimSpace(d.Action))
	if d.Action != "BUY" && d.Action != "SELL" && d.Action != "HOLD" {
		d.Action = "HOLD"
	}
	if d.Confidence < 0 || d.Confidence > 1 {
		d.Confidence = 0.0
	}
}
