package memory

import (
	"context"
	"cybertown/internal/agent/embedding"
	"cybertown/internal/agent/memstore"
	"strings"
)

func LoadMemories(
	ctx context.Context,
	embedder *embedding.Embedder,
	store *memstore.VectorStore,
	query string,
) (string, error) {

	vector, err := embedder.EmbedQuery(query, ctx)
	if err != nil {
		return "", err
	}

	hits, err := store.QueryMemory(store.Collecion, vector, 5, ctx)
	if err != nil {
		return "", err
	}

	// selectedFiles, err := selectRelevantMemories(messages, ctx, 5)
	// if err != nil {
	// 	return "", err
	// }

	if len(hits) == 0 {
		return "", nil
	}

	parts := []string{"<relevant_memories>"}

	for _, hit := range hits {
		content := strings.TrimSpace(hit.Body)

		if content == "" && hit.FileName != "" {
			content, err = readMemoryFile(hit.FileName)
			if err != nil {
				continue
			}
			content = strings.TrimSpace(content)
		}

		if content != "" {
			parts = append(parts, content)
		}

	}

	parts = append(parts, "</relevant_memories>")

	return strings.Join(parts, "\n\n"), nil
}
