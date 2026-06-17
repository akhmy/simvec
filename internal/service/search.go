package service

import (
	"context"
	"fmt"
	"math"
	"sort"
)

type SearchService interface {
	FindSimilar(ctx context.Context, input *FindSimilarInput) ([]SimilarResult, error)
}

type FindSimilarInput struct {
	CollectionName string
	RecordID       string
	Limit          int
	Weights        map[string]float32
}

type SimilarResult struct {
	ID    string
	Score float32
}

type searchService struct {
	records     RecordRepo
	collections CollectionRepo
}

//nolint:ireturn // DIP constructor
func NewSearchService(records RecordRepo, collections CollectionRepo) SearchService {
	return &searchService{records: records, collections: collections}
}

// Множитель кандидатов для ANN-поиска, умножается на лимит, заданный пользователем в запросе
const annCandidateMultiplier = 10

func (svc *searchService) FindSimilar(ctx context.Context, input *FindSimilarInput) ([]SimilarResult, error) {
	collection, err := svc.collections.SelectByName(ctx, input.CollectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	query, err := svc.records.FindByID(ctx, collection, input.RecordID)
	if err != nil {
		return nil, fmt.Errorf("failed to find query record: %w", err)
	}

	// 100 -- минимальное значение для кандидатов ANN-поиска
	annLimit := max(input.Limit*annCandidateMultiplier, 100)

	candidates, err := svc.records.FindSimilar(ctx, collection, query.Vector, annLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to find similar candidates: %w", err)
	}

	// Применение одинаковых весов, если они не заданы
	weights := input.Weights
	if len(weights) == 0 {
		weights = equalWeights(query.Data)
	}

	// Формирование результата и его сортировка
	results := make([]SimilarResult, 0, len(candidates))
	for _, c := range candidates {
		if c.ID == input.RecordID {
			continue
		}

		score := scoreCandidate(c, query.Data, weights)
		results = append(results, SimilarResult{ID: c.ID, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > input.Limit {
		results = results[:input.Limit]
	}

	return results, nil
}

func equalWeights(data map[string]float32) map[string]float32 {
	n := float32(1 + len(data)) // 1 для эмбеддинга + скаляры
	w := float32(1) / n

	weights := make(map[string]float32, len(data)+1)
	weights["vector"] = w

	for fieldName := range data {
		weights[fieldName] = w
	}

	return weights
}

// Функция, вычисляющая итоговую оценку схожести кандидата с запрашиваемым объектом
func scoreCandidate(c SimilarCandidate, queryData map[string]float32, weights map[string]float32) float32 {
	var score, totalWeight float32

	// Сначала к оценке прибавляется косинусное сходство (1 - косинусное расстояние),
	// умноженное на соответствующий вес, если он не равен 0
	if w, ok := weights["vector"]; ok {
		score += w * (1 - c.Dist)
		totalWeight += w
	}

	// Подсчёт оценки скалярных признаков
	for fieldName, queryVal := range queryData {
		// Извлечение веса признака
		w, ok := weights[fieldName]
		if !ok {
			continue
		}
		// Извлечение признака запрошенного объекта
		candidateVal, ok := c.Data[fieldName]
		if !ok {
			continue
		}

		// Оба признака нормализованы в [0,1], поэтому их разность не выйдет
		// за пределы этого диапазона, равно как и схожесть
		sim := float32(1) - float32(math.Abs(float64(queryVal-candidateVal)))
		score += w * sim
		totalWeight += w
	}

	// Если ни один признак не участвовал в измерении сходства, то возвращается 0
	if totalWeight == 0 {
		return 0
	}

	// Нормировка оценки на суммарный вес всех признаков
	return score / totalWeight
}
