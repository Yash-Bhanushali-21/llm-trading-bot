package engine

import (
	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/store"
)

func New(cfg *store.Config, brk interfaces.Broker, d interfaces.Decider) interfaces.Engine {
	return newEngine(cfg, brk, d)
}
