package domain

type FieldType string

// Всевозможные типы признаков, обрабатываемые модулем
const (
	FieldTypeNumber FieldType = "number"
	FieldTypeBool   FieldType = "bool"
	FieldTypeString FieldType = "string"
)

func (fieldType FieldType) IsValid() bool {
	switch fieldType {
	case FieldTypeNumber, FieldTypeBool, FieldTypeString:
		return true
	}

	return false
}
