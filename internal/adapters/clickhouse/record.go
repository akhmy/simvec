package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/akhmy/vecrec-core/internal/domain"
	"github.com/akhmy/vecrec-core/internal/service"
)

type recordRepo struct {
	conn driver.Conn
}

func NewRecordRepo(conn driver.Conn) service.RecordRepo {
	return &recordRepo{conn: conn}
}

func recordsTableName(collectionName string) string {
	return fmt.Sprintf("records_%s", collectionName)
}

// Извелечение скалярных полей из схемы
func scalarFields(collection *domain.Collection) []domain.SchemaField {
	var fields []domain.SchemaField
	for _, f := range collection.SchemaFields {
		if f.Type == domain.FieldTypeNumber || f.Type == domain.FieldTypeBool {
			fields = append(fields, f)
		}
	}
	return fields
}

func (r *recordRepo) InsertBatch(ctx context.Context, collection *domain.Collection, batch []domain.Record) error {
	scalars := scalarFields(collection)

	cols := make([]string, 0, 2+len(scalars)) // 2 под id и vector, остальные под скаляры
	cols = append(cols, "id", "vector")
	for _, f := range scalars {
		cols = append(cols, fmt.Sprintf("`%s`", f.Name))
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s)",
		recordsTableName(collection.Name),
		strings.Join(cols, ", "),
	)

	// Пакетная вставка в ClickHouse
	chBatch, err := r.conn.PrepareBatch(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	for _, record := range batch {
		row := make([]any, 0, 2+len(scalars))
		row = append(row, record.ID, record.Vector)
		for _, f := range scalars {
			row = append(row, record.Data[f.Name])
		}

		if err := chBatch.Append(row...); err != nil {
			return fmt.Errorf("failed to append record: %w", err)
		}
	}

	if err := chBatch.Send(); err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}

	return nil
}

func (r *recordRepo) FindByID(ctx context.Context, collection *domain.Collection, id string) (*domain.Record, error) {
	scalars := scalarFields(collection)

	cols := make([]string, 0, 2+len(scalars))
	cols = append(cols, "id", "vector")
	for _, f := range scalars {
		cols = append(cols, fmt.Sprintf("`%s`", f.Name))
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE id = ? LIMIT 1",
		strings.Join(cols, ", "),
		recordsTableName(collection.Name),
	)

	rec := domain.Record{
		CollectionName: collection.Name,
		Data:           make(map[string]float32),
	}

	scalarVals := make([]float32, len(scalars))
	dest := make([]any, 0, 2+len(scalars))
	dest = append(dest, &rec.ID, &rec.Vector)
	for i := range scalars {
		dest = append(dest, &scalarVals[i])
	}

	if err := r.conn.QueryRow(ctx, query, id).Scan(dest...); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, sql.ErrNoRows) {
			return nil, &service.RecordNotFoundError{RecordID: id}
		}

		return nil, fmt.Errorf("failed to query record: %w", err)
	}

	for i, f := range scalars {
		rec.Data[f.Name] = scalarVals[i]
	}

	return &rec, nil
}

func (r *recordRepo) FindSimilar(ctx context.Context, collection *domain.Collection, vector []float32, limit int) ([]service.SimilarCandidate, error) {
	scalars := scalarFields(collection)

	cols := make([]string, 0, 2+len(scalars))
	cols = append(cols, "id")
	for _, f := range scalars {
		cols = append(cols, fmt.Sprintf("`%s`", f.Name))
	}
	cols = append(cols, "cosineDistance(vector, ?) AS dist")

	query := fmt.Sprintf(
		// Синтаксис запроса, использующего HNSW-индекс
		"SELECT %s FROM %s ORDER BY dist ASC LIMIT ?",
		strings.Join(cols, ", "),
		recordsTableName(collection.Name),
	)

	rows, err := r.conn.Query(ctx, query, vector, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query similar records: %w", err)
	}
	defer rows.Close()

	candidates := make([]service.SimilarCandidate, 0, limit)

	for rows.Next() {
		c := service.SimilarCandidate{
			Data: make(map[string]float32),
		}

		scalarVals := make([]float32, len(scalars))
		var dist float64

		dest := make([]any, 0, 2+len(scalars))
		dest = append(dest, &c.ID)
		for i := range scalars {
			dest = append(dest, &scalarVals[i])
		}
		dest = append(dest, &dist)

		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		for i, f := range scalars {
			c.Data[f.Name] = scalarVals[i]
		}
		c.Dist = float32(dist)

		candidates = append(candidates, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return candidates, nil
}
