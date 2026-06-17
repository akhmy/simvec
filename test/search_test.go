package test

import (
	"context"
	"testing"

	"github.com/akhmy/vecrec-core/internal/adapters/clickhouse"
	"github.com/akhmy/vecrec-core/internal/service"
)

func TestFindSimilar(t *testing.T) {
	conn, err := clickhouse.Connect(context.Background(), "clickhouse:9000", "vecrec", "admin", "secret")
	if err != nil {
		t.Fatal(err)
	}

	svc := service.NewSearchService(
		clickhouse.NewRecordRepo(conn),
		clickhouse.NewCollectionRepo(conn),
	)

	const queryID = "794" // "Reds Knock Padres Out of Wild-Card Lead"

	results, err := svc.FindSimilar(context.Background(), &service.FindSimilarInput{
		CollectionName: "news",
		RecordID:       queryID,
		Limit:          5,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("similar to %q [%s]:", articleByID(queryID)["title"], rawNewsCluster[queryID])
	for _, r := range results {
		t.Logf("  id=%-8s cluster=%-10s score=%.4f", r.ID, rawNewsCluster[r.ID], r.Score)
	}

	if len(results) == 0 {
		t.Error("expected results, got empty list")
	}
}
