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

	"llm-trading-bot/internal/trace"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/types"
)

type OpenAIDecider struct {
	cfg *store.Config
}

func NewOpenAIDecider(cfg *store.Config) *OpenAIDecider {
	return &OpenAIDecider{cfg: cfg}
}

func (d *OpenAIDecider) Decide(ctx context.Context, symbol string, latest types.Candle, inds types.Indicators, ctxmap map[string]any) (types.Decision, error) {
	ctx, span := trace.StartSpan(ctx, "openai-api-call")
	defer span.End()

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return types.Decision{}, errors.New("OPENAI_API_KEY missing")
	}

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

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(bb))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return types.Decision{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return types.Decision{}, fmt.Errorf("openai http %d", resp.StatusCode)
	}

	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return types.Decision{}, err
	}

	if len(r.Choices) == 0 {
		return types.Decision{}, errors.New("no choices")
	}

	out := strings.TrimSpace(r.Choices[0].Message.Content)

	var dres types.Decision
	if err := json.Unmarshal([]byte(out), &dres); err != nil {
		return types.Decision{Action: "HOLD", Reason: "invalid_json", Confidence: 0.0}, nil
	}

	dres.Action = strings.ToUpper(strings.TrimSpace(dres.Action))
	valid := map[string]bool{"BUY": true, "SELL": true, "HOLD": true}
	if !valid[dres.Action] {
		dres.Action = "HOLD"
	}
	if dres.Confidence < 0 || dres.Confidence > 1 {
		dres.Confidence = 0.0
	}

	return dres, nil
}
