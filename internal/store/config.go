package store

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Mode           string   `yaml:"mode"`
	PollSeconds    int      `yaml:"poll_seconds"`
	Exchange       string   `yaml:"exchange"`
	UniverseStatic []string `yaml:"universe_static"`
	Qty            struct {
		DefaultBuy  int            `yaml:"default_buy"`
		DefaultSell int            `yaml:"default_sell"`
		PerSymbol   map[string]int `yaml:"per_symbol"`
	} `yaml:"qty"`
	Risk struct {
		MaxDailyDrawdownPct float64 `yaml:"max_daily_drawdown_pct"`
		PerTradeRiskPct     float64 `yaml:"per_trade_risk_pct"`
	} `yaml:"risk"`
	Stop struct {
		Mode     string  `yaml:"mode"`
		Pct      float64 `yaml:"pct"`
		ATRMult  float64 `yaml:"atr_mult"`
		Trailing bool    `yaml:"trailing"`
		MinTick  float64 `yaml:"min_tick"`
	} `yaml:"stop"`
	Indicators struct {
		SMAWindows []int   `yaml:"sma_windows"`
		RSIPeriod  int     `yaml:"rsi_period"`
		BBWindow   int     `yaml:"bb_window"`
		BBStdDev   float64 `yaml:"bb_stddev"`
		ATRPeriod  int     `yaml:"atr_period"`
	} `yaml:"indicators"`
	LLM struct {
		Provider    string  `yaml:"provider"`
		Model       string  `yaml:"model"`
		MaxTokens   int     `yaml:"max_tokens"`
		Temperature float32 `yaml:"temperature"`
		System      string  `yaml:"system"`
		Schema      string  `yaml:"schema"`
	} `yaml:"llm"`
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.PollSeconds == 0 {
		c.PollSeconds = 15
	}
	return &c, nil
}
