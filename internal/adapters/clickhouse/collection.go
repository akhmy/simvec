package clickhouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/akhmy/vecrec-core/internal/domain"
	"github.com/akhmy/vecrec-core/internal/service"
)

type collectionRepo struct {
	conn driver.Conn
}

//nolint:ireturn // DIP constructor
func NewCollectionRepo(conn driver.Conn) service.CollectionRepo {
	return &collectionRepo{conn: conn}
}

func (r *collectionRepo) Insert(ctx context.Context, collection *domain.Collection) error {
	// Упаковка метаданных в JSON
	schema, err := json.Marshal(collection.SchemaFields)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	minMaxRules, err := json.Marshal(collection.MinMaxRules)
	if err != nil {
		return fmt.Errorf("failed to marshal min max rules: %w", err)
	}

	credentials, err := json.Marshal(collection.EmbedderCredentials)
	if err != nil {
		return fmt.Errorf("failed to marshal embedder credentials: %w", err)
	}

	err = r.conn.Exec(ctx, `
		INSERT INTO collections (name, embedder_type, embedder_full_type, embedder_credentials, schema, min_max_rules)
		VALUES (?, ?, ?, ?, ?, ?)`,
		collection.Name,
		string(collection.EmbedderType),
		collection.EmbedderFullType,
		string(credentials),
		string(schema),
		string(minMaxRules),
	)
	if err != nil {
		return fmt.Errorf("failed to execute sql: %w", err)
	}

	if err := r.createRecordsTable(ctx, collection); err != nil {
		return fmt.Errorf("failed to create records table: %w", err)
	}

	return nil
}

func (r *collectionRepo) createRecordsTable(ctx context.Context, collection *domain.Collection) error {
	// Динамически формируем скалярные колонки на основе схемы коллекции.
	// Поля типа string всегда конкатенируются в один эмбеддинг, поэтому тип для них всегда Array(Float32).
	var scalarCols strings.Builder
	for _, field := range collection.SchemaFields {
		if field.Type == domain.FieldTypeNumber || field.Type == domain.FieldTypeBool {
			scalarCols.WriteString(fmt.Sprintf("    `%s` Float32,\n", field.Name))
		}
	}

	ddl := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS records_%s (
    	id     String,
    	vector Array(Float32),
		%s
		INDEX vector_ann_idx vector
			TYPE vector_similarity('hnsw', 'cosineDistance') GRANULARITY 1
	) ENGINE = MergeTree()
	ORDER BY id
	`,
		collection.Name,
		scalarCols.String(),
	)

	if err := r.conn.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("failed to exec ddl: %w", err)
	}

	return nil
}

func (r *collectionRepo) ExistsByName(ctx context.Context, collectionName string) (bool, error) {
	var count uint64

	err := r.conn.QueryRow(ctx, `SELECT count() FROM collections WHERE name = ?`, collectionName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("execute query: %w", err)
	}

	return count > 0, nil
}

func (r *collectionRepo) SelectByName(ctx context.Context, collectionName string) (*domain.Collection, error) {
	var (
		name, embedderType, embedderFullType, credentialsJSON string
		schemaJSON, minMaxRulesJSON                           string
	)

	err := r.conn.QueryRow(ctx, `
		SELECT name, embedder_type, embedder_full_type, embedder_credentials, schema, min_max_rules
		FROM collections
		WHERE name = ?
	`, collectionName).Scan(&name, &embedderType, &embedderFullType, &credentialsJSON, &schemaJSON, &minMaxRulesJSON)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, &service.CollectionNotFoundError{CollectionName: collectionName}
		}

		return nil, fmt.Errorf("failed to execute select query: %w", err)
	}

	// Распаковка метаданных из JSON
	var credentials map[string]any
	if err = json.Unmarshal([]byte(credentialsJSON), &credentials); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedder credentials: %w", err)
	}

	var schemaFields []domain.SchemaField
	if err = json.Unmarshal([]byte(schemaJSON), &schemaFields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	var minMaxRules map[string]domain.MinMaxRange
	if err = json.Unmarshal([]byte(minMaxRulesJSON), &minMaxRules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal min max rules: %w", err)
	}

	return &domain.Collection{
		Name:                name,
		EmbedderType:        domain.EmbedderType(embedderType),
		EmbedderFullType:    embedderFullType,
		EmbedderCredentials: credentials,
		SchemaFields:        schemaFields,
		MinMaxRules:         minMaxRules,
	}, nil
}
