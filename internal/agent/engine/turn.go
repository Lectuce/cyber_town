package engine

import "github.com/anthropics/anthropic-sdk-go"

// 一次agent执行的边界
type Scope struct {
	WorldID   string
	AgentKind string
	AgentID   string
	PlayerID  string
	SessionID string
	TurnID    string
}

// 一轮agent需要的输入
type TurnRequest struct {
	Scope        Scope
	SystemPrompt string
	Messages     []anthropic.MessageParam
	Input        string
	ToolSet      string
	OnText       func(string) error `json:"-"`
}

// 这一轮agent执行完成后的返回的结果
type TurnResult struct {
	Text       string
	Messages   []anthropic.MessageParam
	StopReason string
}
