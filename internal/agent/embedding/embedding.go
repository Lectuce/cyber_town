package embedding

import (
	"bytes"
	"context"
	"cybertown/internal/agent/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	memoryInstruction = "Instruction:为智能体长期记忆检索生成语义向量\n.\nQuery:"
	queryInstruction  = "Target_modality: text.\n" +
		"Instruction:根据当前问题检索最相关的长期记忆\n" +
		"Query:"
)

func NewEmbedder(config config.EmbedderConfig, client *http.Client) (*Embedder, error) {

	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &Embedder{
		config: config,
		client: *client,
	}, nil
}

func (e *Embedder) EmbedMemory(text string, ctx context.Context) ([]float32, error) {
	return e.embed(text, memoryInstruction, ctx)
}

func (e *Embedder) EmbedQuery(text string, ctx context.Context) ([]float32, error) {
	return e.embed(text, queryInstruction, ctx)
}

func (e *Embedder) embed(text string, instruction string, ctx context.Context) ([]float32, error) {

	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("embedding text is empty.")
	}

	endpoint := strings.TrimRight(e.config.BaseURL, "/") + "/embeddings/multimodal"

	payload := embeddingRequest{
		Model: e.config.Model,
		Input: []embeddingInput{
			{
				Type: "text",
				Text: text,
			},
		},
		Dimensions:     e.config.Dimension,
		EncodingFormat: "float",
		Instructions:   instruction,
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(requestBody),
	)

	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+e.config.APIKey)

	response, err := e.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {

		var apiError apiErrorResponse
		err := json.Unmarshal(responseBody, &apiError)
		if err != nil {
			return nil, err
		}

		if apiError.Er.Message != "" {
			return nil, fmt.Errorf(
				"embedding request failed: status=%v code=%v type=%v message=%v",
				response.StatusCode,
				apiError.Er.Code,
				apiError.Er.Type,
				apiError.Er.Message,
			)
		}

		return nil, fmt.Errorf(
			"embedding request failed: status=%d body=%s",
			response.StatusCode,
			string(responseBody),
		)
	}

	var result embeddingResponse
	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		return nil, err
	}

	if len(result.Data.Embedding) == 0 {
		return nil, fmt.Errorf("returned an empty embedding")
	}

	if len(result.Data.Embedding) != int(e.config.Dimension) {
		return nil, fmt.Errorf("unexpected embedding dimension: got %v, want %v", len(result.Data.Embedding), e.config.Dimension)
	}

	return result.Data.Embedding, nil
}
