package memory

import (
	"context"
	"cybertown/internal/agent/embedding"
	"cybertown/internal/agent/memstore"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

func LoadMemories(messages []anthropic.MessageParam,
	ctx context.Context,
	embedder *embedding.Embedder,
	store *memstore.VectorStore,
	query string,
) (string, error) {

	vector, err := embedder.EmbedQuery(query, ctx)
	if err != nil {
		return "", err
	}

	selectedFiles, err := store.QueryMemory(store.Collecion, vector, 5, ctx)
	if err != nil {
		return "", err
	}

	// selectedFiles, err := selectRelevantMemories(messages, ctx, 5)
	// if err != nil {
	// 	return "", err
	// }

	if len(selectedFiles) == 0 {
		return "", nil
	}

	parts := []string{"<relevant_memories>"}

	for _, filename := range selectedFiles {
		content, err := readMemoryFile(filename.FileName)
		if err != nil {
			continue
		}

		if strings.TrimSpace(content) != "" {
			parts = append(parts, content)
		}
	}

	parts = append(parts, "</relevant_memories>")

	return strings.Join(parts, "\n\n"), nil
}
