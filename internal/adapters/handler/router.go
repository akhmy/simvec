package handler

import (
	"github.com/akhmy/vecrec-core/internal/config"
	"github.com/akhmy/vecrec-core/internal/lib/middleware"
	"github.com/akhmy/vecrec-core/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// Функция, определяющая правила маршрутизации сервера модуля, также подключает middleware
func NewRouter(
	logger *zap.Logger,
	validate *validator.Validate,
	cfg *config.Config,
	collectionSvc service.CollectionService,
	batchSvc service.BatchService,
	searchSvc service.SearchService,
) *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.RequestLogger(logger)) // Подключение логгера запросов
	router.Use(chimw.Heartbeat("/ping")) // Подключение healthcheck-эндпоинта
	router.Use(chimw.Recoverer) // Подключение восстановления после паник

	router.Mount("/debug", chimw.Profiler()) // Подключение профилировщика pprof

	collectionHandler := newCollectionHandler(validate, collectionSvc)
	batchHandler := newBatchHandler(validate, cfg, batchSvc)
	similarHandler := newSimilarHandler(searchSvc)

	router.Route("/collections", func(r chi.Router) {
		r.Post("/", collectionHandler.post)
		r.Post("/{collectionName}/records", batchHandler.post)
		r.Get("/{collectionName}/records", similarHandler.get)
	})

	return router
}
