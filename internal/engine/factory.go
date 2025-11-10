package engine

import (
	"llm-trading-bot/internal/interfaces"
	"llm-trading-bot/internal/store"
	"llm-trading-bot/internal/types"
)

func New(cfg *store.Config, brk interfaces.Broker, d types.Decider) interfaces.Engine {
	return newEngine(cfg, brk, d)
}
