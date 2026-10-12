package engine

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
)

type ToolHandler func(ctx context.Context, scope Scope, input map[string]any) (string, error)

var ToolDefinitions = map[string][]anthropic.ToolParam{
	"npc": NPCTools,
}

var ToolHandlers = map[string]map[string]ToolHandler{
	"npc": NPCToolHandlers,
}
