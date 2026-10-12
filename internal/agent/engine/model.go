package engine

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

type Model interface {
	Generate(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error)
}

type LLM struct {
	Client *anthropic.Client
}

func (m *LLM) Generate(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error) {

	if m.Client == nil {
		return nil, fmt.Errorf("anthropic client is nil")
	}

	return m.Client.Messages.New(ctx, params)
}
