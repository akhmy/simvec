package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/akhmy/vecrec-core/internal/adapters/clickhouse"
	"github.com/akhmy/vecrec-core/internal/adapters/embedders"
	"github.com/akhmy/vecrec-core/internal/adapters/embedders/gigachat"
	"github.com/akhmy/vecrec-core/internal/adapters/handler"
	"github.com/akhmy/vecrec-core/internal/config"
	"github.com/akhmy/vecrec-core/internal/domain"

	"github.com/akhmy/vecrec-core/internal/lib/zaplogger"
	"github.com/akhmy/vecrec-core/internal/service"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	devMode := flag.Bool("dev", false, "load .env file")

	flag.Parse()

	// Инициализация логгера (по умолчанию в JSON, с dev-флагом строками)
	logger, err := handleDevMode(*devMode)
	if err != nil {
		log.Fatalf("failed to handle dev mode: %s", err.Error())
	}

	cfg, err := config.Build()
	if err != nil {
		logger.Fatal("failed to build config", zap.Error(err))
	}

	validate := validator.New()

	// : CLICKHOUSE CONNECTION & MIGRATIONS
	chConn, err := clickhouse.Connect(context.Background(),
		cfg.CHAddr, cfg.CHDatabase, cfg.CHUser, cfg.CHPassword,
	)
	if err != nil {
		logger.Fatal("failed to connect to clickhouse", zap.Error(err))
	}

	err = clickhouse.Migrate(context.Background(), chConn)
	if err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}

	// : CLICKHOUSE REPOS
	collectionRepo, recordRepo := buildRepos(chConn)

	// : SERVICES
	embedderRegistry := embedders.NewRegistry(map[domain.EmbedderType]service.Embedder{
		domain.EmbedderTypeLocal: embedders.NewLocalEmbedder(
			&http.Client{
				Timeout: 10 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 10,
					IdleConnTimeout:     90 * time.Second,
					TLSHandshakeTimeout: 5 * time.Second,
				},
			}, validate),
		domain.EmbedderTypeGigaChatV1: gigachat.NewGigaChatV1Embedder(
			&http.Client{
				Timeout: 10 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 10,
					IdleConnTimeout:     90 * time.Second,
					TLSHandshakeTimeout: 5 * time.Second,
				},
			}, validate),
	})

	collectionSvc := service.NewCollectionService(collectionRepo)
	batchSvc := service.NewBatchService(recordRepo, collectionRepo, embedderRegistry)
	searchSvc := service.NewSearchService(recordRepo, collectionRepo)

	// : SERVER
	server := handler.NewServer(
		handler.NewRouter(
			logger,
			validate,
			cfg,
			collectionSvc,
			batchSvc,
			searchSvc,
		),
		cfg.ServerAddr,
		cfg.ReadHeaderTimeout,
		cfg.ReadTimeout,
		cfg.WriteTimeout,
		cfg.IdleTimeout,
	)

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.ListenAndServe()
	}()

	logger.Info("server started", zap.String("addr", server.Addr))

	// : GRACEFUL SHUTDOWN
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err = <-serverErr:
		logger.Fatal("server error", zap.Error(err))
	case <-shutdown:
		logger.Info("shutting down")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeout)*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		logger.Fatal("failed to shut down gracefully", zap.Error(err))
	}

	logger.Info("shutdown complete")
}

func handleDevMode(devMode bool) (*zap.Logger, error) {
	if devMode {
		logger, err := zaplogger.NewDev()
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}

		err = godotenv.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load .env: %w", err)
		}

		return logger, nil
	}

	logger, err := zaplogger.NewProd()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return logger, nil
}

//nolint:ireturn // dependency inversion
func buildRepos(conn driver.Conn) (service.CollectionRepo, service.RecordRepo) {
	return clickhouse.NewCollectionRepo(conn), clickhouse.NewRecordRepo(conn)
}
