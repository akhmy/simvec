package service

import (
	"context"

	"github.com/akhmy/vecrec-core/internal/domain"
)

type CollectionRepo interface {
	ExistsByName(ctx context.Context, collectionName string) (bool, error)
	Insert(ctx context.Context, collection *domain.Collection) error
	SelectByName(ctx context.Context, collectionName string) (*domain.Collection, error)
}

// Формат результата основного метода сервиса поиска похожих объектов
type SimilarCandidate struct {
	ID   string
	Data map[string]float32
	Dist float32
}

type RecordRepo interface {
	InsertBatch(ctx context.Context, collection *domain.Collection, batch []domain.Record) error
	FindByID(ctx context.Context, collection *domain.Collection, id string) (*domain.Record, error)
	FindSimilar(ctx context.Context, collection *domain.Collection, vector []float32, limit int) ([]SimilarCandidate, error)
}

type Embedder interface {
	Embed(ctx context.Context, strings []string, credentials map[string]any) (embeddings [][]float32, err error)
}

type EmbedderRegistry interface {
	Get(key domain.EmbedderType) (Embedder, error)
}
