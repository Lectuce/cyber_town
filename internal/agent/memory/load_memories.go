package memory

import (
	"context"
	"cybertown/internal/agent/embedding"
	"cybertown/internal/agent/memstore"
	"fmt"
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

	hits, err := store.QueryMemory(store.Collecion, vector, 10, ctx)
	if err != nil {
		return "", err
	}

	// selectedFiles, err := selectRelevantMemories(messages, ctx, 5)
	// if err != nil {
	// 	return "", err
	// }

	rerankHits, err := rerankMemoryHits(query, hits, 5, ctx)
	if err != nil {
		fmt.Printf("[memory rerank error: %v]\n", err)
		if len(hits) > 5 {
			hits = hits[:5]
		}
	} else {
		hits = rerankHits
		for _, hit := range hits {
			fmt.Printf(
				"[memory selected] score=%.4f name=%q\n",
				hit.Score,
				hit.Name,
			)
		}
	}

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
			parts = append(parts, fmt.Sprintf(
				"<memory name=%q type=%q score=\"%.4f\">\n%s\n</memory>",
				hit.Name,
				hit.Type,
				hit.Score,
				content,
			))
		}

	}

	if len(parts) == 1 {
		return "", nil
	}

	parts = append(parts, "</relevant_memories>")

	return strings.Join(parts, "\n\n"), nil
}
