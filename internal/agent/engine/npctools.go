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
	{
		Name: "get_my_quest_status",
		Description: anthropic.String(
			"查询当前角色发布的委托进度，" +
				"包括任务状态、证据是否满足、凭证数量和奖励规则。" +
				"目前仅委托发布者商人可使用。此工具只查询，不执行结算。",
		),
		InputSchema: anthropic.ToolInputSchemaParam{
			Type:       "object",
			Properties: map[string]interface{}{},
			Required:   []string{},
		},
	},
}

var NPCToolsDefinitions = map[string][]anthropic.ToolUnionParam{
	"npc": BuildTools(NPCTools),
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
