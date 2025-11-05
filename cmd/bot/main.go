package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"llm-trading-bot/internal/broker"
	"llm-trading-bot/internal/engine"
	"llm-trading-bot/internal/eod"
	"llm-trading-bot/internal/llm"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/tradelog"
	"llm-trading-bot/internal/types"

	"github.com/joho/godotenv"
)

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	_ = godotenv.Load()
	cfg, err := store.LoadConfig("config.yaml")
	must(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if v := os.Getenv("TRADER_LOG_RETENTION_DAYS"); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		_ = tradelog.CompressOlder(n)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	brk := broker.NewZerodha(cpf(cfg))
	if cfg.Mode == "DRY_RUN" {
		log.Println(">> DRY_RUN mode")
	}

	var decider types.Decider
	if cfg.LLM.Provider == "OPENAI" {
		decider = llm.NewOpenAIDecider(cfg)
	} else {
		decider = llm.NewNoopDecider()
	}

	eng := engine.New(cfg, brk, decider)

	tick := time.NewTicker(time.Duration(cfg.PollSeconds) * time.Second)
	defer tick.Stop()
	eodTick := time.NewTicker(60 * time.Second)
	defer eodTick.Stop()

	log.Println("Bot started.")
	for {
		select {
		case <-tick.C:
			for _, sym := range cfg.UniverseStatic {
				st, err := eng.Step(ctx, sym)
				if err != nil {
					log.Printf("[%s] step error: %v", sym, err)
					continue
				}
				if st != nil {
					b, _ := json.Marshal(st)
					fmt.Println(string(b))
				}
			}
		case <-eodTick.C:
			if ok, _ := eod.ShouldRunNow(); ok {
				if p, err := eod.SummarizeToday(); err == nil && p != "" {
					log.Println("EOD CSV written:", p)
				}
			}
		case <-sigc:
			log.Println("Shutting down...")
			if p, err := eod.SummarizeToday(); err == nil && p != "" {
				log.Println("EOD CSV written:", p)
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

func cpf(c *store.Config) broker.Params {
	return broker.Params{Mode: c.Mode, APIKey: os.Getenv("KITE_API_KEY"), AccessToken: os.Getenv("KITE_ACCESS_TOKEN"), Exchange: c.Exchange}
}
