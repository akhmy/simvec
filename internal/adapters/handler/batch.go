package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	apierrors "github.com/akhmy/vecrec-core/internal/adapters/handler/errors"
	"github.com/akhmy/vecrec-core/internal/config"
	"github.com/akhmy/vecrec-core/internal/lib/ctxtools"
	"github.com/akhmy/vecrec-core/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type postBatchRequest struct {
	RawRecords []rawRecordJSON `json:"records" validate:"required,min=1"`
}

type rawRecordJSON struct {
	ID     string         `json:"id" validate:"required"`
	Fields map[string]any `json:"fields" validate:"required"`
}

type batchHandler struct {
	validate    *validator.Validate
	cfg         *config.Config
	ingestBatch service.BatchService
}

func newBatchHandler(validate *validator.Validate, cfg *config.Config, ingestBatch service.BatchService) *batchHandler {
	return &batchHandler{validate: validate, cfg: cfg, ingestBatch: ingestBatch}
}

func (h *batchHandler) post(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	collectionName := chi.URLParam(r, "collectionName")
	collectionName = strings.ToLower(collectionName)

	logger, ok := ctxtools.Logger(r.Context())
	if !ok {
		log.Print("logger not in context")
		apierrors.WriteJSON(w, apierrors.InternalServerError)

		return
	}

	var req postBatchRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("failed to decode JSON", zap.Error(err))
		apierrors.WriteJSON(w, apierrors.ToAPIError(fmt.Errorf("%w: %w", apierrors.ErrDecodingJSON, err)))

		return
	}

	if err := h.validate.Struct(req); err != nil {
		logger.Warn("failed to validate JSON", zap.Error(err))
		apierrors.WriteJSON(w, apierrors.ToAPIError(fmt.Errorf("%w: %w", apierrors.ErrValidatingDTO, err)))

		return
	}

	// Проверка того, что размер пакета не превышает максимального значения
	if len(req.RawRecords) > h.cfg.MaxRecordsPerBatch {
		logger.Warn("batch too large",
			zap.Int("count", len(req.RawRecords)),
			zap.Int("limit", h.cfg.MaxRecordsPerBatch),
		)
		apierrors.WriteJSON(w, apierrors.ToAPIError(&apierrors.EntityBatchLimitExceeded{Limit: h.cfg.MaxRecordsPerBatch}))

		return
	}

	rawRecords := make([]service.RawRecordInput, 0, len(req.RawRecords))
	for _, r := range req.RawRecords {
		rawRecords = append(rawRecords, service.RawRecordInput{ID: r.ID, Fields: r.Fields})
	}

	if err := h.ingestBatch.Ingest(r.Context(), &service.IngestBatchInput{
		CollectionName: collectionName,
		RawRecords:     rawRecords,
	}); err != nil {
		logger.Error("failed to ingest batch", zap.Error(err))
		apierrors.WriteJSON(w, apierrors.ToAPIError(err))

		return
	}

	w.WriteHeader(http.StatusCreated)
}
