package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// requestLogger logs each request as a structured zap entry, so access logs
// share the format and destination of the rest of the server's logs.
func requestLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info("Request",
				zap.String("request_id", middleware.GetReqID(r.Context())),
				zap.String("method", r.Method),
				zap.String("path", r.URL.RequestURI()),
				zap.Int("status", ww.Status()),
				zap.Int("bytes", ww.BytesWritten()),
				zap.Duration("duration", time.Since(start)),
				zap.String("remote", r.RemoteAddr),
				// Not trusted for anything; logged so requests behind a proxy
				// can still be traced to their client.
				zap.String("forwarded_for", r.Header.Get("X-Forwarded-For")),
			)
		})
	}
}
