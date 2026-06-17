package domain

import "fmt"

type EmbedderType string

// Всевозможные провайдеры эмбеддингов
const (
	EmbedderTypeLocal      EmbedderType = "local"
	EmbedderTypeGigaChatV1 EmbedderType = "gigachat/v1"
)

func (key EmbedderType) IsValid() bool {
	switch key {
	case EmbedderTypeLocal, EmbedderTypeGigaChatV1:
		return true
	}

	return false
}

type MinMaxRange struct {
	Min float32 `json:"min"`
	Max float32 `json:"max"`
}

type SchemaField struct {
	Name string    `json:"name"`
	Type FieldType `json:"type"`
}

type Collection struct {
	Name                string
	EmbedderType        EmbedderType
	EmbedderFullType    string
	EmbedderCredentials map[string]any
	MinMaxRules         map[string]MinMaxRange
	SchemaFields        []SchemaField
}

func NewCollection(
	name string,
	embedderType string,
	embedderModel string,
	embedderCredentials map[string]any,
	schemaFields []SchemaField,
	minMaxRules map[string]MinMaxRange,
) (*Collection, error) {
	if name == "" {
		return nil, &CollectionCreationError{Reason: "collection name is empty"}
	}

	// Приведение строки к типу строковых констант EmbedderType
	typedEmbedderType := EmbedderType(embedderType)
	if !typedEmbedderType.IsValid() {
		return nil, &InvalidEmbedderTypeError{EmbedderKey: embedderType}
	}

	// Ошибка, если в схеме нет строковых полей
	hasStringField := false
	for _, f := range schemaFields {
		if f.Type == FieldTypeString {
			hasStringField = true
			break
		}
	}
	if !hasStringField {
		return nil, &NoStringFieldsError{}
	}

	fieldNameMap := make(map[string]FieldType)

	for _, schemaField := range schemaFields {
		if !schemaField.Type.IsValid() {
			return nil, &InvalidSchemaFieldTypeError{
				CollectionName: name,
				FieldName:      schemaField.Name,
				FieldType:      string(schemaField.Type),
			}
		}

		fieldNameMap[schemaField.Name] = schemaField.Type
	}

	// Проверка того, что диапазоны заданы для существующих и числовых значений
	for fieldName := range minMaxRules {
		fieldType, ok := fieldNameMap[fieldName]
		if !ok {
			return nil, &MinMaxRuleUnknownFieldError{FieldName: fieldName}
		}

		if fieldType != FieldTypeNumber {
			return nil, &MinMaxRuleNonNumericFieldError{FieldName: fieldName, FieldType: string(fieldType)}
		}
	}

	// Проверка того, что для всех числовых признаков заданы диапазоны
	for fieldName, fieldType := range fieldNameMap {
		if fieldType == FieldTypeNumber {
			if _, ok := minMaxRules[fieldName]; !ok {
				return nil, &MinMaxRuleMissingError{FieldName: fieldName}
			}
		}
	}

	return &Collection{
		Name:                name,
		EmbedderType:        typedEmbedderType,
		EmbedderFullType:    fmt.Sprintf("%s/%s", embedderType, embedderModel),
		EmbedderCredentials: embedderCredentials,
		SchemaFields:        schemaFields,
		MinMaxRules:         minMaxRules,
	}, nil
}
