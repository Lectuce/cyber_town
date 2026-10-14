既然现在考虑的是正式开发，就不要增加 `/engine` 命令，也不要增加空的 `"cli"` 工具集。

正式接入应当是：

```text
现有 REPL 输入
→ handleQuery
→ 新 Engine
→ 真实 CLI 工具
→ 更新原 Session
→ 输出结果
```

旧 `AgentLoop` 暂时保留给 Cron 队列，普通 CLI 请求直接切换到新 Engine。

## 1. 让 Engine 接收 `ToolUnionParam`

当前 Engine 使用：

```go
map[string][]anthropic.ToolParam
```

这无法装入现有 CLI 的服务端工具和 MCP 工具。修改 [engine.go](D:/GoProjects/cyber_towm/internal/agent/engine/engine.go)：

```go
type Engine struct {
	model           Model
	config          Config
	toolDefinitions map[string][]anthropic.ToolUnionParam
	toolHandlers    map[string]map[string]ToolHandler
}
```

`New()` 参数也改成：

```go
func New(
	model Model,
	config Config,
	toolDefinitions map[string][]anthropic.ToolUnionParam,
	toolHandlers map[string]map[string]ToolHandler,
) (*Engine, error) {
```

`validateTools()` 同样改成：

```go
func validateTools(
	definitions map[string][]anthropic.ToolUnionParam,
	handlers map[string]map[string]ToolHandler,
) error {
```

遍历定义时，只校验需要本地 Handler 的客户端工具：

```go
for _, definition := range toolDefinitions {
	if definition.OfTool == nil {
		// web_search 等 Anthropic 服务端工具
		// 不需要本地 Handler。
		continue
	}

	name := string(definition.OfTool.Name)

	if name == "" {
		return fmt.Errorf(
			"tool name is empty in set: %s",
			setName,
		)
	}

	if _, exists := names[name]; exists {
		return fmt.Errorf(
			"duplicate tool %q in set %q",
			name,
			setName,
		)
	}

	names[name] = struct{}{}

	handler, ok := setHandlers[name]
	if !ok {
		return fmt.Errorf(
			"handler not found for tool %q in set %q",
			name,
			setName,
		)
	}

	if handler == nil {
		return fmt.Errorf(
			"handler is nil for tool %q in set %q",
			name,
			setName,
		)
	}
}
```

在 `RunTurn()` 中不再转换：

```go
tools := BuildTools(toolParams)
```

改成：

```go
tools := toolParams
```

## 2. 调整 NPC 工具注册

在 Engine 的 [tool.go](D:/GoProjects/cyber_towm/internal/agent/engine/tool.go) 中：

```go
var ToolDefinitions = map[string][]anthropic.ToolUnionParam{
	"npc": BuildTools(NPCTools),
}

var ToolHandlers = map[string]map[string]ToolHandler{
	"npc": NPCToolHandlers,
}
```

`BuildTools()` 可以继续保留，它只负责把 NPC 的 `ToolParam` 转换成 `ToolUnionParam`。

## 3. 创建真正的 CLI 工具适配器

新增：

```text
internal/agent/cli_engine.go
```

内容：

```go
package agent

import (
	"context"
	"fmt"

	"cybertown/internal/agent/config"
	agentengine "cybertown/internal/agent/engine"
	"cybertown/internal/agent/mcpclient"
	"cybertown/internal/agent/subagent"
	"cybertown/internal/agent/tool"

	"github.com/anthropics/anthropic-sdk-go"
)

func newCLIEngine(
	mcpManager *mcpclient.Manager,
) (*agentengine.Engine, error) {
	cliDefinitions := tool.BuildTools()

	mcpDefinitions, err := mcpManager.BuildTools()
	if err != nil {
		return nil, fmt.Errorf(
			"build MCP tools: %w",
			err,
		)
	}

	cliDefinitions = append(
		cliDefinitions,
		mcpDefinitions...,
	)

	cliHandlers := make(
		map[string]agentengine.ToolHandler,
		len(tool.ToolHandlers)+
			len(mcpManager.ToolNames())+
			1,
	)

	// 适配现有本地 CLI 工具。
	for name, oldHandler := range tool.ToolHandlers {
		handler := oldHandler

		cliHandlers[name] = func(
			ctx context.Context,
			scope agentengine.Scope,
			input map[string]any,
		) (string, error) {
			if err := ctx.Err(); err != nil {
				return "", err
			}

			return handler(input)
		}
	}

	// task 原来是在 AgentLoop 中动态添加的。
	cliHandlers["task"] = func(
		ctx context.Context,
		scope agentengine.Scope,
		input map[string]any,
	) (string, error) {
		return subagent.SpawnSubagent(input, ctx)
	}

	// 适配 MCP 工具。
	for _, toolName := range mcpManager.ToolNames() {
		name := toolName

		cliHandlers[name] = func(
			ctx context.Context,
			scope agentengine.Scope,
			input map[string]any,
		) (string, error) {
			return mcpManager.Execute(
				ctx,
				name,
				input,
			)
		}
	}

	definitions := map[string][]anthropic.ToolUnionParam{
		"cli": cliDefinitions,
		"npc": agentengine.BuildTools(
			agentengine.NPCTools,
		),
	}

	handlers := map[string]map[string]agentengine.ToolHandler{
		"cli": cliHandlers,
		"npc": agentengine.NPCToolHandlers,
	}

	return agentengine.New(
		&agentengine.LLM{
			Client: &config.Client,
		},
		agentengine.Config{
			Model:       config.PRIMARY_MODEL,
			MaxTokens:   config.DEFAULT_MAX_TOOKENS,
			MaxRounds:   config.MAX_AGENT_ROUNDS,
			TurnTimeout: config.IDLE_TIMEOUT,
		},
		definitions,
		handlers,
	)
}
```

