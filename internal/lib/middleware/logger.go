package middleware

import (
	"net/http"
	"time"

	"github.com/akhmy/vecrec-core/internal/lib/ctxtools"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func RequestLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqLogger := logger.With(
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remoteAddr", r.RemoteAddr),
			)
			r = r.WithContext(ctxtools.WithLogger(r.Context(), reqLogger))

			start := time.Now()

			//nolint:varnamelen
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			logger.Info("request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.Duration("duration", time.Since(start)),
				zap.String("remoteAddr", r.RemoteAddr),
			)
		})
	}
}
