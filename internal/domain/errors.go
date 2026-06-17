package domain

import "fmt"

type CollectionCreationError struct {
	Reason string
}

func (e *CollectionCreationError) Error() string {
	return fmt.Sprintf("collection creation error: %s", e.Reason)
}

type InvalidSchemaFieldTypeError struct {
	CollectionName string
	FieldName      string
	FieldType      string
}

func (e *InvalidSchemaFieldTypeError) Error() string {
	return fmt.Sprintf(
		"collection '%s': field '%s' has invalid type '%s'",
		e.CollectionName, e.FieldName, e.FieldType,
	)
}

type MinMaxRuleUnknownFieldError struct {
	FieldName string
}

func (e *MinMaxRuleUnknownFieldError) Error() string {
	return fmt.Sprintf("min-max rule references unknown field '%s'", e.FieldName)
}

type MinMaxRuleNonNumericFieldError struct {
	FieldName string
	FieldType string
}

func (e *MinMaxRuleNonNumericFieldError) Error() string {
	return fmt.Sprintf("min-max rule for field '%s' requires type 'number', got '%s'", e.FieldName, e.FieldType)
}

type InvalidEmbedderTypeError struct {
	EmbedderKey string
}

func (e *InvalidEmbedderTypeError) Error() string {
	return fmt.Sprintf("unknown embedder key: '%s'", e.EmbedderKey)
}

type MinMaxRuleMissingError struct {
	FieldName string
}

func (e *MinMaxRuleMissingError) Error() string {
	return fmt.Sprintf("min-max rule missing for number field '%s'", e.FieldName)
}

type NoStringFieldsError struct{}

func (e *NoStringFieldsError) Error() string {
	return "collection schema must have at least one string field"
}