这里接入了：

- 现有本地 CLI 工具。
- 动态 `task` 工具。
- MCP 工具。
- Anthropic 服务端工具。

`scope` 对旧 CLI Handler 暂时没有用途，因此适配器忽略它。

## 4. 在 `App` 中创建共享 Engine

在 [app.go](D:/GoProjects/cyber_towm/internal/agent/app.go) 增加 import：

```go
agentengine "cybertown/internal/agent/engine"
```

增加字段：

```go
type App struct {
	session      *session.SessionManager
	promptCtx    *prompt.PromptContext
	cron         *cron.Manager
	mcp          *mcpclient.Manager
	reader       *readline.Instance
	embedder     *embedding.Embedder
	memoryStore  *memstore.VectorStore
	memoryWorker *memory.ExtractionWorker
	engine       *agentengine.Engine
}
```

在 `setupMCP()` 之后创建：

```go
agentEngine, err := newCLIEngine(mcpManager)
if err != nil {
	return nil, err
}
```

返回时加入：

```go
return &App{
	session:      sessionManager,
	promptCtx:    promptContext,
	cron:         cronManager,
	mcp:          mcpManager,
	reader:       r,
	embedder:     embedder,
	memoryStore:  store,
	memoryWorker: memoryWorker,
	engine:       agentEngine,
}, nil
```

这里叫 `agentEngine`，因为同一 Engine 已经注册了 `"cli"` 和 `"npc"` 两套权限。

## 5. 让现有 `handleQuery()` 直接调用 Engine

保留当前的 Hook、锁、记忆 Worker 和 Inbox 逻辑，只替换：

```go
err := loop.AgentLoop(...)
```

在 [repl.go](D:/GoProjects/cyber_towm/internal/agent/repl.go) 增加 import：

```go
agentengine "cybertown/internal/agent/engine"
"cybertown/internal/agent/memory"
"cybertown/internal/agent/prompt"
"time"
```

把旧的 `AgentLoop` 调用替换为：

```go
currentSession := a.session.CurrentSession()

systemPrompt := prompt.GetSystemPrompt(
	*a.promptCtx,
)

memoriesContent, memoryErr := memory.LoadMemories(
	ctx,
	a.embedder,
	a.memoryStore,
	query,
)
if memoryErr != nil {
	fmt.Printf(
		"[memory load error: %v]\n",
		memoryErr,
	)
} else if memoriesContent != "" {
	systemPrompt += "\n\nRelevant memories:\n" +
		memoriesContent
}

result, err := a.engine.RunTurn(
	ctx,
	agentengine.TurnRequest{
		Scope: agentengine.Scope{
			AgentKind: "cli",
			AgentID:   "coding-agent",
			SessionID: currentSession.ID,
			TurnID: fmt.Sprintf(
				"cli-%d",
				time.Now().UnixNano(),
			),
		},
		SystemPrompt: systemPrompt,
		Messages:     currentSession.Messages,
		Input:        query,
		ToolSet:      "cli",
	},
)
if err != nil {
	return err
}

currentSession.Messages = result.Messages

_ = hook.TriggerHooks(
	hook.Stop,
	&hook.HookContext{
		Messages: currentSession.Messages,
	},
)

fmt.Println(result.Text)
```

后面的代码继续保留：

```go
err = a.memoryWorker.Enqueue(
	currentSession.Messages,
)
```

以及 Inbox 处理。

语义记忆放进本轮 `SystemPrompt`，这样不会把注入内容写入 `currentSession.Messages`。

## 6. 暂时保留旧 `AgentLoop`

普通 CLI 请求已经走：

```text
handleQuery → Engine
```

但 [queue_processor.go](D:/GoProjects/cyber_towm/internal/agent/loop/queue_processor.go) 暂时仍使用旧 `AgentLoop` 处理 Cron。

这一步不要删除：

```text
internal/agent/loop/loop.go
```

因为恢复、压缩、后台工具和 Cron 还没有全部迁入新结构。等这些能力分别迁移后，再删除旧循环。

最终不需要：

```text
/engine 命令
空的 cli 工具集
两套 CLI 输入方式
```

正常 REPL 输入直接进入新 Engine，`ToolSet: "cli"` 使用真实 CLI 工具；未来 NPC 请求使用同一个 Engine，并指定 `ToolSet: "npc"`。