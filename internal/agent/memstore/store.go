package memstore

import "github.com/qdrant/go-client/qdrant"

type VectorStore struct {
	Client    *qdrant.Client
	Collecion string
}
