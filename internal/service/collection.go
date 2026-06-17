package service

import (
	"context"
	"fmt"

	"github.com/akhmy/vecrec-core/internal/domain"
)

type CollectionAlreadyExistsError struct {
	CollectionName string
}

func (e *CollectionAlreadyExistsError) Error() string {
	return fmt.Sprintf("collection '%s' already exists", e.CollectionName)
}

type CollectionService interface {
	Create(ctx context.Context, input *CreateCollectionInput) error
}

type CreateCollectionInput struct {
	Name                string
	EmbedderType        string
	EmbedderModel       string
	EmbedderCredentials map[string]any
	SchemaFields        []domain.SchemaField
	MinMaxRules         map[string]domain.MinMaxRange
}

type collectionService struct {
	collections CollectionRepo
}

//nolint:ireturn // dependency inversion
func NewCollectionService(collections CollectionRepo) CollectionService {
	return &collectionService{collections: collections}
}

func (s *collectionService) Create(ctx context.Context, input *CreateCollectionInput) error {
	collection, err := domain.NewCollection(
		input.Name,
		input.EmbedderType,
		input.EmbedderModel,
		input.EmbedderCredentials,
		input.SchemaFields,
		input.MinMaxRules,
	)
	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	alreadyExists, err := s.collections.ExistsByName(ctx, input.Name)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if alreadyExists {
		return &CollectionAlreadyExistsError{CollectionName: input.Name}
	}

	err = s.collections.Insert(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to insert collection: %w", err)
	}

	return nil
}
