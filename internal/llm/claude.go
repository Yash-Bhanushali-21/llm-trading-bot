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

func (d *ClaudeDecider) Decide(symbol string, latest types.Candle, inds types.Indicators, ctxmap map[string]any) (types.Decision, error) {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		return types.Decision{}, errors.New("CLAUDE_API_KEY missing")
	}

	// Build the state object the model will see
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

	bb, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", d.endpoint, bytes.NewReader(bb))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return types.Decision{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return types.Decision{}, fmt.Errorf("claude http %d: %s", resp.StatusCode, string(body))
	}

	// Read body and try to extract assistant content robustly
	respBytes, _ := io.ReadAll(resp.Body)
	// Try to parse JSON and drill common fields
	var anyResp any
	if err := json.Unmarshal(respBytes, &anyResp); err != nil {
		// Not JSON? treat full body as the text response
		return parseDecisionFromText(string(respBytes))
	}

	// Try common Claude messages structures: { "completion": "..."} or { "messages":[{ "role":"assistant","content":"..."}] } etc
	if m, ok := anyResp.(map[string]any); ok {
		// 1) messages array
		if msgs, found := m["messages"]; found {
			if arr, ok2 := msgs.([]any); ok2 && len(arr) > 0 {
				if first, ok3 := arr[0].(map[string]any); ok3 {
					if cont, ok4 := first["content"].(string); ok4 && strings.TrimSpace(cont) != "" {
						return parseDecisionFromText(cont)
					}
				}
			}
		}
		// 2) completion / output_text / completion_text
		for _, k := range []string{"completion", "output", "output_text", "completion_text", "result"} {
			if v, exists := m[k]; exists {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return parseDecisionFromText(s)
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
									return parseDecisionFromText(s)
								}
							}
						}
					}
					// fallback to text field
					if txt, ex := c0["text"]; ex {
						if s, ok5 := txt.(string); ok5 {
							return parseDecisionFromText(s)
						}
					}
				}
			}
		}
	}

	// final fallback: raw text
	return parseDecisionFromText(string(respBytes))
}

// parseDecisionFromText tries to locate a JSON object in text and unmarshal into types.Decision
func parseDecisionFromText(text string) (types.Decision, error) {
	// Trim and try to find first { ... } JSON substring
	t := strings.TrimSpace(text)
	// If it already looks like JSON object, unmarshal directly
	if strings.HasPrefix(t, "{") {
		var d types.Decision
		if err := json.Unmarshal([]byte(t), &d); err == nil {
			normalizeDecision(&d)
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
			return d, nil
		}
	}
	// If still not parsable, return HOLD
	return types.Decision{Action: "HOLD", Reason: "unable_to_parse_claude_output", Confidence: 0.0}, nil
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
