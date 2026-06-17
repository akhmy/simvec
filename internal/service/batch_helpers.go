package service

import (
	"fmt"

	"github.com/akhmy/vecrec-core/internal/domain"
)

type schemaFieldMeta struct {
	fieldType  domain.FieldType
	fieldIndex int
}

// Функция построения словаря-схемы коллекции для O(1)-доступа
func buildSchemaIndex(collection *domain.Collection) map[string]schemaFieldMeta {
	schema := make(map[string]schemaFieldMeta, len(collection.SchemaFields))

	for i, field := range collection.SchemaFields {
		schema[field.Name] = schemaFieldMeta{fieldType: field.Type, fieldIndex: i}
	}

	return schema
}

// Функция приведения внешнего bool ко внутреннему логическому признаку 
func processBool(value any) (float32, error) {
	v, ok := value.(bool)
	if !ok {
		return 0, &WrongFieldTypeError{HasType: fmt.Sprintf("%T", value), WantType: "bool"}
	}

	return domain.NormalizeBool(v), nil
}

// Функция приведения внешнего float32 ко внутреннему числовому признаку
func processNumber(value any, rng domain.MinMaxRange) (float32, error) {
	v, ok := value.(float64)
	if !ok {
		return 0, &WrongFieldTypeError{HasType: fmt.Sprintf("%T", value), WantType: "number"}
	}

	v32 := float32(v)
	if v32 < rng.Min || v32 > rng.Max {
		return 0, &OutOfMinMaxRangeError{FieldValue: v32, Min: rng.Min, Max: rng.Max}
	}

	return domain.NormalizeNumber(v32, rng), nil
}
