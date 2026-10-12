一个共享模型调用 Engine；
NPCID + PlayerID 隔离的会话；
根据 NPC 数据生成角色提示词；
查询世界状态和关系的只读工具；
一个受游戏逻辑校验的行为工具；
返回文本而不是直接打印到终端。




具体做法是先新增一个独立的 `engine` 包，暂时不改现有 [loop.go](D:/GoProjects/cyber_towm/internal/agent/loop/loop.go)。这样风险最小：新 Engine 测试通过后，再把 CLI 切过去。

`agent-builder` 的原则在这里体现为一个很小的循环：

```text
模型生成
→ 有工具调用就执行工具并继续
→ 没有工具调用就返回文本
```

### 1. 先记录当前基线

在项目根目录执行：

```powershell
go test ./...
go vet ./...
```

记录已有失败。完成 Engine 后还要再执行一次，确认没有引入新问题。

### 2. 创建目录

```text
internal/agent/engine/
├── turn.go
├── model.go
├── tool.go
├── engine.go
└── engine_test.go
```

### 3. 定义调用输入和输出

创建 `internal/agent/engine/turn.go`：

```go
package engine

import "github.com/anthropics/anthropic-sdk-go"

type Scope struct {
	WorldID   string
	AgentKind string
	AgentID   string
	PlayerID  string
	SessionID string
	TurnID    string
}

type TurnRequest struct {
	Scope        Scope
	SystemPrompt string
	Messages     []anthropic.MessageParam
	Input        string
	ToolSet      string
}

type TurnResult struct {
	Text       string
	Messages   []anthropic.MessageParam
	StopReason string
}
```

`Scope` 只允许调用方设置。之后工具参数里即使出现 `player_id`，也不能覆盖这里的 `PlayerID`。

### 4. 抽象模型调用

创建 `internal/agent/engine/model.go`：

```go
package engine

import (
	"context"
	"errors"

	"github.com/anthropics/anthropic-sdk-go"
)

type Model interface {
	Generate(
		ctx context.Context,
		params anthropic.MessageNewParams,
	) (*anthropic.Message, error)
}

type AnthropicModel struct {
	Client *anthropic.Client
}

func (m AnthropicModel) Generate(
	ctx context.Context,
	params anthropic.MessageNewParams,
) (*anthropic.Message, error) {
	if m.Client == nil {
		return nil, errors.New("anthropic client is nil")
	}

	return m.Client.Messages.New(ctx, params)
}
```

后面真实环境可以这样创建：

```go
model := engine.AnthropicModel{
	Client: &config.Client,
}
```

测试使用假模型，不需要 API Key。

### 5. 定义工具和权限集合

创建 `internal/agent/engine/tool.go`：

```go
package engine

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
)

type ToolHandler func(
	ctx context.Context,
	scope Scope,
	input map[string]any,
) (string, error)

type Tool struct {
	Definition anthropic.ToolUnionParam
	Execute    ToolHandler
}

type ToolSet map[string]Tool
```

第一版测试可以注册一个非常简单的工具：

```go
get_agent_name := Tool{
	Definition: anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Type:       "object",
			Properties: map[string]any{},
			Required:   []string{},
		},
		"get_agent_name",
	),
	Execute: func(
		ctx context.Context,
		scope Scope,
		input map[string]any,
	) (string, error) {
		return scope.AgentID, nil
	},
}
```

注册表结构为：

```go
map[string]ToolSet{
	"npc": {
		"get_agent_name": getAgentName,
	},
	"director": {},
	"cli":      {},
}
```

这里的第一层名称由服务端选择。将来玩家发送 HTTP 请求时，不能在请求体里指定 `"tool_set": "cli"`。

### 6. 实现 Engine 构造函数

在 `internal/agent/engine/engine.go` 中先定义配置：

