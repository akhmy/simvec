package test

import (
	"context"
	"testing"

	"github.com/akhmy/vecrec-core/internal/adapters/clickhouse"
	"github.com/akhmy/vecrec-core/internal/service"
)

func articleByID(id string) map[string]any {
	for _, a := range rawNews {
		if a.ID == id {
			return a.Fields
		}
	}
	return nil
}

func avgWordCount(results []service.SimilarResult) float64 {
	var sum float64
	for _, r := range results {
		if f := articleByID(r.ID); f != nil {
			sum += f["word_count"].(float64)
		}
	}
	return sum / float64(len(results))
}

func isGlobalCount(results []service.SimilarResult) int {
	n := 0
	for _, r := range results {
		if f := articleByID(r.ID); f != nil && f["is_global"].(bool) {
			n++
		}
	}
	return n
}

func TestFindSimilarWeights(t *testing.T) {
	conn, err := clickhouse.Connect(context.Background(), "clickhouse:9000", "vecrec", "admin", "secret")
	if err != nil {
		t.Fatal(err)
	}

	svc := service.NewSearchService(
		clickhouse.NewRecordRepo(conn),
		clickhouse.NewCollectionRepo(conn),
	)

	const queryID = "449"
	const queryWordCount = 59.0
	const K = 5

	find := func(weights map[string]float32) []service.SimilarResult {
		t.Helper()
		results, err := svc.FindSimilar(context.Background(), &service.FindSimilarInput{
			CollectionName: "news",
			RecordID:       queryID,
			Limit:          K,
			Weights:        weights,
		})
		if err != nil {
			t.Fatalf("FindSimilar: %v", err)
		}
		return results
	}

	logResults := func(label string, results []service.SimilarResult) {
		t.Logf("%s:", label)
		for _, r := range results {
			f := articleByID(r.ID)
			t.Logf("  id=%-8s cluster=%-10s wc=%-4.0f is_global=%v  score=%.4f",
				r.ID, rawNewsCluster[r.ID], f["word_count"], f["is_global"], r.Score)
		}
	}

	vectorResults := find(map[string]float32{"vector": 1.0})
	logResults("vector only", vectorResults)
	t.Logf("  avg_word_count=%.1f  is_global=%d/%d\n",
		avgWordCount(vectorResults), isGlobalCount(vectorResults), K)

	wcResults := find(map[string]float32{"word_count": 1.0})
	logResults("word_count only", wcResults)
	t.Logf("  avg_word_count=%.1f\n", avgWordCount(wcResults))

	globalResults := find(map[string]float32{"is_global": 1.0})
	logResults("is_global only", globalResults)
	t.Logf("  is_global=%d/%d\n", isGlobalCount(globalResults), K)

	queryCluster := rawNewsCluster[queryID]
	clusterHits := 0
	for _, r := range vectorResults {
		if rawNewsCluster[r.ID] == queryCluster {
			clusterHits++
		}
	}
	if clusterHits < 3 {
		t.Errorf("vector-only: expected >= 3 of top-%d from cluster %q, got %d",
			K, queryCluster, clusterHits)
	}

	const wcTolerance = 25.0
	avg := avgWordCount(wcResults)
	if avg < queryWordCount-wcTolerance || avg > queryWordCount+wcTolerance {
		t.Errorf("word_count-only: avg_word_count=%.1f outside ±%.0f of query %.0f",
			avg, wcTolerance, queryWordCount)
	}

	if isGlobalCount(globalResults) > 0 {
		t.Errorf("is_global-only: expected all %d results to be is_global=false, got %d is_global=true",
			K, isGlobalCount(globalResults))
	}
}
