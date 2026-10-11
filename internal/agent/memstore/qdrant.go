package memstore

import (
	"context"
	"cybertown/internal/agent/config"

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

	_, err := v.Client.Upsert(context.Background(), &qdrant.UpsertPoints{
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

func (v *VectorStore) QueryMemory(collection string, vector []float32, limit uint64, ctx context.Context) ([]string, error) {
	searchResults, err := v.Client.Query(context.Background(), &qdrant.QueryPoints{
		CollectionName: collection,
		Query:          qdrant.NewQueryDense(vector),
		Limit:          qdrant.PtrOf(limit),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(searchResults))

	for _, result := range searchResults {

		payload := qdrant.ValueMapToMap(result.Payload)

		fileName, ok := payload["filename"].(string)
		if ok && fileName != "" {
			files = append(files, fileName)
		}

	}

	return files, nil
}
