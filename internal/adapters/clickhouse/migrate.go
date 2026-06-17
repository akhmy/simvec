package clickhouse

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Упаковка папки с миграциями в итоговый бинарный файл и чтение из него
//go:embed migrations/*.sql
var migrations embed.FS

func Migrate(ctx context.Context, conn driver.Conn) error {
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	for _, file := range files {
		sql, err := migrations.ReadFile("migrations/" + file.Name())
		if err != nil {
			return fmt.Errorf("reading migration '%s': %w", file.Name(), err)
		}

		for _, stmt := range splitStatements(string(sql)) {
			if err = conn.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("executing migration '%s': %w", file.Name(), err)
			}
		}
	}

	return nil
}

func splitStatements(sql string) []string {
	parts := strings.Split(sql, ";")
	stmts := make([]string, 0, len(parts))

	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			stmts = append(stmts, s)
		}
	}

	return stmts
}
