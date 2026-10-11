package memory

import (
	"context"
	"cybertown/internal/agent/embedding"
	"cybertown/internal/agent/memstore"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

var ErrExtractionWorkerClosed = errors.New(
	"memory extraction worker is closed",
)

// 提取协程
type ExtractionWorker struct {
	ctx      context.Context
	embedder *embedding.Embedder
	store    *memstore.VectorStore
	jobs     chan ExtractionJob

	mu        sync.RWMutex
	closed    bool
	closeOnce sync.Once
	wg        sync.WaitGroup
}

type ExtractionJob struct {
	Messages []anthropic.MessageParam
}

func NewExtractionWorker(
	ctx context.Context,
	embedder *embedding.Embedder,
	store *memstore.VectorStore,
	queueSize int,
) (*ExtractionWorker, error) {

	if queueSize <= 0 {
		queueSize = 8
	}

	worker := &ExtractionWorker{
		ctx:      ctx,
		embedder: embedder,
		store:    store,
		jobs:     make(chan ExtractionJob, queueSize),
	}

	worker.wg.Add(1)
	go worker.run()

	return worker, nil

}

func (w *ExtractionWorker) Enqueue(messages []anthropic.MessageParam) error {

	start := len(messages) - 10
	if start < 0 {
		start = 0
	}

	snapshot, err := CloneMessages(messages[start:])
	if err != nil {
		return fmt.Errorf("clone messages for extraction: %w", err)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.closed {
		return ErrExtractionWorkerClosed
	}

	select {
	case w.jobs <- ExtractionJob{
		Messages: snapshot}:

		return nil

	case <-w.ctx.Done():
		return w.ctx.Err()
	}
}

func (w *ExtractionWorker) run() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():

			return

		case job, ok := <-w.jobs:
			if !ok {

				return
			}

			w.process(job)
		}
	}
}

func (w *ExtractionWorker) process(job ExtractionJob) {
	startedAt := time.Now()

	jobCtx, cancel := context.WithTimeout(
		w.ctx,
		2*time.Minute,
	)
	defer cancel()

	err := ExtractMemories(
		job.Messages,
		jobCtx,
		w.embedder,
		w.store,
	)
	if err != nil {
		fmt.Printf("\n[memory extraction error: %v]\n", err)
		return
	}

	fmt.Printf("\n[memory extraction finished in %v]\n", time.Since(startedAt).Round(time.Millisecond))
}

func (w *ExtractionWorker) Close() {
	w.closeOnce.Do(func() {
		w.mu.Lock()
		w.closed = true
		close(w.jobs)
		w.mu.Unlock()

		// 正常退出时等待已入队任务完成。
		w.wg.Wait()
	})
}
