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

func BuildTools(tools []anthropic.ToolParam) []anthropic.ToolUnionParam {

	toolUnions := make([]anthropic.ToolUnionParam, 0, len(tools))

	for i, _ := range tools {
		toolParam := tools[i]
		toolUnions = append(toolUnions,
			anthropic.ToolUnionParam{
				OfTool: &toolParam,
			},
		)
	}

	return toolUnions
}
