package interfaces

import (
	"context"

	"llm-trading-bot/internal/types"
)

type Engine interface {
	Step(ctx context.Context, symbol string) (*types.StepResult, error)
}
