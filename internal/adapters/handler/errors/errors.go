package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/akhmy/vecrec-core/internal/domain"
	"github.com/akhmy/vecrec-core/internal/service"
	"github.com/go-playground/validator/v10"
)

var ErrDecodingJSON = errors.New("error decoding json")
var ErrValidatingDTO = errors.New("error validating dto")

type EntityBatchLimitExceeded struct {
	Limit int
}

func (e *EntityBatchLimitExceeded) Error() string {
	return fmt.Sprintf("entity batch limit exceeded (limit: %d)", e.Limit)
}

type APIError struct {
	Message string
	Status  int
}

var InternalServerError = &APIError{Message: "Internal server error", Status: http.StatusInternalServerError}
var BadRequest = &APIError{Message: "Bad request", Status: http.StatusBadRequest}

func WriteJSON(w http.ResponseWriter, e *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Status)

	body, _ := json.Marshal(map[string]string{"error": e.Message})
	_, _ = w.Write(body)
}

// Функция, преобразующая ошибки приложения в ошибки API
func ToAPIError(e error) *APIError {
	// adapters-layer errors
	var entityBatchLimitExceeded *EntityBatchLimitExceeded

	// 404
	var collectionNotFound *service.CollectionNotFoundError
	var recordNotFound *service.RecordNotFoundError

	// 409
	var collectionAlreadyExists *service.CollectionAlreadyExistsError

	// 400 — domain errors
	var collectionCreation *domain.CollectionCreationError
	var invalidSchemaFieldType *domain.InvalidSchemaFieldTypeError
	var minMaxRuleUnknownField *domain.MinMaxRuleUnknownFieldError
	var minMaxRuleNonNumeric *domain.MinMaxRuleNonNumericFieldError
	var minMaxRuleMissing *domain.MinMaxRuleMissingError
	var invalidEmbedderType *domain.InvalidEmbedderTypeError
	var noStringFields *domain.NoStringFieldsError

	// 400 — service errors
	var fieldNotFound *service.FieldNotFoundError
	var wrongFieldType *service.WrongFieldTypeError
	var outOfMinMaxRange *service.OutOfMinMaxRangeError
	var minMaxRuleNotFound *service.MinMaxRuleNotFoundError
	var emptyRecordText *service.EmptyRecordTextError

	switch {
	case errors.Is(e, ErrDecodingJSON):
		return decodingErrorToAPIError(e)
	case errors.Is(e, ErrValidatingDTO):
		return validationErrorToAPIError(e)
	case errors.As(e, &entityBatchLimitExceeded):
		return &APIError{Message: entityBatchLimitExceeded.Error(), Status: http.StatusBadRequest}

	case errors.As(e, &collectionNotFound):
		return &APIError{Message: collectionNotFound.Error(), Status: http.StatusNotFound}
	case errors.As(e, &recordNotFound):
		return &APIError{Message: recordNotFound.Error(), Status: http.StatusNotFound}

	case errors.As(e, &collectionAlreadyExists):
		return &APIError{Message: collectionAlreadyExists.Error(), Status: http.StatusConflict}

	case errors.As(e, &collectionCreation):
		return &APIError{Message: collectionCreation.Error(), Status: http.StatusBadRequest}
	case errors.As(e, &invalidSchemaFieldType):
		return &APIError{Message: invalidSchemaFieldType.Error(), Status: http.StatusBadRequest}
	case errors.As(e, &minMaxRuleUnknownField):
		return &APIError{Message: minMaxRuleUnknownField.Error(), Status: http.StatusBadRequest}
	case errors.As(e, &minMaxRuleNonNumeric):
		return &APIError{Message: minMaxRuleNonNumeric.Error(), Status: http.StatusBadRequest}
	case errors.As(e, &minMaxRuleMissing):
		return &APIError{Message: minMaxRuleMissing.Error(), Status: http.StatusBadRequest}
	case errors.As(e, &invalidEmbedderType):
		return &APIError{Message: invalidEmbedderType.Error(), Status: http.StatusBadRequest}
	case errors.As(e, &noStringFields):
		return &APIError{Message: noStringFields.Error(), Status: http.StatusBadRequest}

	case errors.As(e, &fieldNotFound):
		return &APIError{Message: fieldNotFound.Error(), Status: http.StatusBadRequest}
	case errors.As(e, &wrongFieldType):
		return &APIError{Message: wrongFieldType.Error(), Status: http.StatusBadRequest}
	case errors.As(e, &outOfMinMaxRange):
		return &APIError{Message: outOfMinMaxRange.Error(), Status: http.StatusBadRequest}
	case errors.As(e, &minMaxRuleNotFound):
		return &APIError{Message: minMaxRuleNotFound.Error(), Status: http.StatusBadRequest}
	case errors.As(e, &emptyRecordText):
		return &APIError{Message: emptyRecordText.Error(), Status: http.StatusBadRequest}

	default:
		return InternalServerError
	}
}

func decodingErrorToAPIError(e error) *APIError {
	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError

	switch {
	case errors.As(e, &syntaxError):
		return &APIError{
			Message: fmt.Sprintf("Syntax error at %d: %s", syntaxError.Offset, syntaxError.Error()),
			Status:  http.StatusBadRequest,
		}
	case errors.As(e, &unmarshalTypeError):
		return &APIError{
			Message: fmt.Sprintf(
				"Field '%s' has wrong type: got '%s', want '%s'",
				unmarshalTypeError.Field, unmarshalTypeError.Value, unmarshalTypeError.Type,
			),
			Status: http.StatusBadRequest,
		}
	default:
		return BadRequest
	}
}

func validationErrorToAPIError(e error) *APIError {
	var validationErrors validator.ValidationErrors

	if errors.As(e, &validationErrors) {
		messages := make([]string, len(validationErrors))
		for i, ve := range validationErrors {
			messages[i] = fmt.Sprintf("Field '%s' failed '%s' validation", ve.Field(), ve.Tag())
		}

		return &APIError{Message: strings.Join(messages, " & "), Status: http.StatusBadRequest}
	}

	return BadRequest
}
