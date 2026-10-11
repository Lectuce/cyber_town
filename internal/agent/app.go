package agent

import (
	"context"
	"cybertown/internal/agent/config"
	"cybertown/internal/agent/cron"
	"cybertown/internal/agent/embedding"
	"cybertown/internal/agent/hook"
	"cybertown/internal/agent/loop"
	"cybertown/internal/agent/mcpclient"
	"cybertown/internal/agent/memstore"
	"cybertown/internal/agent/prompt"
	"cybertown/internal/agent/session"
	"cybertown/internal/agent/skills"
	"cybertown/internal/agent/teams"
	"cybertown/internal/agent/tool"

	"fmt"

	"github.com/chzyer/readline"
)

type App struct {
	session     *session.SessionManager
	promptCtx   *prompt.PromptContext
	cron        *cron.Manager
	mcp         *mcpclient.Manager
	reader      *readline.Instance
	embedder    *embedding.Embedder
	memoryStore *memstore.VectorStore
}

func newApp(ctx context.Context) (*App, error) {
	err := skills.ScanSkill()
	if err != nil {
		return nil, err
	}

	hook.Register()

	sessionManager := session.NewSessionManager()
	promptContext, err := prompt.UpdateContext(prompt.PromptContext{}, nil)
	if err != nil {
		return nil, err
	}

	embedder, err := embedding.NewEmbedder(config.EmbeddingConfig, nil)
	if err != nil {
		return nil, err
	}
	store, err := memstore.NewQdrantClient(config.HOST, config.QDRANT_PORT, config.QDRANT_COLLECTION_NAME)
	if err != nil {
		return nil, err
	}
	err = store.CreateCollection(ctx)
	if err != nil {
		return nil, err
	}

	bus, err := teams.NewMessageBus()
	if err != nil {
		return nil, err
	}
	teams.SetMessageBus(bus)
	tool.SetMessageBus(bus)

	cronManager := cron.NewManager(config.DURABLE_PATH)
	err = cronManager.LoadDurableJobs()
	if err != nil {
		return nil, err
	}
	tool.SetCronManager(cronManager)

	mcpManager, err := setupMCP(ctx)
	if err != nil {
		return nil, err
	}

	r, err := readline.New("agent[default] >> ")
	if err != nil {
		return nil, err
	}

	return &App{
		session:     sessionManager,
		promptCtx:   promptContext,
		cron:        cronManager,
		mcp:         mcpManager,
		reader:      r,
		embedder:    embedder,
		memoryStore: store,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	fmt.Println("输入问题，回车发送。输入 q 退出。")

	go a.cron.SchedulerLoop(ctx)

	go loop.QueueProcessorLoop(
		a.cron,
		ctx,
		a.promptCtx,
		a.session.CurrentSession,
		a.mcp,
		a.embedder,
		a.memoryStore,
	)

	return a.runREPL(ctx)
}

func (a *App) Close() {

	if a.reader != nil {
		_ = a.reader.Close()
	}

	if a.mcp != nil {
		err := a.mcp.Close()
		if err != nil {
			fmt.Printf("[MCP close error] %v\n", err)
		}
	}

}
