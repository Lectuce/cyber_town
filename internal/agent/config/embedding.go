package config

type EmbedderConfig struct {
	Model     string
	BaseURL   string
	APIKey    string
	Dimension uint64
}

var EmbeddingConfig = EmbedderConfig{
	Model:     EMBEDDING_MODEL,
	BaseURL:   EMBEDDING_BASE_URL,
	APIKey:    API_KEY,
	Dimension: EMBEDDING_DIMEMSION,
}
