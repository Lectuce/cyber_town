package agent

import (
	"context"
	"cybertown/internal/agent/config"
	"cybertown/internal/agent/engine"
	"cybertown/internal/agent/mcpclient"
	"cybertown/internal/agent/subagent"
	"cybertown/internal/agent/tool"

	"github.com/anthropics/anthropic-sdk-go"
)

func newEngine(mcpManager *mcpclient.Manager) (*engine.Engine, error) {

	tools := tool.BuildTools()

	mcpTools, err := mcpManager.BuildTools()
	if err != nil {
		return nil, err
	}

	tools = append(tools, mcpTools...)

	handlers := make(map[string]engine.ToolHandler, 0)

	for name, handler := range tool.ToolHandlers {

		handlers[name] = func(ctx context.Context, scope engine.Scope, input map[string]any) (string, error) {
			err := ctx.Err()
			if err != nil {
				return "", err
			}
			return handler(input)
		}

	}

	handlers["task"] = func(ctx context.Context, scope engine.Scope, input map[string]any) (string, error) {
		return subagent.SpawnSubagent(input, ctx)
	}

	for _, toolName := range mcpManager.ToolNames() {
		name := toolName

		handlers[name] = func(ctx context.Context, scope engine.Scope, input map[string]any) (string, error) {
			return mcpManager.Execute(
				ctx,
				name,
				input,
			)
		}
	}

	allTools := map[string][]anthropic.ToolUnionParam{
		"cli": tools,
		"npc": engine.BuildTools(
			engine.NPCTools,
		),
	}

	allHandlers := map[string]map[string]engine.ToolHandler{
		"cli": handlers,
		"npc": engine.NPCToolHandlers,
	}

	return engine.New(
		&engine.LLM{
			Client: &config.Client,
		},
		engine.Config{
			Model:       config.PRIMARY_MODEL,
			MaxTokens:   config.DEFAULT_MAX_TOOKENS,
			MaxRounds:   config.MAX_AGENT_ROUNDS,
			TurnTimeout: config.IDLE_TIMEOUT,
		},
		allTools,
		allHandlers,
	)

}
