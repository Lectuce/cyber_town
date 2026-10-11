package qdrant

import (
	"context"
	"cybertown/internal/agent/config"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
)

type QdrantClient struct {
	client *qdrant.Client
}

func NewQdrantClient() (*qdrant.Client, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: "localhost",
		Port: 6334,
	})

	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *QdrantClient) CreateCollection(ctx context.Context) error {
	err := c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: config.QDRANT_COLLECTION_NAME,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     4,
			Distance: qdrant.Distance_Cosine,
		}),
	})

	if err != nil {
		return err
	}
	return nil
}

func (c *QdrantClient) Upsert(collectionName string, ctx context.Context) error {

	operationInfo, err := c.client.Upsert(context.Background(), &qdrant.UpsertPoints{
		CollectionName: config.QDRANT_COLLECTION_NAME,
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDNum(1),
				Vectors: qdrant.NewVectors(0.05, 0.61, 0.76, 0.74),
				Payload: qdrant.NewValueMap(map[string]any{"city": "Berlin"}),
			},
			{
				Id:      qdrant.NewIDNum(2),
				Vectors: qdrant.NewVectors(0.19, 0.81, 0.75, 0.11),
				Payload: qdrant.NewValueMap(map[string]any{"city": "London"}),
			},
			{
				Id:      qdrant.NewIDNum(3),
				Vectors: qdrant.NewVectors(0.36, 0.55, 0.47, 0.94),
				Payload: qdrant.NewValueMap(map[string]any{"city": "Moscow"}),
			},
		},
	})

	if err != nil {
		return err
	}

	fmt.Println(operationInfo)
	return nil
}

func (c *QdrantClient) Query() error {
	searchResult, err := c.client.Query(context.Background(), &qdrant.QueryPoints{
		CollectionName: config.QDRANT_COLLECTION_NAME,
		Query:          qdrant.NewQuery(0.2, 0.1, 0.9, 0.7),
	})
	if err != nil {
		return err
	}

	fmt.Println(searchResult)

	return nil
}
