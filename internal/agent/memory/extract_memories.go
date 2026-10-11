package memory

import (
	"context"
	"cybertown/internal/agent/config"
	"cybertown/internal/agent/embedding"
	"cybertown/internal/agent/memstore"
	"cybertown/internal/agent/recovery"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

func ExtractMemories(messages []anthropic.MessageParam, ctx context.Context, embedder *embedding.Embedder, store *memstore.VectorStore) error {

	start := len(messages) - 10
	if start < 0 {
		start = 0
	}

	var dialogueParts []string

	for _, msg := range messages[start:] {
		text := strings.TrimSpace(messageText(msg))

		if text == "" {
			continue
		}

		dialogueParts = append(
			dialogueParts,
			fmt.Sprintf(
				"%v: %v",
				msg.Role,
				text,
			),
		)
	}

	dialogue := strings.Join(dialogueParts, "\n")

	if strings.TrimSpace(dialogue) == "" {
		return nil
	}

	existing, err := listMemoryFiles()
	if err != nil {
		return err
	}

	existingDesc := "(none)"

	if len(existing) > 0 {
		var lines []string

		for _, mem := range existing {
			lines = append(
				lines,
				fmt.Sprintf(
					"- %s: %s",
					mem.Name,
					mem.Description,
				),
			)
		}

		existingDesc = strings.Join(lines, "\n")
	}

	if len(dialogue) > 4000 {
		dialogue = dialogue[:4000]
	}

	prompt := "Extract user preferences, constraints, or project facts from this dialogue.\n" +
		"Return a JSON array. Each item: {name, type, description, body}.\n" +
		"- name: short kebab-case identifier (e.g. 'user-preference-tabs')\n" +
		"- type: one of 'user' (user preference), 'feedback' (guidance), " +
		"'project' (project fact), 'reference' (external pointer)\n" +
		"- description: one-line summary for index lookup\n" +
		"- body: full detail in markdown\n" +
		"If nothing new or already covered by existing memories, return [].\n\n" +
		"Existing memories:\n" + existingDesc + "\n\n" +
		"Dialogue:\n" + dialogue

	state := recovery.InitRecoveryState()

	response, err := recovery.WithRetry(
		func() (*anthropic.Message, error) {
			return config.Client.Messages.New(
				ctx,
				anthropic.MessageNewParams{
					Model:     state.CurrentModel,
					MaxTokens: 800,
					Messages: []anthropic.MessageParam{
						anthropic.NewUserMessage(
							anthropic.NewTextBlock(prompt),
						),
					},
				},
			)
		},
		state,
		10,
	)

	if err != nil {
		return nil
	}

	text := extractResponseText(response)

	re := regexp.MustCompile(`(?s)\[.*\]`)
	jsonText := re.FindString(text)

	if jsonText == "" {
		return nil
	}

	var items []ExtractedMemory

	if err := json.Unmarshal(
		[]byte(jsonText),
		&items,
	); err != nil {
		return nil
	}

	count := 0

	for _, mem := range items {
		name := mem.Name

		if name == "" {
			name = fmt.Sprintf(
				"memory_%d",
				time.Now().Unix(),
			)
		}

		memType := mem.Type
		if memType == "" {
			memType = "user"
		}

		if strings.TrimSpace(mem.Description) == "" ||
			strings.TrimSpace(mem.Body) == "" {
			continue
		}

		path, err := writeMemoryFile(
			name,
			memType,
			mem.Description,
			mem.Body,
		)

		if err != nil {
			continue
		}

		count++

		embeddingText := fmt.Sprintf("名称：%s\n类型：%s\n摘要：%s\n内容：%s",
			mem.Name,
			mem.Type,
			mem.Description,
			mem.Body,
		)

		vector, err := embedder.EmbedMemory(embeddingText, ctx)
		if err != nil {
			continue
		}

		fileName := filepath.Base(path)

		store.UpsertMemory(store.Collecion, memoryID(fileName), vector,
			map[string]any{
				"filename":    fileName,
				"name":        mem.Name,
				"type":        memType,
				"description": mem.Description,
			}, ctx)

	}

	if count > 0 {
		fmt.Printf("\n\033[33m[Memory: extracted %d new memories]\033[0m\n", count)
	}

	return nil
}

// 生成memoryID，用于向量数据库
func memoryID(fileName string) uint64 {

	h := fnv.New64a()
	_, _ = h.Write([]byte(fileName))
	id := h.Sum64()
	if id == 0 {
		return 1
	}
	return id

}
