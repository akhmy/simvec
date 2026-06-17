package test

import (
	"context"
	"math"
	"sort"
	"testing"

	"github.com/akhmy/vecrec-core/internal/adapters/clickhouse"
	"github.com/akhmy/vecrec-core/internal/service"
)

func TestPrecision(t *testing.T) {
	ctx := context.Background()

	conn, err := clickhouse.Connect(ctx, "clickhouse:9000", "vecrec", "admin", "secret")
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewSearchService(
		clickhouse.NewRecordRepo(conn),
		clickhouse.NewCollectionRepo(conn),
	)

	clusterSizes := map[string]int{}
	for _, c := range rawNewsCluster {
		clusterSizes[c]++
	}

	Ks := []int{1, 5, 10}
	maxK := Ks[len(Ks)-1]

	type clusterStats struct {
		prec, ndcg, recall map[int]float64
		mrr                float64
		n                  int
	}
	accum := map[string]*clusterStats{}
	for c := range clusterSizes {
		accum[c] = &clusterStats{
			prec:   map[int]float64{},
			ndcg:   map[int]float64{},
			recall: map[int]float64{},
		}
	}

	for _, article := range rawNews {
		cluster := rawNewsCluster[article.ID]
		relevant := clusterSizes[cluster] - 1

		results, err := svc.FindSimilar(ctx, &service.FindSimilarInput{
			CollectionName: "news",
			RecordID:       article.ID,
			Limit:          maxK,
			Weights:        map[string]float32{"vector": 1.0, "word_count": 0.5},
		})
		if err != nil {
			t.Fatalf("FindSimilar(%s): %v", article.ID, err)
		}

		rels := make([]int, len(results))
		for i, r := range results {
			if rawNewsCluster[r.ID] == cluster {
				rels[i] = 1
			}
		}

		st := accum[cluster]
		st.n++
		for i, rel := range rels {
			if rel == 1 {
				st.mrr += 1.0 / float64(i+1)
				break
			}
		}
		for _, k := range Ks {
			lim := min(k, len(rels))
			hits := 0
			for _, rel := range rels[:lim] {
				hits += rel
			}
			st.prec[k] += float64(hits) / float64(k)
			st.ndcg[k] += binaryNDCG(rels, k, relevant)
			if relevant > 0 {
				st.recall[k] += float64(hits) / float64(relevant)
			}
		}
	}

	clusters := make([]string, 0, len(accum))
	for c := range accum {
		clusters = append(clusters, c)
	}
	sort.Strings(clusters)

	totalN := 0
	totalMRR := 0.0
	totPrec := map[int]float64{}
	totNDCG := map[int]float64{}
	totRecall := map[int]float64{}

	for _, c := range clusters {
		st := accum[c]
		n := st.n
		totalN += n
		totalMRR += st.mrr
		for _, k := range Ks {
			totPrec[k] += st.prec[k]
			totNDCG[k] += st.ndcg[k]
			totRecall[k] += st.recall[k]
		}
		t.Logf("%-12s  MRR=%.3f  P@5=%.3f  NDCG@5=%.3f  (n=%d)",
			c, st.mrr/float64(n), st.prec[5]/float64(n), st.ndcg[5]/float64(n), n)
	}
	t.Logf("overall       MRR=%.3f  P@5=%.3f  NDCG@5=%.3f  (n=%d)",
		totalMRR/float64(totalN),
		totPrec[5]/float64(totalN),
		totNDCG[5]/float64(totalN),
		totalN,
	)

	p5 := totPrec[5] / float64(totalN)
	mrr := totalMRR / float64(totalN)
	if p5 < 0.60 {
		t.Errorf("Precision@5 = %.4f, expected >= 0.60", p5)
	}
	if mrr < 0.70 {
		t.Errorf("MRR = %.4f, expected >= 0.70", mrr)
	}
}

func binaryNDCG(rels []int, k, idealCount int) float64 {
	dcg := 0.0
	for i, rel := range rels[:min(k, len(rels))] {
		if rel == 1 {
			dcg += 1.0 / math.Log2(float64(i+2))
		}
	}
	ideal := 0.0
	for i := range min(idealCount, k) {
		ideal += 1.0 / math.Log2(float64(i+2))
	}
	if ideal == 0 {
		return 0
	}
	return dcg / ideal
}
