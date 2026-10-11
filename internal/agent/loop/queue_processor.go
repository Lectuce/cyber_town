package loop

import (
	"context"
	"cybertown/internal/agent/cron"
	"cybertown/internal/agent/embedding"
	"cybertown/internal/agent/mcpclient"
	"cybertown/internal/agent/memstore"
	"cybertown/internal/agent/prompt"
	"cybertown/internal/agent/session"
	"fmt"
	"sync"
	"time"
)

var AgentLock sync.Mutex

func QueueProcessorLoop(cronManager *cron.Manager,
	ctx context.Context,
	promptCtx *prompt.PromptContext,
	getCurrentSession func() *session.Session,
	mcpManager *mcpclient.Manager,
	embedder *embedding.Embedder,
	store *memstore.VectorStore) {
	ticker := time.NewTicker(200 * time.Millisecond)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():
			return

		case <-ticker.C:
			if !cronManager.HasCronQueue() {
				continue
			}

			if !AgentLock.TryLock() {
				continue
			}

			func() {
				defer AgentLock.Unlock()

				if !cronManager.HasCronQueue() {
					return
				}

				fmt.Println("\n\033[35m" +
					"[queue processor] delivering scheduled work" +
					"\033[0m",
				)

				currentSession := getCurrentSession()

				err := AgentLoop(
					"",
					ctx,
					promptCtx,
					currentSession,
					cronManager,
					mcpManager,
					embedder,
					store,
				)

				if err != nil {
					fmt.Printf(
						"[cron agent error] %v\n",
						err,
					)
				}
			}()
		}

	}

}