```go
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

type Config struct {
	Model       string
	MaxTokens   int64
	MaxRounds   int
	TurnTimeout time.Duration
}

type Engine struct {
	model    Model
	config   Config
	toolSets map[string]ToolSet
}

func New(
	model Model,
	config Config,
	toolSets map[string]ToolSet,
) (*Engine, error) {
	if model == nil {
		return nil, errors.New("model is required")
	}
	if config.Model == "" {
		return nil, errors.New("model name is required")
	}
	if config.MaxTokens <= 0 {
		return nil, errors.New("max tokens must be positive")
	}
	if config.MaxRounds <= 0 {
		return nil, errors.New("max rounds must be positive")
	}

	sets := make(map[string]ToolSet, len(toolSets))
	for setName, sourceSet := range toolSets {
		targetSet := make(ToolSet, len(sourceSet))
		for toolName, tool := range sourceSet {
			targetSet[toolName] = tool
		}
		sets[setName] = targetSet
	}

	return &Engine{
		model:    model,
		config:   config,
		toolSets: sets,
	}, nil
}
```

构造时复制工具 map，避免调用方运行过程中修改注册表，造成并发问题。

### 7. 实现工具定义转换

继续在 `engine.go` 中添加：

```go
func buildToolDefinitions(toolSet ToolSet) []anthropic.ToolUnionParam {
	names := make([]string, 0, len(toolSet))
	for name := range toolSet {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]anthropic.ToolUnionParam, 0, len(names))
	for _, name := range names {
		result = append(result, toolSet[name].Definition)
	}

	return result
}
```

排序不是业务要求，但会让测试请求稳定，方便排查问题。

### 8. 实现核心循环

在 `engine.go` 中实现 `RunTurn`：

```go
func (e *Engine) RunTurn(
	ctx context.Context,
	req TurnRequest,
) (TurnResult, error) {
	if strings.TrimSpace(req.SystemPrompt) == "" {
		return TurnResult{}, errors.New("system prompt is required")
	}

	toolSet, ok := e.toolSets[req.ToolSet]
	if !ok {
		return TurnResult{}, fmt.Errorf(
			"unknown tool set: %s",
			req.ToolSet,
		)
	}

	runCtx := ctx
	cancel := func() {}

	if e.config.TurnTimeout > 0 {
		runCtx, cancel = context.WithTimeout(
			ctx,
			e.config.TurnTimeout,
		)
	}
	defer cancel()

	// 创建新的切片，不能直接向 req.Messages append。
	messages := make(
		[]anthropic.MessageParam,
		0,
		len(req.Messages)+1,
	)
	messages = append(messages, req.Messages...)

	if req.Input != "" {
		messages = append(
			messages,
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(req.Input),
			),
		)
	}

	tools := buildToolDefinitions(toolSet)

	for round := 1; round <= e.config.MaxRounds; round++ {
		if err := runCtx.Err(); err != nil {
			return TurnResult{
				Messages: messages,
			}, err
		}

		response, err := e.model.Generate(
			runCtx,
			anthropic.MessageNewParams{
				Model:     e.config.Model,
				MaxTokens: e.config.MaxTokens,
				System: []anthropic.TextBlockParam{
					{Text: req.SystemPrompt},
				},
				Messages: messages,
				Tools:    tools,
			},
		)
		if err != nil {
			return TurnResult{
				Messages: messages,
			}, fmt.Errorf("generate model response: %w", err)
		}
		if response == nil {
			return TurnResult{
				Messages: messages,
			}, errors.New("model returned nil response")
		}

		messages = append(messages, response.ToParam())

		if response.StopReason != anthropic.StopReasonToolUse {
			return TurnResult{
				Text:       collectText(response),
				Messages:   messages,
				StopReason: string(response.StopReason),
			}, nil
		}

		results, err := e.executeTools(
			runCtx,
			req.Scope,
			toolSet,
			response,
		)
		if err != nil {
			return TurnResult{
				Messages: messages,
			}, err
		}

		messages = append(
			messages,
			anthropic.NewUserMessage(results...),
		)
	}

	return TurnResult{
		Messages: messages,
	}, fmt.Errorf(
		"maximum agent rounds exceeded: %d",
		e.config.MaxRounds,
	)
}
```

这里传入的 `req.Messages` 不会被追加或覆盖。Engine 只操作自己创建的切片。

### 9. 实现工具执行

继续在 `engine.go` 添加：

