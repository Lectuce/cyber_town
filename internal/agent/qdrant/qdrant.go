package qdrant

import (
	"context"
	"cybertown/internal/agent/config"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
)

type QdrantClient struct {
	client    *qdrant.Client
	collecion string
}

func NewQdrantClient(host string, port int, collection string) (*QdrantClient, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})

	if err != nil {
		return nil, err
	}

	return &QdrantClient{
		client:    client,
		collecion: collection,
	}, nil
}

func (c *QdrantClient) CreateCollection(ctx context.Context) error {

	ok, err := c.client.CollectionExists(ctx, c.collecion)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	err = c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: c.collecion,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     config.EMBEDDING_DIMEMSION,
			Distance: qdrant.Distance_Cosine,
		}),
	})

	if err != nil {
		return err
	}
	return nil
}

func (c *QdrantClient) UpsertMemory(collection string, id uint64, vector []float32, payload map[string]any, ctx context.Context) error {

	_, err := c.client.Upsert(context.Background(), &qdrant.UpsertPoints{
		CollectionName: collection,
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDNum(id),
				Vectors: qdrant.NewVectorsDense(vector),
				Payload: qdrant.NewValueMap(payload),
			},
		},
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *QdrantClient) QueryMemory(collection string, vector []float32, ctx context.Context) ([]*qdrant.ScoredPoint, error) {
	searchResult, err := c.client.Query(context.Background(), &qdrant.QueryPoints{
		CollectionName: collection,
		Query:          qdrant.NewQueryDense(vector),
	})
	if err != nil {
		return nil, err
	}

	fmt.Println(searchResult)

	return searchResult, nil
}
