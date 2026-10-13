package engine

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

type fakeModel struct {
	responses []*anthropic.Message
	requests  []anthropic.MessageNewParams
	next      int
}

func (m *fakeModel) Generate(
	ctx context.Context,
	params anthropic.MessageNewParams,
) (*anthropic.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	copied := params
	copied.Messages = append(
		[]anthropic.MessageParam(nil),
		params.Messages...,
	)
	m.requests = append(m.requests, copied)

	if m.next >= len(m.responses) {
		return nil, errors.New(
			"no fake response configured",
		)
	}

	response := m.responses[m.next]
	m.next++

	return response, nil
}

func mustMessage(
	t *testing.T,
	source string,
) *anthropic.Message {
	t.Helper()

	var message anthropic.Message
	if err := json.Unmarshal(
		[]byte(source),
		&message,
	); err != nil {
		t.Fatal(err)
	}

	return &message
}

func newTestEngine(
	t *testing.T,
	model Model,
) *Engine {
	t.Helper()

	value, err := New(
		model,
		Config{
			Model:       "test-model",
			MaxTokens:   100,
			MaxRounds:   3,
			TurnTimeout: time.Second,
		},
		map[string][]anthropic.ToolParam{
			"npc": NPCTools,
		},
		map[string]map[string]ToolHandler{
			"npc": NPCToolHandlers,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	return value
}

func TestRunTurnReturnsText(t *testing.T) {
	model := &fakeModel{
		responses: []*anthropic.Message{
			mustMessage(t, `{
				"id": "msg-1",
				"type": "message",
				"role": "assistant",
				"model": "test-model",
				"stop_reason": "end_turn",
				"content": [
					{
						"type": "text",
						"text": "欢迎来到铁匠铺。"
					}
				],
				"usage": {
					"input_tokens": 1,
					"output_tokens": 1
				}
			}`),
		},
	}

	value := newTestEngine(t, model)

	history := []anthropic.MessageParam{
		anthropic.NewUserMessage(
			anthropic.NewTextBlock("你是谁？"),
		),
		anthropic.NewAssistantMessage(
			anthropic.NewTextBlock("我是镇上的铁匠。"),
		),
	}

	result, err := value.RunTurn(
		context.Background(),
		TurnRequest{
			Scope: Scope{
				WorldID:   "world-1",
				AgentKind: "npc",
				AgentID:   "blacksmith",
				PlayerID:  "player-1",
				SessionID: "conversation-1",
				TurnID:    "turn-1",
			},
			SystemPrompt: "你是镇上的铁匠。",
			Messages:     history,
			Input:        "你好。",
			ToolSet:      "npc",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Text != "欢迎来到铁匠铺。" {
		t.Fatalf(
			"unexpected text: %q",
			result.Text,
		)
	}

	if result.StopReason != "end_turn" {
		t.Fatalf(
			"unexpected stop reason: %s",
			result.StopReason,
		)
	}

	if len(result.Messages) != 4 {
		t.Fatalf(
			"expected 4 messages, got %d",
			len(result.Messages),
		)
	}

	if len(history) != 2 {
		t.Fatal("input history was modified")
	}

	if len(model.requests) != 1 {
		t.Fatalf(
			"expected 1 request, got %d",
			len(model.requests),
		)
	}
}

func TestRunTurnExecutesToolThenReturnsText(
	t *testing.T,
) {
	model := &fakeModel{
		responses: []*anthropic.Message{
			mustMessage(t, `{
				"id": "msg-1",
				"type": "message",
				"role": "assistant",
				"model": "test-model",
				"stop_reason": "tool_use",
				"content": [
					{
						"type": "tool_use",
						"id": "tool-1",
						"name": "get_agent_id",
						"input": {}
					}
				],
				"usage": {
					"input_tokens": 1,
					"output_tokens": 1
				}
			}`),
			mustMessage(t, `{
				"id": "msg-2",
				"type": "message",
				"role": "assistant",
				"model": "test-model",
				"stop_reason": "end_turn",
				"content": [
					{
						"type": "text",
						"text": "我是 blacksmith。"
					}
				],
				"usage": {
					"input_tokens": 1,
					"output_tokens": 1
				}
			}`),
		},
	}

	value := newTestEngine(t, model)

	result, err := value.RunTurn(
		context.Background(),
		TurnRequest{
			Scope: Scope{
				WorldID:   "world-1",
				AgentKind: "npc",
				AgentID:   "blacksmith",
				PlayerID:  "player-1",
				SessionID: "conversation-1",
				TurnID:    "turn-1",
			},
			SystemPrompt: "你是镇上的铁匠。",
			Input:        "你是谁？",
			ToolSet:      "npc",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Text != "我是 blacksmith。" {
		t.Fatalf(
			"unexpected text: %q",
			result.Text,
		)
	}

	if len(model.requests) != 2 {
		t.Fatalf(
			"expected 2 requests, got %d",
			len(model.requests),
		)
	}

	secondRequest, err := json.Marshal(
		model.requests[1].Messages,
	)
	if err != nil {
		t.Fatal(err)
	}

	payload := string(secondRequest)

	if !strings.Contains(payload, `"tool_result"`) {
		t.Fatal(
			"second request does not contain tool_result",
		)
	}

	if !strings.Contains(payload, "blacksmith") {
		t.Fatal(
			"second request does not contain tool output",
		)
	}
}