```go
func (e *Engine) executeTools(
	ctx context.Context,
	scope Scope,
	toolSet ToolSet,
	response *anthropic.Message,
) ([]anthropic.ContentBlockParamUnion, error) {
	results := make([]anthropic.ContentBlockParamUnion, 0)
	foundToolUse := false

	for _, content := range response.Content {
		block, ok := content.AsAny().(anthropic.ToolUseBlock)
		if !ok {
			continue
		}

		foundToolUse = true

		var input map[string]any
		if err := json.Unmarshal(block.Input, &input); err != nil {
			results = append(
				results,
				anthropic.NewToolResultBlock(
					block.ID,
					fmt.Sprintf("invalid tool input: %v", err),
					true,
				),
			)
			continue
		}

		registeredTool, ok := toolSet[block.Name]
		if !ok {
			results = append(
				results,
				anthropic.NewToolResultBlock(
					block.ID,
					fmt.Sprintf(
						"tool is not authorized: %s",
						block.Name,
					),
					true,
				),
			)
			continue
		}

		if registeredTool.Execute == nil {
			results = append(
				results,
				anthropic.NewToolResultBlock(
					block.ID,
					"tool handler is not configured",
					true,
				),
			)
			continue
		}

		output, err := registeredTool.Execute(
			ctx,
			scope,
			input,
		)
		if err != nil {
			results = append(
				results,
				anthropic.NewToolResultBlock(
					block.ID,
					err.Error(),
					true,
				),
			)
			continue
		}

		results = append(
			results,
			anthropic.NewToolResultBlock(
				block.ID,
				output,
				false,
			),
		)
	}

	if !foundToolUse {
		return nil, errors.New(
			"model stopped for tool use without a tool_use block",
		)
	}

	return results, nil
}
```

注意错误分两类：

- 模型请求了无权使用的工具：返回 `tool_result`，让模型自行修正。
- 模型协议不合法，例如 `stop_reason=tool_use` 却没有工具块：终止本轮并返回 Go error。

### 10. 收集最终文本

继续添加：

```go
func collectText(response *anthropic.Message) string {
	parts := make([]string, 0)

	for _, content := range response.Content {
		block, ok := content.AsAny().(anthropic.TextBlock)
		if !ok {
			continue
		}

		text := strings.TrimSpace(block.Text)
		if text != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, "\n")
}
```

当前只收集最终一次模型响应中的文本。工具调用前的中间说明不作为最终回答返回。

### 11. 创建假模型

在 `engine_test.go` 中定义：

```go
type fakeModel struct {
	responses []*anthropic.Message
	requests  []anthropic.MessageNewParams
	next      int
}

func (f *fakeModel) Generate(
	ctx context.Context,
	params anthropic.MessageNewParams,
) (*anthropic.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	f.requests = append(f.requests, params)

	if f.next >= len(f.responses) {
		return nil, errors.New("no fake response configured")
	}

	response := f.responses[f.next]
	f.next++

	return response, nil
}
```

为了方便构造 SDK 响应，可以用 JSON：

```go
func mustMessage(t *testing.T, source string) *anthropic.Message {
	t.Helper()

	var result anthropic.Message
	if err := json.Unmarshal([]byte(source), &result); err != nil {
		t.Fatal(err)
	}

	return &result
}
```

### 12. 测试直接回复

