package embedding

import (
	"cybertown/internal/agent/config"
	"net/http"
)

type Embedder struct {
	config config.EmbedderConfig
	client http.Client
}

type embeddingInput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type embeddingRequest struct {
	Model          string           `json:"model"`
	Instructions   string           `json:"insructions"`
	Input          []embeddingInput `json:"input"`
	Dimensions     uint64           `json:"demensions"`
	EncodingFormat string           `json:"encoding_format"`
}

type embeddingResponseData struct {
	Embedding []float32 `json:"embedding"`
}

type sparseElement struct {
	Index uint32  `json:"index"`
	Value float32 `json:"value"`
}

type usage struct {
	PromptTokens        int64   `json:"prompt_tokens"`
	PromptTokensDetails details `json:"prompr_tokens_details"`
	TotalTokens         int64   `json:"total_tokens"`
}

type details struct {
	ImageTokens int64 `json:"image_tokens"`
	TextTokens  int64 `json:"text_tokens"`
}

type embeddingResponse struct {
	Created int64                 `json:"created"`
	Model   string                `json:"model"`
	Data    embeddingResponseData `json:"data"`
	ID      string                `json:"id"`
	Object  string                `json:"object"`
	Usage   usage                 `json:"usage"`
}

type Error struct {
	Code    any    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

type apiErrorResponse struct {
	Er Error `json:"error"`
}
