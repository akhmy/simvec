package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/akhmy/vecrec-core/internal/domain"
)

type BatchService interface {
	Ingest(ctx context.Context, input *IngestBatchInput) error
}

type RawRecordInput struct {
	ID     string
	Fields map[string]any
}

type IngestBatchInput struct {
	CollectionName string
	RawRecords     []RawRecordInput
}

type batchService struct {
	records     RecordRepo
	collections CollectionRepo
	embedders   EmbedderRegistry
}

//nolint:ireturn // DIP constructor
func NewBatchService(records RecordRepo, collections CollectionRepo, embedders EmbedderRegistry) BatchService {
	return &batchService{records: records, collections: collections, embedders: embedders}
}

func (svc *batchService) Ingest(ctx context.Context, input *IngestBatchInput) error {
	collection, err := svc.collections.SelectByName(ctx, input.CollectionName)
	if err != nil {
		return fmt.Errorf("failed to execute select by name: %w", err)
	}

	embedder, err := svc.embedders.Get(collection.EmbedderType)
	if err != nil {
		return fmt.Errorf("failed to get embedder by key '%s': %w", collection.EmbedderType, err)
	}

	schema := buildSchemaIndex(collection)

	textStrings := make([]string, 0, len(input.RawRecords))
	scalarMaps := make([]map[string]float32, 0, len(input.RawRecords))

	// RawRecords - объекты в своём изначальном виде
	for _, rawRecord := range input.RawRecords {
		// Инициализация массива для строк и словаря для скалярных признаков
		texts := make([]string, 0)
		data := make(map[string]float32)

		// Обход всех полей
		for fieldName, fieldValue := range rawRecord.Fields {
			// Если поле с таким именем есть в схеме
			fieldMeta, ok := schema[fieldName]
			if !ok {
				return &FieldNotFoundError{FieldName: fieldName}
			}

			switch fieldMeta.fieldType {
			// Если поле - строка
			case domain.FieldTypeString:
				s, ok := fieldValue.(string)
				if !ok {
					return &WrongFieldTypeError{HasType: fmt.Sprintf("%T", fieldValue), WantType: "string"}
				}

				texts = append(texts, s)

			// Если поле - число
			case domain.FieldTypeNumber:
				// Если найден диапазон для числового признака
				rng, ok := collection.MinMaxRules[fieldName]
				if !ok {
					return &MinMaxRuleNotFoundError{FieldName: fieldName}
				}

				// Нормализация числа
				normalized, err := processNumber(fieldValue, rng)
				if err != nil {
					return fmt.Errorf("error processing '%s': %w", fieldName, err)
				}

				data[fieldName] = normalized

			case domain.FieldTypeBool:
				// Нормализация логического признака
				normalized, err := processBool(fieldValue)
				if err != nil {
					return fmt.Errorf("error processing '%s': %w", fieldName, err)
				}

				data[fieldName] = normalized
			}
		}

		// Добавление извлечённых данных в структуры
		joined := strings.Join(texts, " | ")
		if strings.TrimSpace(joined) == "" {
			return &EmptyRecordTextError{RecordID: rawRecord.ID}
		}
		textStrings = append(textStrings, joined)
		scalarMaps = append(scalarMaps, data)
	}

	embeddings, err := embedder.Embed(ctx, textStrings, collection.EmbedderCredentials)
	if err != nil {
		return fmt.Errorf("failed to embed batch: %w", err)
	}

	// Объединение аккумуляторов эмбеддингов и скаляров в массив объектов
	records := make([]domain.Record, 0, len(input.RawRecords))
	for i, rawRecord := range input.RawRecords {
		record := domain.NewRecord(rawRecord.ID, collection.Name, embeddings[i], scalarMaps[i])
		records = append(records, *record)
	}

	err = svc.records.InsertBatch(ctx, collection, records)
	if err != nil {
		return fmt.Errorf("failed to insert batch: %w", err)
	}

	return nil
}
