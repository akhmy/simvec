package test

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/akhmy/vecrec-core/internal/adapters/clickhouse"
	"github.com/akhmy/vecrec-core/internal/adapters/embedders"
	gigachatembedder "github.com/akhmy/vecrec-core/internal/adapters/embedders/gigachat"
	"github.com/akhmy/vecrec-core/internal/domain"
	"github.com/akhmy/vecrec-core/internal/service"
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	conn, err := clickhouse.Connect(ctx, "clickhouse:9000", "vecrec", "admin", "secret")
	if err != nil {
		log.Fatalf("setup: connect: %v", err)
	}

	_ = conn.Exec(ctx, "DROP TABLE IF EXISTS records_news")
	_ = conn.Exec(ctx, "DROP TABLE IF EXISTS collections")

	if err = clickhouse.Migrate(ctx, conn); err != nil {
		log.Fatalf("setup: migrate: %v", err)
	}

	collectionRepo := clickhouse.NewCollectionRepo(conn)
	recordRepo := clickhouse.NewRecordRepo(conn)

	collectionSvc := service.NewCollectionService(collectionRepo)
	if err = collectionSvc.Create(ctx, &service.CreateCollectionInput{
		Name:          "news",
		EmbedderType:  "gigachat/v1",
		EmbedderModel: "Embeddings",
		EmbedderCredentials: map[string]any{
			"embed_url":                "https://gigachat.devices.sberbank.ru/api/v1/embeddings",
			"refresh_access_token_url": "https://ngw.devices.sberbank.ru:9443/api/v2/oauth",
			"auth_token":               mustReadSecret("gigachat_auth_token"),
			"model":                    "Embeddings",
			"scope":                    "GIGACHAT_API_PERS",
		},
		SchemaFields: []domain.SchemaField{
			{Name: "title", Type: domain.FieldTypeString},
			{Name: "description", Type: domain.FieldTypeString},
			{Name: "word_count", Type: domain.FieldTypeNumber},
			{Name: "is_global", Type: domain.FieldTypeBool},
		},
		MinMaxRules: map[string]domain.MinMaxRange{
			"word_count": {Min: 5, Max: 200},
		},
	}); err != nil {
		log.Fatalf("setup: create collection: %v", err)
	}

	embedderFactory := embedders.NewRegistry(map[domain.EmbedderType]service.Embedder{
		domain.EmbedderTypeLocal: embedders.NewLocalEmbedder(
			http.DefaultClient,
			validator.New(),
		),
		domain.EmbedderTypeGigaChatV1: gigachatembedder.NewGigaChatV1Embedder(
			http.DefaultClient,
			validator.New(),
		),
	})

	batchSvc := service.NewBatchService(recordRepo, collectionRepo, embedderFactory)
	if err = batchSvc.Ingest(ctx, &service.IngestBatchInput{
		CollectionName: "news",
		RawRecords:     rawNews,
	}); err != nil {
		log.Fatalf("setup: ingest: %v", err)
	}

	os.Exit(m.Run())
}

func mustReadSecret(name string) string {
	b, err := os.ReadFile("/run/secrets/" + name)
	if err != nil {
		log.Fatalf("secret %q: %v", name, err)
	}
	return strings.TrimSpace(string(b))
}
