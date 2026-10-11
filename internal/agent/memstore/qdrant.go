package memstore

import (
	"context"
	"cybertown/internal/agent/config"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
)

func NewQdrantClient(host string, port int, collection string) (*VectorStore, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})

	if err != nil {
		return nil, err
	}

	return &VectorStore{
		Client:    client,
		Collecion: collection,
	}, nil
}

func (v *VectorStore) CreateCollection(ctx context.Context) error {

	ok, err := v.Client.CollectionExists(ctx, v.Collecion)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	err = v.Client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: v.Collecion,
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

func (v *VectorStore) UpsertMemory(collection string, id uint64, vector []float32, payload map[string]any, ctx context.Context) error {

	_, err := v.Client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collection,
		Wait:           qdrant.PtrOf(true),
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

func (v *VectorStore) QueryMemory(collection string, vector []float32, limit uint64, ctx context.Context) ([]MemoryHit, error) {
	searchResults, err := v.Client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collection,
		Query:          qdrant.NewQueryDense(vector),
		Limit:          qdrant.PtrOf(limit),
		WithPayload:    qdrant.NewWithPayload(true),
	})

	// threshold := float32(0.5)

	// searchResults, err := v.Client.Query(ctx, &qdrant.QueryPoints{
	// 	CollectionName: v.Collecion,
	// 	Query:          qdrant.NewQueryDense(vector),
	// 	Limit:          qdrant.PtrOf(uint64(5)),
	// 	ScoreThreshold: qdrant.PtrOf(threshold),
	// 	WithPayload:    qdrant.NewWithPayload(true),
	// })

	if err != nil {
		return nil, err
	}

	hits := make([]MemoryHit, 0, len(searchResults))

	for _, result := range searchResults {

		payload := qdrant.ValueMapToMap(result.Payload)

		hits = append(hits, MemoryHit{
			FileName:    stringValue(payload, "filename"),
			Name:        stringValue(payload, "name"),
			Type:        stringValue(payload, "type"),
			Description: stringValue(payload, "description"),
			Body:        stringValue(payload, "body"),
			UpdatedAt:   stringValue(payload, "updated_at"),
			Score:       result.Score,
		})

		fmt.Printf(
			"[memory vector hit] score=%.4f name=%q\n",
			result.Score,
			stringValue(payload, "name"),
		)

	}

	return hits, nil
}

func stringValue(payload map[string]any, key string) string {

	v, ok := payload[key].(string)
	if !ok || v == "" {
		return ""
	}
	return v
}