```go
func TestRunTurnReturnsText(t *testing.T) {
	model := &fakeModel{
		responses: []*anthropic.Message{
			mustMessage(t, `{
				"id": "msg_1",
				"type": "message",
				"role": "assistant",
				"model": "test-model",
				"stop_reason": "end_turn",
				"content": [
					{"type": "text", "text": "欢迎来到铁匠铺。"}
				],
				"usage": {
					"input_tokens": 1,
					"output_tokens": 1
				}
			}`),
		},
	}

	value, err := New(
		model,
		Config{
			Model:     "test-model",
			MaxTokens: 100,
			MaxRounds: 3,
		},
		map[string]ToolSet{
			"npc": {},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	original := []anthropic.MessageParam{}

	result, err := value.RunTurn(
		context.Background(),
		TurnRequest{
			Scope: Scope{
				WorldID:   "world-1",
				AgentKind: "npc",
				AgentID:   "blacksmith",
				PlayerID:  "player-1",
				SessionID: "session-1",
				TurnID:    "turn-1",
			},
			SystemPrompt: "你是城镇铁匠。",
			Messages:     original,
			Input:        "你好",
			ToolSet:      "npc",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Text != "欢迎来到铁匠铺。" {
		t.Fatalf("unexpected text: %q", result.Text)
	}
	if len(result.Messages) != 2 {
		t.Fatalf(
			"expected 2 messages, got %d",
			len(result.Messages),
		)
	}
	if len(original) != 0 {
		t.Fatal("input messages were modified")
	}
	if len(model.requests) != 1 {
		t.Fatalf(
			"expected 1 model request, got %d",
			len(model.requests),
		)
	}
}
```

### 13. 测试工具调用后回复

假模型准备两次响应：

```go
toolCall := mustMessage(t, `{
	"id": "msg_1",
	"type": "message",
	"role": "assistant",
	"model": "test-model",
	"stop_reason": "tool_use",
	"content": [
		{
			"type": "tool_use",
			"id": "tool_1",
			"name": "get_agent_name",
			"input": {}
		}
	],
	"usage": {
		"input_tokens": 1,
		"output_tokens": 1
	}
}`)

finalReply := mustMessage(t, `{
	"id": "msg_2",
	"type": "message",
	"role": "assistant",
	"model": "test-model",
	"stop_reason": "end_turn",
	"content": [
		{"type": "text", "text": "我是铁匠布兰。"}
	],
	"usage": {
		"input_tokens": 1,
		"output_tokens": 1
	}
}`)
```

注册测试工具，并记录它收到的 Scope：

```go
var receivedScope Scope
callCount := 0

getAgentName := Tool{
	Definition: anthropic.ToolUnionParamOfTool(
		anthropic.ToolInputSchemaParam{
			Type:       "object",
			Properties: map[string]any{},
			Required:   []string{},
		},
		"get_agent_name",
	),
	Execute: func(
		ctx context.Context,
		scope Scope,
		input map[string]any,
	) (string, error) {
		callCount++
		receivedScope = scope
		return "铁匠布兰", nil
	},
}
```

运行后断言：

```go
if result.Text != "我是铁匠布兰。" {
	t.Fatalf("unexpected text: %q", result.Text)
}
if callCount != 1 {
	t.Fatalf("expected tool call once, got %d", callCount)
}
if receivedScope.AgentID != "blacksmith" {
	t.Fatalf(
		"unexpected agent ID: %s",
		receivedScope.AgentID,
	)
}
if receivedScope.PlayerID != "player-1" {
	t.Fatalf(
		"unexpected player ID: %s",
		receivedScope.PlayerID,
	)
}
if len(model.requests) != 2 {
	t.Fatalf(
		"expected 2 model requests, got %d",
		len(model.requests),
	)
}
```

还可以将第二次请求序列化，检查里面确实包含工具结果：

```go
encoded, err := json.Marshal(model.requests[1].Messages)
if err != nil {
	t.Fatal(err)
}

payload := string(encoded)
if !strings.Contains(payload, `"tool_result"`) {
	t.Fatal("second request does not contain tool result")
}
if !strings.Contains(payload, "铁匠布兰") {
	t.Fatal("second request does not contain tool output")
}
```

### 14. 运行验收

先只测试新包：

```powershell
go test ./internal/agent/engine -v
go test -race ./internal/agent/engine
```

再检查整个项目：

```powershell
go test ./...
go vet ./...
```

这一阶段不要修改 [repl.go](D:/GoProjects/cyber_towm/internal/agent/repl.go) 或删除原来的 `AgentLoop`。第一个小目标完成的标志是：

```text
直接文本测试通过
工具调用闭环测试通过
工具收到正确 Scope
原始消息没有被修改
测试不需要 API Key、Qdrant、MCP、Cron 或 readline
```

随后第二个小目标才是把 CLI 的一次 `handleQuery` 改为调用这个 Engine。