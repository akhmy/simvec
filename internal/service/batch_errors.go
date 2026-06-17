package service

import "fmt"

type CollectionNotFoundError struct {
	CollectionName string
}

func (e *CollectionNotFoundError) Error() string {
	return fmt.Sprintf("collection '%s' not found", e.CollectionName)
}

type FieldNotFoundError struct {
	FieldName string
}

func (e *FieldNotFoundError) Error() string {
	return fmt.Sprintf("field '%s' not found in schema", e.FieldName)
}

type WrongFieldTypeError struct {
	HasType  string
	WantType string
}

func (e *WrongFieldTypeError) Error() string {
	return fmt.Sprintf("wrong field type: have '%s', want '%s'", e.HasType, e.WantType)
}

type MinMaxRuleNotFoundError struct {
	FieldName string
}

func (e *MinMaxRuleNotFoundError) Error() string {
	return fmt.Sprintf("min-max rule for field '%s' not found", e.FieldName)
}

type OutOfMinMaxRangeError struct {
	FieldValue float32
	Min        float32
	Max        float32
}

func (e *OutOfMinMaxRangeError) Error() string {
	return fmt.Sprintf("value %g is out of min-max range [%g, %g]", e.FieldValue, e.Min, e.Max)
}

type RecordNotFoundError struct {
	RecordID string
}

func (e *RecordNotFoundError) Error() string {
	return fmt.Sprintf("record '%s' not found", e.RecordID)
}

type EmptyRecordTextError struct {
	RecordID string
}

func (e *EmptyRecordTextError) Error() string {
	return fmt.Sprintf("record '%s': all text fields are empty", e.RecordID)
}
