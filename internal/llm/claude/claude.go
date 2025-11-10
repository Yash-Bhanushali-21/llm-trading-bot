package claude

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

	"llm-trading-bot/internal/trace"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/types"
)

type ClaudeDecider struct {
	cfg      *store.Config
	endpoint string
}

func NewClaudeDecider(cfg *store.Config) *ClaudeDecider {
	endpoint := "https://api.anthropic.com/v1/messages"
	if ep := os.Getenv("CLAUDE_API_ENDPOINT"); ep != "" {
		endpoint = ep
	}
	return &ClaudeDecider{cfg: cfg, endpoint: endpoint}
}

func (d *ClaudeDecider) Decide(ctx context.Context, symbol string, latest types.Candle, inds types.Indicators, ctxmap map[string]any) (types.Decision, error) {
	ctx, span := trace.StartSpan(ctx, "claude-api-call")
	defer span.End()

	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		return types.Decision{}, errors.New("CLAUDE_API_KEY missing")
	}

	state := map[string]any{
		"symbol":     symbol,
		"latest":     latest,
		"indicators": inds,
		"context":    ctxmap,
	}
	stateB, _ := json.Marshal(state)

	system := d.cfg.LLM.System
	if system == "" {
		system = "You are a disciplined equities trader. Output STRICT JSON with BUY/SELL/HOLD."
	}
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
	req, _ := http.NewRequestWithContext(ctx, "POST", d.endpoint, bytes.NewReader(bb))
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

	respBytes, _ := io.ReadAll(resp.Body)

	var anyResp any
	if err := json.Unmarshal(respBytes, &anyResp); err != nil {
		return parseDecisionFromText(string(respBytes))
	}

	if m, ok := anyResp.(map[string]any); ok {
		if msgs, found := m["messages"]; found {
			if arr, ok2 := msgs.([]any); ok2 && len(arr) > 0 {
				if first, ok3 := arr[0].(map[string]any); ok3 {
					if cont, ok4 := first["content"].(string); ok4 && strings.TrimSpace(cont) != "" {
						return parseDecisionFromText(cont)
					}
				}
			}
		}
		for _, k := range []string{"completion", "output", "output_text", "completion_text", "result"} {
			if v, exists := m[k]; exists {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return parseDecisionFromText(s)
				}
			}
		}
		if choices, ok := m["choices"]; ok {
			if arr, ok2 := choices.([]any); ok2 && len(arr) > 0 {
				if c0, ok3 := arr[0].(map[string]any); ok3 {
					if msg, ex := c0["message"]; ex {
						if mm, ok4 := msg.(map[string]any); ok4 {
							if cont, ex2 := mm["content"]; ex2 {
								if s, ok5 := cont.(string); ok5 {
									return parseDecisionFromText(s)
								}
							}
						}
					}
					if txt, ex := c0["text"]; ex {
						if s, ok5 := txt.(string); ok5 {
							return parseDecisionFromText(s)
						}
					}
				}
			}
		}
	}

	return parseDecisionFromText(string(respBytes))
}

func parseDecisionFromText(text string) (types.Decision, error) {
	t := strings.TrimSpace(text)

	if strings.HasPrefix(t, "{") {
		var d types.Decision
		if err := json.Unmarshal([]byte(t), &d); err == nil {
			normalizeDecision(&d)
			return d, nil
		}
	}

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
