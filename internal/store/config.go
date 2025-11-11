package store

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Mode           string   `yaml:"mode"`
	DataSource     string   `yaml:"data_source"`
	PollSeconds    int      `yaml:"poll_seconds"`
	Exchange       string   `yaml:"exchange"`
	UniverseMode   string   `yaml:"universe_mode"`
	UniverseStatic []string `yaml:"universe_static"`
	Universe       struct {
		Static  []string `yaml:"static"`
		Dynamic struct {
			TopN            int      `yaml:"top_n"`
			RunPreopen      bool     `yaml:"run_preopen"`
			PreopenTime     string   `yaml:"preopen_time"`
			RefreshMidday   string   `yaml:"refresh_midday"`
			CandidateList   []string `yaml:"candidate_list"`
			Filters         struct {
				MinPrice       float64 `yaml:"min_price"`
				MaxPrice       float64 `yaml:"max_price"`
				MinTurnoverCr  float64 `yaml:"min_turnover_cr"`
				ATRPctMin      float64 `yaml:"atr_pct_min"`
				ATRPctMax      float64 `yaml:"atr_pct_max"`
				RSIMin         float64 `yaml:"rsi_min"`
				RSIMax         float64 `yaml:"rsi_max"`
				ExcludeT2T     bool    `yaml:"exclude_t2t"`
			} `yaml:"filters"`
			Scoring struct {
				WeightTrend      float64 `yaml:"weight_trend"`
				WeightRSI        float64 `yaml:"weight_rsi"`
				WeightTurnover   float64 `yaml:"weight_turnover"`
				WeightVolatility float64 `yaml:"weight_volatility"`
			} `yaml:"scoring"`
		} `yaml:"dynamic"`
	} `yaml:"universe"`
	Qty struct {
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
	PEAD struct {
		Enabled              bool    `yaml:"enabled"`
		MinDaysSinceEarnings int     `yaml:"min_days_since_earnings"`
		MaxDaysSinceEarnings int     `yaml:"max_days_since_earnings"`
		MinCompositeScore    float64 `yaml:"min_composite_score"`
		MinEarningsSurprise  float64 `yaml:"min_earnings_surprise"`
		MinRevenueGrowth     float64 `yaml:"min_revenue_growth"`
		MinEPSGrowth         float64 `yaml:"min_eps_growth"`
		Weights              struct {
			EarningsSurprise    float64 `yaml:"earnings_surprise"`
			RevenueSurprise     float64 `yaml:"revenue_surprise"`
			EarningsGrowth      float64 `yaml:"earnings_growth"`
			RevenueGrowth       float64 `yaml:"revenue_growth"`
			MarginExpansion     float64 `yaml:"margin_expansion"`
			Consistency         float64 `yaml:"consistency"`
			RevenueAcceleration float64 `yaml:"revenue_acceleration"`
		} `yaml:"weights"`
		DataSource string `yaml:"data_source"`
		APIKeyEnv  string `yaml:"api_key_env"`
	} `yaml:"pead"`
}

func (c *Config) Validate() error {
	if c.Mode != "DRY_RUN" && c.Mode != "LIVE" {
		return fmt.Errorf("invalid mode '%s': must be 'DRY_RUN' or 'LIVE'", c.Mode)
	}
	if c.DataSource != "STATIC" && c.DataSource != "LIVE" {
		return fmt.Errorf("invalid data_source '%s': must be 'STATIC' or 'LIVE'", c.DataSource)
	}
	// Support both old and new universe config
	if len(c.UniverseStatic) == 0 && len(c.Universe.Static) == 0 {
		return errors.New("universe_static cannot be empty")
	}
	if c.Risk.PerTradeRiskPct <= 0 || c.Risk.PerTradeRiskPct > 100 {
		return fmt.Errorf("risk.per_trade_risk_pct must be between 0-100, got %.2f", c.Risk.PerTradeRiskPct)
	}
	if c.Stop.Mode != "FIXED" && c.Stop.Mode != "ATR" && c.Stop.Mode != "PCT" {
		return fmt.Errorf("stop.mode must be 'FIXED', 'ATR', or 'PCT', got '%s'", c.Stop.Mode)
	}
	return nil
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
	if c.DataSource == "" {
		c.DataSource = "STATIC"
	}

	// Backward compatibility: copy UniverseStatic to Universe.Static if present
	if len(c.UniverseStatic) > 0 && len(c.Universe.Static) == 0 {
		c.Universe.Static = c.UniverseStatic
	}

	// Set PEAD defaults if not configured
	if c.PEAD.MaxDaysSinceEarnings == 0 {
		c.PEAD.MaxDaysSinceEarnings = 60
	}
	if c.PEAD.MinCompositeScore == 0 {
		c.PEAD.MinCompositeScore = 40
	}

	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &c, nil
}
