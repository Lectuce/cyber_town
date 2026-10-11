package memstore

import "github.com/qdrant/go-client/qdrant"

type VectorStore struct {
	Client    *qdrant.Client
	Collecion string
}

type MemoryHit struct {
	FileName    string
	Name        string
	Type        string
	Description string
	Body        string
	UpdatedAt   string
	Score       float32
}
