package engine

import (
	"context"
	"fmt"
)

func (e *Engine) RunTurn(ctx context.Context, req TurnRequest) (TurnResult, error) {

	if req.SystemPrompt == "" {
		return TurnResult{}, fmt.Errorf("system prompt is required")
	}

	round := 0

	for {
		round++
		if round > e.config.MaxRounds {
			return TurnResult{}, nil
		}

	}

	return TurnResult{}, nil
}
