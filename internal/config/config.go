package config

import (
	"fmt"

	cenv "github.com/caarlos0/env/v11"
)

type Config struct {
	ServerAddr         string `env:"SERVER_ADDR" envDefault:":8080"`
	MaxRecordsPerBatch int    `env:"MAX_RECORDS_PER_BATCH" envDefault:"1000"`
	ReadHeaderTimeout  int    `env:"READ_HEADER_TIMEOUT_S" envDefault:"5"`
	ReadTimeout        int    `env:"READ_TIMEOUT_S" envDefault:"10"`
	WriteTimeout       int    `env:"WRITE_TIMEOUT_S" envDefault:"10"`
	IdleTimeout        int    `env:"IDLE_TIMEOUT_S" envDefault:"60"`
	ShutdownTimeout    int    `env:"SHUTDOWN_TIMEOUT_S" envDefault:"30"`

	CHAddr        string `env:"CLICKHOUSE_ADDR,required"`
	CHDatabase    string `env:"CLICKHOUSE_DATABASE,required"`
	CHUser        string `env:"CLICKHOUSE_USER,required"`
	CHPassword    string `env:"CLICKHOUSE_PASSWORD,required"`
	CHInitTimeout int    `env:"CLICKHOUSE_INIT_TIMEOUT" envDefault:"60"`
}

func Build() (*Config, error) {
	var cfg Config

	err := cenv.Parse(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}
