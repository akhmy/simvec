package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	apierrors "github.com/akhmy/vecrec-core/internal/adapters/handler/errors"
	"github.com/akhmy/vecrec-core/internal/domain"
	"github.com/akhmy/vecrec-core/internal/lib/ctxtools"
	"github.com/akhmy/vecrec-core/internal/service"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type collectionHandler struct {
	validate    *validator.Validate
	collections service.CollectionService
}

func newCollectionHandler(validate *validator.Validate, collections service.CollectionService) *collectionHandler {
	return &collectionHandler{validate: validate, collections: collections}
}

type postCollectionRequest struct {
	Name                string                     `json:"name" validate:"required"`
	EmbedderType        string                     `json:"embedderType" validate:"required"`
	EmbedderModel       string                     `json:"embedderModel" validate:"required"`
	EmbedderCredentials map[string]any             `json:"embedderCredentials"`
	MinMaxRules         map[string]minMaxRangeJSON `json:"minMaxRules"`
	SchemaFields        []schemaFieldJSON          `json:"schema" validate:"required"`
}

type schemaFieldJSON struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type minMaxRangeJSON struct {
	Min float32 `json:"min"`
	Max float32 `json:"max"`
}

func (h *collectionHandler) post(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	logger, ok := ctxtools.Logger(r.Context())
	if !ok {
		log.Print("logger not in context")
		apierrors.WriteJSON(w, apierrors.InternalServerError)

		return
	}

	var req postCollectionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("decode error", zap.Error(err))
		apierrors.WriteJSON(w, apierrors.ToAPIError(fmt.Errorf("%w: %w", apierrors.ErrDecodingJSON, err)))

		return
	}

	// Имя коллекции автоматически приводится к нижнему регистру
	req.Name = strings.ToLower(req.Name)

	if err := h.validate.Struct(req); err != nil {
		logger.Warn("validation error", zap.Error(err))
		apierrors.WriteJSON(w, apierrors.ToAPIError(fmt.Errorf("%w: %w", apierrors.ErrValidatingDTO, err)))

		return
	}

	schemaFields := make([]domain.SchemaField, 0, len(req.SchemaFields))
	for _, f := range req.SchemaFields {
		schemaFields = append(schemaFields, domain.SchemaField{Name: f.Name, Type: domain.FieldType(f.Type)})
	}

	minMaxRules := make(map[string]domain.MinMaxRange, len(req.MinMaxRules))
	for k, v := range req.MinMaxRules {
		minMaxRules[k] = domain.MinMaxRange{Min: v.Min, Max: v.Max}
	}

	err := h.collections.Create(r.Context(), &service.CreateCollectionInput{
		Name:                req.Name,
		EmbedderType:        req.EmbedderType,
		EmbedderModel:       req.EmbedderModel,
		EmbedderCredentials: req.EmbedderCredentials,
		SchemaFields:        schemaFields,
		MinMaxRules:         minMaxRules,
	})
	if err != nil {
		logger.Error("create collection error", zap.Error(err))
		apierrors.WriteJSON(w, apierrors.ToAPIError(err))

		return
	}

	w.WriteHeader(http.StatusCreated)
}
