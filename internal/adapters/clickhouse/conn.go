package clickhouse

import (
	"context"
	"fmt"

	goch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func Connect(ctx context.Context, addr, database, user, password string) (driver.Conn, error) {
	conn, err := goch.Open(&goch.Options{
		Addr: []string{addr},
		Auth: goch.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
		Settings: goch.Settings{
			"allow_experimental_annoy_index":             1,
			"allow_experimental_vector_similarity_index": 1,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}

	if err = conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}

	return conn, nil
}
