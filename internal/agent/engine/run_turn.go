package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

func (e *Engine) RunTurn(ctx context.Context, req TurnRequest) (TurnResult, error) {

	runCtx := ctx

	cancel := func() {}

	if e.config.TurnTimeout > 0 {
		runCtx, cancel = context.WithTimeout(
			ctx,
			e.config.TurnTimeout,
		)
	}
	defer cancel()

	if req.SystemPrompt == "" {
		return TurnResult{}, fmt.Errorf("system prompt is required")
	}
	messages := make([]anthropic.MessageParam, 0)
	messages = append(messages, req.Messages...)

	if req.Input != "" {
		messages = append(messages, anthropic.NewUserMessage(
			anthropic.NewTextBlock(req.Input),
		))
	}

	round := 0

	for {
		round++
		if round > e.config.MaxRounds {
			return TurnResult{
				Messages: messages,
			}, fmt.Errorf("maximum agent rounds exceeded: %v", e.config.MaxRounds)
		}

		toolParams, ok := e.toolDefinitions[req.ToolSet]
		if !ok {
			return TurnResult{}, fmt.Errorf("no tools in set %v", req.ToolSet)
		}
		tools := toolParams

		response, err := e.model.Generate(runCtx, anthropic.MessageNewParams{
			MaxTokens: e.config.MaxTokens,
			Model:     e.config.Model,
			System: []anthropic.TextBlockParam{
				{
					Text: req.SystemPrompt,
				},
			},
			Messages: messages,
			Tools:    tools,
		})

		if err != nil {
			return TurnResult{
				Messages: messages,
			}, fmt.Errorf("llm response: %v", err)
		}

		if response == nil {
			return TurnResult{
				Messages: messages,
			}, fmt.Errorf("llm return nil response")
		}

		messages = append(messages,
			anthropic.NewAssistantMessage(response.ToParam().Content...),
		)

		if response.StopReason != anthropic.StopReasonToolUse {
			parts := make([]string, 0)
			for _, block := range response.Content {
				text, ok := block.AsAny().(anthropic.TextBlock)
				if ok {
					if text.Text != "" {
						parts = append(parts, strings.TrimSpace(text.Text))
					}

				}
			}
			var lines string
			lines = strings.Join(parts, "\n")

			if response.StopReason == anthropic.StopReasonMaxTokens {
				return TurnResult{
					Text:       lines,
					Messages:   messages,
					StopReason: string(response.StopReason),
				}, fmt.Errorf("LLM output reached max tokens")
			}

			return TurnResult{
				Messages:   messages,
				StopReason: string(response.StopReason),
				Text:       lines,
			}, nil
		}

		toolResults := []anthropic.ContentBlockParamUnion{}
		for _, block := range response.Content {
			switch block := block.AsAny().(type) {

			case anthropic.ToolUseBlock:

				var input map[string]any
				err := json.Unmarshal(block.Input, &input)
				if err != nil {
					toolResults = append(toolResults,
						anthropic.NewToolResultBlock(
							block.ID,
							fmt.Sprintf("invalid tool input: %v", err),
							true,
						),
					)
					continue
				}

				handers, ok := e.toolHandlers[req.ToolSet]
				if !ok {
					return TurnResult{
						Messages: messages,
					}, fmt.Errorf("tool handler not found: %v", req.ToolSet)
				}

				hander, ok := handers[block.Name]
				if !ok {
					toolResults = append(toolResults,
						anthropic.NewToolResultBlock(
							block.ID,
							fmt.Sprintf("tool is not authorized: %v", block.Name),
							true,
						),
					)
					continue
				}

				output, err := hander(runCtx, req.Scope, input)
				if err != nil {
					toolResults = append(toolResults,
						anthropic.NewToolResultBlock(
							block.ID,
							fmt.Sprintf("tool execution failed: %v", err),
							true,
						),
					)
					continue
				}
				toolResults = append(toolResults,
					anthropic.NewToolResultBlock(
						block.ID,
						output,
						false,
					),
				)

			}

		}

		if len(toolResults) == 0 {
			return TurnResult{
					Messages:   messages,
					StopReason: string(response.StopReason),
				}, fmt.Errorf(
					"tool_use stop reason without tool call",
				)
		}

		toolResultMessages := anthropic.NewUserMessage(toolResults...)
		messages = append(messages, toolResultMessages)
	}

}
