package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	apierrors "github.com/akhmy/vecrec-core/internal/adapters/handler/errors"
	"github.com/akhmy/vecrec-core/internal/lib/ctxtools"
	"github.com/akhmy/vecrec-core/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Лимит результатов по умолчанию
const defaultSimilarLimit = 10

type similarHandler struct {
	search service.SearchService
}

func newSimilarHandler(search service.SearchService) *similarHandler {
	return &similarHandler{search: search}
}

type similarResult struct {
	ID    string  `json:"id"`
	Score float32 `json:"score"`
}

type getSimilarResponse struct {
	Results []similarResult `json:"results"`
}

func (h *similarHandler) get(w http.ResponseWriter, r *http.Request) {
	collectionName := strings.ToLower(chi.URLParam(r, "collectionName"))

	logger, ok := ctxtools.Logger(r.Context())
	if !ok {
		log.Print("logger not in context")
		apierrors.WriteJSON(w, apierrors.InternalServerError)

		return
	}

	q := r.URL.Query()

	recordID := q.Get("similar-to")
	if recordID == "" {
		apierrors.WriteJSON(w, &apierrors.APIError{Message: "similar-to is required", Status: http.StatusBadRequest})

		return
	}

	limit := defaultSimilarLimit

	if raw := q.Get("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			logger.Warn("invalid limit param", zap.String("limit", raw))
			apierrors.WriteJSON(w, &apierrors.APIError{Message: "limit must be a positive integer", Status: http.StatusBadRequest})

			return
		}

		limit = v
	}

	weights, err := parseWeights(q)
	if err != nil {
		logger.Warn("invalid weight param", zap.Error(err))
		apierrors.WriteJSON(w, &apierrors.APIError{Message: err.Error(), Status: http.StatusBadRequest})

		return
	}

	results, err := h.search.FindSimilar(r.Context(), &service.FindSimilarInput{
		CollectionName: collectionName,
		RecordID:       recordID,
		Limit:          limit,
		Weights:        weights,
	})
	if err != nil {
		logger.Error("failed to find similar", zap.Error(err))
		apierrors.WriteJSON(w, apierrors.ToAPIError(err))

		return
	}

	out := make([]similarResult, len(results))
	for i, res := range results {
		out[i] = similarResult{ID: res.ID, Score: res.Score}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := encodeJSON(w, getSimilarResponse{Results: out}); err != nil {
		logger.Error("failed to encode response", zap.Error(err))
	}
}

func parseWeights(q url.Values) (map[string]float32, error) {
	weights := make(map[string]float32)

	for key, values := range q {
		if !strings.HasPrefix(key, "w[") || !strings.HasSuffix(key, "]") {
			continue
		}

		fieldName := key[2 : len(key)-1]

		v, err := strconv.ParseFloat(values[0], 32)
		if err != nil {
			return nil, fmt.Errorf("weight for field '%s' must be a number", fieldName)
		}

		weights[fieldName] = float32(v)
	}

	if len(weights) == 0 {
		return nil, nil
	}

	return weights, nil
}

func encodeJSON(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}
