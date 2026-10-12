package engine

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
)

var NPCToolHandlers = map[string]ToolHandler{
	"get_agent_id": func(ctx context.Context, scope Scope, input map[string]any) (string, error) {
		return scope.AgentID, nil
	},
}

var NPCTools = []anthropic.ToolParam{

	{

		Name:        "get_agent_id",
		Description: anthropic.String("Get the current NPC's ID."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Type:       "object",
			Properties: map[string]interface{}{},
			Required:   []string{},
		},
	},
}

func BuildNPCTools() []anthropic.ToolUnionParam {

	tools := make([]anthropic.ToolUnionParam, 0, len(NPCTools))

	for i, _ := range NPCTools {
		toolParam := NPCTools[i]
		tools = append(tools,
			anthropic.ToolUnionParam{
				OfTool: &toolParam,
			},
		)
	}

	return tools
}
