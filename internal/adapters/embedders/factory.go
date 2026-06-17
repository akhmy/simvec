package embedders

import (
	"fmt"

	"github.com/akhmy/vecrec-core/internal/domain"
	"github.com/akhmy/vecrec-core/internal/service"
)

type registry map[domain.EmbedderType]service.Embedder

//nolint:ireturn // DIP constructor
func NewRegistry(embedders map[domain.EmbedderType]service.Embedder) service.EmbedderRegistry {
	return registry(embedders)
}

//nolint:ireturn // DIP
func (r registry) Get(key domain.EmbedderType) (service.Embedder, error) {
	v, ok := r[key]
	if !ok {
		return nil, &EmbedderNotFoundError{EmbedderKey: string(key)}
	}

	return v, nil
}

type EmbedderNotFoundError struct {
	EmbedderKey string
}

func (e *EmbedderNotFoundError) Error() string {
	return fmt.Sprintf("embedder '%s' not found", e.EmbedderKey)
}
